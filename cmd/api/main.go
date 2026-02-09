package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jeremyjsx/entries/internal/config"
	"github.com/jeremyjsx/entries/internal/events"
	"github.com/jeremyjsx/entries/internal/handlers"
	"github.com/jeremyjsx/entries/internal/middleware"
	"github.com/jeremyjsx/entries/internal/posts"
	"github.com/jeremyjsx/entries/internal/storage"
	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	if cfg.S3Bucket == "" {
		logger.Error("S3_BUCKET is required")
		os.Exit(1)
	}

	db, err := openDB(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		logger.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			o.UsePathStyle = true
		}
	})
	store := storage.NewS3Storage(s3Client, cfg.S3Bucket)
	s3PublicBaseURL := ""
	if cfg.S3Endpoint != "" {
		s3PublicBaseURL = strings.TrimSuffix(cfg.S3Endpoint, "/") + "/" + cfg.S3Bucket
	}

	var publisher events.Publisher = events.NoopPublisher{}
	if cfg.RabbitMQURL != "" {
		rmq, err := events.NewRabbitMQPublisher(cfg.RabbitMQURL)
		if err != nil {
			logger.Error("failed to connect to RabbitMQ", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := rmq.Close(); err != nil {
				logger.Warn("rabbitmq close on shutdown", "error", err)
			}
		}()
		publisher = rmq
		logger.Info("event publisher connected", "broker", "rabbitmq")
	} else {
		logger.Info("event publisher disabled", "broker", "none")
	}

	repo := posts.NewPostgresRepository(db)
	svc := posts.NewService(repo, store, publisher, logger, posts.ServiceConfig{
		S3Bucket:        cfg.S3Bucket,
		AWSRegion:       cfg.AWSRegion,
		S3PublicBaseURL: s3PublicBaseURL,
	})
	postsHandler := handlers.NewPostsHandler(svc, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlers.Health(&handlers.HealthDeps{
		DB: db, Storage: store, RabbitMQURL: cfg.RabbitMQURL,
	}))
	mux.HandleFunc("GET /posts", postsHandler.List())
	mux.HandleFunc("POST /posts", postsHandler.Create())
	mux.HandleFunc("GET /posts/{slug}/content", postsHandler.GetContent())
	mux.HandleFunc("GET /posts/{slug}", postsHandler.GetBySlug())
	mux.HandleFunc("PUT /posts/{slug}", postsHandler.Update())
	mux.HandleFunc("DELETE /posts/{slug}", postsHandler.Delete())
	mux.HandleFunc("PATCH /posts/{slug}/publish", postsHandler.Publish())

	handler := middleware.Recovery(logger)(
		middleware.RequestID(
			middleware.Logging(logger)(
				middleware.APIKey(cfg.APIKey)(mux),
			),
		),
	)
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server started", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}

func openDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
