<script setup>
import { computed } from 'vue'
import { assetURL } from '@/api/client.js'
import BrowseListTools from '@/components/BrowseListTools.vue'
import { formatProgress } from '@/domain/readingOrders.js'

const props = defineProps({
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

defineEmits(['update:search', 'update:filter', 'update:sort', 'update:direction', 'open-character', 'toggle-favorite'])

const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'appearances', label: 'Appearances' },
  { value: 'aliases', label: 'Aliases' },
  { value: 'progress', label: 'Progress' },
]

const visibleCharacters = computed(() => props.characters)
const visibleSections = computed(() => {
  if (props.filter === 'favorites') return sectionList('Favorites', visibleCharacters.value)
  if (props.filter === 'other') return sectionList('Other Characters', visibleCharacters.value)
  return sectionList('All Characters', visibleCharacters.value)
})
const hasFilters = computed(() => props.searchTerm || props.filter !== 'all')

function sectionList(title, characters) {
  return characters.length ? [{ key: props.filter, title, characters }] : []
}

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
          @update:search="$emit('update:search', $event)"
          @update:filter="$emit('update:filter', $event)"
          @update:sort="$emit('update:sort', $event)"
          @update:direction="$emit('update:direction', $event)"
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
