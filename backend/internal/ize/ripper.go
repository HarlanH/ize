package ize

import (
	"fmt"
	"ize/internal/algolia"
	"ize/internal/logger"
	"math"
)

// RipperGroup represents a group of items sharing a facet value
type RipperGroup struct {
	FacetName  string
	FacetValue string
	Items      []Result
	// TotalCount is the total number of items in the current result set
	// that have this facet value (including items that were assigned to
	// earlier groups). This is used for display in the UI so that the
	// count reflects the full set size, not just the remaining items
	// when the group was selected.
	TotalCount int
}

// RipperResult represents the output of the RIPPER algorithm
type RipperResult struct {
	Groups     []RipperGroup
	OtherGroup []Result
}

// ProcessRipper implements the RIPPER-inspired faceting algorithm
// It greedily selects the top 5 facet values that maximize information gain
func ProcessRipper(query string, algoliaResults *algolia.SearchResult, log *logger.Logger) (*RipperResult, error) {
	if log == nil {
		log = logger.Default()
	}

	log.Debug("ProcessRipper started",
		"query", query,
		"hits_count", func() int {
			if algoliaResults == nil {
				return 0
			}
			return len(algoliaResults.Hits)
		}(),
	)

	if algoliaResults == nil || len(algoliaResults.Hits) == 0 {
		log.Debug("ProcessRipper: empty results, returning empty groups")
		return &RipperResult{
			Groups:     []RipperGroup{},
			OtherGroup: []Result{},
		}, nil
	}

	// Convert Algolia hits to Results
	allItems := make([]Result, 0, len(algoliaResults.Hits))
	for _, hit := range algoliaResults.Hits {
		allItems = append(allItems, Result{
			ID:          hit.ObjectID,
			Name:        hit.Name,
			Description: hit.Description,
			Image:       hit.Image,
		})
	}

	totalItems := len(allItems)
	if totalItems == 0 {
		return &RipperResult{
			Groups:     []RipperGroup{},
			OtherGroup: []Result{},
		}, nil
	}

	// Calculate minimum group size: max(ceil(total * 0.05), 2)
	minGroupSize := int(math.Ceil(float64(totalItems) * 0.05))
	if minGroupSize < 2 {
		minGroupSize = 2
	}

	log.Debug("ProcessRipper: calculated parameters",
		"total_items", totalItems,
		"min_group_size", minGroupSize,
	)

	// Extract facet values from all items
	// Map: facetName -> facetValue -> []item indices
	facetValueMap := make(map[string]map[string][]int)
	for i, hit := range algoliaResults.Hits {
		if hit.Facets == nil {
			continue
		}
		for facetName, facetValue := range hit.Facets {
			if facetValue == nil {
				continue
			}
			// Handle both string and []string facet values
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
				// Skip non-string facet values
				continue
			}

			if facetValueMap[facetName] == nil {
				facetValueMap[facetName] = make(map[string][]int)
			}
			for _, value := range values {
				facetValueMap[facetName][value] = append(facetValueMap[facetName][value], i)
			}
		}
	}

	// Calculate initial counts for all facet values (P values for information gain)
	initialCounts := make(map[string]map[string]int) // facetName -> facetValue -> count
	totalFacetValues := 0
	for facetName, values := range facetValueMap {
		initialCounts[facetName] = make(map[string]int)
		for value, indices := range values {
			count := len(indices)
			initialCounts[facetName][value] = count
			totalFacetValues++
		}
	}

	log.Debug("ProcessRipper: extracted facets",
		"facet_names_count", len(facetValueMap),
		"total_facet_values", totalFacetValues,
	)

	// Greedy selection: select top 5 facet values
	selectedGroups := make([]RipperGroup, 0, 5)
	selectedFacetValues := make(map[string]map[string]bool) // facetName -> facetValue -> true
	assignedItems := make(map[int]bool)                     // Track which items have been assigned to groups

	for iteration := 0; iteration < 5; iteration++ {
		// Calculate information gain for all facet values using unassigned items
		bestFacetName := ""
		bestFacetValue := ""
		bestGain := math.Inf(-1)
		bestIndices := []int{}

		// Count unassigned items
		totalUnassigned := totalItems - len(assignedItems)

		log.Debug("ProcessRipper: iteration started",
			"iteration", iteration+1,
			"total_unassigned", totalUnassigned,
			"assigned_items", len(assignedItems),
			"selected_groups_count", len(selectedGroups),
		)

		// If no unassigned items remain, stop
		if totalUnassigned < minGroupSize {
			log.Debug("ProcessRipper: stopping early, insufficient unassigned items",
				"total_unassigned", totalUnassigned,
				"min_group_size", minGroupSize,
			)
			break
		}

		for facetName, values := range facetValueMap {
			for value, allIndices := range values {
				// Skip if this facet value was already selected
				if selectedFacetValues[facetName] != nil && selectedFacetValues[facetName][value] {
					continue
				}

				// Filter to only unassigned items
				unassignedIndices := make([]int, 0)
				for _, idx := range allIndices {
					if !assignedItems[idx] {
						unassignedIndices = append(unassignedIndices, idx)
					}
				}

				p := len(unassignedIndices)
				if p < minGroupSize {
					continue // Skip facet values that don't meet minimum size
				}

				t := totalUnassigned

				// Calculate information gain using entropy-based approach
				// Information gain measures how much we learn by splitting on this facet value
				//
				// If facet applies to ALL items (p = t): gain = 0 (no information gained)
				// If facet applies to NONE (p = 0): gain = 0 (no information gained)
				// Maximum gain occurs when split is balanced (p ≈ t/2)
				//
				// We use the entropy of the split: H = -p/t * log2(p/t) - (1-p/t) * log2(1-p/t)
				// This measures the "surprise" or information content of the split
				// Higher entropy = more balanced split = more information gain

				var gain float64
				if p == 0 || t == 0 || p == t {
					// No information gain if all items match or none match
					gain = 0
				} else {
					ratio := float64(p) / float64(t)

					// Entropy of the binary split
					// Maximum when ratio = 0.5 (perfectly balanced)
					entropySplit := -ratio*math.Log2(ratio) - (1-ratio)*math.Log2(1-ratio)

					// Weight by number of items in the group to prefer larger groups
					// But also weight by (1-ratio) to penalize when ratio approaches 1
					// This ensures facets covering all items get zero gain
					gain = entropySplit * float64(p) * (1 - ratio)
				}

				// Log top candidates (only log if gain is positive and significant)
				if gain > 0 && gain > bestGain-1 {
					log.Debug("ProcessRipper: evaluating facet value",
						"iteration", iteration+1,
						"facet_name", facetName,
						"facet_value", value,
						"p", p,
						"t", t,
						"ratio", fmt.Sprintf("%.4f", float64(p)/float64(t)),
						"gain", fmt.Sprintf("%.4f", gain),
					)
				}

				// Break ties: prefer higher coverage, then alphabetical
				if gain > bestGain || (gain == bestGain && (p > len(bestIndices) || (p == len(bestIndices) && fmt.Sprintf("%s:%s", facetName, value) < fmt.Sprintf("%s:%s", bestFacetName, bestFacetValue)))) {
					bestGain = gain
					bestFacetName = facetName
					bestFacetValue = value
					bestIndices = unassignedIndices
				}
			}
		}

		// If no valid facet value found, stop
		if bestFacetName == "" || len(bestIndices) == 0 {
			log.Debug("ProcessRipper: no valid facet value found, stopping",
				"iteration", iteration+1,
			)
			break
		}

		log.Debug("ProcessRipper: selected best facet value",
			"iteration", iteration+1,
			"facet_name", bestFacetName,
			"facet_value", bestFacetValue,
			"items_count", len(bestIndices),
			"information_gain", fmt.Sprintf("%.4f", bestGain),
		)

		// Mark this facet value as selected
		if selectedFacetValues[bestFacetName] == nil {
			selectedFacetValues[bestFacetName] = make(map[string]bool)
		}
		selectedFacetValues[bestFacetName][bestFacetValue] = true

		// Mark items as assigned
		for _, idx := range bestIndices {
			assignedItems[idx] = true
		}

		// Create group for selected facet value
		groupItems := make([]Result, 0, len(bestIndices))
		for _, idx := range bestIndices {
			groupItems = append(groupItems, allItems[idx])
		}

		// Use Algolia's facet counts if available (reflects entire result set),
		// otherwise fall back to counts from hits (top N only)
		totalCount := initialCounts[bestFacetName][bestFacetValue]
		if algoliaResults.Facets != nil {
			if facetValues, ok := algoliaResults.Facets[bestFacetName]; ok {
				if count, ok := facetValues[bestFacetValue]; ok {
					totalCount = int(count)
				}
			}
		}

		selectedGroups = append(selectedGroups, RipperGroup{
			FacetName:  bestFacetName,
			FacetValue: bestFacetValue,
			Items:      groupItems,
			TotalCount: totalCount,
		})
	}

	// Create "Other" group with items that weren't assigned to any selected group
	otherGroup := make([]Result, 0)
	for i, item := range allItems {
		if !assignedItems[i] {
			otherGroup = append(otherGroup, item)
		}
	}

	log.Debug("ProcessRipper: completed",
		"selected_groups_count", len(selectedGroups),
		"other_group_count", len(otherGroup),
		"total_assigned", len(assignedItems),
		"total_items", totalItems,
	)

	return &RipperResult{
		Groups:     selectedGroups,
		OtherGroup: otherGroup,
	}, nil
}
