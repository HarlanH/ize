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

// Tests for DecisionList

func TestDecisionList_ToAlgoliaFilter(t *testing.T) {
	tests := []struct {
		name     string
		rule     DecisionList
		expected [][]string
	}{
		{
			name:     "empty rule",
			rule:     DecisionList{},
			expected: nil,
		},
		{
			name: "single facet single value",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung"}},
				},
			},
			expected: [][]string{{"brand:Samsung"}},
		},
		{
			name: "single facet multiple values",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung", "LG"}},
				},
			},
			expected: [][]string{{"brand:Samsung", "brand:LG"}},
		},
		{
			name: "multiple facets",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung", "LG"}},
					{FacetName: "color", Values: []string{"Black"}},
				},
			},
			expected: [][]string{{"brand:Samsung", "brand:LG"}, {"color:Black"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.rule.ToAlgoliaFilter()
			if len(result) != len(tt.expected) {
				t.Errorf("ToAlgoliaFilter() length = %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if len(result[i]) != len(tt.expected[i]) {
					t.Errorf("ToAlgoliaFilter()[%d] length = %d, want %d", i, len(result[i]), len(tt.expected[i]))
					continue
				}
				for j := range result[i] {
					if result[i][j] != tt.expected[i][j] {
						t.Errorf("ToAlgoliaFilter()[%d][%d] = %q, want %q", i, j, result[i][j], tt.expected[i][j])
					}
				}
			}
		})
	}
}

