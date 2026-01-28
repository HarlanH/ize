package algolia

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

// Client wraps the Algolia search client
type Client struct {
	client     *search.APIClient
	indexName  string
}

// NewClient creates a new Algolia client
func NewClient(appID, apiKey, indexName string) (*Client, error) {
	client, err := search.NewClient(appID, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create algolia client: %w", err)
	}

	return &Client{
		client:    client,
		indexName: indexName,
	}, nil
}

// Hit represents a single search result from Algolia
type Hit struct {
	ObjectID    string                 `json:"objectID"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Image       string                 `json:"image"`
	Facets      map[string]interface{} `json:"facets,omitempty"`
}

// SearchResult represents the full search response from Algolia
type SearchResult struct {
	Hits []Hit `json:"hits"`
}

// Search performs a search query against Algolia
func (c *Client) Search(ctx context.Context, query string) (*SearchResult, error) {
	// Create SearchParamsObject with the query
	searchParamsObject := search.SearchParamsObject{
		Query: &query,
	}
	searchParams := search.SearchParamsObjectAsSearchParams(&searchParamsObject)
	
	// Create the request with the index name using the proper API method
	request := c.client.NewApiSearchSingleIndexRequest(c.indexName)
	
	// Use WithSearchParams to set the search parameters
	request = request.WithSearchParams(searchParams)
	
	res, err := c.client.SearchSingleIndex(request)
	if err != nil {
		return nil, fmt.Errorf("algolia search failed: %w", err)
	}

	// Extract hits from the response using JSON marshaling/unmarshaling
	var hits []Hit
	if res.Hits != nil {
		// Marshal the hits to JSON and then unmarshal into our Hit structs
		hitsJSON, err := json.Marshal(res.Hits)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal hits: %w", err)
		}
		
		if err := json.Unmarshal(hitsJSON, &hits); err != nil {
			return nil, fmt.Errorf("failed to unmarshal hits: %w", err)
		}
	}

	return &SearchResult{Hits: hits}, nil
}
