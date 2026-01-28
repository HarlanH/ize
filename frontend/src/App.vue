<template>
  <div class="app-container">
    <!-- Top: Search Bar -->
    <div class="search-bar-container">
      <SearchBar @search="handleSearch" />
    </div>

    <!-- Main Content Area -->
    <div class="main-content">
      <!-- Left: Refinements -->
      <div class="refinement-panel">
        <FacetedSearch
          :facets="facets"
          :selected="selectedFacetValues"
          @toggle="onFacetToggle"
          @clear="clearFacetFilters"
        />
      </div>

      <!-- Right: Results Grid -->
      <div class="results-container">
        <ResultsGrid :results="results" :loading="loading" :error="error" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import SearchBar from './components/SearchBar.vue'
import ResultsGrid from './components/ResultsGrid.vue'
import FacetedSearch from './components/FacetedSearch.vue'
import { searchWithFacetGroups } from './api/search'
import type { SearchResult } from './types'

const results = ref<SearchResult[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const facets = ref<Record<string, Record<string, number>> | undefined>(undefined)
const lastQuery = ref('')
const selectedFacetValues = ref<Record<string, string[]>>({})

function onFacetToggle(payload: { facet: string; value: string; checked: boolean }) {
  const curr = selectedFacetValues.value[payload.facet] ?? []
  const set = new Set(curr)
  if (payload.checked) set.add(payload.value)
  else set.delete(payload.value)
  
  const updated: Record<string, string[]> = { ...selectedFacetValues.value }
  const newValues = Array.from(set).sort((a, b) => a.localeCompare(b))
  if (newValues.length > 0) {
    updated[payload.facet] = newValues
  } else {
    delete updated[payload.facet]
  }
  selectedFacetValues.value = updated
  
  // Auto-apply filters immediately
  const groups: string[][] = Object.entries(selectedFacetValues.value)
    .filter(([, values]) => values.length > 0)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([facet, values]) => values.map((v) => `${facet}:${v}`))
  void runSearch(lastQuery.value, groups)
}

function clearFacetFilters() {
  selectedFacetValues.value = {}
  // Re-run the last query without filters (including empty string browse)
  void runSearch(lastQuery.value, [])
}

async function runSearch(query: string, facetGroups: string[][]): Promise<boolean> {
  loading.value = true
  error.value = null
  try {
    const response = await searchWithFacetGroups(query, facetGroups)
    results.value = response.hits
    facets.value = response.facets
    return true
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Search failed'
    results.value = []
    facets.value = undefined
    return false
  } finally {
    loading.value = false
  }
}

const handleSearch = async (query: string) => {
  const normalized = query.trim()
  lastQuery.value = normalized
  // New search resets filters for now.
  selectedFacetValues.value = {}
  await runSearch(normalized, [])
}

onMounted(() => {
  // Initial browse: empty query shows default ranking + facets.
  const maxAttempts = 6
  const baseDelayMs = 200

  const tryInitialBrowse = async (attempt: number) => {
    const ok = await runSearch('', [])
    if (ok) return

    // runSearch already sets error state; we just retry a few times in case
    // the frontend starts before the backend proxy is ready.
    if (attempt >= maxAttempts) return
    const delay = baseDelayMs * Math.pow(2, attempt - 1)
    setTimeout(() => void tryInitialBrowse(attempt + 1), delay)
  }

  void tryInitialBrowse(1)
})
</script>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100%;
}

.search-bar-container {
  padding: 1rem;
  border-bottom: 1px solid #e0e0e0;
  background-color: #f5f5f5;
}

.main-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.refinement-panel {
  width: 300px;
  border-right: 1px solid #e0e0e0;
  background-color: #fafafa;
  padding: 1rem;
  /* Reserved for future controls */
}

.results-container {
  flex: 1;
  overflow-y: auto;
  padding: 0.75rem;
}

@media (prefers-color-scheme: dark) {
  .search-bar-container {
    background-color: #1a1a1a;
    border-bottom-color: #333;
  }
  
  .refinement-panel {
    background-color: #1a1a1a;
    border-right-color: #333;
  }
}
</style>
