package ize

import (
	"fmt"
	"sort"

	"ize/internal/algolia"
	"ize/internal/logger"
)

// ClusterGroup represents a cluster of items with similar facet profiles
type ClusterGroup struct {
	Name        string        // LLM-generated label (or fallback)
	Items       []Result      // Items in this cluster
	TopFacets   []FacetCount  // Most common facet:value pairs in this cluster
	Stats       ClusterStats  // Statistics for LLM labeling
	Rule        *DecisionList // Filter rule that defines this cluster (nil if not fitted)
	RuleQuality *RuleQuality  // Quality metrics for the fitted rule (nil if not fitted)
}

// FacetCount represents a facet:value pair with its count and percentage
type FacetCount struct {
	FacetName  string
	FacetValue string
	Count      int
	Percentage float64
}

// ClusterStats holds statistics about a cluster for LLM labeling
type ClusterStats struct {
	Size      int
	TopFacets []FacetCount
}

// ClusterResult represents the output of the clustering algorithm
type ClusterResult struct {
	Groups       []ClusterGroup
	OtherGroup   []Result
	ClusterCount int // The selected k value
}

// FacetSet represents an item's facets as a set of "facetName:facetValue" strings
type FacetSet map[string]bool

// Minimum cluster size - clusters smaller than this go to "Other"
const minClusterSize = 2

// ProcessCluster implements facet-space clustering using Jaccard similarity
// and agglomerative hierarchical clustering with silhouette-based k selection
func ProcessCluster(query string, algoliaResults *algolia.SearchResult, log *logger.Logger) (*ClusterResult, error) {
	if log == nil {
		log = logger.Default()
	}

	log.Debug("ProcessCluster started",
		"query", query,
		"hits_count", hitsCount(algoliaResults),
	)

	if algoliaResults == nil || len(algoliaResults.Hits) == 0 {
		log.Debug("ProcessCluster: empty results, returning empty groups")
		return &ClusterResult{
			Groups:       []ClusterGroup{},
			OtherGroup:   []Result{},
			ClusterCount: 0,
		}, nil
	}

	// Convert Algolia hits to Results and extract facet sets
	allItems, facetSets := extractItemsAndFacets(algoliaResults)

	totalItems := len(allItems)
	log.Debug("ProcessCluster: extracted facet sets", "total_items", totalItems)

	// Handle edge cases
	if result := handleEdgeCases(allItems, facetSets, log); result != nil {
		return result, nil
	}

	// Build distance matrix and find optimal clustering
	distMatrix := buildDistanceMatrix(facetSets)
	log.Debug("ProcessCluster: built distance matrix", "matrix_size", len(distMatrix))

	optimalK, assignments, silhouetteScores := selectOptimalK(distMatrix, facetSets, log)
	logSilhouetteScores(log, silhouetteScores, optimalK)

	// Build cluster groups from similarity clustering
	groups, otherItems := buildClusterGroups(allItems, facetSets, assignments, optimalK, log)
	log.Debug("ProcessCluster: similarity clustering complete",
		"initial_clusters", len(groups),
		"other_count", len(otherItems),
	)

	// Fit decision list rules and reassign items based on rules
	groups = fitAndReassign(groups, allItems, facetSets, log)

	actualClusterCount := len(groups)
	log.Info("ProcessCluster: completed",
		"selected_k", optimalK,
		"actual_clusters", actualClusterCount,
		"other_count", len(otherItems),
	)

	return &ClusterResult{
		Groups:       groups,
		OtherGroup:   otherItems,
		ClusterCount: actualClusterCount,
	}, nil
}

// hitsCount safely returns the number of hits
func hitsCount(results *algolia.SearchResult) int {
	if results == nil {
		return 0
	}
	return len(results.Hits)
}

// extractItemsAndFacets converts Algolia hits to Results and extracts facet sets
func extractItemsAndFacets(algoliaResults *algolia.SearchResult) ([]Result, []FacetSet) {
	allItems := make([]Result, 0, len(algoliaResults.Hits))
	facetSets := make([]FacetSet, 0, len(algoliaResults.Hits))

	for _, hit := range algoliaResults.Hits {
		allItems = append(allItems, Result{
			ID:          hit.ObjectID,
			Name:        hit.Name,
			Description: hit.Description,
			Image:       hit.Image,
		})
		facetSets = append(facetSets, extractFacetSet(hit))
	}

	return allItems, facetSets
}

// handleEdgeCases checks for conditions that prevent clustering
func handleEdgeCases(allItems []Result, facetSets []FacetSet, log *logger.Logger) *ClusterResult {
	if len(allItems) < 2 {
		log.Debug("ProcessCluster: too few items for clustering")
		return &ClusterResult{
			Groups:       []ClusterGroup{},
			OtherGroup:   allItems,
			ClusterCount: 0,
		}
	}

	if !hasAnyFacets(facetSets) {
		log.Debug("ProcessCluster: no items have facets, returning all as Other")
		return &ClusterResult{
			Groups:       []ClusterGroup{},
			OtherGroup:   allItems,
			ClusterCount: 0,
		}
	}

	return nil
}

// hasAnyFacets checks if any items have facets
func hasAnyFacets(facetSets []FacetSet) bool {
	for _, fs := range facetSets {
		if len(fs) > 0 {
			return true
		}
	}
	return false
}

// logSilhouetteScores logs silhouette scores prominently for debugging
func logSilhouetteScores(log *logger.Logger, scores map[int]float64, selectedK int) {
	log.Info("ProcessCluster: silhouette scores by k",
		"k=2", fmt.Sprintf("%.3f", scores[2]),
		"k=3", fmt.Sprintf("%.3f", scores[3]),
		"k=4", fmt.Sprintf("%.3f", scores[4]),
		"k=5", fmt.Sprintf("%.3f", scores[5]),
		"k=6", fmt.Sprintf("%.3f", scores[6]),
		"selected_k", selectedK,
	)
}

