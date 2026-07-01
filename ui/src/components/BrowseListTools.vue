<script setup>
import { computed } from 'vue'

const props = defineProps({
  search: {
    type: String,
    default: '',
  },
  searchPlaceholder: {
    type: String,
    default: 'Search',
  },
  filter: {
    type: String,
    required: true,
  },
  sort: {
    type: String,
    required: true,
  },
  direction: {
    type: String,
    required: true,
  },
  sortOptions: {
    type: Array,
    required: true,
  },
})

const emit = defineEmits(['update:search', 'update:filter', 'update:sort', 'update:direction'])

const searchModel = computed({
  get: () => props.search,
  set: value => emit('update:search', value),
})
</script>

<template>
  <div class="comic-list-tools browse-list-tools">
    <input v-model="searchModel" type="search" :placeholder="searchPlaceholder" />
    <div class="inline-filter-tabs" role="tablist" aria-label="Favorite filter">
      <button type="button" :class="{ active: filter === 'all' }" role="tab" :aria-selected="filter === 'all'" @click="$emit('update:filter', 'all')">
        All
      </button>
      <button type="button" :class="{ active: filter === 'favorites' }" role="tab" :aria-selected="filter === 'favorites'" @click="$emit('update:filter', 'favorites')">
        Favorites
      </button>
      <button type="button" :class="{ active: filter === 'other' }" role="tab" :aria-selected="filter === 'other'" @click="$emit('update:filter', 'other')">
        Other
      </button>
    </div>
    <select :value="sort" aria-label="Sort list" @change="$emit('update:sort', $event.target.value)">
      <option v-for="option in sortOptions" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
    <select :value="direction" aria-label="Sort direction" @change="$emit('update:direction', $event.target.value)">
      <option value="asc">Ascending</option>
      <option value="desc">Descending</option>
    </select>
  </div>
</template>
