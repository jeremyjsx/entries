package posts

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	Draft     Status = "draft"
	Published Status = "published"
)

type Post struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	S3Key     string    `json:"s3_key"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListParams struct {
	Limit  int
	Offset int
	Status *Status
}

type ListResult struct {
	Posts      []*Post `json:"data"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	TotalPages int     `json:"total_pages"`
}
