<template>
  <div class="ripper-view">
    <div v-if="props.loading" class="loading">Loading RIPPER groups...</div>
    <div v-else-if="props.error" class="error">{{ props.error }}</div>
    <div v-else-if="!groups || groups.length === 0" class="empty">
      <div class="empty__title">No RIPPER groups yet</div>
      <div class="empty__hint">Run a search to generate RIPPER groups.</div>
    </div>
    <div v-else class="groups">
      <button
        v-for="group in groups"
        :key="`${group.facetName}:${group.facetValue}`"
        class="group-item"
        type="button"
        @click="emit('select', { facetName: group.facetName, facetValue: group.facetValue })"
      >
        <span class="group-item__label">{{ group.facetName }}: {{ group.facetValue }}</span>
        <span class="group-item__count">{{ group.count }}</span>
      </button>
      <button
        v-if="otherGroup && otherGroup.length > 0"
        class="group-item"
        type="button"
        @click="emit('select-other')"
      >
        <span class="group-item__label">Other</span>
        <span class="group-item__count">{{ otherGroup.length }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { RipperGroup, SearchResult } from '../types'

const props = defineProps<{
  groups?: RipperGroup[]
  otherGroup?: SearchResult[]
  loading?: boolean
  error?: string | null
}>()

const emit = defineEmits<{
  (e: 'select', payload: { facetName: string; facetValue: string }): void
  (e: 'select-other'): void
}>()

const groups = computed(() => props.groups ?? [])
</script>

<style scoped>
.ripper-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow-y: auto;
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

.empty__title {
  font-weight: 700;
  color: #111;
  margin-bottom: 0.25rem;
}

.empty__hint {
  color: #666;
  font-size: 0.9rem;
}

.groups {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.5rem 0;
}

.group-item {
  appearance: none;
  border: 1px solid #ddd;
  background: #fff;
  padding: 0.75rem 1rem;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  text-align: left;
  transition: all 0.2s;
}

.group-item:hover {
  background: #f5f5f5;
  border-color: #bbb;
}


.group-item__label {
  font-weight: 500;
  color: #111;
  flex: 1;
}

.group-item__count {
  font-size: 0.9rem;
  color: #666;
  font-weight: 600;
  margin-left: 1rem;
}

@media (prefers-color-scheme: dark) {
  .loading,
  .empty {
    color: #aaa;
  }

  .empty__title {
    color: #eee;
  }

  .empty__hint {
    color: #aaa;
  }

  .group-item {
    border-color: #333;
    background: #121212;
    color: #eee;
  }

  .group-item:hover {
    background: #1a1a1a;
    border-color: #555;
  }


  .group-item__label {
    color: #eee;
  }

  .group-item__count {
    color: #aaa;
  }
}
</style>