func TestDecisionList_Matches(t *testing.T) {
	tests := []struct {
		name     string
		rule     DecisionList
		facetSet FacetSet
		expected bool
	}{
		{
			name:     "empty rule matches everything",
			rule:     DecisionList{},
			facetSet: FacetSet{"brand:Samsung": true, "color:Black": true},
			expected: true,
		},
		{
			name: "single clause matches",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung"}},
				},
			},
			facetSet: FacetSet{"brand:Samsung": true, "color:Black": true},
			expected: true,
		},
		{
			name: "single clause no match",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Apple"}},
				},
			},
			facetSet: FacetSet{"brand:Samsung": true, "color:Black": true},
			expected: false,
		},
		{
			name: "OR within clause - first value matches",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung", "LG"}},
				},
			},
			facetSet: FacetSet{"brand:Samsung": true},
			expected: true,
		},
		{
			name: "OR within clause - second value matches",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung", "LG"}},
				},
			},
			facetSet: FacetSet{"brand:LG": true},
			expected: true,
		},
		{
			name: "AND across clauses - both match",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung"}},
					{FacetName: "color", Values: []string{"Black"}},
				},
			},
			facetSet: FacetSet{"brand:Samsung": true, "color:Black": true},
			expected: true,
		},
		{
			name: "AND across clauses - only first matches",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung"}},
					{FacetName: "color", Values: []string{"Black"}},
				},
			},
			facetSet: FacetSet{"brand:Samsung": true, "color:White": true},
			expected: false,
		},
		{
			name: "complex rule",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung", "LG"}},
					{FacetName: "color", Values: []string{"Black", "White"}},
					{FacetName: "size", Values: []string{"Large"}},
				},
			},
			facetSet: FacetSet{"brand:LG": true, "color:Black": true, "size:Large": true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.rule.Matches(tt.facetSet)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDecisionList_String(t *testing.T) {
	tests := []struct {
		name     string
		rule     DecisionList
		expected string
	}{
		{
			name:     "empty rule",
			rule:     DecisionList{},
			expected: "(empty rule)",
		},
		{
			name: "single value",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung"}},
				},
			},
			expected: "brand:Samsung",
		},
		{
			name: "OR values",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung", "LG"}},
				},
			},
			expected: "(brand:Samsung OR brand:LG)",
		},
		{
			name: "AND clauses",
			rule: DecisionList{
				Clauses: []Clause{
					{FacetName: "brand", Values: []string{"Samsung"}},
					{FacetName: "color", Values: []string{"Black"}},
				},
			},
			expected: "brand:Samsung AND color:Black",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.rule.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFitDecisionList_BasicCase(t *testing.T) {
	// Create facet sets where items 0,1,2 share "brand:A" and items 3,4,5 share "brand:B"
	facetSets := []FacetSet{
		{"brand:A": true, "color:Red": true},
		{"brand:A": true, "color:Blue": true},
		{"brand:A": true, "color:Green": true},
		{"brand:B": true, "color:Red": true},
		{"brand:B": true, "color:Blue": true},
		{"brand:B": true, "color:Green": true},
	}

	// Cluster is items 0,1,2 (brand:A items)
	positiveIndices := []int{0, 1, 2}

	rule, quality := fitDecisionList(positiveIndices, facetSets, logger.Default())

	// Rule should capture brand:A
	if rule == nil || len(rule.Clauses) == 0 {
		t.Fatal("fitDecisionList() returned empty rule")
	}

	// Should have high recall (capture all brand:A items)
	if quality.Recall < 0.9 {
		t.Errorf("fitDecisionList() recall = %.3f, want >= 0.9", quality.Recall)
	}

	// Verify the rule matches all positive items
	for _, idx := range positiveIndices {
		if !rule.Matches(facetSets[idx]) {
			t.Errorf("fitDecisionList() rule doesn't match positive item %d", idx)
		}
	}
}

func TestFitDecisionList_EmptyPositives(t *testing.T) {
	facetSets := []FacetSet{
		{"brand:A": true},
		{"brand:B": true},
	}

	rule, quality := fitDecisionList([]int{}, facetSets, logger.Default())

	if len(rule.Clauses) != 0 {
		t.Errorf("fitDecisionList() with empty positives should return empty rule")
	}
	if quality.Recall != 0 || quality.Precision != 0 {
		t.Errorf("fitDecisionList() with empty positives should have zero quality metrics")
	}
}

func TestComputeRuleQuality(t *testing.T) {
	facetSets := []FacetSet{
		{"brand:A": true}, // 0 - positive
		{"brand:A": true}, // 1 - positive
		{"brand:A": true}, // 2 - negative but matches rule
		{"brand:B": true}, // 3 - negative
	}
	positiveIndices := []int{0, 1}

	// Rule that matches items 0, 1, 2 (all brand:A)
	rule := DecisionList{
		Clauses: []Clause{
			{FacetName: "brand", Values: []string{"A"}},
		},
	}

	quality := computeRuleQuality(rule, positiveIndices, facetSets)

	// Recall: 2/2 = 1.0 (all positives match)
	if math.Abs(quality.Recall-1.0) > 0.001 {
		t.Errorf("computeRuleQuality() recall = %.3f, want 1.0", quality.Recall)
	}

	// Precision: 2/3 = 0.667 (2 true positives out of 3 matches)
	expectedPrecision := 2.0 / 3.0
	if math.Abs(quality.Precision-expectedPrecision) > 0.001 {
		t.Errorf("computeRuleQuality() precision = %.3f, want %.3f", quality.Precision, expectedPrecision)
	}

	// F1: 2 * (0.667 * 1.0) / (0.667 + 1.0) = 0.8
	expectedF1 := 2 * expectedPrecision * 1.0 / (expectedPrecision + 1.0)
	if math.Abs(quality.F1-expectedF1) > 0.001 {
		t.Errorf("computeRuleQuality() F1 = %.3f, want %.3f", quality.F1, expectedF1)
	}
}

func TestProcessCluster_HasRules(t *testing.T) {
	// Create items with clear cluster structure
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			// Cluster 1: Electronics
			{ObjectID: "1", Name: "Phone 1", Facets: map[string]interface{}{"category": "Electronics", "brand": "Samsung"}},
			{ObjectID: "2", Name: "Phone 2", Facets: map[string]interface{}{"category": "Electronics", "brand": "Samsung"}},
			{ObjectID: "3", Name: "Phone 3", Facets: map[string]interface{}{"category": "Electronics", "brand": "Apple"}},
			// Cluster 2: Clothing
			{ObjectID: "4", Name: "Shirt 1", Facets: map[string]interface{}{"category": "Clothing", "brand": "Nike"}},
			{ObjectID: "5", Name: "Shirt 2", Facets: map[string]interface{}{"category": "Clothing", "brand": "Nike"}},
			{ObjectID: "6", Name: "Shirt 3", Facets: map[string]interface{}{"category": "Clothing", "brand": "Adidas"}},
		},
	}

	result, err := ProcessCluster("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessCluster() error = %v", err)
	}

	// Check that clusters have rules
	for i, group := range result.Groups {
		if group.Rule == nil {
			t.Errorf("ProcessCluster() group %d has nil Rule", i)
			continue
		}
		if len(group.Rule.Clauses) == 0 {
			t.Errorf("ProcessCluster() group %d has empty Rule", i)
		}
		if group.RuleQuality == nil {
			t.Errorf("ProcessCluster() group %d has nil RuleQuality", i)
		}
	}
}
