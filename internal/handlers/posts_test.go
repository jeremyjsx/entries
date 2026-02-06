package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jeremyjsx/entries/internal/posts"
	"github.com/jeremyjsx/entries/internal/storage"
)

type testMockRepo struct {
	create    func(ctx context.Context, title, slug, s3Key string) (*posts.Post, error)
	getBySlug func(ctx context.Context, slug string) (*posts.Post, error)
	list      func(ctx context.Context, params posts.ListParams) ([]*posts.Post, error)
	count     func(ctx context.Context, status *posts.Status) (int64, error)
	update    func(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*posts.Post, error)
	delete    func(ctx context.Context, slug string) error
	publish   func(ctx context.Context, slug string) (*posts.Post, error)
}

func (m *testMockRepo) Create(ctx context.Context, title, slug, s3Key string) (*posts.Post, error) {
	if m.create != nil {
		return m.create(ctx, title, slug, s3Key)
	}
	return nil, posts.ErrNotFound
}

func (m *testMockRepo) GetBySlug(ctx context.Context, slug string) (*posts.Post, error) {
	if m.getBySlug != nil {
		return m.getBySlug(ctx, slug)
	}
	return nil, posts.ErrNotFound
}

func (m *testMockRepo) List(ctx context.Context, params posts.ListParams) ([]*posts.Post, error) {
	if m.list != nil {
		return m.list(ctx, params)
	}
	return nil, nil
}

func (m *testMockRepo) Count(ctx context.Context, status *posts.Status) (int64, error) {
	if m.count != nil {
		return m.count(ctx, status)
	}
	return 0, nil
}

func (m *testMockRepo) Update(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*posts.Post, error) {
	if m.update != nil {
		return m.update(ctx, id, title, slug, s3Key)
	}
	return nil, posts.ErrNotFound
}

func (m *testMockRepo) Delete(ctx context.Context, slug string) error {
	if m.delete != nil {
		return m.delete(ctx, slug)
	}
	return nil
}

func (m *testMockRepo) Publish(ctx context.Context, slug string) (*posts.Post, error) {
	if m.publish != nil {
		return m.publish(ctx, slug)
	}
	return nil, posts.ErrNotFound
}

type testMockStorage struct {
	upload       func(ctx context.Context, key string, body io.Reader, contentType string) error
	download     func(ctx context.Context, key string) (io.ReadCloser, error)
	delete       func(ctx context.Context, key string) error
	deletePrefix func(ctx context.Context, prefix string) error
	exists       func(ctx context.Context, key string) (bool, error)
}

func (m *testMockStorage) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	if m.upload != nil {
		return m.upload(ctx, key, body, contentType)
	}
	return nil
}

func (m *testMockStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.download != nil {
		return m.download(ctx, key)
	}
	return nil, storage.ErrNotFound
}

func (m *testMockStorage) Delete(ctx context.Context, key string) error {
	if m.delete != nil {
		return m.delete(ctx, key)
	}
	return nil
}

func (m *testMockStorage) DeletePrefix(ctx context.Context, prefix string) error {
	if m.deletePrefix != nil {
		return m.deletePrefix(ctx, prefix)
	}
	return nil
}

func (m *testMockStorage) Exists(ctx context.Context, key string) (bool, error) {
	if m.exists != nil {
		return m.exists(ctx, key)
	}
	return false, nil
}

func testHandler(t *testing.T) (*PostsHandler, *testMockRepo, *testMockStorage) {
	repo := &testMockRepo{}
	st := &testMockStorage{}
	svc := posts.NewService(repo, st, "b", "r", "")
	h := NewPostsHandler(svc, slog.Default())
	return h, repo, st
}

func testMux(h *PostsHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /posts", h.List())
	mux.HandleFunc("POST /posts", h.Create())
	mux.HandleFunc("GET /posts/{slug}/content", h.GetContent())
	mux.HandleFunc("GET /posts/{slug}", h.GetBySlug())
	mux.HandleFunc("PUT /posts/{slug}", h.Update())
	mux.HandleFunc("DELETE /posts/{slug}", h.Delete())
	mux.HandleFunc("PATCH /posts/{slug}/publish", h.Publish())
	return mux
}

