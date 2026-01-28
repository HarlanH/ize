# RIPPER Faceting Implementation Plan

## Algorithm Overview

Adapt RIPPER's greedy information gain maximization for faceting:

1. **Input**: Relevance-weighted sample of items from Algolia (up to 100 hits)
2. **Process**: Greedily select facet values that maximize information gain
3. **Output**: Top 5 facet groups + "Other" group

### Information Gain Calculation

Since items are treated uniformly (no explicit relevance weighting), we'll use a simplified information gain metric:

**FOIL's Information Gain**: `p * (log(p/t) - log(P/T))`

Where:
- `p` = items matching the facet value (in current candidate set)
- `t` = total items in current candidate set
- `P` = items matching the facet value (in original full set)
- `T` = total items in original full set

This measures how much information we gain by knowing an item belongs to this facet value, relative to the overall distribution.

### Greedy Selection Algorithm

```
1. Start with all items unassigned
2. For each iteration (up to 5):
   a. Calculate information gain for all facet values using unassigned items
   b. Select facet value with highest information gain
   c. Assign matching items to this group
   d. Remove assigned items from unassigned set
3. Assign remaining unassigned items to "Other" group
```

### Minimum Group Size

- **Threshold**: 5% of total items (or absolute minimum of 2 items, whichever is larger)
- **Validation**: Only consider facet values that meet minimum size threshold
- **Edge case**: If remaining items < minimum threshold, assign to "Other" instead of creating new group

## Edge Cases & Problems Identified

### 1. **Small Result Sets**
- **Problem**: With < 20 items, 5% = 1 item, which may be too small
- **Solution**: Use `max(ceil(total * 0.05), 2)` as minimum threshold

### 2. **Multiple Facet Membership**
- **Problem**: Items can belong to multiple facet values (user confirmed: items appear in multiple groups)
- **Solution**: When assigning items to a group, don't remove them from consideration for other groups. Each group is independent.

### 3. **Insufficient Facet Coverage**
- **Problem**: Some items may not have any facet values, or facets may be sparse
- **Solution**: Items without the selected facet values go to "Other". If no valid facets exist, show only "Other" group.

### 4. **Tie-Breaking**
- **Problem**: Multiple facet values may have identical information gain
- **Solution**: Break ties by:
  1. Higher coverage (more items)
  2. Alphabetical order (deterministic)

### 5. **Empty or Very Small "Other" Group**
- **Problem**: If top 5 facets cover most items, "Other" may be tiny or empty
- **Solution**: Always show "Other" group, even if empty (for consistency). Consider showing count of 0.

### 6. **Facet Value Selection Across Different Facets**
- **Problem**: Should we consider facet values from different facet attributes, or only one facet at a time?
- **Solution**: Consider all facet values across all facets simultaneously. The algorithm selects the best facet value regardless of which facet attribute it belongs to.

### 7. **Performance with 100 Items**
- **Problem**: Calculating information gain for all facet values may be expensive
- **Solution**: Should be fine for 100 items. If performance becomes an issue, we can optimize by:
  - Pre-filtering facet values that don't meet minimum size
  - Caching calculations

### 8. **Algolia Result Limit**
- **Problem**: What if search returns exactly 100 items but there are more?
- **Solution**: Use the 100-item sample. The algorithm works on whatever sample we have. This is acceptable since we're working with a relevance-weighted sample.

### 9. **Zero Information Gain**
- **Problem**: If all facet values have zero or negative information gain
- **Solution**: Still select the top 5 by information gain (even if all are 0 or negative). This ensures we always show 5 groups + Other.

### 10. **Facet Value Format**
- **Problem**: Need to display both facet name and value (e.g., "Category: Electronics")
- **Solution**: Store facet name with each group. Display as "FacetName: Value" in UI.

## Implementation Plan

### Backend Changes

#### 1. Update Algolia Client (`backend/internal/algolia/client.go`)
- Add `HitsPerPage: 100` to `SearchParamsObject` for RIPPER requests
- Optionally extract `_score` field from hits (for future relevance weighting, though not used initially)

