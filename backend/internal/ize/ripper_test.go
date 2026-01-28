package ize

import (
	"ize/internal/algolia"
	"ize/internal/logger"
	"testing"
)

func TestProcessRipper_EmptyResults(t *testing.T) {
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
			result, err := ProcessRipper("test", tt.algoliaResults, logger.Default())
			if err != nil {
				t.Fatalf("ProcessRipper() error = %v", err)
			}
			if len(result.Groups) != 0 {
				t.Errorf("ProcessRipper() groups count = %d, want 0", len(result.Groups))
			}
			if len(result.OtherGroup) != 0 {
				t.Errorf("ProcessRipper() other group count = %d, want 0", len(result.OtherGroup))
			}
		})
	}
}

func TestProcessRipper_SmallResultSet(t *testing.T) {
	// Test with 10 items (5% = 0.5, so minGroupSize should be 2)
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{ObjectID: "1", Name: "Item 1", Facets: map[string]interface{}{"category": "A"}},
			{ObjectID: "2", Name: "Item 2", Facets: map[string]interface{}{"category": "A"}},
			{ObjectID: "3", Name: "Item 3", Facets: map[string]interface{}{"category": "B"}},
			{ObjectID: "4", Name: "Item 4", Facets: map[string]interface{}{"category": "B"}},
			{ObjectID: "5", Name: "Item 5", Facets: map[string]interface{}{"category": "C"}},
			{ObjectID: "6", Name: "Item 6", Facets: map[string]interface{}{"category": "C"}},
			{ObjectID: "7", Name: "Item 7", Facets: map[string]interface{}{"category": "D"}},
			{ObjectID: "8", Name: "Item 8", Facets: map[string]interface{}{"category": "D"}},
			{ObjectID: "9", Name: "Item 9", Facets: map[string]interface{}{"category": "E"}},
			{ObjectID: "10", Name: "Item 10", Facets: map[string]interface{}{"category": "E"}},
		},
	}

	result, err := ProcessRipper("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessRipper() error = %v", err)
	}

	// Should select up to 5 groups, each with at least 2 items
	if len(result.Groups) > 5 {
		t.Errorf("ProcessRipper() groups count = %d, want <= 5", len(result.Groups))
	}

	for _, group := range result.Groups {
		if len(group.Items) < 2 {
			t.Errorf("ProcessRipper() group %s:%s has %d items, want >= 2", group.FacetName, group.FacetValue, len(group.Items))
		}
	}
}

func TestProcessRipper_NoFacets(t *testing.T) {
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{ObjectID: "1", Name: "Item 1"},
			{ObjectID: "2", Name: "Item 2"},
			{ObjectID: "3", Name: "Item 3"},
		},
	}

	result, err := ProcessRipper("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessRipper() error = %v", err)
	}

	// Should have no groups, all items in Other
	if len(result.Groups) != 0 {
		t.Errorf("ProcessRipper() groups count = %d, want 0", len(result.Groups))
	}
	if len(result.OtherGroup) != 3 {
		t.Errorf("ProcessRipper() other group count = %d, want 3", len(result.OtherGroup))
	}
}

func TestProcessRipper_MultipleFacetValues(t *testing.T) {
	// Items with multiple facet values
	// Note: Items are assigned to only one group (the first selected)
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{
				ObjectID: "1",
				Name:     "Item 1",
				Facets: map[string]interface{}{
					"category": "Electronics",
					"brand":    "Apple",
				},
			},
			{
				ObjectID: "2",
				Name:     "Item 2",
				Facets: map[string]interface{}{
					"category": "Electronics",
					"brand":    "Samsung",
				},
			},
			{
				ObjectID: "3",
				Name:     "Item 3",
				Facets: map[string]interface{}{
					"category": "Clothing",
					"brand":    "Apple",
				},
			},
			{
				ObjectID: "4",
				Name:     "Item 4",
				Facets: map[string]interface{}{
					"category": "Clothing",
					"brand":    "Samsung",
				},
			},
		},
	}

	result, err := ProcessRipper("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessRipper() error = %v", err)
	}

	// Should create groups (items are assigned to only one group)
	// Verify that items appear in exactly one group
	itemGroups := make(map[string]int) // item ID -> count of groups it appears in
	for _, group := range result.Groups {
		for _, item := range group.Items {
			itemGroups[item.ID]++
		}
	}

	// Each item should appear in exactly one group (or Other)
	for itemID, count := range itemGroups {
		if count > 1 {
			t.Errorf("Item %s appears in %d groups, should appear in at most 1", itemID, count)
		}
	}

	// Verify we have some groups
	if len(result.Groups) == 0 {
		t.Error("Expected at least one group to be created")
	}
}

