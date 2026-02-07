package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jeremyjsx/entries/internal/config"
	"github.com/jeremyjsx/entries/internal/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := config.Load()
	if cfg.RabbitMQURL == "" {
		logger.Error("RABBITMQ_URL is required")
		os.Exit(1)
	}

	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		logger.Error("failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logger.Error("failed to open channel", "error", err)
		os.Exit(1)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(events.ExchangeName, "topic", true, false, false, false, nil); err != nil {
		logger.Error("failed to declare exchange", "error", err)
		os.Exit(1)
	}

	q, err := ch.QueueDeclare(events.QueueName, true, false, false, false, nil)
	if err != nil {
		logger.Error("failed to declare queue", "error", err)
		os.Exit(1)
	}

	if err := ch.QueueBind(q.Name, events.RoutingKey, events.ExchangeName, false, nil); err != nil {
		logger.Error("failed to bind queue", "error", err)
		os.Exit(1)
	}

	deliveries, err := ch.Consume(q.Name, "newsletter-worker", false, false, false, false, nil)
	if err != nil {
		logger.Error("failed to start consuming", "error", err)
		os.Exit(1)
	}

	logger.Info("newsletter worker started", "queue", q.Name)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-quit:
			logger.Info("worker shutting down")
			return
		case d, ok := <-deliveries:
			if !ok {
				logger.Warn("delivery channel closed")
				return
			}
			handlePostPublished(logger, d)
		}
	}
}

func handlePostPublished(logger *slog.Logger, d amqp.Delivery) {
	var e events.PostPublished
	if err := json.Unmarshal(d.Body, &e); err != nil {
		logger.Error("invalid event body", "error", err)
		_ = d.Nack(false, false)
		return
	}
	if e.Type != events.TypePostPublished {
		logger.Debug("ignoring event type", "type", e.Type)
		_ = d.Ack(false)
		return
	}
	logger.Info("post published event received",
		"post_id", e.Payload.PostID,
		"slug", e.Payload.Slug,
		"title", e.Payload.Title,
	)

	if err := d.Ack(false); err != nil {
		logger.Error("failed to ack", "error", err)
	}
}
