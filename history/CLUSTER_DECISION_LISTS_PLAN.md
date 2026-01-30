# Cluster Decision Lists Implementation

## Overview

Extend clustering to fit decision list rules (CNF: conjunction of disjunctions, max 3 facets) for each similarity cluster. Rules become the cluster definition, enabling Algolia queries for full cluster expansion.

Add a rule-fitting phase after similarity clustering. Each cluster gets a decision list that defines membership. Rules are CNF (conjunction of disjunctions) with max 3 facets, mapping directly to Algolia's `facetFilters` syntax.

## Data Structures

Add to `backend/internal/ize/cluster.go`:

```go
// DecisionList represents a cluster's filter rule
// Structure: (facet1:val1 OR facet1:val2) AND (facet2:val3) AND ...
type DecisionList struct {
    Clauses []Clause  // AND of these clauses (max 3)
}

type Clause struct {
    FacetName string
    Values    []string  // OR of these values
}

// Methods
func (d DecisionList) ToAlgoliaFilter() [][]string  // Convert to facetFilters format
func (d DecisionList) Matches(facetSet FacetSet) bool  // Test if item matches rule
```

Update `ClusterGroup`:

```go
type ClusterGroup struct {
    Name        string
    Items       []Result
    TopFacets   []FacetCount
    Stats       ClusterStats
    Rule        *DecisionList  // NEW: the fitted rule
    RuleQuality RuleQuality    // NEW: precision, recall, F1
}
```

## Algorithm: `fitDecisionList`

**Input:** Cluster members (positive), all other items (negative), all facet sets

**Output:** DecisionList with max 3 clauses, optimized for recall

**Approach:** Greedy facet selection using information gain

1. **Candidate generation**: For each facet, compute which values appear more frequently in positives than negatives (lift > 1)
2. **Scoring**: For each candidate facet, compute recall improvement if we add it as a clause (include all high-lift values for that facet)
3. **Selection**: Greedily add facets that maximize recall while keeping precision above a minimum threshold (e.g., 0.3)
4. **Termination**: Stop at 3 facets or when no facet improves recall

**Key insight**: Since we want high recall, we're permissive with values — include any value that appears notably more in cluster than outside. The AND across facets provides specificity.

## Integration Point

Modify `ProcessCluster` to call rule fitting after clustering:

```go
func ProcessCluster(...) (*ClusterResult, error) {
    // ... existing similarity clustering ...
    groups, otherItems := buildClusterGroups(...)
    
    // NEW: Fit rules and reassign items
    groups = fitAndReassign(groups, allItems, facetSets)
    
    return &ClusterResult{...}, nil
}
```

The `fitAndReassign` function:

1. Fits a decision list for each group
2. Re-evaluates all items against all rules
3. Assigns items to every cluster whose rule they match (allows overlap)
4. Recalculates TopFacets based on new membership

## Algolia Filter Format

The `ToAlgoliaFilter()` method produces:

```go
// Rule: (brand:Samsung OR brand:LG) AND (color:Black)
// Output: [["brand:Samsung", "brand:LG"], ["color:Black"]]
```

This plugs directly into the existing `Search` method's `facetFilters` parameter.

## Files to Modify

- `backend/internal/ize/cluster.go` — Add `DecisionList` types and `fitDecisionList` algorithm
- `backend/internal/httpapi/dto.go` — Expose rule in API response (for frontend "load more")

## Testing Strategy

- Unit test `fitDecisionList` with synthetic facet data
- Test edge cases: empty clusters, no discriminative facets, overlapping clusters
- Integration test: verify rule-based membership matches expected recall targets

## Design Decisions

- **High recall over precision**: Rules are permissive to capture all cluster members, accepting some false positives
- **Overlapping clusters**: Items can belong to multiple clusters if they match multiple rules
- **Rules define clusters**: After fitting, rules become the source of truth for membership (not original similarity grouping)
- **Max 3 facets**: Keeps rules simple and Algolia queries efficient
