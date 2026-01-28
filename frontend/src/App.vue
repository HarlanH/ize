<template>
  <div class="app-container">
    <!-- Top: Search Bar -->
    <div class="search-bar-container">
      <SearchBar @search="handleSearch" />
    </div>

    <!-- Main Content Area -->
    <div class="main-content">
      <!-- Left: Empty container for refinements -->
      <div class="refinement-panel">
        <!-- Reserved for future refinement controls -->
      </div>

      <!-- Right: Results Grid -->
      <div class="results-container">
        <ResultsGrid :results="results" :loading="loading" :error="error" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SearchBar from './components/SearchBar.vue'
import ResultsGrid from './components/ResultsGrid.vue'
import { search } from './api/search'
import type { SearchResult } from './types'

const results = ref<SearchResult[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const handleSearch = async (query: string) => {
  if (!query.trim()) {
    results.value = []
    return
  }

  loading.value = true
  error.value = null

  try {
    const response = await search(query)
    results.value = response.hits
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Search failed'
    results.value = []
  } finally {
    loading.value = false
  }
}
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
