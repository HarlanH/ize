package httpapi

import (
	"encoding/json"
	"net/http"

	"ize/internal/algolia"
	"ize/internal/anthropic"
	"ize/internal/config"
	"ize/internal/ize"
	"ize/internal/logger"
)

type SearchHandler struct {
	algoliaClient   algolia.ClientInterface
	anthropicClient anthropic.ClientInterface
	logger          *logger.Logger
}

func NewSearchHandler(cfg *config.Config, log *logger.Logger) (*SearchHandler, error) {
	algoliaClient, err := algolia.NewClient(cfg.AlgoliaAppID, cfg.AlgoliaAPIKey, cfg.AlgoliaIndexName, log)
	if err != nil {
		return nil, err
	}

	// Anthropic client is optional - cluster naming will use fallback if not configured
	var anthropicClient anthropic.ClientInterface
	if cfg.AnthropicAPIKey != "" {
		anthropicClient, err = anthropic.NewClient(cfg.AnthropicAPIKey, log)
		if err != nil {
			log.Warn("failed to create anthropic client, cluster naming will use fallback", "error", err)
		}
	} else {
		log.Info("anthropic API key not configured, cluster naming will use fallback labels")
	}

	return &SearchHandler{
		algoliaClient:   algoliaClient,
		anthropicClient: anthropicClient,
		logger:          log,
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

	log.Debug("processing search request",
		"query", req.Query,
		"facet_filters", req.FacetFilters,
	)

	// Search Algolia
	algoliaResults, err := h.algoliaClient.Search(r.Context(), req.Query, req.FacetFilters)
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
		Hits:   results,
		Facets: algoliaResults.Facets,
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

func (h *SearchHandler) HandleRipper(w http.ResponseWriter, r *http.Request) {
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

	log.Debug("processing RIPPER request",
		"query", req.Query,
		"facet_filters", req.FacetFilters,
	)

	// Search Algolia with 100 hits per page
	algoliaResults, err := h.algoliaClient.SearchRipper(r.Context(), req.Query, req.FacetFilters)
	if err != nil {
		log.ErrorWithErr("algolia search failed for RIPPER", err, "query", req.Query)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	log.Debug("algolia search completed for RIPPER",
		"query", req.Query,
		"hits_count", len(algoliaResults.Hits),
	)

	// Process through RIPPER algorithm
	ripperResult, err := ize.ProcessRipper(req.Query, algoliaResults, log)
	if err != nil {
		log.ErrorWithErr("RIPPER processing failed", err, "query", req.Query)
		http.Error(w, "RIPPER processing failed", http.StatusInternalServerError)
		return
	}

	log.Debug("RIPPER processing completed",
		"query", req.Query,
		"groups_count", len(ripperResult.Groups),
		"other_group_count", len(ripperResult.OtherGroup),
	)

	// Convert ize.RipperGroup to httpapi.RipperGroup
	groups := make([]RipperGroup, len(ripperResult.Groups))
	for i, group := range ripperResult.Groups {
		items := make([]SearchResult, len(group.Items))
		for j, item := range group.Items {
			items[j] = SearchResult{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				Image:       item.Image,
			}
		}
		groups[i] = RipperGroup{
			FacetName:  group.FacetName,
			FacetValue: group.FacetValue,
			Items:      items,
			// Use the TotalCount from the algorithm so the count shown
			// in the UI reflects all items with this facet value in the
			// current (possibly filtered) result set, not just the
			// remaining unassigned items when the group was selected.
			Count:      group.TotalCount,
		}
	}

	// Convert ize.Result to httpapi.SearchResult for Other group
	otherGroup := make([]SearchResult, len(ripperResult.OtherGroup))
	for i, item := range ripperResult.OtherGroup {
		otherGroup[i] = SearchResult{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Image:       item.Image,
		}
	}

	response := RipperResponse{
		Groups:     groups,
		OtherGroup: otherGroup,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.ErrorWithErr("failed to encode RIPPER response", err, "query", req.Query)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info("RIPPER request completed successfully",
		"query", req.Query,
		"groups_count", len(groups),
		"other_group_count", len(otherGroup),
	)
}

func (h *SearchHandler) HandleCluster(w http.ResponseWriter, r *http.Request) {
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

	log.Debug("processing Cluster request",
		"query", req.Query,
		"facet_filters", req.FacetFilters,
	)

	// Search Algolia with 100 hits per page (same as RIPPER)
	algoliaResults, err := h.algoliaClient.SearchRipper(r.Context(), req.Query, req.FacetFilters)
	if err != nil {
		log.ErrorWithErr("algolia search failed for Cluster", err, "query", req.Query)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	log.Debug("algolia search completed for Cluster",
		"query", req.Query,
		"hits_count", len(algoliaResults.Hits),
	)

	// Process through clustering algorithm
	clusterResult, err := ize.ProcessCluster(req.Query, algoliaResults, log)
	if err != nil {
		log.ErrorWithErr("Cluster processing failed", err, "query", req.Query)
		http.Error(w, "Cluster processing failed", http.StatusInternalServerError)
		return
	}

	log.Debug("Cluster processing completed",
		"query", req.Query,
		"cluster_count", clusterResult.ClusterCount,
		"other_group_count", len(clusterResult.OtherGroup),
	)

	// Generate LLM-based cluster names if Anthropic client is available
	if h.anthropicClient != nil && len(clusterResult.Groups) > 0 {
		statsSlice := make([]anthropic.ClusterStats, len(clusterResult.Groups))
		for i, group := range clusterResult.Groups {
			facetInfos := make([]anthropic.FacetInfo, len(group.TopFacets))
			for j, f := range group.TopFacets {
				facetInfos[j] = anthropic.FacetInfo{
					Name:       f.FacetName,
					Value:      f.FacetValue,
					Percentage: f.Percentage,
				}
			}
			statsSlice[i] = anthropic.ClusterStats{
				Size:      group.Stats.Size,
				TopFacets: facetInfos,
			}
		}

		names, err := h.anthropicClient.GenerateClusterNames(r.Context(), statsSlice)
		if err != nil {
			log.Warn("failed to generate cluster names, using fallbacks", "error", err)
		} else {
			for i, name := range names {
				if i < len(clusterResult.Groups) {
					clusterResult.Groups[i].Name = name
				}
			}
		}
	}

	// Convert ize.ClusterGroup to httpapi.ClusterGroup
	groups := make([]ClusterGroup, len(clusterResult.Groups))
	for i, group := range clusterResult.Groups {
		items := make([]SearchResult, len(group.Items))
		for j, item := range group.Items {
			items[j] = SearchResult{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				Image:       item.Image,
			}
		}

		topFacets := make([]FacetCount, len(group.TopFacets))
		for j, f := range group.TopFacets {
			topFacets[j] = FacetCount{
				FacetName:  f.FacetName,
				FacetValue: f.FacetValue,
				Count:      f.Count,
				Percentage: f.Percentage,
			}
		}

		groups[i] = ClusterGroup{
			Name:      group.Name,
			Items:     items,
			TopFacets: topFacets,
		}
	}

	// Convert ize.Result to httpapi.SearchResult for Other group
	otherGroup := make([]SearchResult, len(clusterResult.OtherGroup))
	for i, item := range clusterResult.OtherGroup {
		otherGroup[i] = SearchResult{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Image:       item.Image,
		}
	}

	response := ClusterResponse{
		Groups:       groups,
		OtherGroup:   otherGroup,
		ClusterCount: clusterResult.ClusterCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.ErrorWithErr("failed to encode Cluster response", err, "query", req.Query)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info("Cluster request completed successfully",
		"query", req.Query,
		"cluster_count", len(groups),
		"other_group_count", len(otherGroup),
	)
}
