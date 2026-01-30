package ize

import (
	"math"
	"sort"
)

// clusterNode represents a node in the hierarchical clustering dendrogram
type clusterNode struct {
	id      int // Unique identifier
	left    *clusterNode
	right   *clusterNode
	height  float64 // Distance at which this cluster was formed
	members []int   // Indices of original items in this cluster
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
		minI, minJ, minDist := findClosestClusters(clusters, active, distMatrix)

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

// findClosestClusters finds the two closest clusters using average linkage
func findClosestClusters(clusters []*clusterNode, active []int, distMatrix [][]float64) (int, int, float64) {
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

	return minI, minJ, minDist
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

	// BFS to get all internal nodes sorted by height (descending)
	internalNodes := collectInternalNodes(root)

	// Sort by height descending
	sort.Slice(internalNodes, func(i, j int) bool {
		return internalNodes[i].height > internalNodes[j].height
	})

	// Start with root as single cluster, then split top k-1 merges
	currentClusters := []*clusterNode{root}

	for i := 0; i < k-1 && i < len(internalNodes); i++ {
		maxHeightIdx := findHighestMergeCluster(currentClusters)
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

// collectInternalNodes collects all internal (non-leaf) nodes from the dendrogram
func collectInternalNodes(root *clusterNode) []*clusterNode {
	var nodes []*clusterNode
	queue := []*clusterNode{root}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node.left != nil && node.right != nil {
			nodes = append(nodes, node)
			queue = append(queue, node.left, node.right)
		}
	}

	return nodes
}

// findHighestMergeCluster finds the cluster with the highest merge height that can be split
func findHighestMergeCluster(clusters []*clusterNode) int {
	maxHeightIdx := -1
	maxHeight := -1.0

	for j, cluster := range clusters {
		if cluster.left != nil && cluster.right != nil && cluster.height > maxHeight {
			maxHeight = cluster.height
			maxHeightIdx = j
		}
	}

	return maxHeightIdx
}