func TestPostsHandler_Create(t *testing.T) {
	h, repo, st := testHandler(t)
	repo.create = func(ctx context.Context, title, slug, s3Key string) (*posts.Post, error) {
		return &posts.Post{ID: uuid.New(), Title: title, Slug: slug, S3Key: s3Key, Status: posts.Draft}, nil
	}
	st.upload = func(context.Context, string, io.Reader, string) error { return nil }

	body := bytes.NewBufferString(`{"title":"Hello","slug":"hello","content":"# Hi"}`)
	req := httptest.NewRequest(http.MethodPost, "/posts", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Create: status %d, body %s", rec.Code, rec.Body.Bytes())
	}
	var post posts.Post
	if err := json.NewDecoder(rec.Body).Decode(&post); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if post.Slug != "hello" || post.Title != "Hello" {
		t.Errorf("got %+v", post)
	}
}

func TestPostsHandler_Create_InvalidJSON(t *testing.T) {
	h, _, _ := testHandler(t)
	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/posts", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestPostsHandler_Create_ValidationError(t *testing.T) {
	h, _, _ := testHandler(t)
	body := bytes.NewBufferString(`{"title":"","slug":"","content":""}`)
	req := httptest.NewRequest(http.MethodPost, "/posts", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestPostsHandler_Create_Conflict(t *testing.T) {
	h, repo, st := testHandler(t)
	repo.create = func(context.Context, string, string, string) (*posts.Post, error) {
		return nil, posts.ErrSlugExists
	}
	st.upload = func(context.Context, string, io.Reader, string) error { return nil }

	body := bytes.NewBufferString(`{"title":"X","slug":"x","content":"c"}`)
	req := httptest.NewRequest(http.MethodPost, "/posts", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestPostsHandler_GetBySlug(t *testing.T) {
	h, repo, _ := testHandler(t)
	want := &posts.Post{ID: uuid.New(), Title: "A", Slug: "a", Status: posts.Draft}
	repo.getBySlug = func(context.Context, string) (*posts.Post, error) { return want, nil }

	req := httptest.NewRequest(http.MethodGet, "/posts/a", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GetBySlug: status %d", rec.Code)
	}
	var post posts.Post
	if err := json.NewDecoder(rec.Body).Decode(&post); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if post.Slug != "a" {
		t.Errorf("got slug %q", post.Slug)
	}
}

func TestPostsHandler_GetBySlug_NotFound(t *testing.T) {
	h, repo, _ := testHandler(t)
	repo.getBySlug = func(context.Context, string) (*posts.Post, error) { return nil, posts.ErrNotFound }

	req := httptest.NewRequest(http.MethodGet, "/posts/missing", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestPostsHandler_GetContent(t *testing.T) {
	h, repo, st := testHandler(t)
	repo.getBySlug = func(context.Context, string) (*posts.Post, error) {
		return &posts.Post{Slug: "a", S3Key: "posts/a.md"}, nil
	}
	st.download = func(context.Context, string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader([]byte("# Hello"))), nil
	}

	req := httptest.NewRequest(http.MethodGet, "/posts/a/content", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GetContent: status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/markdown; charset=utf-8" {
		t.Errorf("Content-Type %q", ct)
	}
	if rec.Body.String() != "# Hello" {
		t.Errorf("body %q", rec.Body.String())
	}
}

func TestPostsHandler_GetContent_NotFound(t *testing.T) {
	h, repo, _ := testHandler(t)
	repo.getBySlug = func(context.Context, string) (*posts.Post, error) { return nil, posts.ErrNotFound }

	req := httptest.NewRequest(http.MethodGet, "/posts/missing/content", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestPostsHandler_List(t *testing.T) {
	h, repo, _ := testHandler(t)
	repo.list = func(context.Context, posts.ListParams) ([]*posts.Post, error) {
		return []*posts.Post{{ID: uuid.New(), Slug: "one"}}, nil
	}
	repo.count = func(context.Context, *posts.Status) (int64, error) { return 1, nil }

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("List: status %d", rec.Code)
	}
}

func TestPostsHandler_List_InvalidStatus(t *testing.T) {
	h, _, _ := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/posts?status=invalid", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status, got %d", rec.Code)
	}
}

func TestPostsHandler_Update(t *testing.T) {
	h, repo, st := testHandler(t)
	pid := uuid.New()
	repo.getBySlug = func(context.Context, string) (*posts.Post, error) {
		return &posts.Post{ID: pid, Title: "Old", Slug: "old", S3Key: "posts/old.md"}, nil
	}
	repo.update = func(ctx context.Context, id uuid.UUID, title, slug, s3Key string) (*posts.Post, error) {
		return &posts.Post{ID: id, Title: title, Slug: slug, S3Key: s3Key}, nil
	}
	st.upload = func(context.Context, string, io.Reader, string) error { return nil }

	body := bytes.NewBufferString(`{"title":"New Title"}`)
	req := httptest.NewRequest(http.MethodPut, "/posts/old", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Update: status %d, body %s", rec.Code, rec.Body.Bytes())
	}
}

func TestPostsHandler_Update_InvalidJSON(t *testing.T) {
	h, _, _ := testHandler(t)
	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPut, "/posts/slug", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestPostsHandler_Update_NoFields(t *testing.T) {
	h, _, _ := testHandler(t)
	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPut, "/posts/slug", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestPostsHandler_Update_NotFound(t *testing.T) {
	h, repo, st := testHandler(t)
	repo.getBySlug = func(context.Context, string) (*posts.Post, error) { return nil, posts.ErrNotFound }
	st.upload = func(context.Context, string, io.Reader, string) error { return nil }

	body := bytes.NewBufferString(`{"title":"X"}`)
	req := httptest.NewRequest(http.MethodPut, "/posts/missing", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestPostsHandler_Update_Conflict(t *testing.T) {
	h, repo, st := testHandler(t)
	pid := uuid.New()
	repo.getBySlug = func(context.Context, string) (*posts.Post, error) {
		return &posts.Post{ID: pid, Title: "Old", Slug: "old", S3Key: "posts/old.md"}, nil
	}
	repo.update = func(context.Context, uuid.UUID, string, string, string) (*posts.Post, error) {
		return nil, posts.ErrSlugExists
	}
	st.upload = func(context.Context, string, io.Reader, string) error { return nil }

	body := bytes.NewBufferString(`{"slug":"taken","content":"body"}`)
	req := httptest.NewRequest(http.MethodPut, "/posts/old", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestPostsHandler_Delete(t *testing.T) {
	h, repo, st := testHandler(t)
	repo.getBySlug = func(context.Context, string) (*posts.Post, error) {
		return &posts.Post{Slug: "d", S3Key: "posts/d.md"}, nil
	}
	repo.delete = func(context.Context, string) error { return nil }
	st.delete = func(context.Context, string) error { return nil }
	st.deletePrefix = func(context.Context, string) error { return nil }

	req := httptest.NewRequest(http.MethodDelete, "/posts/d", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("Delete: status %d", rec.Code)
	}
}

func TestPostsHandler_Publish(t *testing.T) {
	h, repo, _ := testHandler(t)
	repo.publish = func(context.Context, string) (*posts.Post, error) {
		return &posts.Post{ID: uuid.New(), Slug: "p", Status: posts.Published}, nil
	}

	req := httptest.NewRequest(http.MethodPatch, "/posts/p/publish", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Publish: status %d", rec.Code)
	}
}

func TestPostsHandler_Publish_NotFound(t *testing.T) {
	h, repo, _ := testHandler(t)
	repo.publish = func(context.Context, string) (*posts.Post, error) { return nil, posts.ErrNotFound }

	req := httptest.NewRequest(http.MethodPatch, "/posts/missing/publish", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestPostsHandler_Delete_NotFound(t *testing.T) {
	h, repo, _ := testHandler(t)
	repo.getBySlug = func(context.Context, string) (*posts.Post, error) { return nil, posts.ErrNotFound }

	req := httptest.NewRequest(http.MethodDelete, "/posts/missing", nil)
	rec := httptest.NewRecorder()
	testMux(h).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
