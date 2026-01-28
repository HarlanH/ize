package algolia

import "context"

// ClientInterface defines the interface for Algolia operations
// This allows us to mock the client in tests
type ClientInterface interface {
	// facetFilters is interpreted as AND across outer slices and OR within each inner slice.
	// Example: [["brand:Apple","brand:Samsung"],["category:Phone"]] means
	// (brand:Apple OR brand:Samsung) AND (category:Phone).
	Search(ctx context.Context, query string, facetFilters [][]string) (*SearchResult, error)
	// SearchRipper performs a search with 100 hits per page for RIPPER algorithm
	SearchRipper(ctx context.Context, query string, facetFilters [][]string) (*SearchResult, error)
}
