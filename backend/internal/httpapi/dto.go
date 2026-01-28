package httpapi

// SearchRequest represents the incoming search request
type SearchRequest struct {
	Query        string   `json:"query"`
	FacetFilters [][]string `json:"facetFilters,omitempty"`
}

// SearchResponse represents the search response
type SearchResponse struct {
	Hits   []SearchResult             `json:"hits"`
	Facets map[string]map[string]int32 `json:"facets,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
}
