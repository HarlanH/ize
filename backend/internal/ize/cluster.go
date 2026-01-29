package ize

import (
	"fmt"
	"math"
	"sort"

	"ize/internal/algolia"
	"ize/internal/logger"
)

// ClusterGroup represents a cluster of items with similar facet profiles
type ClusterGroup struct {
	Name      string            // LLM-generated label (or fallback)
	Items     []Result          // Items in this cluster
	TopFacets []FacetCount      // Most common facet:value pairs in this cluster
	Stats     ClusterStats      // Statistics for LLM labeling
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

// ProcessCluster implements facet-space clustering using Jaccard similarity
// and agglomerative hierarchical clustering with silhouette-based k selection
func ProcessCluster(query string, algoliaResults *algolia.SearchResult, log *logger.Logger) (*ClusterResult, error) {
	if log == nil {
		log = logger.Default()
	}

	log.Debug("ProcessCluster started",
		"query", query,
		"hits_count", func() int {
			if algoliaResults == nil {
				return 0
			}
			return len(algoliaResults.Hits)
		}(),
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

	totalItems := len(allItems)
	log.Debug("ProcessCluster: extracted facet sets",
		"total_items", totalItems,
	)

	// Handle edge cases
	if totalItems < 2 {
		log.Debug("ProcessCluster: too few items for clustering")
		return &ClusterResult{
			Groups:       []ClusterGroup{},
			OtherGroup:   allItems,
			ClusterCount: 0,
		}, nil
	}

	// Check if any items have facets
	hasAnyFacets := false
	for _, fs := range facetSets {
		if len(fs) > 0 {
			hasAnyFacets = true
			break
		}
	}
	if !hasAnyFacets {
		log.Debug("ProcessCluster: no items have facets, returning all as Other")
		return &ClusterResult{
			Groups:       []ClusterGroup{},
			OtherGroup:   allItems,
			ClusterCount: 0,
		}, nil
	}

	// Build distance matrix using Jaccard distance
	distMatrix := buildDistanceMatrix(facetSets)
	log.Debug("ProcessCluster: built distance matrix",
		"matrix_size", len(distMatrix),
	)

	// Find optimal k using silhouette score
	optimalK, assignments, silhouetteScores := selectOptimalK(distMatrix, facetSets, log)
	
	// Log silhouette scores prominently for easy debugging
	log.Info("ProcessCluster: silhouette scores by k",
		"k=2", fmt.Sprintf("%.3f", silhouetteScores[2]),
		"k=3", fmt.Sprintf("%.3f", silhouetteScores[3]),
		"k=4", fmt.Sprintf("%.3f", silhouetteScores[4]),
		"k=5", fmt.Sprintf("%.3f", silhouetteScores[5]),
		"k=6", fmt.Sprintf("%.3f", silhouetteScores[6]),
		"selected_k", optimalK,
	)

	// Build cluster groups
	groups, otherItems := buildClusterGroups(allItems, facetSets, assignments, optimalK, log)

	// Actual cluster count is the number of groups after filtering small clusters
	actualClusterCount := len(groups)

	log.Info("ProcessCluster: completed",
		"selected_k", optimalK,
		"actual_clusters", actualClusterCount,
		"other_count", len(otherItems),
	)

	return &ClusterResult{
		Groups:       groups,
		OtherGroup:   otherItems,
		ClusterCount: actualClusterCount, // Use actual count, not selected k
	}, nil
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
			continue
		}

		for _, value := range values {
			key := fmt.Sprintf("%s:%s", facetName, value)
			fs[key] = true
		}
	}

	return fs
}

// jaccardDistance calculates 1 - Jaccard similarity between two facet sets
// Returns 1.0 if both sets are empty (maximum distance for no information)
func jaccardDistance(a, b FacetSet) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0 // No shared information
	}

	// Calculate intersection and union sizes
	intersection := 0
	union := len(a)

	for key := range b {
		if a[key] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 1.0
	}

	similarity := float64(intersection) / float64(union)
	return 1.0 - similarity
}

// buildDistanceMatrix creates a symmetric distance matrix using Jaccard distance
func buildDistanceMatrix(facetSets []FacetSet) [][]float64 {
	n := len(facetSets)
	matrix := make([][]float64, n)

	for i := 0; i < n; i++ {
		matrix[i] = make([]float64, n)
		matrix[i][i] = 0.0 // Distance to self is 0
	}

	// Fill upper triangle and mirror to lower
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			dist := jaccardDistance(facetSets[i], facetSets[j])
			matrix[i][j] = dist
			matrix[j][i] = dist
		}
	}

	return matrix
}

// clusterNode represents a node in the hierarchical clustering dendrogram
type clusterNode struct {
	id       int     // Unique identifier
	left     *clusterNode
	right    *clusterNode
	height   float64 // Distance at which this cluster was formed
	members  []int   // Indices of original items in this cluster
}

