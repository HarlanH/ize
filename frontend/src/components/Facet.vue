<template>
  <section class="facet">
    <div class="facet__header">
      <button class="facet__toggle" type="button" @click="collapsed = !collapsed">
        <span class="facet__chev" aria-hidden="true">{{ collapsed ? '▸' : '▾' }}</span>
        <span class="facet__name">{{ facet.displayName }}</span>
      </button>
      <div class="facet__meta" v-if="selectedCount > 0">{{ selectedCount }}</div>
    </div>

    <ul v-show="!collapsed" class="facet__values">
      <li v-for="v in facet.values" :key="v.value" class="facet__value">
        <label class="facet__label">
          <input
            type="checkbox"
            :checked="selectedValuesSet.has(v.value)"
            @change="onToggle(v.value, ($event.target as HTMLInputElement).checked)"
          />
          <span class="facet__valueText">{{ displayValue(v.value) }}</span>
          <span class="facet__count">{{ v.count }}</span>
        </label>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Facet as FacetType } from '../types'

const props = defineProps<{
  facet: FacetType
  selectedValues?: string[]
}>()

const emit = defineEmits<{
  (e: 'toggle', payload: { facet: string; value: string; checked: boolean }): void
}>()

const selectedValuesSet = computed(() => new Set(props.selectedValues ?? []))
const selectedCount = computed(() => selectedValuesSet.value.size)

const collapsed = ref(false)

// Strip the configured prefix from a value for display purposes
function displayValue(value: string): string {
  const prefix = props.facet.removePrefix
  if (prefix && value.startsWith(prefix)) {
    return value.slice(prefix.length)
  }
  return value
}

function onToggle(value: string, checked: boolean) {
  emit('toggle', { facet: props.facet.name, value, checked })
}
</script>

<style scoped>
.facet {
  padding: 0.4rem 0;
  border-bottom: 1px solid #eee;
}

.facet__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.2rem;
}

.facet__toggle {
  appearance: none;
  border: 0;
  background: transparent;
  padding: 0;
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  cursor: pointer;
  text-align: left;
  min-width: 0;
}

.facet__chev {
  width: 0.9rem;
  color: #666;
  flex: 0 0 auto;
}

.facet__name {
  font-weight: 700;
  color: #111;
  font-size: 0.95rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.facet__meta {
  font-variant-numeric: tabular-nums;
  color: #666;
  font-size: 0.8rem;
  padding: 0 0.35rem;
  border: 1px solid #ddd;
  border-radius: 999px;
}

.facet__values {
  list-style: none;
  padding: 0;
  margin: 0.25rem 0 0;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.facet__label {
  display: grid;
  grid-template-columns: 16px 1fr auto;
  gap: 0.4rem;
  align-items: center;
  color: #222;
  cursor: pointer;
}

.facet__valueText {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.facet__count {
  font-variant-numeric: tabular-nums;
  color: #666;
  font-size: 0.85rem;
}

@media (prefers-color-scheme: dark) {
  .facet {
    border-bottom-color: #333;
  }
  .facet__chev {
    color: #aaa;
  }
  .facet__name {
    color: #eee;
  }
  .facet__meta {
    border-color: #333;
    color: #aaa;
  }
  .facet__label {
    color: #ddd;
  }
  .facet__count {
    color: #aaa;
  }
}
</style>
