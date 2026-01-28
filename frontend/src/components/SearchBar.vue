<template>
  <div class="search-bar">
    <input
      v-model="query"
      type="text"
      placeholder="Search products..."
      class="search-input"
      @keyup.enter="handleSubmit"
      @input="handleInput"
    />
    <div class="search-actions">
      <button
        v-if="query"
        @click="handleClear"
        class="clear-button"
        type="button"
      >
        Clear
      </button>
      <button
        @click="handleSubmit"
        class="search-button"
        type="button"
        :disabled="loading"
      >
        {{ loading ? 'Searching...' : 'Search' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  search: [query: string]
}>()

const query = ref('')
const loading = ref(false)

const handleSubmit = () => {
  loading.value = true
  // Allow empty string search ("browse default ranking")
  emit('search', query.value.trim())
  // Reset loading after a short delay (actual loading handled by parent)
  setTimeout(() => {
    loading.value = false
  }, 100)
}

const handleClear = () => {
  query.value = ''
  emit('search', '')
}

const handleInput = () => {
  // Could implement debounced search here in the future
}
</script>

<style scoped>
.search-bar {
  display: flex;
  gap: 0.5rem;
  max-width: 800px;
  margin: 0 auto;
}

.search-input {
  flex: 1;
  padding: 0.75rem 1rem;
  font-size: 1rem;
  border: 1px solid #ccc;
  border-radius: 4px;
  outline: none;
}

.search-input:focus {
  border-color: #646cff;
  box-shadow: 0 0 0 2px rgba(100, 108, 255, 0.2);
}

.search-actions {
  display: flex;
  gap: 0.5rem;
}

.search-button,
.clear-button {
  padding: 0.75rem 1.5rem;
  font-size: 1rem;
  border: 1px solid #ccc;
  border-radius: 4px;
  cursor: pointer;
  background-color: white;
  transition: background-color 0.2s;
}

.search-button {
  background-color: #646cff;
  color: white;
  border-color: #646cff;
}

.search-button:hover:not(:disabled) {
  background-color: #535bf2;
}

.search-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.clear-button:hover {
  background-color: #f0f0f0;
}

@media (prefers-color-scheme: dark) {
  .search-input {
    background-color: #1a1a1a;
    border-color: #333;
    color: #fff;
  }

  .search-button,
  .clear-button {
    background-color: #1a1a1a;
    border-color: #333;
    color: #fff;
  }

  .clear-button:hover {
    background-color: #2a2a2a;
  }
}
</style>
