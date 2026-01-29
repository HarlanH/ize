package ize

import (
	"ize/internal/algolia"
	"ize/internal/logger"
	"math"
	"testing"
)

func TestProcessCluster_EmptyResults(t *testing.T) {
	tests := []struct {
		name           string
		algoliaResults *algolia.SearchResult
	}{
		{
			name:           "nil results",
			algoliaResults: nil,
		},
		{
			name: "empty hits",
			algoliaResults: &algolia.SearchResult{
				Hits: []algolia.Hit{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessCluster("test", tt.algoliaResults, logger.Default())
			if err != nil {
				t.Fatalf("ProcessCluster() error = %v", err)
			}
			if len(result.Groups) != 0 {
				t.Errorf("ProcessCluster() groups count = %d, want 0", len(result.Groups))
			}
			if len(result.OtherGroup) != 0 {
				t.Errorf("ProcessCluster() other group count = %d, want 0", len(result.OtherGroup))
			}
			if result.ClusterCount != 0 {
				t.Errorf("ProcessCluster() cluster count = %d, want 0", result.ClusterCount)
			}
		})
	}
}

func TestProcessCluster_SingleItem(t *testing.T) {
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{ObjectID: "1", Name: "Item 1", Facets: map[string]interface{}{"category": "A"}},
		},
	}

	result, err := ProcessCluster("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessCluster() error = %v", err)
	}

	// Single item can't be clustered, should be in Other
	if len(result.Groups) != 0 {
		t.Errorf("ProcessCluster() groups count = %d, want 0", len(result.Groups))
	}
	if len(result.OtherGroup) != 1 {
		t.Errorf("ProcessCluster() other group count = %d, want 1", len(result.OtherGroup))
	}
}

func TestProcessCluster_NoFacets(t *testing.T) {
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{ObjectID: "1", Name: "Item 1"},
			{ObjectID: "2", Name: "Item 2"},
			{ObjectID: "3", Name: "Item 3"},
		},
	}

	result, err := ProcessCluster("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessCluster() error = %v", err)
	}

	// Should have no groups, all items in Other
	if len(result.Groups) != 0 {
		t.Errorf("ProcessCluster() groups count = %d, want 0", len(result.Groups))
	}
	if len(result.OtherGroup) != 3 {
		t.Errorf("ProcessCluster() other group count = %d, want 3", len(result.OtherGroup))
	}
}

func TestProcessCluster_TwoClearClusters(t *testing.T) {
	// Create two clearly distinct clusters
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			// Cluster 1: Electronics
			{ObjectID: "1", Name: "iPhone", Facets: map[string]interface{}{"category": "Electronics", "brand": "Apple", "type": "Phone"}},
			{ObjectID: "2", Name: "iPad", Facets: map[string]interface{}{"category": "Electronics", "brand": "Apple", "type": "Tablet"}},
			{ObjectID: "3", Name: "MacBook", Facets: map[string]interface{}{"category": "Electronics", "brand": "Apple", "type": "Laptop"}},
			{ObjectID: "4", Name: "Galaxy", Facets: map[string]interface{}{"category": "Electronics", "brand": "Samsung", "type": "Phone"}},
			// Cluster 2: Clothing
			{ObjectID: "5", Name: "T-Shirt", Facets: map[string]interface{}{"category": "Clothing", "brand": "Nike", "type": "Top"}},
			{ObjectID: "6", Name: "Jeans", Facets: map[string]interface{}{"category": "Clothing", "brand": "Levi", "type": "Bottom"}},
			{ObjectID: "7", Name: "Hoodie", Facets: map[string]interface{}{"category": "Clothing", "brand": "Nike", "type": "Top"}},
			{ObjectID: "8", Name: "Shorts", Facets: map[string]interface{}{"category": "Clothing", "brand": "Adidas", "type": "Bottom"}},
		},
	}

	result, err := ProcessCluster("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessCluster() error = %v", err)
	}

	// Should have 2-6 clusters
	if result.ClusterCount < 2 || result.ClusterCount > 6 {
		t.Errorf("ProcessCluster() cluster count = %d, want between 2 and 6", result.ClusterCount)
	}

	// All items should be assigned to groups or Other
	totalItems := 0
	for _, group := range result.Groups {
		totalItems += len(group.Items)
	}
	totalItems += len(result.OtherGroup)

	if totalItems != 8 {
		t.Errorf("ProcessCluster() total items = %d, want 8", totalItems)
	}
}

func TestProcessCluster_ClusterHasTopFacets(t *testing.T) {
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{ObjectID: "1", Name: "Item 1", Facets: map[string]interface{}{"category": "A", "brand": "X"}},
			{ObjectID: "2", Name: "Item 2", Facets: map[string]interface{}{"category": "A", "brand": "X"}},
			{ObjectID: "3", Name: "Item 3", Facets: map[string]interface{}{"category": "B", "brand": "Y"}},
			{ObjectID: "4", Name: "Item 4", Facets: map[string]interface{}{"category": "B", "brand": "Y"}},
		},
	}

	result, err := ProcessCluster("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessCluster() error = %v", err)
	}

	// Each group should have top facets
	for _, group := range result.Groups {
		if len(group.TopFacets) == 0 {
			t.Error("ProcessCluster() group has no top facets")
		}
		// Each top facet should have valid percentage
		for _, f := range group.TopFacets {
			if f.Percentage < 0 || f.Percentage > 100 {
				t.Errorf("ProcessCluster() facet percentage = %f, want between 0 and 100", f.Percentage)
			}
		}
	}
}

