package httpapi

// SearchRequest represents the incoming search request
type SearchRequest struct {
	Query        string     `json:"query"`
	FacetFilters [][]string `json:"facetFilters,omitempty"`
}

// FacetMeta provides display metadata for a facet field
type FacetMeta struct {
	Field        string `json:"field"`                  // Algolia facet field name
	DisplayName  string `json:"displayName"`            // User-friendly name for UI
	RemovePrefix string `json:"removePrefix,omitempty"` // Optional prefix to strip from facet values
}

// SearchResponse represents the search response
type SearchResponse struct {
	Hits      []SearchResult              `json:"hits"`
	Facets    map[string]map[string]int32 `json:"facets,omitempty"`
	FacetMeta []FacetMeta                 `json:"facetMeta,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

// RipperGroup represents a group of items sharing a facet value
type RipperGroup struct {
	FacetName  string         `json:"facetName"`
	FacetValue string         `json:"facetValue"`
	Items      []SearchResult `json:"items"`
	Count      int            `json:"count"` // Accurate count from Algolia facets
}

// RipperResponse represents the RIPPER algorithm response
type RipperResponse struct {
	Groups     []RipperGroup  `json:"groups"`
	OtherGroup []SearchResult `json:"otherGroup"`
	FacetMeta  []FacetMeta    `json:"facetMeta,omitempty"`
}

// FacetCount represents a facet:value pair with its count and percentage
type FacetCount struct {
	FacetName  string  `json:"facetName"`
	FacetValue string  `json:"facetValue"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// RuleQuality represents quality metrics for a cluster's decision list rule
type RuleQuality struct {
	Precision float64 `json:"precision"` // Fraction of rule matches that are true cluster members
	Recall    float64 `json:"recall"`    // Fraction of cluster members that match the rule
	F1        float64 `json:"f1"`        // Harmonic mean of precision and recall
}

// ClusterGroup represents a cluster of items with similar facet profiles
type ClusterGroup struct {
	Name            string         `json:"name"` // LLM-generated label
	Items           []SearchResult `json:"items"`
	Percentage      float64        `json:"percentage"`                // Approximate percentage of total results (~X%)
	TopFacets       []FacetCount   `json:"topFacets"`                 // For transparency
	Rule            [][]string     `json:"rule,omitempty"`            // Algolia filter format for "load more"
	RuleDescription string         `json:"ruleDescription,omitempty"` // Human-readable rule
	RuleQuality     *RuleQuality   `json:"ruleQuality,omitempty"`     // Rule quality metrics
}

// ClusterResponse represents the clustering algorithm response
type ClusterResponse struct {
	Groups       []ClusterGroup `json:"groups"`
	OtherGroup   []SearchResult `json:"otherGroup"`
	ClusterCount int            `json:"clusterCount"` // Selected k value
	TotalHits    int            `json:"totalHits"`    // Total matching records from Algolia
}
