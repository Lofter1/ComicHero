<script setup>
import { computed } from 'vue'
import { assetURL } from '@/api/client.js'
import BrowseListTools from '@/components/BrowseListTools.vue'
import { formatProgress } from '@/domain/readingOrders.js'

const props = defineProps({
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
]
const filterOptions = [
  { value: 'all', label: 'All' },
  { value: 'favorites', label: 'Favorites' },
  { value: 'started', label: 'Started' },
  { value: 'other', label: 'Other' },
]

const visibleArcs = computed(() => props.arcs)
const visibleSections = computed(() => {
  if (props.filter === 'favorites') return sectionList('Favorites', visibleArcs.value)
  if (props.filter === 'other') return sectionList('Other Arcs', visibleArcs.value)
  if (props.filter === 'started') return sectionList('Started Arcs', visibleArcs.value)
  return sectionList('All Arcs', visibleArcs.value)
})

function sectionList(title, arcs) {
  return arcs.length ? [{ key: props.filter, title, arcs }] : []
}
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
            :filter-options="filterOptions"
            @update:search="$emit('update:search', $event)"
            @update:filter="$emit('update:filter', $event)"
            @update:sort="$emit('update:sort', $event)"
            @update:direction="$emit('update:direction', $event)"
          />
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
                    <span v-if="arc.startedAt" class="started-pill">Started</span>
                  </span>
                </button>
                <button
                  v-if="!readOnly"
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
    </div>
  </div>
</template>
