package algolia

import "context"

// ClientInterface defines the interface for Algolia operations
// This allows us to mock the client in tests
type ClientInterface interface {
	Search(ctx context.Context, query string) (*SearchResult, error)
}
