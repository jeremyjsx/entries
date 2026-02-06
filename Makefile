.PHONY: dev down run build test test-integration coverage coverage-integration lint format migrate sqlc-generate sqlc-check bucket ci help

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

# Build and run unit tests
test:
	go test -short ./...

TEST_DB_URL = postgres://entries:entries@localhost:5433/entries?sslmode=disable
# Run integration tests (ephemeral Postgres, then tear down)
test-integration:
	docker compose -f docker-compose.test.yml up -d && \
	sleep 5 && \
	( \
	  TEST_DATABASE_URL='$(TEST_DB_URL)' goose -dir internal/db/migrations postgres '$(TEST_DB_URL)' up && \
	  TEST_DATABASE_URL='$(TEST_DB_URL)' go test -tags=integration ./...; \
	  r=$$?; docker compose -f docker-compose.test.yml down; exit $$r \
	)

# Run unit tests and generate coverage (fast; internal/posts + internal/handlers)
coverage:
	go test -short -coverprofile=coverage.out ./internal/posts/... ./internal/handlers/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage written to coverage.html"

# Run unit + integration tests and generate coverage (includes repo/DB)
coverage-integration:
	docker compose -f docker-compose.test.yml up -d && \
	sleep 5 && \
	( \
	  TEST_DATABASE_URL='$(TEST_DB_URL)' goose -dir internal/db/migrations postgres '$(TEST_DB_URL)' up && \
	  TEST_DATABASE_URL='$(TEST_DB_URL)' go test -tags=integration -coverprofile=coverage.out ./... && \
	  go tool cover -html=coverage.out -o coverage.html; \
	  r=$$?; docker compose -f docker-compose.test.yml down; exit $$r \
	)
	@echo "Coverage (with integration) written to coverage.html"

# Run CI steps locally (lint, build, test, sqlc check)
ci: lint build test sqlc-check

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

# Run database migrations (requires DATABASE_URL in env)
migrate:
	goose -dir internal/db/migrations postgres "$(DATABASE_URL)" up

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
	@echo "  make test         Run unit tests"
	@echo "  make test-integration     Run integration tests (ephemeral Postgres, then tear down)"
	@echo "  make coverage             Unit-test coverage → coverage.html"
	@echo "  make coverage-integration  Unit + integration coverage (includes repo/DB)"
	@echo "  make ci           Run CI locally (lint, build, test, sqlc check)"
	@echo "  make help         Show this help"
