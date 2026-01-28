package main

import (
	"fmt"
	"net/http"

	"ize/internal/config"
	"ize/internal/httpapi"
	"ize/internal/logger"
)

// corsMiddleware adds CORS headers to allow requests from the frontend
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func main() {
	log := logger.Default()
	
	cfg, err := config.Load()
	if err != nil {
		log.ErrorWithErr("failed to load configuration", err)
		panic(err)
	}

	log.Info("configuration loaded successfully",
		"port", cfg.Port,
		"algolia_app_id", cfg.AlgoliaAppID,
		"algolia_index", cfg.AlgoliaIndexName,
	)

	mux := http.NewServeMux()
	
	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Search endpoint
	searchHandler, err := httpapi.NewSearchHandler(cfg, log)
	if err != nil {
		log.ErrorWithErr("failed to create search handler", err)
		panic(err)
	}
	
	log.Info("search handler initialized")
	
	// Handle both with and without trailing slash, and handle OPTIONS preflight
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			// Handle preflight
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		searchHandler.HandleSearch(w, r)
	})

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Info("server starting", "address", addr)
	
	// Chain middleware: request ID logging -> CORS -> mux
	handler := logger.RequestIDMiddleware(log, corsMiddleware(mux))
	
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.ErrorWithErr("server failed to start", err, "address", addr)
		panic(err)
	}
}
