package posts

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jeremyjsx/entries/internal/db"
	"github.com/lib/pq"
)

var _ Repository = (*postgresRepository)(nil)

type postgresRepository struct {
	queries *db.Queries
}

func NewPostgresRepository(sqlDB *sql.DB) Repository {
	return &postgresRepository{queries: db.New(sqlDB)}
}

func (r *postgresRepository) Create(ctx context.Context, title, slug, s3Key string) (*Post, error) {
	dbPost, err := r.queries.CreatePost(ctx, db.CreatePostParams{
		Title:  title,
		Slug:   slug,
		S3Key:  s3Key,
		Status: string(Draft),
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrSlugExists
		}
		return nil, err
	}
	return toPost(dbPost), nil
}

func (r *postgresRepository) GetBySlug(ctx context.Context, slug string) (*Post, error) {
	dbPost, err := r.queries.GetPostBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toPost(dbPost), nil
}

func (r *postgresRepository) List(ctx context.Context, params ListParams) ([]*Post, error) {
	var status sql.NullString
	if params.Status != nil {
		status = sql.NullString{String: string(*params.Status), Valid: true}
	}

	dbPosts, err := r.queries.ListPosts(ctx, db.ListPostsParams{
		Limit:  int32(params.Limit),
		Offset: int32(params.Offset),
		Status: status,
	})
	if err != nil {
		return nil, err
	}

	posts := make([]*Post, len(dbPosts))
	for i, p := range dbPosts {
		posts[i] = toPost(p)
	}
	return posts, nil
}

func (r *postgresRepository) Count(ctx context.Context, status *Status) (int64, error) {
	var s sql.NullString
	if status != nil {
		s = sql.NullString{String: string(*status), Valid: true}
	}
	return r.queries.CountPosts(ctx, s)
}

func (r *postgresRepository) Update(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*Post, error) {
	dbPost, err := r.queries.UpdatePost(ctx, db.UpdatePostParams{
		ID:    id,
		Title: title,
		Slug:  slug,
		S3Key: s3Key,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrSlugExists
		}
		return nil, err
	}
	return toPost(dbPost), nil
}

func (r *postgresRepository) Delete(ctx context.Context, slug string) error {
	return r.queries.DeletePostBySlug(ctx, slug)
}

func (r *postgresRepository) Publish(ctx context.Context, slug string) (*Post, error) {
	dbPost, err := r.queries.PublishPost(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toPost(dbPost), nil
}

func toPost(p db.Post) *Post {
	return &Post{
		ID:        p.ID,
		Title:     p.Title,
		Slug:      p.Slug,
		S3Key:     p.S3Key,
		Status:    Status(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
