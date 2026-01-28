<template>
  <div class="results-grid-container">
    <div v-if="loading" class="loading">Loading...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="results.length === 0" class="empty">
      No results found. Try a different search query.
    </div>
    <div v-else class="results-grid">
      <ResultCard
        v-for="result in results"
        :key="result.id"
        :result="result"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import ResultCard from './ResultCard.vue'
import type { SearchResult } from '../types'

defineProps<{
  results: SearchResult[]
  loading: boolean
  error: string | null
}>()
</script>

<style scoped>
.results-grid-container {
  width: 100%;
}

.loading,
.error,
.empty {
  text-align: center;
  padding: 2rem;
  color: #666;
}

.error {
  color: #d32f2f;
}

.results-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 1rem;
  padding: 0.5rem 0;
}

@media (prefers-color-scheme: dark) {
  .loading,
  .empty {
    color: #aaa;
  }
}
</style>
