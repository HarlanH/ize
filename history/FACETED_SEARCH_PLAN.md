# Faceted Search Implementation Plan

## Overview

Implement classical faceted search functionality where:

- Facets are always displayed (even with empty query)
- Filters apply immediately when selected/deselected
- Multiple values within a facet use OR logic
- Multiple facets combine with AND logic
- All facet logic lives in the `algolia` module (not `ize`)

## Architecture

### Data Flow

```
Frontend (App.vue)
  ↓ (manages facet filter state)
FacetedSearch Component
  ↓ (displays facets, handles selection)
Search API Call (with filters)
  ↓
Backend Handler
  ↓ (passes filters to Algolia client)
Algolia Client
  ↓ (requests facets, applies filters)
Algolia API
  ↓ (returns hits + facet counts)
Backend Response (hits + facets)
  ↓
Frontend (updates results + facet counts)
```

## Backend Changes

### 1. Extend Algolia Client (`backend/internal/algolia/client.go`)

**Add methods to `Client`:**

- Update `Search` method signature to accept optional facet filters
- Request facets in search: Add `Facets` parameter to `SearchParamsObject` (request all facets or specific ones)
- Apply facet filters: Add `FacetFilters` parameter to `SearchParamsObject`
- Extract facet counts: Parse `FacetsStats` or `Facets` from Algolia response

**Update `SearchResult` struct:**

- Add `Facets` field to hold facet data with counts: `map[string]map[string]int` (facet name → value → count)

**Update `ClientInterface` (`backend/internal/algolia/interface.go`):**

- Update `Search` method signature to accept filters: `Search(ctx context.Context, query string, facetFilters []string) (*SearchResult, error)`

### 2. Update HTTP API DTOs (`backend/internal/httpapi/dto.go`)

**Update `SearchRequest`:**

- Add optional `FacetFilters` field: `FacetFilters []string \`json:"facetFilters,omitempty"\``

**Update `SearchResponse`:**

- Add `Facets` field: `Facets map[string]map[string]int \`json:"facets"\`` (facet name → value → count)

### 3. Update Search Handler (`backend/internal/httpapi/handler.go`)

**Update `HandleSearch`:**

- Extract `FacetFilters` from request
- Pass filters to `algoliaClient.Search`
- Include facets in response

## Frontend Changes

### 1. Update Types (`frontend/src/types.ts`)

**Add facet-related types:**

```typescript
export interface FacetValue {
  value: string
  count: number
}

export interface Facet {
  name: string
  values: FacetValue[]
}

export interface SearchRequest {
  query: string
  facetFilters?: string[]  // Format: ["category:Electronics", "brand:Nike"]
}

export interface SearchResponse {
  hits: SearchResult[]
  facets: Record<string, Record<string, number>>  // facet name → value → count
}
```

### 2. Create FacetedSearch Component (`frontend/src/components/FacetedSearch.vue`)

**Features:**

- Tab header: "Faceted Search"
- Display list of facets (each facet shows name and list of values with counts)
- Checkboxes for each facet value
- Visual indication of selected filters
- Handle selection/deselection (immediate application)

**Props:**

- `facets: Record<string, Record<string, number>>`
- `selectedFilters: string[]`
- `loading: boolean`

**Events:**

- `filter-changed: (filters: string[]) => void`

### 3. Create Facet Component (`frontend/src/components/Facet.vue`)

**Features:**

- Display facet name as header
- List of facet values with checkboxes
- Show count for each value
- Handle individual value selection

**Props:**

- `facetName: string`
- `values: Record<string, number>` (value → count)
- `selectedValues: string[]`

**Events:**

- `value-toggled: (facetName: string, value: string, selected: boolean) => void`

### 4. Update App.vue (`frontend/src/App.vue`)

**State management:**

- Add `selectedFacetFilters: ref<string[]>([])`
- Add `facets: ref<Record<string, Record<string, number>>>({})`

**Update `handleSearch`:**

- Pass `selectedFacetFilters` to search API
- Extract and store facets from response

**Update refinement panel:**

- Replace empty div with tabbed interface
- Add "Faceted Search" tab
- Integrate `FacetedSearch` component
- Pass facets, selected filters, and loading state

**Filter management:**

- Handle filter changes from `FacetedSearch` component
- Update `selectedFacetFilters` state
- Trigger new search with updated filters

### 5. Update Search API (`frontend/src/api/search.ts`)

**Update `search` function:**

- Accept optional `facetFilters` parameter
- Include filters in request body
- Return facets in response

### 6. Update Refinement Panel UI

**Add tab structure:**

- Create tabs component or simple tab UI
- "Faceted Search" as first tab (others can be added later)
- Style tabs appropriately

## Implementation Details

### Facet Filter Format

Algolia uses the format `"facetName:facetValue"` for facet filters. Examples:

- `"category:Electronics"`
- `"brand:Nike"`
- `"brand:Adidas"`

Multiple values in same facet: `["brand:Nike", "brand:Adidas"]` (OR logic)
Multiple facets: `["category:Electronics", "brand:Nike"]` (AND logic)

### Algolia Search Parameters

- `Facets`: Array of facet attribute names to retrieve (or `["*"]` for all)
- `FacetFilters`: Array of filter strings in format `"facet:value"`
- Response includes `Facets` object with counts per facet value

### Empty Query Handling

When query is empty, still perform search with empty string to get all records and their facets. Algolia supports empty queries.

## Testing Considerations

- Test with empty query (should show all facets)
- Test with query + no filters
- Test with query + single filter
- Test with query + multiple filters (same facet - OR)
- Test with query + multiple filters (different facets - AND)
- Test filter removal
- Test facet count updates when filters applied

## Files to Modify

**Backend:**

- `backend/internal/algolia/client.go` - Add facet support to Search method
- `backend/internal/algolia/interface.go` - Update interface signature
- `backend/internal/httpapi/dto.go` - Add facet fields to request/response
- `backend/internal/httpapi/handler.go` - Extract and pass facet filters

**Frontend:**

- `frontend/src/types.ts` - Add facet types
- `frontend/src/components/FacetedSearch.vue` - New component
- `frontend/src/components/Facet.vue` - New component (optional, can be inline)
- `frontend/src/App.vue` - Integrate faceted search, manage filter state
- `frontend/src/api/search.ts` - Update to accept and return facets

## Open Questions / Assumptions

1. **Facet attribute names**: Assumes Algolia index has `attributesForFaceting` configured. May need to document which facets are available or make configurable.
2. **Facet ordering**: Will display facets in order returned by Algolia (or alphabetically if needed)
3. **Empty state**: When no facets available, show empty state message
4. **Loading state**: Show loading indicator in facet panel during search
5. **Facet value display**: Show all facet values or limit to top N? (Assumption: show all for now)
6. **Facet value sorting**: Sort by count (descending) or alphabetically? (Assumption: by count descending, as Algolia default)
