package posts

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jeremyjsx/entries/internal/events"
	"github.com/jeremyjsx/entries/internal/storage"
)

const maxImageSize = 5 << 20

var dataURLImageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(data:image/([a-zA-Z]+);base64,([^)]+)\)`)

type ServiceConfig struct {
	S3Bucket        string
	AWSRegion       string
	S3PublicBaseURL string
}

type Service struct {
	repo            Repository
	storage         storage.Storage
	publisher       events.Publisher
	logger          *slog.Logger
	s3Bucket        string
	awsRegion       string
	s3PublicBaseURL string
}

func NewService(repo Repository, storage storage.Storage, publisher events.Publisher, logger *slog.Logger, opts ServiceConfig) *Service {
	if publisher == nil {
		publisher = events.NoopPublisher{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:            repo,
		storage:         storage,
		publisher:       publisher,
		logger:          logger,
		s3Bucket:        opts.S3Bucket,
		awsRegion:       opts.AWSRegion,
		s3PublicBaseURL: opts.S3PublicBaseURL,
	}
}

func (s *Service) s3PublicURL(key string) string {
	if s.s3PublicBaseURL != "" {
		return s.s3PublicBaseURL + "/" + key
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.s3Bucket, s.awsRegion, key)
}

func (s *Service) processMarkdownImages(ctx context.Context, slug, content string) string {
	allowedTypes := map[string]string{
		"png":  "image/png",
		"jpeg": "image/jpeg",
		"jpg":  "image/jpeg",
		"webp": "image/webp",
		"gif":  "image/gif",
	}

	result := dataURLImageRegex.ReplaceAllStringFunc(content, func(match string) string {
		subs := dataURLImageRegex.FindStringSubmatch(match)
		if len(subs) != 4 {
			return match
		}
		alt, ext, b64 := subs[1], strings.ToLower(subs[2]), subs[3]
		contentType, ok := allowedTypes[ext]
		if !ok {
			return match
		}
		b64Clean := strings.ReplaceAll(strings.ReplaceAll(b64, "\n", ""), "\r", "")
		data, err := base64.StdEncoding.DecodeString(b64Clean)
		if err != nil || len(data) > maxImageSize {
			return match
		}
		key := fmt.Sprintf("posts/%s/images/%s.%s", slug, uuid.New().String(), ext)
		if err := s.storage.Upload(ctx, key, strings.NewReader(string(data)), contentType); err != nil {
			return match
		}
		url := s.s3PublicURL(key)
		return fmt.Sprintf("![%s](%s)", alt, url)
	})

	return result
}

func (s *Service) CreatePost(ctx context.Context, title, slug, content string) (*Post, error) {
	s3Key := fmt.Sprintf("posts/%s.md", slug)
	post, err := s.repo.Create(ctx, title, slug, s3Key)
	if err != nil {
		return nil, err
	}

	content = s.processMarkdownImages(ctx, slug, content)
	if err := s.storage.Upload(ctx, s3Key, strings.NewReader(content), "text/markdown"); err != nil {
		_ = s.repo.Delete(ctx, slug)
		return nil, fmt.Errorf("upload to s3: %w", err)
	}

	return post, nil
}

func (s *Service) GetPostBySlug(ctx context.Context, slug string) (*Post, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *Service) GetPostContent(ctx context.Context, slug string) ([]byte, error) {
	post, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	body, err := s.storage.Download(ctx, post.S3Key)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("download from s3: %w", err)
	}
	defer body.Close()
	return io.ReadAll(body)
}

func (s *Service) ListPosts(ctx context.Context, page, perPage int, status *Status) (*ListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	posts, err := s.repo.List(ctx, ListParams{
		Limit:  perPage,
		Offset: offset,
		Status: status,
	})
	if err != nil {
		return nil, err
	}

	total, err := s.repo.Count(ctx, status)
	if err != nil {
		return nil, err
	}

	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return &ListResult{
		Posts:      posts,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) UpdatePost(ctx context.Context, currentSlug string, title, newSlug, content *string) (*Post, error) {
	post, err := s.repo.GetBySlug(ctx, currentSlug)
	if err != nil {
		return nil, err
	}

	titleToUse := post.Title
	if title != nil {
		titleToUse = *title
	}
	slugToUse := post.Slug
	if newSlug != nil {
		slugToUse = *newSlug
	}

	var s3Key string
	if content != nil {
		processed := s.processMarkdownImages(ctx, slugToUse, *content)
		s3Key = fmt.Sprintf("posts/%s.md", slugToUse)
		if err := s.storage.Upload(ctx, s3Key, strings.NewReader(processed), "text/markdown"); err != nil {
			return nil, fmt.Errorf("upload to s3: %w", err)
		}
		if currentSlug != slugToUse {
			oldKey := fmt.Sprintf("posts/%s.md", currentSlug)
			_ = s.storage.Delete(ctx, oldKey)
		}
	} else {
		if slugToUse != currentSlug {
			newKey := fmt.Sprintf("posts/%s.md", slugToUse)
			body, err := s.storage.Download(ctx, post.S3Key)
			if err != nil {
				return nil, fmt.Errorf("download current content: %w", err)
			}
			defer body.Close()
			data, err := io.ReadAll(body)
			if err != nil {
				return nil, fmt.Errorf("read content: %w", err)
			}
			if err := s.storage.Upload(ctx, newKey, bytes.NewReader(data), "text/markdown"); err != nil {
				return nil, fmt.Errorf("upload to s3: %w", err)
			}
			_ = s.storage.Delete(ctx, post.S3Key)
			s3Key = newKey
		} else {
			s3Key = post.S3Key
		}
	}

	return s.repo.Update(ctx, post.ID, titleToUse, slugToUse, s3Key)
}

func (s *Service) DeletePost(ctx context.Context, slug string) error {
	post, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}
	if post.S3Key != "" {
		if delErr := s.storage.Delete(ctx, post.S3Key); delErr != nil {
			return fmt.Errorf("delete from s3: %w", delErr)
		}
	}
	imagesPrefix := fmt.Sprintf("posts/%s/images/", post.Slug)
	if delErr := s.storage.DeletePrefix(ctx, imagesPrefix); delErr != nil {
		return fmt.Errorf("delete images from s3: %w", delErr)
	}
	return s.repo.Delete(ctx, slug)
}

func (s *Service) PublishPost(ctx context.Context, slug string) (*Post, error) {
	post, err := s.repo.Publish(ctx, slug)
	if err != nil {
		return nil, err
	}
	evt := events.NewPostPublished(post.ID, post.Slug, post.Title)
	if err := s.publisher.PublishPostPublished(ctx, evt); err != nil {
		s.logger.Warn("failed to publish post.published event", "slug", post.Slug, "error", err)
	}
	return post, nil
}
