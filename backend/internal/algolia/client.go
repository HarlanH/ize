package algolia

import (
	"context"
	"encoding/json"
	"fmt"

	"ize/internal/config"
	"ize/internal/logger"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
)

// ptr returns a pointer to the given value (helper for inline pointer creation)
func ptr[T any](v T) *T {
	return &v
}

// Client wraps the Algolia search client
type Client struct {
	client         *search.APIClient
	indexName      string
	logger         *logger.Logger
	fieldMapping   *config.FieldMapping
	facetFields    []string
	facetFieldsSet map[string]bool // For quick lookup when filtering hit facets
}

// NewClient creates a new Algolia client
func NewClient(appID, apiKey, indexName string, log *logger.Logger) (*Client, error) {
	return NewClientWithConfig(appID, apiKey, indexName, nil, nil, log)
}

// NewClientWithConfig creates a new Algolia client with field mapping and facet configuration
func NewClientWithConfig(appID, apiKey, indexName string, fieldMapping *config.FieldMapping, facetFields []string, log *logger.Logger) (*Client, error) {
	client, err := search.NewClient(appID, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create algolia client: %w", err)
	}

	// Default to all facets if none specified
	if len(facetFields) == 0 {
		facetFields = []string{"*"}
	}

	// Build set for quick lookup when filtering hit facets
	facetFieldsSet := make(map[string]bool)
	for _, f := range facetFields {
		if f != "*" {
			facetFieldsSet[f] = true
		}
	}

	log.Info("algolia client initialized",
		"app_id", appID,
		"index_name", indexName,
		"field_mapping_configured", fieldMapping != nil,
		"facet_fields_count", len(facetFields),
	)

	return &Client{
		client:         client,
		indexName:      indexName,
		logger:         log,
		fieldMapping:   fieldMapping,
		facetFields:    facetFields,
		facetFieldsSet: facetFieldsSet,
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

// extractHitFields extracts name, description, and image from raw hit data using field mapping
func (c *Client) extractHitFields(rawHit map[string]interface{}) Hit {
	hit := Hit{
		Facets: make(map[string]interface{}),
	}

	// Always extract objectID
	if objID, ok := rawHit["objectID"].(string); ok {
		hit.ObjectID = objID
	}

	// Use field mapping if configured, otherwise fall back to direct field access
	if c.fieldMapping != nil {
		hit.Name = config.ExtractField(rawHit, c.fieldMapping.Name)
		hit.Description = config.ExtractField(rawHit, c.fieldMapping.Description)
		hit.Image = config.ExtractField(rawHit, c.fieldMapping.Image)
	} else {
		// Legacy behavior: direct field access
		if name, ok := rawHit["name"].(string); ok {
			hit.Name = name
		}
		if desc, ok := rawHit["description"].(string); ok {
			hit.Description = desc
		}
		if img, ok := rawHit["image"].(string); ok {
			hit.Image = img
		}
	}

	// Store facet fields in Facets map
	// If specific facets are configured, extract those nested paths and store with full path as key
	// Otherwise (facetFieldsSet is empty, meaning "*" was used), include all top-level fields
	if len(c.facetFieldsSet) > 0 {
		// Extract configured facet values using their nested paths
		for facetField := range c.facetFieldsSet {
			value := config.ExtractFieldValue(rawHit, facetField)
			if value != nil {
				hit.Facets[facetField] = value
			}
		}
	} else {
		// Include all top-level fields when no specific facets configured (legacy behavior)
		knownFields := map[string]bool{
			"objectID":         true,
			"_highlightResult": true,
			"_snippetResult":   true,
			"_rankingInfo":     true,
		}
		for key, value := range rawHit {
			if !knownFields[key] {
				hit.Facets[key] = value
			}
		}
	}

	return hit
}

// SearchResult represents the full search response from Algolia
type SearchResult struct {
	Hits      []Hit                       `json:"hits"`
	Facets    map[string]map[string]int32 `json:"facets,omitempty"`
	TotalHits int                         `json:"nbHits"` // Total number of matching records
}

// Search performs a search query against Algolia
func (c *Client) Search(ctx context.Context, query string, facetFilters [][]string) (*SearchResult, error) {
	log := c.logger.WithContext(ctx)

	log.Debug("executing algolia search",
		"query", query,
		"facet_filters", facetFilters,
		"index_name", c.indexName,
		"facet_fields", c.facetFields,
	)

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

	// Request all attributes to be retrieved so field mapping can access any field
	attributesToRetrieve := []string{"*"}
	// Disable highlighting to avoid SDK unmarshalling issues with complex highlight results
	attributesToHighlight := []string{}

	searchParamsObject := search.SearchParamsObject{
		Query:                 &query,
		Facets:                c.facetFields,
		FacetFilters:          facetFiltersParam,
		AttributesToRetrieve:  attributesToRetrieve,
		AttributesToHighlight: attributesToHighlight,
		Analytics:             ptr(false), // Disable analytics to avoid corrupting production metrics
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
		// Marshal the hits to JSON and then unmarshal into raw maps
		hitsJSON, err := json.Marshal(res.Hits)
		if err != nil {
			log.ErrorWithErr("failed to marshal algolia hits", err,
				"query", query,
			)
			return nil, fmt.Errorf("failed to marshal hits: %w", err)
		}

		var rawHits []map[string]interface{}
		if err := json.Unmarshal(hitsJSON, &rawHits); err != nil {
			log.ErrorWithErr("failed to unmarshal algolia hits", err,
				"query", query,
			)
			return nil, fmt.Errorf("failed to unmarshal hits: %w", err)
		}

		// Convert to Hit structs using field mapping
		hits = make([]Hit, 0, len(rawHits))
		for _, rawHit := range rawHits {
			hits = append(hits, c.extractHitFields(rawHit))
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
		Hits:      hits,
		Facets:    facets,
		TotalHits: int(res.NbHits),
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
		"facet_fields", c.facetFields,
	)

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
	// Disable highlighting to avoid SDK unmarshalling issues with complex highlight results
	attributesToHighlight := []string{}

	searchParamsObject := search.SearchParamsObject{
		Query:                 &query,
		Facets:                c.facetFields,
		FacetFilters:          facetFiltersParam,
		HitsPerPage:           &hitsPerPage,
		AttributesToRetrieve:  attributesToRetrieve,
		AttributesToHighlight: attributesToHighlight,
		Analytics:             ptr(false), // Disable analytics to avoid corrupting production metrics
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

		// Convert to Hit structs using field mapping
		hits = make([]Hit, 0, len(rawHits))
		for _, rawHit := range rawHits {
			hits = append(hits, c.extractHitFields(rawHit))
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
		Hits:      hits,
		Facets:    facets,
		TotalHits: int(res.NbHits),
	}, nil
}