// extractFacetSet converts a hit's facets to a set of "facetName:facetValue" strings
func extractFacetSet(hit algolia.Hit) FacetSet {
	fs := make(FacetSet)
	if hit.Facets == nil {
		return fs
	}

	for facetName, facetValue := range hit.Facets {
		if facetValue == nil {
			continue
		}
		addFacetValues(fs, facetName, facetValue)
	}

	return fs
}

// addFacetValues adds facet values to the facet set, handling different types
func addFacetValues(fs FacetSet, facetName string, facetValue interface{}) {
	var values []string
	switch v := facetValue.(type) {
	case string:
		values = []string{v}
	case []interface{}:
		for _, val := range v {
			if str, ok := val.(string); ok {
				values = append(values, str)
			}
		}
	default:
		return
	}

	for _, value := range values {
		key := fmt.Sprintf("%s:%s", facetName, value)
		fs[key] = true
	}
}

// buildClusterGroups creates ClusterGroup objects from cluster assignments
// Clusters with fewer than minClusterSize items are moved to "Other"
func buildClusterGroups(allItems []Result, facetSets []FacetSet, assignments []int, k int, log *logger.Logger) ([]ClusterGroup, []Result) {
	clusterItems, otherIndices := partitionByCluster(assignments, k, log)

	// Build ClusterGroup for each non-empty cluster
	groups := make([]ClusterGroup, 0, k)
	for clusterIdx, indices := range clusterItems {
		if len(indices) == 0 {
			continue
		}

		items := collectItems(allItems, indices)
		topFacets := computeTopFacetsForIndices(facetSets, indices)
		fallbackName := fmt.Sprintf("Cluster %d", clusterIdx+1)

		groups = append(groups, ClusterGroup{
			Name:      fallbackName,
			Items:     items,
			TopFacets: topFacets,
			Stats: ClusterStats{
				Size:      len(items),
				TopFacets: topFacets,
			},
		})
	}

	otherItems := collectItems(allItems, otherIndices)
	return groups, otherItems
}

// partitionByCluster groups item indices by cluster, moving small clusters to Other
func partitionByCluster(assignments []int, k int, log *logger.Logger) ([][]int, []int) {
	clusterItems := make([][]int, k)
	for i := 0; i < k; i++ {
		clusterItems[i] = []int{}
	}

	var otherIndices []int

	for i, cluster := range assignments {
		if cluster < 0 || cluster >= k {
			otherIndices = append(otherIndices, i)
		} else {
			clusterItems[cluster] = append(clusterItems[cluster], i)
		}
	}

	// Move items from small clusters to "Other"
	for clusterIdx, indices := range clusterItems {
		if len(indices) > 0 && len(indices) < minClusterSize {
			log.Debug("moving small cluster to Other",
				"cluster_idx", clusterIdx,
				"size", len(indices),
				"min_size", minClusterSize,
			)
			otherIndices = append(otherIndices, indices...)
			clusterItems[clusterIdx] = []int{}
		}
	}

	return clusterItems, otherIndices
}

// collectItems gathers Result items by their indices
func collectItems(allItems []Result, indices []int) []Result {
	items := make([]Result, len(indices))
	for i, idx := range indices {
		items[i] = allItems[idx]
	}
	return items
}

// computeTopFacetsForIndices calculates top facets for items at given indices
func computeTopFacetsForIndices(facetSets []FacetSet, indices []int) []FacetCount {
	facetCounts := make(map[string]int)
	for _, idx := range indices {
		for facet := range facetSets[idx] {
			facetCounts[facet]++
		}
	}

	return sortAndLimitFacets(facetCounts, len(indices), 5)
}

// calculateTopFacets computes the most common facets for a set of items
func calculateTopFacets(items []Result, facetSets []FacetSet, itemIndex map[string]int) []FacetCount {
	facetCounts := make(map[string]int)

	for _, item := range items {
		idx, ok := itemIndex[item.ID]
		if !ok {
			continue
		}
		for facet := range facetSets[idx] {
			facetCounts[facet]++
		}
	}

	return sortAndLimitFacets(facetCounts, len(items), 5)
}

// sortAndLimitFacets sorts facets by count and returns top N
func sortAndLimitFacets(facetCounts map[string]int, totalItems, topN int) []FacetCount {
	type facetWithCount struct {
		facet string
		count int
	}

	var sortedFacets []facetWithCount
	for facet, count := range facetCounts {
		sortedFacets = append(sortedFacets, facetWithCount{facet, count})
	}

	sort.Slice(sortedFacets, func(i, j int) bool {
		if sortedFacets[i].count != sortedFacets[j].count {
			return sortedFacets[i].count > sortedFacets[j].count
		}
		return sortedFacets[i].facet < sortedFacets[j].facet
	})

	if len(sortedFacets) < topN {
		topN = len(sortedFacets)
	}

	topFacets := make([]FacetCount, topN)
	for i := 0; i < topN; i++ {
		facetName, facetValue := parseFacetKey(sortedFacets[i].facet)
		topFacets[i] = FacetCount{
			FacetName:  facetName,
			FacetValue: facetValue,
			Count:      sortedFacets[i].count,
			Percentage: float64(sortedFacets[i].count) / float64(totalItems) * 100,
		}
	}

	return topFacets
}

// parseFacetKey splits a "facetName:facetValue" string into its components
func parseFacetKey(key string) (string, string) {
	for i, c := range key {
		if c == ':' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

// joinStrings joins strings with a separator (simple helper to avoid importing strings)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
