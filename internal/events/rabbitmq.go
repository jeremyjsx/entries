package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "entries.events"
	RoutingKey   = "post.published"
	QueueName    = "newsletter.post_published"
)

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.Mutex
	once    sync.Once
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}
	if err := ch.ExchangeDeclare(ExchangeName, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}
	return &RabbitMQPublisher{conn: conn, channel: ch}, nil
}

func (p *RabbitMQPublisher) PublishPostPublished(ctx context.Context, e PostPublished) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.channel == nil {
		return fmt.Errorf("publisher closed")
	}
	err = p.channel.PublishWithContext(ctx, ExchangeName, RoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}

func (p *RabbitMQPublisher) Close() error {
	var err error
	p.once.Do(func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.channel != nil {
			err = p.channel.Close()
			p.channel = nil
		}
		if p.conn != nil {
			if closeErr := p.conn.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			p.conn = nil
		}
	})
	return err
}