func TestProcessRipper_MinimumThreshold(t *testing.T) {
	// Test with 100 items: 5% = 5, so minGroupSize should be 5
	// Create groups with varying sizes
	hits := make([]algolia.Hit, 100)
	for i := 0; i < 100; i++ {
		category := "A"
		if i < 10 {
			category = "Small" // 10 items - should be selected
		} else if i < 20 {
			category = "Medium" // 10 items - should be selected
		} else if i < 30 {
			category = "Large" // 10 items - should be selected
		} else if i < 40 {
			category = "XL" // 10 items - should be selected
		} else if i < 50 {
			category = "XXL" // 10 items - should be selected
		} else if i < 52 {
			category = "Tiny" // 2 items - should NOT be selected (below threshold of 5)
		}
		hits[i] = algolia.Hit{
			ObjectID: string(rune('0' + (i % 10))),
			Name:     "Item",
			Facets:   map[string]interface{}{"category": category},
		}
	}

	algoliaResults := &algolia.SearchResult{Hits: hits}

	result, err := ProcessRipper("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessRipper() error = %v", err)
	}

	// Verify all groups meet minimum threshold
	for _, group := range result.Groups {
		if len(group.Items) < 5 {
			t.Errorf("ProcessRipper() group %s:%s has %d items, want >= 5", group.FacetName, group.FacetValue, len(group.Items))
		}
	}

	// Verify "Tiny" group is not selected (should be in Other)
	tinyInGroups := false
	for _, group := range result.Groups {
		if group.FacetValue == "Tiny" {
			tinyInGroups = true
		}
	}
	if tinyInGroups {
		t.Error("Tiny group should not be selected (below minimum threshold)")
	}
}

func TestProcessRipper_TieBreaking(t *testing.T) {
	// Create scenario where multiple facet values have same information gain
	// Use alphabetical tie-breaking
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{ObjectID: "1", Name: "Item 1", Facets: map[string]interface{}{"category": "Zebra"}},
			{ObjectID: "2", Name: "Item 2", Facets: map[string]interface{}{"category": "Zebra"}},
			{ObjectID: "3", Name: "Item 3", Facets: map[string]interface{}{"category": "Zebra"}},
			{ObjectID: "4", Name: "Item 4", Facets: map[string]interface{}{"category": "Apple"}},
			{ObjectID: "5", Name: "Item 5", Facets: map[string]interface{}{"category": "Apple"}},
			{ObjectID: "6", Name: "Item 6", Facets: map[string]interface{}{"category": "Apple"}},
		},
	}

	result, err := ProcessRipper("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessRipper() error = %v", err)
	}

	// Should select groups (alphabetical order should prefer "Apple" over "Zebra" if tied)
	if len(result.Groups) > 0 {
		// Verify groups are created
		hasApple := false
		hasZebra := false
		for _, group := range result.Groups {
			if group.FacetValue == "Apple" {
				hasApple = true
			}
			if group.FacetValue == "Zebra" {
				hasZebra = true
			}
		}
		// Both should be selected if they meet threshold
		if !hasApple && !hasZebra {
			t.Error("Expected at least one group to be selected")
		}
	}
}

