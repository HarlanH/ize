package algolia

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"ize/internal/logger"
)

// Client wraps the Algolia search client
type Client struct {
	client     *search.APIClient
	indexName  string
	logger     *logger.Logger
}

// NewClient creates a new Algolia client
func NewClient(appID, apiKey, indexName string, log *logger.Logger) (*Client, error) {
	client, err := search.NewClient(appID, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create algolia client: %w", err)
	}

	log.Info("algolia client initialized",
		"app_id", appID,
		"index_name", indexName,
	)

	return &Client{
		client:    client,
		indexName: indexName,
		logger:    log,
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
	log := c.logger.WithContext(ctx)
	
	log.Debug("executing algolia search",
		"query", query,
		"index_name", c.indexName,
	)
	
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
		log.ErrorWithErr("algolia search API call failed", err,
			"query", query,
			"index_name", c.indexName,
		)
		return nil, fmt.Errorf("algolia search failed: %w", err)
	}

	// Extract hits from the response using JSON marshaling/unmarshaling
	var hits []Hit
	if res.Hits != nil {
		// Marshal the hits to JSON and then unmarshal into our Hit structs
		hitsJSON, err := json.Marshal(res.Hits)
		if err != nil {
			log.ErrorWithErr("failed to marshal algolia hits", err,
				"query", query,
			)
			return nil, fmt.Errorf("failed to marshal hits: %w", err)
		}
		
		if err := json.Unmarshal(hitsJSON, &hits); err != nil {
			log.ErrorWithErr("failed to unmarshal algolia hits", err,
				"query", query,
			)
			return nil, fmt.Errorf("failed to unmarshal hits: %w", err)
		}
	}

	log.Debug("algolia search completed successfully",
		"query", query,
		"hits_count", len(hits),
	)

	return &SearchResult{Hits: hits}, nil
}
