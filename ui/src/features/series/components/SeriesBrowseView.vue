<script setup>
import { computed } from 'vue'
import BrowseListTools from '@/shared/components/browse/BrowseListTools.vue'
import BrowseEntityRow from '@/shared/components/browse/BrowseEntityRow.vue'
import BrowseListSection from '@/shared/components/browse/BrowseListSection.vue'
import BrowseRowStats from '@/shared/components/browse/BrowseRowStats.vue'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'
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
  <div class="browse-view min-w-0 w-full">
    <div class="list-pane grid gap-3">
      <div
        class="browse-list-sticky max-w-none sticky [top:var(--comic-list-sticky-top)] z-18 grid gap-2.5 [margin-inline:calc(var(--sticky-toolbar-inline-offset)_*_-1)] [padding:12px_var(--sticky-toolbar-inline-offset)] border-b border-sticky-border bg-sticky-bg shadow-sticky-soft backdrop-blur-ui down-tablet:[&_.comic-list-header]:items-stretch down-tablet:[&_.comic-list-header]:flex-col down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:pb-3 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none"
      >
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
      <LoadingState v-if="loading && !series.length" />
      <div v-else-if="series.length" class="grid gap-4">
        <BrowseListSection :title="sectionTitle" :items="series">
          <template #item="{ item }">
            <BrowseEntityRow
              :title="`${item.name}${seriesYearLabel(item)}`"
              :subtitle="seriesPublisherLabel(item)"
              :image="item.coverImage"
              main-class="flex items-center gap-2.5 [&_>_span:last-child]:min-w-0"
              :selected="selectedSeriesId === item.id"
              :favorite="item.favorite"
              :can-favorite="!readOnly"
              :progress="formatProgress(item.progress)"
              progress-label="Series read progress"
              @open="$emit('open-series', item)"
              @toggle-favorite="$emit('toggle-favorite', item)"
            >
              <template #byline>
                <span
                  v-if="item.startedAt"
                  class="started-pill inline-flex items-center w-fit mt-2 border border-primary rounded-full bg-primary-soft text-primary-strong py-1 px-2 text-xs font-extrabold leading-tight"
                  >Started</span
                >
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
      <div
        v-else
        class="empty-state grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4"
      >
        {{ hasFilters ? 'No series match these filters.' : 'No series available yet.' }}
        <button
          v-if="!hasFilters && !readOnly"
          class="secondary-button min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]"
          type="button"
          @click="$emit('new-comic')"
        >
          <span
            aria-hidden="true"
            class="button-icon inline-flex items-center justify-center size-5 text-xl font-extrabold leading-none"
            >+</span
          >
          Add the first comic
        </button>
      </div>
    </div>
  </div>
</template>
