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
	Hits   []Hit                       `json:"hits"`
	Facets map[string]map[string]int32 `json:"facets,omitempty"`
}

// Search performs a search query against Algolia
func (c *Client) Search(ctx context.Context, query string, facetFilters [][]string) (*SearchResult, error) {
	log := c.logger.WithContext(ctx)
	
	log.Debug("executing algolia search",
		"query", query,
		"facet_filters", facetFilters,
		"index_name", c.indexName,
	)
	
	// Create SearchParamsObject with the query.
	// Request all facets so the UI can render facet counts.
	// (Later we can scope this list to a configured set of facets.)
	allFacets := []string{"*"}

	var facetFiltersParam *search.FacetFilters
	if len(facetFilters) > 0 {
		// Represent `[[a,b], c]` style where outer array is AND and inner arrays are OR.
		// Algolia v4 SDK uses a union type for this.
		outer := make([]search.FacetFilters, 0, len(facetFilters))
		for _, group := range facetFilters {
			if len(group) == 0 {
				continue
			}
			if len(group) == 1 {
				outer = append(outer, *search.StringAsFacetFilters(group[0]))
				continue
			}

			inner := make([]search.FacetFilters, 0, len(group))
			for _, f := range group {
				inner = append(inner, *search.StringAsFacetFilters(f))
			}
			outer = append(outer, *search.ArrayOfFacetFiltersAsFacetFilters(inner))
		}
		if len(outer) > 0 {
			facetFiltersParam = search.ArrayOfFacetFiltersAsFacetFilters(outer)
		}
	}
	searchParamsObject := search.SearchParamsObject{
		Query: &query,
		Facets: allFacets,
		FacetFilters: facetFiltersParam,
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

	var facets map[string]map[string]int32
	if res.Facets != nil {
		facets = *res.Facets
	}

	return &SearchResult{
		Hits:   hits,
		Facets: facets,
	}, nil
}

// SearchRipper performs a search query against Algolia with 100 hits per page for RIPPER algorithm
func (c *Client) SearchRipper(ctx context.Context, query string, facetFilters [][]string) (*SearchResult, error) {
	log := c.logger.WithContext(ctx)
	
	log.Debug("executing algolia search for RIPPER",
		"query", query,
		"facet_filters", facetFilters,
		"index_name", c.indexName,
		"hits_per_page", 100,
	)
	
	// Create SearchParamsObject with the query and 100 hits per page
	allFacets := []string{"*"}
	hitsPerPage := int32(100)

	var facetFiltersParam *search.FacetFilters
	if len(facetFilters) > 0 {
		outer := make([]search.FacetFilters, 0, len(facetFilters))
		for _, group := range facetFilters {
			if len(group) == 0 {
				continue
			}
			if len(group) == 1 {
				outer = append(outer, *search.StringAsFacetFilters(group[0]))
				continue
			}

			inner := make([]search.FacetFilters, 0, len(group))
			for _, f := range group {
				inner = append(inner, *search.StringAsFacetFilters(f))
			}
			outer = append(outer, *search.ArrayOfFacetFiltersAsFacetFilters(inner))
		}
		if len(outer) > 0 {
			facetFiltersParam = search.ArrayOfFacetFiltersAsFacetFilters(outer)
		}
	}
	// Request all attributes to be retrieved so facet values are included in hits
	attributesToRetrieve := []string{"*"}
	
	searchParamsObject := search.SearchParamsObject{
		Query:              &query,
		Facets:             allFacets,
		FacetFilters:       facetFiltersParam,
		HitsPerPage:        &hitsPerPage,
		AttributesToRetrieve: attributesToRetrieve,
	}
	searchParams := search.SearchParamsObjectAsSearchParams(&searchParamsObject)
	
	request := c.client.NewApiSearchSingleIndexRequest(c.indexName)
	request = request.WithSearchParams(searchParams)
	
	res, err := c.client.SearchSingleIndex(request)
	if err != nil {
		log.ErrorWithErr("algolia search API call failed for RIPPER", err,
			"query", query,
			"index_name", c.indexName,
		)
		return nil, fmt.Errorf("algolia search failed: %w", err)
	}

	var hits []Hit
	if res.Hits != nil {
		hitsJSON, err := json.Marshal(res.Hits)
		if err != nil {
			log.ErrorWithErr("failed to marshal algolia hits", err,
				"query", query,
			)
			return nil, fmt.Errorf("failed to marshal hits: %w", err)
		}
		
		// Unmarshal into a slice of maps first to capture all fields
		var rawHits []map[string]interface{}
		if err := json.Unmarshal(hitsJSON, &rawHits); err != nil {
			log.ErrorWithErr("failed to unmarshal algolia hits", err,
				"query", query,
			)
			return nil, fmt.Errorf("failed to unmarshal hits: %w", err)
		}
		
		// Convert to Hit structs, preserving all attributes in Facets map
		hits = make([]Hit, 0, len(rawHits))
		for _, rawHit := range rawHits {
			hit := Hit{
				Facets: make(map[string]interface{}),
			}
			
			// Extract known fields
			if objID, ok := rawHit["objectID"].(string); ok {
				hit.ObjectID = objID
			}
			if name, ok := rawHit["name"].(string); ok {
				hit.Name = name
			}
			if desc, ok := rawHit["description"].(string); ok {
				hit.Description = desc
			}
			if img, ok := rawHit["image"].(string); ok {
				hit.Image = img
			}
			
			// Store all other fields (including facet attributes) in Facets map
			knownFields := map[string]bool{
				"objectID": true,
				"name":     true,
				"description": true,
				"image":    true,
				"_highlightResult": true,
				"_snippetResult": true,
				"_rankingInfo": true,
			}
			for key, value := range rawHit {
				if !knownFields[key] {
					hit.Facets[key] = value
				}
			}
			
			hits = append(hits, hit)
		}
	}

	log.Debug("algolia search completed successfully for RIPPER",
		"query", query,
		"hits_count", len(hits),
	)

	var facets map[string]map[string]int32
	if res.Facets != nil {
		facets = *res.Facets
	}

	return &SearchResult{
		Hits:   hits,
		Facets: facets,
	}, nil
}
