package events

import "context"

type NoopPublisher struct{}

func (NoopPublisher) PublishPostPublished(context.Context, PostPublished) error {
	return nil
}

var _ Publisher = (*NoopPublisher)(nil)
