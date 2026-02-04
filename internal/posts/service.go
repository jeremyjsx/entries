package posts

import (
	"context"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreatePost(ctx context.Context, title, slug, s3Key string) (*Post, error) {
	return s.repo.Create(ctx, title, slug, s3Key)
}

func (s *Service) GetPostBySlug(ctx context.Context, slug string) (*Post, error) {
	return s.repo.GetBySlug(ctx, slug)
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

func (s *Service) UpdatePost(ctx context.Context, currentSlug, title, newSlug, s3Key string) (*Post, error) {
	post, err := s.repo.GetBySlug(ctx, currentSlug)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, post.ID, title, newSlug, s3Key)
}

func (s *Service) DeletePost(ctx context.Context, slug string) error {
	return s.repo.Delete(ctx, slug)
}

func (s *Service) PublishPost(ctx context.Context, slug string) (*Post, error) {
	return s.repo.Publish(ctx, slug)
}