// agglomerativeCluster performs hierarchical agglomerative clustering
// using average linkage and returns the root of the dendrogram
func agglomerativeCluster(distMatrix [][]float64) *clusterNode {
	n := len(distMatrix)
	if n == 0 {
		return nil
	}

	// Initialize each item as its own cluster
	clusters := make([]*clusterNode, n)
	for i := 0; i < n; i++ {
		clusters[i] = &clusterNode{
			id:      i,
			members: []int{i},
			height:  0,
		}
	}

	// Active clusters (indices into clusters slice)
	active := make([]int, n)
	for i := 0; i < n; i++ {
		active[i] = i
	}

	nextID := n

	// Merge until only one cluster remains
	for len(active) > 1 {
		// Find the two closest clusters (average linkage)
		minDist := math.Inf(1)
		minI, minJ := 0, 1

		for i := 0; i < len(active); i++ {
			for j := i + 1; j < len(active); j++ {
				dist := averageLinkageDistance(clusters[active[i]], clusters[active[j]], distMatrix)
				if dist < minDist {
					minDist = dist
					minI, minJ = i, j
				}
			}
		}

		// Create new merged cluster
		leftCluster := clusters[active[minI]]
		rightCluster := clusters[active[minJ]]

		newMembers := make([]int, 0, len(leftCluster.members)+len(rightCluster.members))
		newMembers = append(newMembers, leftCluster.members...)
		newMembers = append(newMembers, rightCluster.members...)

		newCluster := &clusterNode{
			id:      nextID,
			left:    leftCluster,
			right:   rightCluster,
			height:  minDist,
			members: newMembers,
		}
		nextID++

		clusters = append(clusters, newCluster)

		// Update active list: remove minJ first (larger index), then minI
		active = append(active[:minJ], active[minJ+1:]...)
		active = append(active[:minI], active[minI+1:]...)
		active = append(active, len(clusters)-1)
	}

	return clusters[active[0]]
}

// averageLinkageDistance calculates the average distance between all pairs
// of items from two clusters
func averageLinkageDistance(a, b *clusterNode, distMatrix [][]float64) float64 {
	totalDist := 0.0
	count := 0

	for _, i := range a.members {
		for _, j := range b.members {
			totalDist += distMatrix[i][j]
			count++
		}
	}

	if count == 0 {
		return math.Inf(1)
	}

	return totalDist / float64(count)
}

// cutDendrogram cuts the dendrogram to produce k clusters
func cutDendrogram(root *clusterNode, k int) [][]int {
	if root == nil || k <= 0 {
		return nil
	}

	// Collect all nodes with their heights
	type nodeWithHeight struct {
		node   *clusterNode
		height float64
	}

	// BFS to get all internal nodes sorted by height (descending)
	var internalNodes []nodeWithHeight
	queue := []*clusterNode{root}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node.left != nil && node.right != nil {
			internalNodes = append(internalNodes, nodeWithHeight{node, node.height})
			queue = append(queue, node.left, node.right)
		}
	}

	// Sort by height descending
	sort.Slice(internalNodes, func(i, j int) bool {
		return internalNodes[i].height > internalNodes[j].height
	})

	// Start with root as single cluster, then split top k-1 merges
	currentClusters := []*clusterNode{root}

	for i := 0; i < k-1 && i < len(internalNodes); i++ {
		// Find the cluster with highest merge height and split it
		maxHeightIdx := -1
		maxHeight := -1.0

		for j, cluster := range currentClusters {
			if cluster.left != nil && cluster.right != nil && cluster.height > maxHeight {
				maxHeight = cluster.height
				maxHeightIdx = j
			}
		}

		if maxHeightIdx == -1 {
			break // No more splits possible
		}

		// Split this cluster
		splitCluster := currentClusters[maxHeightIdx]
		currentClusters = append(currentClusters[:maxHeightIdx], currentClusters[maxHeightIdx+1:]...)
		currentClusters = append(currentClusters, splitCluster.left, splitCluster.right)
	}

	// Extract member indices from each cluster
	result := make([][]int, len(currentClusters))
	for i, cluster := range currentClusters {
		result[i] = cluster.members
	}

	return result
}

