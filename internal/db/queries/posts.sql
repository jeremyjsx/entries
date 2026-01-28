-- name: CreatePost :one
INSERT INTO posts (title, slug, s3_key, status)
VALUES ($1, $2, $3, $4)
RETURNING id, title, slug, s3_key, status, created_at, updated_at;
