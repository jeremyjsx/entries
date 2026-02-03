package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jeremyjsx/entries/internal/posts"
)

type PostsHandler struct {
	svc    *posts.Service
	logger *slog.Logger
}

func NewPostsHandler(svc *posts.Service, logger *slog.Logger) *PostsHandler {
	return &PostsHandler{
		svc:    svc,
		logger: logger,
	}
}

type CreatePostRequest struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
	S3Key string `json:"s3_key"`
}

func (h *PostsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreatePostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body", nil)
			return
		}

		errs := make(map[string]string)
		if req.Title == "" {
			errs["title"] = "required"
		}
		if req.Slug == "" {
			errs["slug"] = "required"
		}
		if len(errs) > 0 {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "validation failed", errs)
			return
		}

		post, err := h.svc.CreatePost(r.Context(), req.Title, req.Slug, req.S3Key)
		if err != nil {
			if errors.Is(err, posts.ErrSlugExists) {
				writeError(w, http.StatusConflict, "CONFLICT", "slug already exists", nil)
				return
			}
			h.logger.Error("create post failed", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			return
		}

		writeJSON(w, http.StatusCreated, post)
	}
}

func (h *PostsHandler) GetBySlug() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if slug == "" {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "slug is required", nil)
			return
		}

		post, err := h.svc.GetPostBySlug(r.Context(), slug)
		if err != nil {
			if errors.Is(err, posts.ErrNotFound) {
				writeError(w, http.StatusNotFound, "NOT_FOUND", "post not found", nil)
				return
			}
			h.logger.Error("get post failed", "slug", slug, "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			return
		}

		writeJSON(w, http.StatusOK, post)
	}
}
