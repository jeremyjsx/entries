.PHONY: dev down run build lint format migrate sqlc-generate sqlc-check bucket ci help

# Start infrastructure (PostgreSQL, LocalStack)
dev:
	docker compose up -d

# Stop infrastructure
down:
	docker compose down

# Run the API server
run:
	go run ./cmd/api

# Build the API binary
build:
	go build -o entries ./cmd/api

# Run CI steps locally (lint, build, sqlc check)
ci: lint build sqlc-check

# Verify sqlc generated code is up to date
sqlc-check:
	sqlc generate
	@git diff --exit-code internal/db/ || (echo "sqlc: generated code out of date. Run 'sqlc generate' and commit."; exit 1)

# Lint code
lint:
	gofmt -l . | grep -q . && (echo "Run 'make format' first"; exit 1) || true
	go vet ./...
	golangci-lint run

# Format code
format:
	gofmt -w .
	go mod tidy

# Run database migrations (requires DEV services up and DATABASE_URL in .env)
migrate:
	goose -dir internal/db/migrations postgres "$${DATABASE_URL}" up

# Generate sqlc code
sqlc-generate:
	sqlc generate

# Create S3 bucket in LocalStack (run after 'make dev')
bucket:
	aws --endpoint-url=http://localhost:4566 s3 mb s3://entries-content --region us-east-1 2>/dev/null || true

help:
	@echo "Entries - Make targets"
	@echo ""
	@echo "  make dev          Start PostgreSQL + LocalStack"
	@echo "  make down         Stop infrastructure"
	@echo "  make run          Run API server"
	@echo "  make build        Build binary"
	@echo "  make lint         Lint (gofmt, vet, golangci-lint)"
	@echo "  make format       Format code"
	@echo "  make migrate      Apply DB migrations"
	@echo "  make sqlc-generate  Generate sqlc code"
	@echo "  make bucket       Create S3 bucket in LocalStack"
	@echo "  make ci           Run CI locally (lint, build, sqlc check)"
	@echo "  make help         Show this help"
