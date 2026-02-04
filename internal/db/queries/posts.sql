-- name: CreatePost :one
INSERT INTO posts (title, slug, s3_key, status)
VALUES ($1, $2, $3, $4)
RETURNING id, title, slug, s3_key, status, created_at, updated_at;

-- name: GetPostBySlug :one
SELECT id, title, slug, s3_key, status, created_at, updated_at FROM posts WHERE slug = $1;

-- name: ListPosts :many
SELECT id, title, slug, s3_key, status, created_at, updated_at FROM posts
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountPosts :one
SELECT COUNT(*) FROM posts
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'));

-- name: UpdatePost :one
UPDATE posts SET title = $2, slug = $3, s3_key = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, title, slug, s3_key, status, created_at, updated_at;

-- name: DeletePostBySlug :exec
DELETE FROM posts WHERE slug = $1;

-- name: PublishPost :one
UPDATE posts SET status = 'published', updated_at = NOW()
WHERE slug = $1 AND status = 'draft'
RETURNING id, title, slug, s3_key, status, created_at, updated_at;
