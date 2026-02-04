package posts

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, title, slug, s3Key string) (*Post, error)
	GetBySlug(ctx context.Context, slug string) (*Post, error)
	List(ctx context.Context, params ListParams) ([]*Post, error)
	Count(ctx context.Context, status *Status) (int64, error)
	Update(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*Post, error)
	Delete(ctx context.Context, slug string) error
	Publish(ctx context.Context, slug string) (*Post, error)
}
