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
  arcs: {
    type: Array,
    default: () => [],
  },
  sections: {
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
})

defineEmits(['update:search', 'update:filter', 'update:sort', 'update:direction', 'new-arc', 'open-arc', 'toggle-favorite'])

const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'progress', label: 'Progress' },
]

const visibleArcs = computed(() => props.arcs)
const visibleSections = computed(() => {
  if (props.filter === 'favorites') return sectionList('Favorites', visibleArcs.value)
  if (props.filter === 'other') return sectionList('Other Arcs', visibleArcs.value)
  return sectionList('All Arcs', visibleArcs.value)
})
const hasFilters = computed(() => props.searchTerm || props.filter !== 'all')

function sectionList(title, arcs) {
  return arcs.length ? [{ key: props.filter, title, arcs }] : []
}
</script>

<template>
  <div class="browse-view">
    <div class="list-pane">
      <div class="browse-list-sticky">
        <div class="overview-strip">
          <span>
            <strong>{{ totalCount }}</strong>
            <small>Arcs</small>
          </span>
          <span>
            <strong>{{ favoriteCount }}</strong>
            <small>Favorites</small>
          </span>
        </div>
        <div class="comic-list-header">
          <BrowseListTools
            :search="search"
            search-placeholder="Search arcs"
            :filter="filter"
            :sort="sort"
            :direction="direction"
            :sort-options="sortOptions"
            @update:search="$emit('update:search', $event)"
            @update:filter="$emit('update:filter', $event)"
            @update:sort="$emit('update:sort', $event)"
            @update:direction="$emit('update:direction', $event)"
          />
          <button
            class="primary-button icon-text-button"
            type="button"
            aria-label="New arc"
            title="New arc"
            @click="$emit('new-arc')"
          >
            <span aria-hidden="true" class="button-icon">+</span>
          </button>
        </div>
      </div>
      <div v-if="visibleArcs.length" class="sectioned-list">
        <section v-for="section in visibleSections" :key="section.key" class="list-section">
          <div class="list-section-header">
            <p class="eyebrow">{{ section.title }}</p>
            <small>{{ section.arcs.length }}</small>
          </div>
          <div class="list">
            <div
              v-for="arc in section.arcs"
              :key="arc.id"
              class="row order-row"
              :class="{ selected: selectedArcId === arc.id }"
            >
              <span class="order-row-content">
                <button class="row-main arc-row-main" type="button" @click="$emit('open-arc', arc)">
                  <span v-if="arc.image" class="issue-list-cover" aria-hidden="true">
                    <img :src="assetURL(arc.image)" alt="" loading="lazy" />
                  </span>
                  <span>
                    <strong>{{ arc.name }}</strong>
                    <small>{{ arc.description || 'No description' }}</small>
                  </span>
                </button>
                <button
                  type="button"
                  class="favorite-toggle"
                  :class="{ active: arc.favorite }"
                  :disabled="quickSavingArcId === arc.id"
                  :aria-label="arc.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  :title="arc.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  @click="$emit('toggle-favorite', arc)"
                >
                  <span aria-hidden="true">{{ arc.favorite ? '★' : '☆' }}</span>
                </button>
              </span>
              <span class="row-progress" aria-label="Arc progress">
                <span :style="{ width: formatProgress(arc.progress) }"></span>
              </span>
            </div>
          </div>
        </section>
      </div>
      <div v-else class="empty-state">
        {{ hasFilters ? 'No arcs match these filters.' : 'No arcs yet.' }}
        <button v-if="!hasFilters" class="secondary-button" type="button" @click="$emit('new-arc')">
          <span aria-hidden="true" class="button-icon">+</span>
          Create the first arc
        </button>
      </div>
    </div>
  </div>
</template>