func TestProcessRipper_OtherGroup(t *testing.T) {
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{ObjectID: "1", Name: "Item 1", Facets: map[string]interface{}{"category": "A"}},
			{ObjectID: "2", Name: "Item 2", Facets: map[string]interface{}{"category": "A"}},
			{ObjectID: "3", Name: "Item 3", Facets: map[string]interface{}{"category": "A"}},
			{ObjectID: "4", Name: "Item 4", Facets: map[string]interface{}{"category": "B"}},
			{ObjectID: "5", Name: "Item 5", Facets: map[string]interface{}{"category": "B"}},
			{ObjectID: "6", Name: "Item 6", Facets: map[string]interface{}{"category": "B"}},
			{ObjectID: "7", Name: "Item 7"}, // No facets - should be in Other
			{ObjectID: "8", Name: "Item 8"}, // No facets - should be in Other
		},
	}

	result, err := ProcessRipper("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessRipper() error = %v", err)
	}

	// Items 7 and 8 should be in Other group
	otherIDs := make(map[string]bool)
	for _, item := range result.OtherGroup {
		otherIDs[item.ID] = true
	}

	if !otherIDs["7"] {
		t.Error("Item 7 should be in Other group")
	}
	if !otherIDs["8"] {
		t.Error("Item 8 should be in Other group")
	}
}

func TestProcessRipper_ArrayFacetValues(t *testing.T) {
	// Test handling of array facet values
	// Note: Items are assigned to only one group (the first selected)
	algoliaResults := &algolia.SearchResult{
		Hits: []algolia.Hit{
			{
				ObjectID: "1",
				Name:     "Item 1",
				Facets: map[string]interface{}{
					"tags": []interface{}{"red", "blue"},
				},
			},
			{
				ObjectID: "2",
				Name:     "Item 2",
				Facets: map[string]interface{}{
					"tags": []interface{}{"red", "green"},
				},
			},
			{
				ObjectID: "3",
				Name:     "Item 3",
				Facets: map[string]interface{}{
					"tags": []interface{}{"blue"},
				},
			},
		},
	}

	result, err := ProcessRipper("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessRipper() error = %v", err)
	}

	// Should handle array facet values correctly
	// Item 1 has both "red" and "blue" tags, but will be assigned to only one group
	item1GroupCount := 0
	item1InRed := false
	item1InBlue := false
	for _, group := range result.Groups {
		for _, item := range group.Items {
			if item.ID == "1" {
				item1GroupCount++
				if group.FacetName == "tags" && group.FacetValue == "red" {
					item1InRed = true
				}
				if group.FacetName == "tags" && group.FacetValue == "blue" {
					item1InBlue = true
				}
			}
		}
	}

	// Item 1 should appear in exactly one group (either red or blue, depending on which has higher info gain)
	if item1GroupCount != 1 {
		t.Errorf("Item 1 should appear in exactly 1 group, but appears in %d", item1GroupCount)
	}
	
	// Item 1 should appear in either red or blue group (but not both)
	if !item1InRed && !item1InBlue {
		t.Error("Item 1 should appear in either red or blue tags group")
	}
	if item1InRed && item1InBlue {
		t.Error("Item 1 should not appear in both red and blue groups")
	}
}

func TestProcessRipper_MaxFiveGroups(t *testing.T) {
	// Create more than 5 valid groups
	hits := make([]algolia.Hit, 60)
	for i := 0; i < 60; i++ {
		category := string(rune('A' + (i / 10)))
		hits[i] = algolia.Hit{
			ObjectID: string(rune('0' + (i % 10))),
			Name:     "Item",
			Facets:   map[string]interface{}{"category": category},
		}
	}

	algoliaResults := &algolia.SearchResult{Hits: hits}

	result, err := ProcessRipper("test", algoliaResults, logger.Default())
	if err != nil {
		t.Fatalf("ProcessRipper() error = %v", err)
	}

	// Should select at most 5 groups
	if len(result.Groups) > 5 {
		t.Errorf("ProcessRipper() groups count = %d, want <= 5", len(result.Groups))
	}
}
