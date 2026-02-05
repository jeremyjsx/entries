# Entries

Event-driven content backend with Markdown publishing and async distribution. Built with **Go**: post CRUD, content in S3 and inline images.

## Features

- **Posts**: Full CRUD with slug, title, status (draft/published), pagination
- **Content in S3**: Markdown stored in S3; metadata in PostgreSQL
- **Inline images**: Base64 images in markdown are uploaded to S3 and replaced with URLs
- **Partial update**: PUT accepts only the fields you want to change
- **LocalStack**: Path-style S3 and public image URLs for local dev

## Quick Start

### Prerequisites

- Go 1.23+
- Docker & Docker Compose
- Make (optional; you can run the commands manually)
- [goose](https://github.com/pressly/goose) for migrations (`go install github.com/pressly/goose/v3/cmd/goose@latest`)
- [sqlc](https://sqlc.dev/) if you change queries (`sqlc generate`)

### 1. Clone and setup

```bash
git clone https://github.com/jeremyjsx/entries.git
cd entries

cp .env.example .env
# Edit .env if needed (defaults work with docker-compose)
```

### 2. Start services

```bash
# Start PostgreSQL and LocalStack (S3)
make dev

# Create S3 bucket (LocalStack)
make bucket

# Apply migrations
make migrate

# Run the API
make run
```

### 3. Use the API

- **API**: http://localhost:8080
- **Health**: http://localhost:8080/health
- **Posts**: `GET /posts`, `POST /posts`, `GET /posts/{slug}`, `GET /posts/{slug}/content`, `PUT /posts/{slug}`, `DELETE /posts/{slug}`, `PATCH /posts/{slug}/publish`

## Development

```bash
make dev      # Start infrastructure only
make run      # Run API
make lint     # Lint (gofmt, vet, golangci-lint)
make format   # Format code
make migrate  # Run migrations
make sqlc-generate  # Regenerate sqlc after editing queries
```

## Project structure

```
cmd/api/           # API entrypoint
internal/
  config/          # Env configuration
  db/              # sqlc-generated DB layer, migrations
  handlers/        # HTTP handlers (posts, health)
  middleware/      # Logging, request ID, recovery
  posts/           # Domain: repository, service, model
  storage/         # S3 interface and implementation
```

## Tech stack

| Category   | Technology        |
|-----------|-------------------|
| Language  | Go (stdlib net/http) |
| Database  | PostgreSQL        |
| Storage   | S3 (LocalStack locally) |
| Migrations| Goose             |
| SQL       | sqlc              |
| Config    | godotenv + env    |

## Environment variables

See `.env.example`. Main ones:

- `PORT`: Server port (default 8080)
- `DATABASE_URL`: PostgreSQL connection string
- `S3_BUCKET`: Bucket name
- `S3_ENDPOINT`: Set for LocalStack (e.g. `http://localhost:4566`); leave empty for AWS
