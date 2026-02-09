package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

func APIKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			if r.URL.Path == "/health" && r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			key := extractAPIKey(r)
			if key == "" || subtle.ConstantTimeCompare([]byte(apiKey), []byte(key)) != 1 {
				writeUnauthorized(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractAPIKey(r *http.Request) string {
	if s := r.Header.Get("X-API-Key"); s != "" {
		return s
	}
	const prefix = "Bearer "
	if s := r.Header.Get("Authorization"); strings.HasPrefix(s, prefix) {
		return strings.TrimSpace(s[len(prefix):])
	}
	return ""
}

func writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":       "UNAUTHORIZED",
			"message":    "missing or invalid API key",
			"request_id": GetRequestID(r.Context()),
		},
	})
}
