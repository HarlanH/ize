import type { SearchRequest, SearchResponse } from '../types'

// Use relative URL to leverage Vite proxy in development
// In production, set VITE_API_URL environment variable if backend is on different domain
const API_BASE_URL = import.meta.env.VITE_API_URL || ''

export async function search(query: string): Promise<SearchResponse> {
  const response = await fetch(`${API_BASE_URL}/api/search`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ query } as SearchRequest),
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(`Search failed: ${response.status} ${errorText}`)
  }

  return response.json() as Promise<SearchResponse>
}

export async function searchWithFacets(query: string, facetFilters: string[] = []): Promise<SearchResponse> {
  const requestBody: SearchRequest = { query }
  // Back-end expects grouped facetFilters: AND across groups, OR within a group.
  // For the simple case of an AND list, pass each filter as a single-item group.
  if (facetFilters.length > 0) requestBody.facetFilters = facetFilters.map((f) => [f])

  const response = await fetch(`${API_BASE_URL}/api/search`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(requestBody),
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(`Search failed: ${response.status} ${errorText}`)
  }

  return response.json() as Promise<SearchResponse>
}

export async function searchWithFacetGroups(
  query: string,
  facetFilterGroups: string[][] = [],
): Promise<SearchResponse> {
  const requestBody: SearchRequest = { query }
  if (facetFilterGroups.length > 0) requestBody.facetFilters = facetFilterGroups

  const response = await fetch(`${API_BASE_URL}/api/search`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(requestBody),
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(`Search failed: ${response.status} ${errorText}`)
  }

  return response.json() as Promise<SearchResponse>
}
