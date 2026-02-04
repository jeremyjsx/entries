package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/jeremyjsx/entries/internal/posts"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

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

type PostRequest struct {
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
}

func (h *PostsHandler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body", nil)
			return
		}

		if errs := validatePostRequest(req.Title, req.Slug); len(errs) > 0 {
			writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "validation failed", errs)
			return
		}

		post, err := h.svc.CreatePost(r.Context(), req.Title, req.Slug, req.Content)
		if err != nil {
			if errors.Is(err, posts.ErrSlugExists) {
				writeError(w, r, http.StatusConflict, "CONFLICT", "slug already exists", nil)
				return
			}
			h.logger.Error("create post failed", "error", err)
			writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			return
		}

		writeJSON(w, http.StatusCreated, post)
	}
}

func (h *PostsHandler) GetBySlug() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if slug == "" {
			writeError(w, r, http.StatusBadRequest, "BAD_REQUEST", "slug is required", nil)
			return
		}

		post, err := h.svc.GetPostBySlug(r.Context(), slug)
		if err != nil {
			if errors.Is(err, posts.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "NOT_FOUND", "post not found", nil)
				return
			}
			h.logger.Error("get post failed", "slug", slug, "error", err)
			writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			return
		}

		writeJSON(w, http.StatusOK, post)
	}
}

func (h *PostsHandler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

		var status *posts.Status
		if s := r.URL.Query().Get("status"); s != "" {
			st := posts.Status(s)
			if st != posts.Draft && st != posts.Published {
				writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid status", nil)
				return
			}
			status = &st
		}

		result, err := h.svc.ListPosts(r.Context(), page, perPage, status)
		if err != nil {
			h.logger.Error("list posts failed", "error", err)
			writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func (h *PostsHandler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if slug == "" {
			writeError(w, r, http.StatusBadRequest, "BAD_REQUEST", "slug is required", nil)
			return
		}

		var req PostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body", nil)
			return
		}

		if errs := validatePostRequest(req.Title, req.Slug); len(errs) > 0 {
			writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "validation failed", errs)
			return
		}

		post, err := h.svc.UpdatePost(r.Context(), slug, req.Title, req.Slug, req.Content)
		if err != nil {
			if errors.Is(err, posts.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "NOT_FOUND", "post not found", nil)
				return
			}
			h.logger.Error("update post failed", "slug", slug, "error", err)
			writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			return
		}

		writeJSON(w, http.StatusOK, post)
	}
}

func (h *PostsHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if slug == "" {
			writeError(w, r, http.StatusBadRequest, "BAD_REQUEST", "slug is required", nil)
			return
		}

		if err := h.svc.DeletePost(r.Context(), slug); err != nil {
			h.logger.Error("delete post failed", "slug", slug, "error", err)
			writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *PostsHandler) Publish() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if slug == "" {
			writeError(w, r, http.StatusBadRequest, "BAD_REQUEST", "slug is required", nil)
			return
		}

		post, err := h.svc.PublishPost(r.Context(), slug)
		if err != nil {
			if errors.Is(err, posts.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "NOT_FOUND", "post not found or already published", nil)
				return
			}
			h.logger.Error("publish post failed", "slug", slug, "error", err)
			writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
			return
		}

		writeJSON(w, http.StatusOK, post)
	}
}

func validatePostRequest(title, slug string) map[string]string {
	errs := make(map[string]string)
	if title == "" {
		errs["title"] = "required"
	} else if len(title) > 200 {
		errs["title"] = "max 200 characters"
	}
	if slug == "" {
		errs["slug"] = "required"
	} else if len(slug) > 100 {
		errs["slug"] = "max 100 characters"
	} else if !slugRegex.MatchString(slug) {
		errs["slug"] = "must be lowercase alphanumeric with hyphens"
	}
	return errs
}
