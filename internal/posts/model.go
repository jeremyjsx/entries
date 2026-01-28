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
