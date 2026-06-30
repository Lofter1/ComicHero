<script setup>
import { computed, ref } from 'vue'
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
  appearanceCount: {
    type: Number,
    default: 0,
  },
  characters: {
    type: Array,
    default: () => [],
  },
  sections: {
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
})

defineEmits(['update:search', 'open-character', 'toggle-favorite'])

const filter = ref('all')
const sort = ref('name')
const direction = ref('asc')
const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'appearances', label: 'Appearances' },
  { value: 'aliases', label: 'Aliases' },
  { value: 'progress', label: 'Progress' },
]

const visibleCharacters = computed(() => {
  return [...props.characters]
    .filter(character => {
      if (filter.value === 'favorites') return character.favorite
      if (filter.value === 'other') return !character.favorite
      return true
    })
    .sort((a, b) => {
      const result = compareCharacters(a, b)
      return direction.value === 'desc' ? -result : result
    })
})
const visibleSections = computed(() => {
  if (filter.value === 'favorites') return sectionList('Favorites', visibleCharacters.value)
  if (filter.value === 'other') return sectionList('Other Characters', visibleCharacters.value)

  const favorites = visibleCharacters.value.filter(character => character.favorite)
  if (!favorites.length) return sectionList('All Characters', visibleCharacters.value)
  return [
    { key: 'favorites', title: 'Favorites', characters: favorites },
    { key: 'other', title: 'Other Characters', characters: visibleCharacters.value.filter(character => !character.favorite) },
  ].filter(section => section.characters.length)
})
const hasFilters = computed(() => props.searchTerm || filter.value !== 'all')

function sectionList(title, characters) {
  return characters.length ? [{ key: filter.value, title, characters }] : []
}

function compareCharacters(a, b) {
  if (sort.value === 'appearances') return (a.appearanceCount ?? 0) - (b.appearanceCount ?? 0) || compareText(a.name, b.name)
  if (sort.value === 'aliases') return (a.aliases?.length ?? 0) - (b.aliases?.length ?? 0) || compareText(a.name, b.name)
  if (sort.value === 'progress') return (a.progress ?? 0) - (b.progress ?? 0) || compareText(a.name, b.name)
  return compareText(a.name, b.name)
}

function compareText(a, b) {
  return String(a || '').localeCompare(String(b || ''), undefined, { numeric: true, sensitivity: 'base' })
}

function characterProgress(character) {
  return formatProgress(character?.progress ?? 0)
}
</script>

<template>
  <div class="browse-view">
    <div class="list-pane">
      <div class="browse-list-sticky">
        <div class="overview-strip">
          <span>
            <strong>{{ totalCount }}</strong>
            <small>Characters</small>
          </span>
          <span>
            <strong>{{ favoriteCount }}</strong>
            <small>Favorites</small>
          </span>
          <span>
            <strong>{{ appearanceCount }}</strong>
            <small>Appearances</small>
          </span>
        </div>
        <BrowseListTools
          :search="search"
          search-placeholder="Search characters"
          :filter="filter"
          :sort="sort"
          :direction="direction"
          :sort-options="sortOptions"
          @update:search="$emit('update:search', $event)"
          @update:filter="filter = $event"
          @update:sort="sort = $event"
          @update:direction="direction = $event"
        />
      </div>
      <div v-if="visibleCharacters.length" class="sectioned-list">
        <section v-for="section in visibleSections" :key="section.key" class="list-section">
          <div class="list-section-header">
            <p class="eyebrow">{{ section.title }}</p>
            <small>{{ section.characters.length }}</small>
          </div>
          <div class="list">
            <div
              v-for="character in section.characters"
              :key="character.id"
              class="row character-row"
              :class="{ selected: selectedCharacterId === character.id }"
            >
              <span class="order-row-content">
                <button class="row-main character-row-main" type="button" @click="$emit('open-character', character)">
                  <span v-if="character.image" class="character-list-avatar" aria-hidden="true">
                    <img :src="assetURL(character.image)" alt="" loading="lazy" />
                  </span>
                  <span>
                    <strong>{{ character.name }}</strong>
                    <small v-if="character.aliases?.length">{{ character.aliases.join(', ') }}</small>
                    <small v-else>No aliases saved</small>
                  </span>
                </button>
                <button
                  type="button"
                  class="favorite-toggle"
                  :class="{ active: character.favorite }"
                  :disabled="quickSavingCharacterId === character.id"
                  :aria-label="character.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  :title="character.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  @click="$emit('toggle-favorite', character)"
                >
                  <span aria-hidden="true">{{ character.favorite ? '★' : '☆' }}</span>
                </button>
              </span>
              <span class="row-meta">
                <span class="status-pill">{{ character.appearanceCount }} appearances</span>
                <span class="status-pill">{{ characterProgress(character) }}</span>
              </span>
              <span class="row-progress" aria-label="Character read progress">
                <span :style="{ width: characterProgress(character) }"></span>
              </span>
            </div>
          </div>
        </section>
      </div>
      <div v-else class="empty-state">
        {{ hasFilters ? 'No characters match these filters.' : 'No characters imported yet.' }}
      </div>
    </div>
  </div>
</template>
