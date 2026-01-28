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
