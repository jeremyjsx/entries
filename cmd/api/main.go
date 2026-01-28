package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jeremyjsx/entries/internal/config"
	"github.com/jeremyjsx/entries/internal/posts"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := openDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := posts.NewPostgresRepository(db)
	svc := posts.NewService(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("POST /posts", handleCreatePost(svc))

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}
	log.Printf("entries: server started on port %s", cfg.Port)
	log.Fatal(server.ListenAndServe())
}

func openDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleCreatePost(svc *posts.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Title string `json:"title"`
			Slug  string `json:"slug"`
			S3Key string `json:"s3_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.Title == "" || req.Slug == "" {
			http.Error(w, "title and slug are required", http.StatusBadRequest)
			return
		}

		post, err := svc.CreatePost(r.Context(), req.Title, req.Slug, req.S3Key)
		if err != nil {
			log.Printf("create post: %v", err)
			http.Error(w, "failed to create post", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(post)
	}
}
