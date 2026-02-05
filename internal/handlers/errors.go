package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jeremyjsx/entries/internal/middleware"
)

type APIError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("writeJSON encode: %v", err)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, details map[string]string) {
	writeJSON(w, status, map[string]any{
		"error": APIError{
			Code:      code,
			Message:   message,
			RequestID: middleware.GetRequestID(r.Context()),
			Details:   details,
		},
	})
}
