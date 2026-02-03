package posts

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jeremyjsx/entries/internal/db"
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