func TestProcessCluster_GroupHasFallbackName(t *testing.T) {
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{ObjectID: "1", Name: "Item 1", Facets: map[string]interface{}{"category": "A"}},
			{ObjectID: "2", Name: "Item 2", Facets: map[string]interface{}{"category": "A"}},
			{ObjectID: "3", Name: "Item 3", Facets: map[string]interface{}{"category": "B"}},
			{ObjectID: "4", Name: "Item 4", Facets: map[string]interface{}{"category": "B"}},
		},
	}

	result, err := ProcessCluster("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessCluster() error = %v", err)
	}

	// Each group should have a name (fallback "Cluster N")
	for i, group := range result.Groups {
		if group.Name == "" {
			t.Errorf("ProcessCluster() group %d has empty name", i)
		}
	}
}

func TestExtractFacetSet(t *testing.T) {
	tests := []struct {
		name     string
		hit      algolia.Hit
		expected FacetSet
	}{
		{
			name:     "empty facets",
			hit:      algolia.Hit{ObjectID: "1", Facets: nil},
			expected: FacetSet{},
		},
		{
			name: "string facet",
			hit: algolia.Hit{
				ObjectID: "1",
				Facets:   map[string]interface{}{"category": "Electronics"},
			},
			expected: FacetSet{"category:Electronics": true},
		},
		{
			name: "array facet",
			hit: algolia.Hit{
				ObjectID: "1",
				Facets:   map[string]interface{}{"tags": []interface{}{"red", "blue"}},
			},
			expected: FacetSet{"tags:red": true, "tags:blue": true},
		},
		{
			name: "multiple facets",
			hit: algolia.Hit{
				ObjectID: "1",
				Facets: map[string]interface{}{
					"category": "Electronics",
					"brand":    "Apple",
				},
			},
			expected: FacetSet{"category:Electronics": true, "brand:Apple": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFacetSet(tt.hit)
			if len(result) != len(tt.expected) {
				t.Errorf("extractFacetSet() size = %d, want %d", len(result), len(tt.expected))
			}
			for key := range tt.expected {
				if !result[key] {
					t.Errorf("extractFacetSet() missing key %s", key)
				}
			}
		})
	}
}

func TestJaccardDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        FacetSet
		b        FacetSet
		expected float64
	}{
		{
			name:     "both empty",
			a:        FacetSet{},
			b:        FacetSet{},
			expected: 1.0, // No shared information
		},
		{
			name:     "identical",
			a:        FacetSet{"a": true, "b": true},
			b:        FacetSet{"a": true, "b": true},
			expected: 0.0, // Identical = distance 0
		},
		{
			name:     "completely different",
			a:        FacetSet{"a": true, "b": true},
			b:        FacetSet{"c": true, "d": true},
			expected: 1.0, // No overlap = distance 1
		},
		{
			name:     "partial overlap",
			a:        FacetSet{"a": true, "b": true},
			b:        FacetSet{"b": true, "c": true},
			expected: 1.0 - 1.0/3.0, // intersection=1, union=3, similarity=1/3
		},
		{
			name:     "one empty",
			a:        FacetSet{"a": true},
			b:        FacetSet{},
			expected: 1.0, // No shared information
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jaccardDistance(tt.a, tt.b)
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("jaccardDistance() = %f, want %f", result, tt.expected)
			}
		})
	}
}

func TestBuildDistanceMatrix(t *testing.T) {
	facetSets := []FacetSet{
		{"a": true, "b": true},
		{"a": true, "b": true},
		{"c": true, "d": true},
	}

	matrix := buildDistanceMatrix(facetSets)

	// Check matrix dimensions
	if len(matrix) != 3 {
		t.Fatalf("buildDistanceMatrix() rows = %d, want 3", len(matrix))
	}
	for i, row := range matrix {
		if len(row) != 3 {
			t.Fatalf("buildDistanceMatrix() row %d cols = %d, want 3", i, len(row))
		}
	}

	// Check diagonal is 0
	for i := 0; i < 3; i++ {
		if matrix[i][i] != 0.0 {
			t.Errorf("buildDistanceMatrix() diagonal[%d] = %f, want 0", i, matrix[i][i])
		}
	}

	// Check symmetry
	for i := 0; i < 3; i++ {
		for j := i + 1; j < 3; j++ {
			if matrix[i][j] != matrix[j][i] {
				t.Errorf("buildDistanceMatrix() not symmetric at [%d][%d]: %f != %f", i, j, matrix[i][j], matrix[j][i])
			}
		}
	}

	// Check specific values
	// Items 0 and 1 are identical, distance should be 0
	if matrix[0][1] != 0.0 {
		t.Errorf("buildDistanceMatrix() [0][1] = %f, want 0 (identical items)", matrix[0][1])
	}

	// Items 0 and 2 are completely different, distance should be 1
	if matrix[0][2] != 1.0 {
		t.Errorf("buildDistanceMatrix() [0][2] = %f, want 1 (completely different)", matrix[0][2])
	}
}

