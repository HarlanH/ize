<template>
  <div class="faceted-search">
    <div class="tabs">
      <button class="tab tab--active" type="button">Faceted Search</button>
    </div>

    <div class="actions">
      <button
        class="btn-clear"
        type="button"
        :disabled="!canClear"
        @click="emit('clear')"
        title="Clear all filters"
      >
        Clear
      </button>
      <div class="actions__meta" v-if="selectedCount > 0">{{ selectedCount }} selected</div>
    </div>

    <div class="content">
      <div v-if="facetList.length === 0" class="empty">
        <div class="empty__title">No facets yet</div>
        <div class="empty__hint">Run a search to load facet counts.</div>
      </div>

      <div v-else class="facet-list">
        <Facet
          v-for="f in facetList"
          :key="f.name"
          :facet="f"
          :selected-values="selected[f.name] ?? []"
          @toggle="emit('toggle', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Facet from './Facet.vue'
import type { Facet as FacetType } from '../types'

const props = defineProps<{
  facets?: Record<string, Record<string, number>>
  selected: Record<string, string[]>
}>()

const emit = defineEmits<{
  (e: 'toggle', payload: { facet: string; value: string; checked: boolean }): void
  (e: 'clear'): void
}>()

const facetList = computed<FacetType[]>(() => {
  const facets = props.facets ?? {}
  return Object.entries(facets)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, valuesMap]) => {
      const values = Object.entries(valuesMap ?? {})
        .map(([value, count]) => ({ value, count }))
        .sort((a, b) => b.count - a.count || a.value.localeCompare(b.value))
      return { name, values }
    })
})

const selectedCount = computed(() => {
  return Object.values(props.selected).reduce((sum, arr) => sum + (arr?.length ?? 0), 0)
})
const canClear = computed(() => selectedCount.value > 0)
</script>

<style scoped>
.faceted-search {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.tabs {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid #e0e0e0;
  margin-bottom: 0.5rem;
}

.tab {
  appearance: none;
  border: 1px solid #ddd;
  background: #fff;
  color: #111;
  font-weight: 600;
  padding: 0.4rem 0.6rem;
  border-radius: 8px;
  cursor: default;
}

.tab--active {
  border-color: #cfd8ff;
  background: #f3f5ff;
}

.btn-clear {
  appearance: none;
  border: 1px solid #ddd;
  background: #fff;
  color: #111;
  padding: 0.25rem 0.45rem;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
}

.btn-clear:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.actions__meta {
  margin-left: auto;
  color: #666;
  font-size: 0.8rem;
}

.content {
  overflow-y: auto;
  padding-right: 0.25rem;
}

.empty__title {
  font-weight: 700;
  color: #111;
}

.empty__hint {
  color: #666;
  margin-top: 0.25rem;
  font-size: 0.9rem;
}

@media (prefers-color-scheme: dark) {
  .tabs {
    border-bottom-color: #333;
  }
  .tab {
    border-color: #333;
    background: #121212;
    color: #eee;
  }
  .tab--active {
    border-color: #3b4aa1;
    background: rgba(70, 87, 204, 0.18);
  }
  .empty__title {
    color: #eee;
  }
  .empty__hint {
    color: #aaa;
  }

  .btn-clear {
    border-color: #333;
    background: #121212;
    color: #eee;
  }

  .actions__meta {
    color: #aaa;
  }
}
</style>
