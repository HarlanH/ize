package httpapi

import (
	"encoding/json"
	"net/http"

	"ize/internal/algolia"
	"ize/internal/config"
	"ize/internal/ize"
	"ize/internal/logger"
)

type SearchHandler struct {
	algoliaClient algolia.ClientInterface
	logger        *logger.Logger
}

func NewSearchHandler(cfg *config.Config, log *logger.Logger) (*SearchHandler, error) {
	client, err := algolia.NewClient(cfg.AlgoliaAppID, cfg.AlgoliaAPIKey, cfg.AlgoliaIndexName, log)
	if err != nil {
		return nil, err
	}

	return &SearchHandler{
		algoliaClient: client,
		logger:        log,
	}, nil
}

func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	log := h.logger.WithContext(r.Context())
	
	if r.Method != http.MethodPost {
		log.Warn("method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.ErrorWithErr("failed to decode request body", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		log.Warn("empty query received")
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	log.Debug("processing search request", "query", req.Query)

	// Search Algolia
	algoliaResults, err := h.algoliaClient.Search(r.Context(), req.Query)
	if err != nil {
		log.ErrorWithErr("algolia search failed", err, "query", req.Query)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	log.Debug("algolia search completed",
		"query", req.Query,
		"hits_count", len(algoliaResults.Hits),
	)

	// Process through ize module
	izeResults := ize.Process(req.Query, algoliaResults)

	log.Debug("ize processing completed",
		"query", req.Query,
		"results_count", len(izeResults),
	)

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
		log.ErrorWithErr("failed to encode response", err, "query", req.Query)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info("search request completed successfully",
		"query", req.Query,
		"results_count", len(results),
	)
}
