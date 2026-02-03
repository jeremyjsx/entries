package posts

import "errors"

var (
	ErrNotFound   = errors.New("post not found")
	ErrSlugExists = errors.New("slug already exists")
)
