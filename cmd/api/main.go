package main

import (
	"log"
	"net/http"

	"github.com/jeremyjsx/entries/internal/config"
)

func main() {
	config := config.Load()
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: mux,
	}

	log.Printf("entries: server started on port %s", config.Port)
	log.Fatal(server.ListenAndServe())
}
