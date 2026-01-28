// Type definitions matching backend DTOs

export interface SearchRequest {
  query: string
}

export interface SearchResponse {
  hits: SearchResult[]
}

export interface SearchResult {
  id: string
  name: string
  description: string
  image: string
}
