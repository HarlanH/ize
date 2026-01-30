<template>
  <div class="cluster-view">
    <div v-if="props.loading" class="loading">Loading clusters...</div>
    <div v-else-if="props.error" class="error">{{ props.error }}</div>
    <div v-else-if="!groups || groups.length === 0" class="empty">
      <div class="empty__title">No clusters yet</div>
      <div class="empty__hint">Run a search to generate clusters.</div>
    </div>
    <div v-else class="groups">
      <div class="cluster-count">{{ props.clusterCount }} clusters found</div>
      <div
        v-for="(group, index) in groups"
        :key="index"
        class="group-item"
        :class="{ 'group-item--selected': props.selectedName === group.name }"
      >
        <button
          class="group-item__main"
          type="button"
          @click="emit('select', { index, name: group.name, rule: group.rule })"
        >
          <span class="group-item__name">{{ group.name }}</span>
          <span class="group-item__count">{{ group.items.length }}</span>
        </button>
        <button
          v-if="group.topFacets && group.topFacets.length > 0"
          class="group-item__toggle"
          type="button"
          @click.stop="toggleFacets(index)"
          :title="expandedFacets.has(index) ? 'Hide details' : 'Show details'"
        >
          <span class="toggle-icon">{{ expandedFacets.has(index) ? '▼' : '▶' }}</span>
          <span class="toggle-label">Details</span>
        </button>
        <div v-if="expandedFacets.has(index)" class="group-item__details">
          <!-- Rule description -->
          <div v-if="group.ruleDescription" class="rule-section">
            <div class="rule-label">Filter rule:</div>
            <code class="rule-code">{{ group.ruleDescription }}</code>
          </div>
          <!-- Top facets -->
          <div class="facets-section">
            <div class="facets-label">Top facets:</div>
            <div class="facet-tags">
              <span
                v-for="facet in group.topFacets"
                :key="`${facet.facetName}:${facet.facetValue}`"
                class="facet-tag"
              >
                {{ facet.facetName }}: {{ facet.facetValue }}
                <span class="facet-tag__pct">{{ Math.round(facet.percentage) }}%</span>
              </span>
            </div>
          </div>
        </div>
      </div>
      <div
        v-if="otherGroup && otherGroup.length > 0"
        class="group-item group-item--other"
      >
        <button
          class="group-item__main"
          type="button"
          @click="emit('select-other')"
        >
          <span class="group-item__name">Other</span>
          <span class="group-item__count">{{ otherGroup.length }}</span>
        </button>
        <div class="group-item__hint">Items that don't fit well in any cluster</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ClusterGroup, SearchResult } from '../types'

const props = defineProps<{
  groups?: ClusterGroup[]
  otherGroup?: SearchResult[]
  clusterCount?: number
  loading?: boolean
  error?: string | null
  selectedName?: string | null
}>()

const emit = defineEmits<{
  (e: 'select', payload: { index: number; name: string; rule?: string[][] }): void
  (e: 'select-other'): void
}>()

const groups = computed(() => props.groups ?? [])
const otherGroup = computed(() => props.otherGroup ?? [])

// Track which clusters have expanded facets (default: all collapsed)
const expandedFacets = ref<Set<number>>(new Set())

function toggleFacets(index: number) {
  const newSet = new Set(expandedFacets.value)
  if (newSet.has(index)) {
    newSet.delete(index)
  } else {
    newSet.add(index)
  }
  expandedFacets.value = newSet
}
</script>

<style scoped>
.cluster-view {
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

.cluster-count {
  font-size: 0.8rem;
  color: #666;
  margin-bottom: 0.25rem;
}

.group-item {
  border: 1px solid #ddd;
  background: #fff;
  border-radius: 6px;
  overflow: hidden;
}

.group-item--other {
  border-style: dashed;
}

.group-item__main {
  appearance: none;
  border: none;
  background: transparent;
  width: 100%;
  padding: 0.75rem 1rem;
  cursor: pointer;
  text-align: left;
  display: flex;
  justify-content: space-between;
  align-items: center;
  transition: background 0.2s;
}

.group-item__main:hover {
  background: #f5f5f5;
}

.group-item__name {
  font-weight: 600;
  color: #111;
  flex: 1;
}

.group-item__count {
  font-size: 0.9rem;
  color: #666;
  font-weight: 600;
  margin-left: 1rem;
}

.group-item__toggle {
  appearance: none;
  border: none;
  border-top: 1px solid #eee;
  background: #fafafa;
  width: 100%;
  padding: 0.4rem 1rem;
  cursor: pointer;
  text-align: left;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  color: #666;
  transition: background 0.2s;
}

.group-item__toggle:hover {
  background: #f0f0f0;
}

.toggle-icon {
  font-size: 0.6rem;
  color: #999;
}

.toggle-label {
  color: #888;
}

.group-item--selected {
  border-color: #1976d2;
  box-shadow: 0 0 0 1px #1976d2;
}

.group-item__details {
  padding: 0.75rem 1rem;
  background: #fafafa;
  border-top: 1px solid #eee;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.rule-section {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.rule-label,
.facets-label {
  font-size: 0.7rem;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.rule-code {
  font-family: 'SF Mono', Menlo, Monaco, monospace;
  font-size: 0.75rem;
  background: #e8e8e8;
  padding: 0.35rem 0.5rem;
  border-radius: 4px;
  color: #333;
  word-break: break-all;
}

.facets-section {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.facet-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.group-item__hint {
  font-size: 0.75rem;
  color: #888;
  font-style: italic;
  padding: 0 1rem 0.5rem;
}

.facet-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  background: #e8e8e8;
  color: #555;
  font-size: 0.75rem;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
}

.facet-tag__pct {
  color: #888;
  font-size: 0.7rem;
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

  .cluster-count {
    color: #aaa;
  }

  .group-item {
    border-color: #333;
    background: #121212;
  }

  .group-item--selected {
    border-color: #42a5f5;
    box-shadow: 0 0 0 1px #42a5f5;
  }

  .group-item__main:hover {
    background: #1a1a1a;
  }

  .group-item__name {
    color: #eee;
  }

  .group-item__count {
    color: #aaa;
  }

  .group-item__toggle {
    border-top-color: #2a2a2a;
    background: #1a1a1a;
    color: #888;
  }

  .group-item__toggle:hover {
    background: #222;
  }

  .toggle-icon {
    color: #666;
  }

  .toggle-label {
    color: #777;
  }

  .group-item__details {
    background: #1a1a1a;
    border-top-color: #2a2a2a;
  }

  .rule-label,
  .facets-label {
    color: #777;
  }

  .rule-code {
    background: #2a2a2a;
    color: #ddd;
  }

  .group-item__hint {
    color: #666;
  }

  .facet-tag {
    background: #2a2a2a;
    color: #bbb;
  }

  .facet-tag__pct {
    color: #777;
  }
}
</style>
