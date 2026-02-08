package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/jeremyjsx/entries/internal/storage"
	amqp "github.com/rabbitmq/amqp091-go"
)

type HealthDeps struct {
	DB          *sql.DB
	Storage     storage.Storage
	RabbitMQURL string
}

type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func Health(deps *HealthDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		checks := map[string]string{}
		status := "healthy"

		if err := deps.DB.PingContext(ctx); err != nil {
			checks["db"] = "unhealthy"
			status = "unhealthy"
		} else {
			checks["db"] = "ok"
		}

		if _, err := deps.Storage.Exists(ctx, "__health__"); err != nil {
			checks["s3"] = "unhealthy"
			status = "unhealthy"
		} else {
			checks["s3"] = "ok"
		}

		if deps.RabbitMQURL != "" {
			conn, err := amqp.Dial(deps.RabbitMQURL)
			if err != nil {
				checks["rabbitmq"] = "unhealthy"
				status = "degraded"
			} else {
				_ = conn.Close()
				checks["rabbitmq"] = "ok"
			}
		} else {
			checks["rabbitmq"] = "skipped"
		}

		code := http.StatusOK
		if status == "unhealthy" {
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, healthResponse{Status: status, Checks: checks})
	}
}
