package events

import "context"

type Publisher interface {
	PublishPostPublished(ctx context.Context, e PostPublished) error
}
