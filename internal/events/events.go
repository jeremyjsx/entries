package events

import (
	"time"

	"github.com/google/uuid"
)

const TypePostPublished = "post.published"

type PostPublishedPayload struct {
	PostID uuid.UUID `json:"post_id"`
	Slug   string    `json:"slug"`
	Title  string    `json:"title"`
}

type PostPublished struct {
	Type      string               `json:"type"`
	Timestamp time.Time            `json:"timestamp"`
	Payload   PostPublishedPayload `json:"payload"`
}

func NewPostPublished(postID uuid.UUID, slug, title string) PostPublished {
	return PostPublished{
		Type:      TypePostPublished,
		Timestamp: time.Now().UTC(),
		Payload: PostPublishedPayload{
			PostID: postID,
			Slug:   slug,
			Title:  title,
		},
	}
}
