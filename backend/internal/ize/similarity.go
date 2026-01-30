package ize

import (
	"fmt"
	"math"

	"ize/internal/logger"
)

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

// silhouetteScore calculates the silhouette score for a clustering
// Returns a value between -1 and 1, where higher is better
func silhouetteScore(distMatrix [][]float64, assignments []int, k int) float64 {
	n := len(assignments)
	if n < 2 || k < 2 {
		return 0
	}

	// Group items by cluster
	clusters := groupByCluster(assignments, k)

	// Calculate silhouette for each point
	totalSilhouette := 0.0
	validPoints := 0

	for i := 0; i < n; i++ {
		s, valid := computePointSilhouette(i, assignments[i], clusters, distMatrix, k)
		if valid {
			totalSilhouette += s
			validPoints++
		}
	}

	if validPoints == 0 {
		return 0
	}

	return totalSilhouette / float64(validPoints)
}

// groupByCluster groups item indices by their cluster assignment
func groupByCluster(assignments []int, k int) [][]int {
	clusters := make([][]int, k)
	for i := 0; i < k; i++ {
		clusters[i] = []int{}
	}
	for i, cluster := range assignments {
		if cluster >= 0 && cluster < k {
			clusters[cluster] = append(clusters[cluster], i)
		}
	}
	return clusters
}

// computePointSilhouette calculates the silhouette coefficient for a single point
func computePointSilhouette(i, cluster int, clusters [][]int, distMatrix [][]float64, k int) (float64, bool) {
	if cluster < 0 || cluster >= k {
		return 0, false
	}

	clusterMembers := clusters[cluster]
	if len(clusterMembers) <= 1 {
		return 0, false // Skip singleton clusters
	}

	// a(i) = average distance to other points in same cluster
	a := computeIntraClusterDistance(i, clusterMembers, distMatrix)

	// b(i) = minimum average distance to points in other clusters
	b := computeNearestClusterDistance(i, cluster, clusters, distMatrix, k)
	if b == math.Inf(1) {
		return 0, false
	}

	// Silhouette coefficient
	maxAB := math.Max(a, b)
	if maxAB == 0 {
		return 0, false
	}

	return (b - a) / maxAB, true
}

// computeIntraClusterDistance calculates average distance from point i to other members of its cluster
func computeIntraClusterDistance(i int, members []int, distMatrix [][]float64) float64 {
	sum := 0.0
	for _, j := range members {
		if j != i {
			sum += distMatrix[i][j]
		}
	}
	return sum / float64(len(members)-1)
}

// computeNearestClusterDistance calculates minimum average distance to points in other clusters
func computeNearestClusterDistance(i, ownCluster int, clusters [][]int, distMatrix [][]float64, k int) float64 {
	minDist := math.Inf(1)

	for otherCluster := 0; otherCluster < k; otherCluster++ {
		if otherCluster == ownCluster {
			continue
		}
		members := clusters[otherCluster]
		if len(members) == 0 {
			continue
		}

		avgDist := 0.0
		for _, j := range members {
			avgDist += distMatrix[i][j]
		}
		avgDist /= float64(len(members))

		if avgDist < minDist {
			minDist = avgDist
		}
	}

	return minDist
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
			continue // Not enough clusters possible
		}

		assignments := clustersToAssignments(clusters, n)
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

// clustersToAssignments converts cluster membership lists to an assignment array
func clustersToAssignments(clusters [][]int, n int) []int {
	assignments := make([]int, n)
	for i := range assignments {
		assignments[i] = -1
	}
	for clusterIdx, members := range clusters {
		for _, itemIdx := range members {
			assignments[itemIdx] = clusterIdx
		}
	}
	return assignments
}
