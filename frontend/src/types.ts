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

export interface RipperGroup {
  facetName: string
  facetValue: string
  items: SearchResult[]
  count: number
}

export interface RipperResponse {
  groups: RipperGroup[]
  otherGroup: SearchResult[]
}

export interface FacetCount {
  facetName: string
  facetValue: string
  count: number
  percentage: number
}

export interface ClusterGroup {
  name: string
  items: SearchResult[]
  topFacets: FacetCount[]
}

export interface ClusterResponse {
  groups: ClusterGroup[]
  otherGroup: SearchResult[]
  clusterCount: number
}
