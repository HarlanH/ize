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
          :facet-meta="facetMeta"
          :selected="selectedFacetValues"
          :active-tab="activeTab"
          :ripper-groups="ripperGroups"
          :ripper-other-group="ripperOtherGroup"
          :ripper-facet-meta="ripperFacetMeta"
          :ripper-loading="ripperLoading"
          :ripper-error="ripperError"
          :ripper-filter-path="ripperFilterPath"
          :cluster-groups="clusterGroups"
          :cluster-other-group="clusterOtherGroup"
          :cluster-count="clusterCount"
          :cluster-loading="clusterLoading"
          :cluster-error="clusterError"
          :cluster-selected-name="clusterSelectedName"
          @toggle="onFacetToggle"
          @clear="clearFacetFilters"
          @tab-change="onTabChange"
          @ripper-select="onRipperSelect"
          @ripper-select-other="onRipperSelectOther"
          @ripper-clear="clearRipperFilters"
          @cluster-select="onClusterSelect"
          @cluster-select-other="onClusterSelectOther"
          @cluster-clear="clearClusterSelection"
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
import { searchWithFacetGroups, searchRipper, searchCluster } from './api/search'
import type { SearchResult, RipperGroup, ClusterGroup, FacetMeta } from './types'

const results = ref<SearchResult[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const facets = ref<Record<string, Record<string, number>> | undefined>(undefined)
const facetMeta = ref<FacetMeta[] | undefined>(undefined)
const lastQuery = ref('')
const selectedFacetValues = ref<Record<string, string[]>>({})

// RIPPER state
const activeTab = ref<'faceted' | 'ripper' | 'cluster'>('faceted')
const ripperGroups = ref<RipperGroup[] | undefined>(undefined)
const ripperOtherGroup = ref<SearchResult[] | undefined>(undefined)
const ripperFacetMeta = ref<FacetMeta[] | undefined>(undefined)
const ripperLoading = ref(false)
const ripperError = ref<string | null>(null)
const ripperFilters = ref<string[][]>([]) // Track RIPPER filter path
const ripperFilterPath = ref<string[]>([]) // Display path for UI

// Cluster state
const clusterGroups = ref<ClusterGroup[] | undefined>(undefined)
const clusterOtherGroup = ref<SearchResult[] | undefined>(undefined)
const clusterCount = ref<number>(0)
const clusterLoading = ref(false)
const clusterError = ref<string | null>(null)
const clusterSelectedName = ref<string | null>(null) // Track selected cluster for Clear button
const clusterFilters = ref<string[][]>([]) // Accumulated filters for drill-down

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
    facetMeta.value = response.facetMeta
    return true
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Search failed'
    results.value = []
    facets.value = undefined
    facetMeta.value = undefined
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
  ripperFilters.value = []
  ripperFilterPath.value = []
  await runSearch(normalized, [])
  // If RIPPER tab is active, also run RIPPER search
  if (activeTab.value === 'ripper') {
    await runRipperSearch(normalized, [])
  }
  // If Cluster tab is active, also run Cluster search
  if (activeTab.value === 'cluster') {
    await runClusterSearch(normalized, [])
  }
}

async function runRipperSearch(query: string, facetFilters: string[][] = []): Promise<boolean> {
  ripperLoading.value = true
  ripperError.value = null
  try {
    const response = await searchRipper(query, facetFilters)
    ripperGroups.value = response.groups
    ripperOtherGroup.value = response.otherGroup
    ripperFacetMeta.value = response.facetMeta
    return true
  } catch (err) {
    ripperError.value = err instanceof Error ? err.message : 'RIPPER search failed'
    ripperGroups.value = undefined
    ripperOtherGroup.value = undefined
    ripperFacetMeta.value = undefined
    return false
  } finally {
    ripperLoading.value = false
  }
}

function onRipperSelect(payload: { facetName: string; facetValue: string }) {
  // Add the selected facet filter to the RIPPER filter path
  const filterString = `${payload.facetName}:${payload.facetValue}`
  ripperFilters.value = [...ripperFilters.value, [filterString]]
  ripperFilterPath.value = [...ripperFilterPath.value, filterString]
  
  // Re-run RIPPER search with the new filter (for next 5 groups)
  void runRipperSearch(lastQuery.value, ripperFilters.value)

  // Also update the main results grid to reflect the currently applied RIPPER filters
  // (so the user sees the filtered result set as they drill down)
  void runSearch(lastQuery.value, ripperFilters.value)
}

