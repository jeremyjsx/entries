package posts

import "context"

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
