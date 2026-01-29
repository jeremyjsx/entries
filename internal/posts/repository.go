package posts

import "context"

type Repository interface {
	Create(ctx context.Context, title, slug, s3Key string) (*Post, error)
	GetBySlug(ctx context.Context, slug string) (*Post, error)
}