func TestSilhouetteScore(t *testing.T) {
	// Create a simple distance matrix with clear clusters
	// Items 0,1 are close (distance 0.1), items 2,3 are close (distance 0.1)
	// Items from different clusters are far (distance 0.9)
	distMatrix := [][]float64{
		{0.0, 0.1, 0.9, 0.9},
		{0.1, 0.0, 0.9, 0.9},
		{0.9, 0.9, 0.0, 0.1},
		{0.9, 0.9, 0.1, 0.0},
	}

	// Perfect clustering: items 0,1 in cluster 0, items 2,3 in cluster 1
	assignments := []int{0, 0, 1, 1}

	score := silhouetteScore(distMatrix, assignments, 2)

	// Should have high positive score (good clustering)
	if score <= 0 {
		t.Errorf("silhouetteScore() = %f, want > 0 for good clustering", score)
	}

	// Bad clustering: items 0,2 in cluster 0, items 1,3 in cluster 1
	badAssignments := []int{0, 1, 0, 1}
	badScore := silhouetteScore(distMatrix, badAssignments, 2)

	// Good clustering should score higher than bad clustering
	if score <= badScore {
		t.Errorf("silhouetteScore() good=%f should be > bad=%f", score, badScore)
	}
}

func TestParseFacetKey(t *testing.T) {
	tests := []struct {
		key       string
		wantName  string
		wantValue string
	}{
		{"category:Electronics", "category", "Electronics"},
		{"brand:Apple", "brand", "Apple"},
		{"tags:red", "tags", "red"},
		{"no_colon", "no_colon", ""},
		{"multiple:colons:here", "multiple", "colons:here"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			name, value := parseFacetKey(tt.key)
			if name != tt.wantName || value != tt.wantValue {
				t.Errorf("parseFacetKey(%s) = (%s, %s), want (%s, %s)", tt.key, name, value, tt.wantName, tt.wantValue)
			}
		})
	}
}

func TestProcessCluster_MaxSixClusters(t *testing.T) {
	// Create many distinct clusters
	hits := make([]algolia.Hit, 80)
	for i := 0; i < 80; i++ {
		category := string(rune('A' + (i / 10)))
		hits[i] = algolia.Hit{
			ObjectID: string(rune('0' + (i % 10))),
			Name:     "Item",
			Facets:   map[string]interface{}{"category": category},
		}
	}

	algoliaResults := &algolia.SearchResult{Hits: hits}

	result, err := ProcessCluster("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessCluster() error = %v", err)
	}

	// Should select at most 6 clusters
	if result.ClusterCount > 6 {
		t.Errorf("ProcessCluster() cluster count = %d, want <= 6", result.ClusterCount)
	}
}

func TestAgglomerativeCluster(t *testing.T) {
	// Simple test case: 4 items with clear pair structure
	distMatrix := [][]float64{
		{0.0, 0.1, 0.9, 0.9},
		{0.1, 0.0, 0.9, 0.9},
		{0.9, 0.9, 0.0, 0.1},
		{0.9, 0.9, 0.1, 0.0},
	}

	root := agglomerativeCluster(distMatrix)

	// Root should contain all 4 items
	if root == nil {
		t.Fatal("agglomerativeCluster() returned nil")
	}
	if len(root.members) != 4 {
		t.Errorf("agglomerativeCluster() root members = %d, want 4", len(root.members))
	}
}

func TestCutDendrogram(t *testing.T) {
	// Build a dendrogram and cut it at various levels
	distMatrix := [][]float64{
		{0.0, 0.1, 0.9, 0.9},
		{0.1, 0.0, 0.9, 0.9},
		{0.9, 0.9, 0.0, 0.1},
		{0.9, 0.9, 0.1, 0.0},
	}

	root := agglomerativeCluster(distMatrix)

	// Cut into 2 clusters
	clusters2 := cutDendrogram(root, 2)
	if len(clusters2) != 2 {
		t.Errorf("cutDendrogram(k=2) = %d clusters, want 2", len(clusters2))
	}

	// Each cluster should have 2 items
	for i, cluster := range clusters2 {
		if len(cluster) != 2 {
			t.Errorf("cutDendrogram(k=2) cluster %d has %d items, want 2", i, len(cluster))
		}
	}

	// All items should be accounted for
	totalItems := 0
	for _, cluster := range clusters2 {
		totalItems += len(cluster)
	}
	if totalItems != 4 {
		t.Errorf("cutDendrogram(k=2) total items = %d, want 4", totalItems)
	}
}
