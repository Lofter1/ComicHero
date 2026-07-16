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
  <div class="browse-view">
    <div class="list-pane">
      <div class="browse-list-sticky">
        <div class="comic-list-header">
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
      <div v-if="arcs.length" class="sectioned-list">
        <BrowseListSection :title="sectionTitle" :items="arcs">
          <template #item="{ item: arc }">
            <BrowseEntityRow
              :title="arc.name"
              :subtitle="arc.description || 'No description'"
              :image="arc.image"
              main-class="arc-row-main"
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
                <span v-if="arc.startedAt" class="started-pill">Started</span>
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
