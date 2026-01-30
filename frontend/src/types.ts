// Type definitions matching backend DTOs

export interface SearchRequest {
  query: string
  facetFilters?: string[][]
}

export interface FacetMeta {
  field: string
  displayName: string
  removePrefix?: string
}

export interface SearchResponse {
  hits: SearchResult[]
  facets?: Record<string, Record<string, number>>
  facetMeta?: FacetMeta[]
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
  name: string         // The actual field name (used for filtering)
  displayName: string  // User-friendly name for UI display
  removePrefix?: string // Optional prefix to strip from facet values
  values: FacetValue[]
}

export interface RipperGroup {
  facetName: string
  facetValue: string
  items: SearchResult[]
  count: number // Accurate count from Algolia facets
}

export interface RipperResponse {
  groups: RipperGroup[]
  otherGroup: SearchResult[]
  facetMeta?: FacetMeta[]
}

export interface FacetCount {
  facetName: string
  facetValue: string
  count: number
  percentage: number
}

export interface RuleQuality {
  precision: number
  recall: number
  f1: number
}

export interface ClusterGroup {
  name: string
  items: SearchResult[]
  percentage: number // Approximate percentage (~X%)
  topFacets: FacetCount[]
  rule?: string[][] // Algolia filter format for "load more"
  ruleDescription?: string // Human-readable rule
  ruleQuality?: RuleQuality // Rule quality metrics
}

export interface ClusterResponse {
  groups: ClusterGroup[]
  otherGroup: SearchResult[]
  clusterCount: number
  totalHits: number
}
