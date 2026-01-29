package posts

import (
	"context"
	"database/sql"

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

	return &Post{
		ID:        dbPost.ID,
		Title:     dbPost.Title,
		Slug:      dbPost.Slug,
		S3Key:     dbPost.S3Key,
		Status:    Status(dbPost.Status),
		CreatedAt: dbPost.CreatedAt,
		UpdatedAt: dbPost.UpdatedAt,
	}, nil
}

func (r *postgresRepository) GetBySlug(ctx context.Context, slug string) (*Post, error) {
	dbPost, err := r.queries.GetPostBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return &Post{
		ID:        dbPost.ID,
		Title:     dbPost.Title,
		Slug:      dbPost.Slug,
		S3Key:     dbPost.S3Key,
		Status:    Status(dbPost.Status),
		CreatedAt: dbPost.CreatedAt,
		UpdatedAt: dbPost.UpdatedAt,
	}, nil
}
