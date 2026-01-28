// Type definitions matching backend DTOs

export interface SearchRequest {
  query: string
  facetFilters?: string[][]
}

export interface SearchResponse {
  hits: SearchResult[]
  facets?: Record<string, Record<string, number>>
}

export interface SearchResult {
  id: string
  name: string
  description: string
  image: string
}

export interface FacetValue {
  value: string
  count: number
}

export interface Facet {
  name: string
  values: FacetValue[]
}
