<script setup>
import { computed } from 'vue'
import { assetURL } from '@/api/client.js'
import BrowseListTools from '@/components/BrowseListTools.vue'
import { formatProgress } from '@/domain/readingOrders.js'

const props = defineProps({
  totalCount: {
    type: Number,
    default: 0,
  },
  favoriteCount: {
    type: Number,
    default: 0,
  },
  entryCount: {
    type: Number,
    default: 0,
  },
  series: {
    type: Array,
    default: () => [],
  },
  sections: {
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
})

defineEmits(['update:search', 'update:filter', 'update:sort', 'update:direction', 'open-series', 'toggle-favorite', 'new-comic'])

const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'year', label: 'Year' },
  { value: 'publisher', label: 'Publisher' },
  { value: 'entries', label: 'Entries' },
  { value: 'progress', label: 'Progress' },
]

const visibleSeries = computed(() => props.series)
const visibleSections = computed(() => {
  if (props.filter === 'favorites') return sectionList('Favorites', visibleSeries.value)
  if (props.filter === 'other') return sectionList('Other Series', visibleSeries.value)
  return sectionList('All Series', visibleSeries.value)
})
const hasFilters = computed(() => props.searchTerm || props.filter !== 'all')

function sectionList(title, series) {
  return series.length ? [{ key: props.filter, title, series }] : []
}

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
        <div class="overview-strip">
          <span>
            <strong>{{ totalCount }}</strong>
            <small>Series</small>
          </span>
          <span>
            <strong>{{ favoriteCount }}</strong>
            <small>Favorites</small>
          </span>
          <span>
            <strong>{{ entryCount }}</strong>
            <small>Entries</small>
          </span>
        </div>
        <BrowseListTools
          :search="search"
          search-placeholder="Search series"
          :filter="filter"
          :sort="sort"
          :direction="direction"
          :sort-options="sortOptions"
          @update:search="$emit('update:search', $event)"
          @update:filter="$emit('update:filter', $event)"
          @update:sort="$emit('update:sort', $event)"
          @update:direction="$emit('update:direction', $event)"
        />
      </div>
      <div v-if="visibleSeries.length" class="sectioned-list">
        <section v-for="section in visibleSections" :key="section.key" class="list-section">
          <div class="list-section-header">
            <p class="eyebrow">{{ section.title }}</p>
            <small>{{ section.series.length }}</small>
          </div>
          <div class="list">
            <div
              v-for="item in section.series"
              :key="item.id"
              class="row series-row"
              :class="{ selected: selectedSeriesId === item.id }"
            >
              <span class="order-row-content">
                <button class="row-main series-row-main" type="button" @click="$emit('open-series', item)">
                  <span v-if="item.coverImage" class="series-list-cover" aria-hidden="true">
                    <img :src="assetURL(item.coverImage)" alt="" loading="lazy" />
                  </span>
                  <span>
                    <strong>{{ item.name }}{{ seriesYearLabel(item) }}</strong>
                    <small>{{ seriesPublisherLabel(item) }}</small>
                  </span>
                </button>
                <button
                  type="button"
                  class="favorite-toggle"
                  :class="{ active: item.favorite }"
                  :aria-label="item.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  :title="item.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  @click="$emit('toggle-favorite', item)"
                >
                  <span aria-hidden="true">{{ item.favorite ? '★' : '☆' }}</span>
                </button>
              </span>
              <span class="row-meta">
                <span class="status-pill">{{ item.entryCount }} entries</span>
                <span class="status-pill">{{ formatProgress(item.progress) }}</span>
              </span>
              <span class="row-progress" aria-label="Series read progress">
                <span :style="{ width: formatProgress(item.progress) }"></span>
              </span>
            </div>
          </div>
        </section>
      </div>
      <div v-else class="empty-state">
        {{ hasFilters ? 'No series match these filters.' : 'No series available yet.' }}
        <button v-if="!hasFilters" class="secondary-button" type="button" @click="$emit('new-comic')">
          <span aria-hidden="true" class="button-icon">+</span>
          Add the first comic
        </button>
      </div>
    </div>
  </div>
</template>