// silhouetteScore calculates the silhouette score for a clustering
// Returns a value between -1 and 1, where higher is better
func silhouetteScore(distMatrix [][]float64, assignments []int, k int) float64 {
	n := len(assignments)
	if n < 2 || k < 2 {
		return 0
	}

	// Group items by cluster
	clusters := make([][]int, k)
	for i := 0; i < k; i++ {
		clusters[i] = []int{}
	}
	for i, cluster := range assignments {
		if cluster >= 0 && cluster < k {
			clusters[cluster] = append(clusters[cluster], i)
		}
	}

	// Calculate silhouette for each point
	totalSilhouette := 0.0
	validPoints := 0

	for i := 0; i < n; i++ {
		cluster := assignments[i]
		if cluster < 0 || cluster >= k {
			continue
		}

		clusterMembers := clusters[cluster]

		// Skip if this is a singleton cluster
		if len(clusterMembers) <= 1 {
			continue
		}

		// a(i) = average distance to other points in same cluster
		a := 0.0
		for _, j := range clusterMembers {
			if j != i {
				a += distMatrix[i][j]
			}
		}
		a /= float64(len(clusterMembers) - 1)

		// b(i) = minimum average distance to points in other clusters
		b := math.Inf(1)
		for otherCluster := 0; otherCluster < k; otherCluster++ {
			if otherCluster == cluster {
				continue
			}
			otherMembers := clusters[otherCluster]
			if len(otherMembers) == 0 {
				continue
			}
			avgDist := 0.0
			for _, j := range otherMembers {
				avgDist += distMatrix[i][j]
			}
			avgDist /= float64(len(otherMembers))
			if avgDist < b {
				b = avgDist
			}
		}

		// Silhouette coefficient for this point
		if b == math.Inf(1) {
			continue
		}
		maxAB := math.Max(a, b)
		if maxAB == 0 {
			continue
		}
		s := (b - a) / maxAB
		totalSilhouette += s
		validPoints++
	}

	if validPoints == 0 {
		return 0
	}

	return totalSilhouette / float64(validPoints)
}

// selectOptimalK finds the optimal number of clusters using silhouette score
// Returns the optimal k, cluster assignments, and all silhouette scores tried
func selectOptimalK(distMatrix [][]float64, facetSets []FacetSet, log *logger.Logger) (int, []int, map[int]float64) {
	n := len(distMatrix)
	silhouetteScores := make(map[int]float64)

	// Build dendrogram once
	root := agglomerativeCluster(distMatrix)

	// Maximum k is min(6, n-1)
	maxK := 6
	if n-1 < maxK {
		maxK = n - 1
	}
	if maxK < 2 {
		maxK = 2
	}

	bestK := 2
	bestScore := math.Inf(-1)
	var bestAssignments []int

	for k := 2; k <= maxK; k++ {
		clusters := cutDendrogram(root, k)
		if len(clusters) < k {
			// Not enough clusters possible
			continue
		}

		// Convert cluster membership lists to assignment array
		assignments := make([]int, n)
		for i := range assignments {
			assignments[i] = -1
		}
		for clusterIdx, members := range clusters {
			for _, itemIdx := range members {
				assignments[itemIdx] = clusterIdx
			}
		}

		score := silhouetteScore(distMatrix, assignments, k)
		silhouetteScores[k] = score

		log.Debug("ProcessCluster: evaluated k",
			"k", k,
			"silhouette_score", fmt.Sprintf("%.4f", score),
		)

		if score > bestScore {
			bestScore = score
			bestK = k
			bestAssignments = assignments
		}
	}

	return bestK, bestAssignments, silhouetteScores
}

// Minimum cluster size - clusters smaller than this go to "Other"
const minClusterSize = 2

// buildClusterGroups creates ClusterGroup objects from cluster assignments
// Clusters with fewer than minClusterSize items are moved to "Other"
func buildClusterGroups(allItems []Result, facetSets []FacetSet, assignments []int, k int, log *logger.Logger) ([]ClusterGroup, []Result) {
	// Group items by cluster
	clusterItems := make([][]int, k)
	for i := 0; i < k; i++ {
		clusterItems[i] = []int{}
	}

	otherIndices := []int{}

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
			clusterItems[clusterIdx] = []int{} // Clear the cluster
		}
	}

	// Build ClusterGroup for each cluster
	groups := make([]ClusterGroup, 0, k)
	for clusterIdx, indices := range clusterItems {
		if len(indices) == 0 {
			continue
		}

		// Collect items
		items := make([]Result, len(indices))
		for i, idx := range indices {
			items[i] = allItems[idx]
		}

		// Calculate top facets for this cluster
		facetCounts := make(map[string]int)
		for _, idx := range indices {
			for facet := range facetSets[idx] {
				facetCounts[facet]++
			}
		}

		// Sort facets by count and take top 5
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

		topN := 5
		if len(sortedFacets) < topN {
			topN = len(sortedFacets)
		}

		topFacets := make([]FacetCount, topN)
		for i := 0; i < topN; i++ {
			// Parse facet:value
			facetStr := sortedFacets[i].facet
			facetName, facetValue := parseFacetKey(facetStr)

			topFacets[i] = FacetCount{
				FacetName:  facetName,
				FacetValue: facetValue,
				Count:      sortedFacets[i].count,
				Percentage: float64(sortedFacets[i].count) / float64(len(indices)) * 100,
			}
		}

		// Generate fallback name (will be replaced by LLM later)
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

	// Build Other group
	otherItems := make([]Result, len(otherIndices))
	for i, idx := range otherIndices {
		otherItems[i] = allItems[idx]
	}

	return groups, otherItems
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