#### 2. Create RIPPER Processor (`backend/internal/ize/ripper.go`)
- New `RipperProcessor` struct implementing `Processor` interface
- `Process()` method that:
  - Takes Algolia results with up to 100 hits
  - Extracts facet data from hits (each hit has `Facets` map)
  - Runs greedy information gain algorithm
  - Returns grouped results with facet group assignments

#### 3. Add RIPPER Response Type (`backend/internal/httpapi/dto.go`)
- New `RipperGroup` struct:
  ```go
  type RipperGroup struct {
    FacetName string   `json:"facetName"`
    FacetValue string  `json:"facetValue"`
    Items []SearchResult `json:"items"`
    Count int          `json:"count"`
  }
  ```
- New `RipperResponse` struct:
  ```go
  type RipperResponse struct {
    Groups []RipperGroup `json:"groups"`
    OtherGroup []SearchResult `json:"otherGroup"`
  }
  ```

#### 4. Add RIPPER Endpoint (`backend/internal/httpapi/handler.go`)
- New `HandleRipper` method
- Searches Algolia with `HitsPerPage: 100`
- Processes through RIPPER algorithm
- Returns grouped results

#### 5. Update Router (`backend/cmd/server/main.go`)
- Add `/api/ripper` POST endpoint

### Frontend Changes

#### 6. Add RIPPER Tab (`frontend/src/components/FacetedSearch.vue`)
- Add "RIPPER" tab button alongside "Faceted Search"
- Tab switching logic to show different content based on active tab
- Conditional rendering: show `RipperView` component when RIPPER tab active

#### 7. Create RIPPER Component (`frontend/src/components/RipperView.vue`)
- Display top 5 facet groups + Other group
- Each group shows:
  - Facet name and value (e.g., "Category: Electronics")
  - Item count
  - List of items (reuse `ResultsGrid` or similar)
- Loading and error states

#### 8. Add RIPPER API Function (`frontend/src/api/search.ts`)
- `searchRipper(query: string): Promise<RipperResponse>`
- Calls `/api/ripper` endpoint

#### 9. Update App State (`frontend/src/App.vue`)
- Add state for active tab ("Faceted Search" vs "RIPPER")
- Add state for RIPPER groups
- Handle tab switching
- Trigger RIPPER search when RIPPER tab is active

## Key Design Decisions

1. **Separate Endpoint**: `/api/ripper` separate from `/api/search` to avoid mixing concerns
2. **Per-Query Computation**: RIPPER facets recomputed on each search (user confirmed)
3. **Multiple Membership**: Items can appear in multiple groups (user confirmed)
4. **Uniform Weighting**: All items treated equally for information gain (user confirmed)
5. **Minimum Threshold**: `max(ceil(total * 0.05), 2)` items per group

## Testing Considerations

- Test with small result sets (< 20 items)
- Test with exactly 100 items
- Test with items that have no facets
- Test with items that have multiple facet values
- Test tie-breaking scenarios
- Test minimum threshold enforcement
- Test "Other" group with various sizes (empty, small, large)

## Files to Create/Modify

**Backend:**
- `backend/internal/ize/ripper.go` (new)
- `backend/internal/ize/ripper_test.go` (new)
- `backend/internal/algolia/client.go` (modify - add HitsPerPage)
- `backend/internal/httpapi/dto.go` (modify - add Ripper types)
- `backend/internal/httpapi/handler.go` (modify - add HandleRipper)
- `backend/cmd/server/main.go` (modify - add route)

**Frontend:**
- `frontend/src/components/RipperView.vue` (new)
- `frontend/src/components/FacetedSearch.vue` (modify - add tab)
- `frontend/src/api/search.ts` (modify - add searchRipper)
- `frontend/src/App.vue` (modify - add RIPPER state/logic)
- `frontend/src/types.ts` (modify - add RipperResponse types)
