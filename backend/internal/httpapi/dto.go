package httpapi

// SearchRequest represents the incoming search request
type SearchRequest struct {
	Query string `json:"query"`
}

// SearchResponse represents the search response
type SearchResponse struct {
	Hits []SearchResult `json:"hits"`
}

// SearchResult represents a single search result
type SearchResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
}
