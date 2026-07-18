<script setup>
import { computed } from 'vue'
import BrowseListTools from '@/shared/components/browse/BrowseListTools.vue'
import BrowseEntityRow from '@/shared/components/browse/BrowseEntityRow.vue'
import BrowseListSection from '@/shared/components/browse/BrowseListSection.vue'
import BrowseRowStats from '@/shared/components/browse/BrowseRowStats.vue'
import { ENGAGEMENT_FILTER_OPTIONS } from '@/shared/browseOptions.js'
import { formatProgress } from '@/features/reading-orders/model.js'

const props = defineProps({
  characters: {
    type: Array,
    default: () => [],
  },
  selectedCharacterId: {
    type: Number,
    default: null,
  },
  quickSavingCharacterId: {
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
  'open-character',
  'toggle-favorite',
  'add-to-collection',
])

const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'appearances', label: 'Appearances' },
  { value: 'aliases', label: 'Aliases' },
  { value: 'progress', label: 'Progress' },
  { value: 'favoriteCount', label: 'Favorites' },
  { value: 'startedCount', label: 'Currently reading' },
]

const sectionTitle = computed(
  () =>
    ({
      favorites: 'Favorites',
      other: 'Other Characters',
      started: 'Started Characters',
    })[props.filter] || 'All Characters',
)
const hasFilters = computed(() => props.searchTerm || props.filter !== 'all')

function characterProgress(character) {
  return formatProgress(character?.progress ?? 0)
}
</script>

<template>
  <div class="browse-view">
    <div class="list-pane">
      <div class="browse-list-sticky">
        <BrowseListTools
          :search="search"
          search-placeholder="Search characters"
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
      <div v-if="characters.length" class="sectioned-list">
        <BrowseListSection :title="sectionTitle" :items="characters">
          <template #item="{ item: character }">
            <BrowseEntityRow
              :title="character.name"
              :subtitle="
                character.aliases?.length ? character.aliases.join(', ') : 'No aliases saved'
              "
              :image="character.image"
              main-class="character-row-main flex items-center gap-2"
              :selected="selectedCharacterId === character.id"
              :favorite="character.favorite"
              :can-favorite="!readOnly"
              :favorite-saving="quickSavingCharacterId === character.id"
              :progress="characterProgress(character)"
              progress-label="Character read progress"
              @open="$emit('open-character', character)"
              @toggle-favorite="$emit('toggle-favorite', character)"
            >
              <template #byline>
                <span v-if="character.startedAt" class="started-pill">Started</span>
                <BrowseRowStats
                  :items="[
                    `${character.appearanceCount} appearances`,
                    characterProgress(character),
                    `${character.favoriteCount} favorites`,
                    `${character.startedCount} reading`,
                  ]"
                />
              </template>
              <template v-if="!readOnly" #actions>
                <button
                  class="secondary-action collection-row-action [min-height:34px] [padding:6px_10px] [font-size:0.78rem] whitespace-nowrap"
                  type="button"
                  @click="$emit('add-to-collection', character)"
                >
                  Add to collection
                </button>
              </template>
            </BrowseEntityRow>
          </template>
        </BrowseListSection>
      </div>
      <div v-else class="empty-state">
        {{ hasFilters ? 'No characters match these filters.' : 'No characters imported yet.' }}
      </div>
    </div>
  </div>
</template>
