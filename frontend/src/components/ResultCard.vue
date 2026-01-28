<template>
  <div class="result-card">
    <div class="card-image-container">
      <img
        v-if="result.image"
        :src="result.image"
        :alt="result.name"
        class="card-image"
        @error="handleImageError"
      />
      <div v-else class="card-image-placeholder">No Image</div>
    </div>
    <div class="card-content">
      <h3 class="card-title">{{ result.name }}</h3>
      <p class="card-description">{{ result.description }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { SearchResult } from '../types'

defineProps<{
  result: SearchResult
}>()

const imageError = ref(false)

const handleImageError = () => {
  imageError.value = true
}
</script>

<style scoped>
.result-card {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  overflow: hidden;
  background-color: white;
  transition: box-shadow 0.2s;
  cursor: pointer;
}

.result-card:hover {
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.card-image-container {
  width: 100%;
  aspect-ratio: 4/3;
  overflow: hidden;
  background-color: #f5f5f5;
}

.card-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.card-image-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #999;
  font-size: 0.9rem;
}

.card-content {
  padding: 0.75rem;
}

.card-title {
  margin: 0 0 0.25rem 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: #333;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.3;
}

.card-description {
  margin: 0;
  font-size: 0.85rem;
  color: #666;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

@media (prefers-color-scheme: dark) {
  .result-card {
    background-color: #1a1a1a;
    border-color: #333;
  }

  .card-image-container {
    background-color: #2a2a2a;
  }

  .card-title {
    color: #fff;
  }

  .card-description {
    color: #aaa;
  }
}
</style>
