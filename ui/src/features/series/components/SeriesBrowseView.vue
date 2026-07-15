<script setup>
import { computed } from 'vue'
import BrowseListTools from '@/shared/components/browse/BrowseListTools.vue'
import BrowseEntityRow from '@/shared/components/browse/BrowseEntityRow.vue'
import BrowseListSection from '@/shared/components/browse/BrowseListSection.vue'
import BrowseRowStats from '@/shared/components/browse/BrowseRowStats.vue'
import { ENGAGEMENT_FILTER_OPTIONS } from '@/shared/browseOptions.js'
import { formatProgress } from '@/features/reading-orders/model.js'

const props = defineProps({
  loading: {
    type: Boolean,
    default: false,
  },
  series: {
    type: Array,
    default: () => [],
  },
  selectedSeriesId: {
    type: Number,
    default: null,
  },
  search: {
    type: String,
    default: '',
  },
  searchTerm: {
    type: String,
    default: '',
  },
  filter: {
    type: String,
    default: 'all',
  },
  sort: {
    type: String,
    default: 'name',
  },
  direction: {
    type: String,
    default: 'asc',
  },
  readOnly: {
    type: Boolean,
    default: false,
  },
})

defineEmits([
  'update:search',
  'update:filter',
  'update:sort',
  'update:direction',
  'open-series',
  'toggle-favorite',
  'new-comic',
])

const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'year', label: 'Year' },
  { value: 'publisher', label: 'Publisher' },
  { value: 'entries', label: 'Entries' },
  { value: 'progress', label: 'Progress' },
  { value: 'favoriteCount', label: 'Favorites' },
  { value: 'startedCount', label: 'Currently reading' },
]

const sectionTitle = computed(
  () =>
    ({ favorites: 'Favorites', other: 'Other Series', started: 'Started Series' })[props.filter] ||
    'All Series',
)
const hasFilters = computed(() => props.searchTerm || props.filter !== 'all')

function seriesYearLabel(series) {
  return series?.seriesYear ? ` (${series.seriesYear})` : ''
}

function seriesPublisherLabel(series) {
  if (!series?.publishers?.length) return 'No publisher saved'
  return series.publishers.join(', ')
}
</script>

<template>
  <div class="browse-view">
    <div class="list-pane">
      <div class="browse-list-sticky">
        <BrowseListTools
          :search="search"
          search-placeholder="Search series"
          :filter="filter"
          :sort="sort"
          :direction="direction"
          :sort-options="sortOptions"
          :filter-options="ENGAGEMENT_FILTER_OPTIONS"
          @update:search="$emit('update:search', $event)"
          @update:filter="$emit('update:filter', $event)"
          @update:sort="$emit('update:sort', $event)"
          @update:direction="$emit('update:direction', $event)"
        />
      </div>
      <div
        v-if="loading && !series.length"
        class="inline-loading-panel"
        role="status"
        aria-live="polite"
      >
        <span class="loading-spinner small" aria-hidden="true"></span>
        <strong>Loading series...</strong>
      </div>
      <div v-else-if="series.length" class="sectioned-list">
        <BrowseListSection :title="sectionTitle" :items="series">
          <template #item="{ item }">
            <BrowseEntityRow
              :title="`${item.name}${seriesYearLabel(item)}`"
              :subtitle="seriesPublisherLabel(item)"
              :image="item.coverImage"
              main-class="series-row-main"
              :selected="selectedSeriesId === item.id"
              :favorite="item.favorite"
              :can-favorite="!readOnly"
              :progress="formatProgress(item.progress)"
              progress-label="Series read progress"
              @open="$emit('open-series', item)"
              @toggle-favorite="$emit('toggle-favorite', item)"
            >
              <template #byline>
                <span v-if="item.startedAt" class="started-pill">Started</span>
                <BrowseRowStats
                  :items="[
                    `${item.entryCount} entries`,
                    formatProgress(item.progress),
                    `${item.favoriteCount} favorites`,
                    `${item.startedCount} reading`,
                  ]"
                />
              </template>
            </BrowseEntityRow>
          </template>
        </BrowseListSection>
      </div>
      <div v-else class="empty-state">
        {{ hasFilters ? 'No series match these filters.' : 'No series available yet.' }}
        <button
          v-if="!hasFilters && !readOnly"
          class="secondary-button"
          type="button"
          @click="$emit('new-comic')"
        >
          <span aria-hidden="true" class="button-icon">+</span>
          Add the first comic
        </button>
      </div>
    </div>
  </div>
</template>
