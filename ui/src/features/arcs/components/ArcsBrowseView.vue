<script setup>
import { computed } from 'vue'
import BrowseListTools from '@/shared/components/browse/BrowseListTools.vue'
import BrowseEntityRow from '@/shared/components/browse/BrowseEntityRow.vue'
import BrowseListSection from '@/shared/components/browse/BrowseListSection.vue'
import BrowseRowStats from '@/shared/components/browse/BrowseRowStats.vue'
import { ENGAGEMENT_FILTER_OPTIONS } from '@/shared/browseOptions.js'
import { formatProgress } from '@/features/reading-orders/model.js'

const props = defineProps({
  arcs: {
    type: Array,
    default: () => [],
  },
  selectedArcId: {
    type: Number,
    default: null,
  },
  quickSavingArcId: {
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
  'open-arc',
  'toggle-favorite',
])

const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'progress', label: 'Progress' },
  { value: 'favoriteCount', label: 'Favorites' },
  { value: 'startedCount', label: 'Currently reading' },
]

const sectionTitle = computed(
  () =>
    ({ favorites: 'Favorites', other: 'Other Arcs', started: 'Started Arcs' })[props.filter] ||
    'All Arcs',
)
</script>

<template>
  <div class="browse-view min-w-0 w-full">
    <div class="list-pane grid gap-3">
      <div
        class="browse-list-sticky max-w-none sticky [top:var(--comic-list-sticky-top)] z-18 grid gap-2.5 [margin-inline:calc(var(--sticky-toolbar-inline-offset)_*_-1)] [padding:12px_var(--sticky-toolbar-inline-offset)] border-b border-sticky-border bg-sticky-bg shadow-sticky-soft backdrop-blur-ui down-tablet:[&_.comic-list-header]:items-stretch down-tablet:[&_.comic-list-header]:flex-col down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:pb-3 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none"
      >
        <div
          class="comic-list-header flex items-center justify-between gap-3 [&_>_*]:min-w-0 [&_.eyebrow]:mb-0.5 [&_small]:text-muted desktop-compact:items-stretch desktop-compact:flex-wrap"
        >
          <BrowseListTools
            :search="search"
            search-placeholder="Search arcs"
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
      </div>
      <div v-if="arcs.length" class="grid gap-4">
        <BrowseListSection :title="sectionTitle" :items="arcs">
          <template #item="{ item: arc }">
            <BrowseEntityRow
              :title="arc.name"
              :subtitle="arc.description || 'No description'"
              :image="arc.image"
              main-class="[&_>_span:last-child]:min-w-0 flex items-center gap-2.5"
              :selected="selectedArcId === arc.id"
              :favorite="arc.favorite"
              :can-favorite="!readOnly"
              :favorite-saving="quickSavingArcId === arc.id"
              :progress="formatProgress(arc.progress)"
              progress-label="Arc progress"
              @open="$emit('open-arc', arc)"
              @toggle-favorite="$emit('toggle-favorite', arc)"
            >
              <template #byline>
                <span
                  v-if="arc.startedAt"
                  class="started-pill inline-flex items-center w-fit mt-2 border border-primary rounded-full bg-primary-soft text-primary-strong py-1 px-2 text-xs font-extrabold leading-tight"
                  >Started</span
                >
                <BrowseRowStats
                  :items="[`${arc.favoriteCount} favorites`, `${arc.startedCount} reading`]"
                />
              </template>
            </BrowseEntityRow>
          </template>
        </BrowseListSection>
      </div>
    </div>
  </div>
</template>
