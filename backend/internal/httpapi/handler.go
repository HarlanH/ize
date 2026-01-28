package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"ize/internal/algolia"
	"ize/internal/config"
	"ize/internal/ize"
)

type SearchHandler struct {
	algoliaClient algolia.ClientInterface
}

func NewSearchHandler(cfg *config.Config) (*SearchHandler, error) {
	client, err := algolia.NewClient(cfg.AlgoliaAppID, cfg.AlgoliaAPIKey, cfg.AlgoliaIndexName)
	if err != nil {
		return nil, err
	}

	return &SearchHandler{
		algoliaClient: client,
	}, nil
}

func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Search Algolia
	algoliaResults, err := h.algoliaClient.Search(r.Context(), req.Query)
	if err != nil {
		log.Printf("Search failed: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Process through ize module
	izeResults := ize.Process(req.Query, algoliaResults)

	// Convert ize.Result to httpapi.SearchResult
	results := make([]SearchResult, len(izeResults))
	for i, r := range izeResults {
		results[i] = SearchResult{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Image:       r.Image,
		}
	}

	response := SearchResponse{
		Hits: results,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