function onRipperSelectOther() {
  // When "Other" is clicked, we want to show items that don't match any selected facet values
  // Algolia supports negated facet filters using the "-" prefix: "facetName:-value"
  // 
  // We need to exclude all selected facet values. For each selected group,
  // we add a negated filter for that facet value.
  // 
  // Example: If we have selected groups:
  //   - brand:Samsung
  //   - category:Electronics
  // Then "Other" filters would be:
  //   [["brand:-Samsung"], ["category:-Electronics"]]
  // This means: brand != Samsung AND category != Electronics
  
  const excludeFilters: string[][] = []
  if (ripperGroups.value) {
    for (const group of ripperGroups.value) {
      // Add negated filter for this facet value
      const negatedFilter = `${group.facetName}:-${group.facetValue}`
      excludeFilters.push([negatedFilter])
    }
  }
  
  // Combine RIPPER filters (if any) with the exclusion filters
  // RIPPER filters narrow the set, exclusion filters remove selected groups
  const combinedFilters = [...ripperFilters.value, ...excludeFilters]
  
  // Update the main results grid to show only "Other" items
  void runSearch(lastQuery.value, combinedFilters)
}

function clearRipperFilters() {
  ripperFilters.value = []
  ripperFilterPath.value = []
  // Re-run RIPPER search without filters
  void runRipperSearch(lastQuery.value, [])
  
  // Reset the main results grid to the unfiltered set for the current query
  void runSearch(lastQuery.value, [])
}

function onTabChange(tab: 'faceted' | 'ripper' | 'cluster') {
  activeTab.value = tab
  // When switching to RIPPER tab, trigger RIPPER search if we have a query
  if (tab === 'ripper' && lastQuery.value !== undefined) {
    // Reset RIPPER filters when switching to RIPPER tab
    ripperFilters.value = []
    ripperFilterPath.value = []
    void runRipperSearch(lastQuery.value, [])
  }
  // When switching to Cluster tab, trigger Cluster search if we have a query
  if (tab === 'cluster' && lastQuery.value !== undefined) {
    void runClusterSearch(lastQuery.value, [])
  }
}

async function runClusterSearch(query: string, facetFilters: string[][] = []): Promise<boolean> {
  clusterLoading.value = true
  clusterError.value = null
  try {
    const response = await searchCluster(query, facetFilters)
    clusterGroups.value = response.groups
    clusterOtherGroup.value = response.otherGroup
    clusterCount.value = response.clusterCount
    return true
  } catch (err) {
    clusterError.value = err instanceof Error ? err.message : 'Cluster search failed'
    clusterGroups.value = undefined
    clusterOtherGroup.value = undefined
    clusterCount.value = 0
    return false
  } finally {
    clusterLoading.value = false
  }
}

async function onClusterSelect(payload: { index: number; name: string; rule?: string[][] }) {
  // When a cluster is selected, use its rule to drill down
  if (!payload.rule || payload.rule.length === 0) {
    // No rule, just show the items (fallback)
    if (clusterGroups.value && clusterGroups.value[payload.index]) {
      results.value = clusterGroups.value[payload.index].items
      clusterSelectedName.value = payload.name
    }
    return
  }

  // Add the rule clauses to accumulated filters and re-run clustering
  clusterFilters.value = [...clusterFilters.value, ...payload.rule]
  clusterSelectedName.value = payload.name

  // Re-run the main search with accumulated filters
  await runSearch(lastQuery.value, clusterFilters.value)

  // Re-run clustering with accumulated filters to get sub-clusters
  await runClusterSearch(lastQuery.value, clusterFilters.value)
}

function onClusterSelectOther() {
  // When "Other" is clicked, show the other group items in the results grid
  if (clusterOtherGroup.value) {
    results.value = clusterOtherGroup.value
    clusterSelectedName.value = 'Other'
  }
}

function clearClusterSelection() {
  // Reset filters and re-run clustering from scratch
  clusterFilters.value = []
  clusterSelectedName.value = null

  // Re-run the search without cluster filters
  void runSearch(lastQuery.value, [])

  // Re-run clustering without filters
  void runClusterSearch(lastQuery.value, [])
}

onMounted(() => {
  // Initial browse: empty query shows default ranking + facets.
  const maxAttempts = 6
  const baseDelayMs = 200

  const tryInitialBrowse = async (attempt: number) => {
    const ok = await runSearch('', [])
    if (ok) {
      // Also run RIPPER search if RIPPER tab is active
      if (activeTab.value === 'ripper') {
        ripperFilters.value = []
        ripperFilterPath.value = []
        await runRipperSearch('', [])
      }
      // Also run Cluster search if Cluster tab is active
      if (activeTab.value === 'cluster') {
        await runClusterSearch('', [])
      }
      return
    }

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
