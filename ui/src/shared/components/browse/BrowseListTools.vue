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
  filterOptions: {
    type: Array,
    default: () => [
      { value: 'all', label: 'All' },
      { value: 'favorites', label: 'Favorites' },
      { value: 'other', label: 'Other' },
    ],
  },
})

const emit = defineEmits(['update:search', 'update:filter', 'update:sort', 'update:direction'])

const searchModel = computed({
  get: () => props.search,
  set: (value) => emit('update:search', value),
})
</script>

<template>
  <div
    class="comic-list-tools browse-list-tools flex [flex:1_1_auto] flex-wrap items-center gap-2 w-full"
  >
    <input v-model="searchModel" type="search" :placeholder="searchPlaceholder" />
    <div
      class="inline-filter-tabs"
      :class="{ 'four-filter-tabs': filterOptions.length === 4 }"
      role="tablist"
      aria-label="List filter"
    >
      <button
        v-for="option in filterOptions"
        :key="option.value"
        type="button"
        :class="{ active: filter === option.value }"
        role="tab"
        :aria-selected="filter === option.value"
        @click="$emit('update:filter', option.value)"
      >
        {{ option.label }}
      </button>
    </div>
    <select
      class="list-sort-select"
      :value="sort"
      aria-label="Sort list"
      @change="$emit('update:sort', $event.target.value)"
    >
      <option v-for="option in sortOptions" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
    <select
      class="list-direction-select"
      :value="direction"
      aria-label="Sort direction"
      @change="$emit('update:direction', $event.target.value)"
    >
      <option value="asc">Ascending</option>
      <option value="desc">Descending</option>
    </select>
    <details
      class="mobile-list-options down-mobile:relative down-mobile:block down-mobile:[flex:1_1_auto] down-mobile:[min-width:150px]"
    >
      <summary>Filter &amp; sort</summary>
      <div>
        <div
          class="inline-filter-tabs"
          :class="{ 'four-filter-tabs': filterOptions.length === 4 }"
          role="tablist"
          aria-label="List filter"
        >
          <button
            v-for="option in filterOptions"
            :key="option.value"
            type="button"
            :class="{ active: filter === option.value }"
            role="tab"
            :aria-selected="filter === option.value"
            @click="$emit('update:filter', option.value)"
          >
            {{ option.label }}
          </button>
        </div>
        <select
          :value="sort"
          aria-label="Sort list"
          @change="$emit('update:sort', $event.target.value)"
        >
          <option v-for="option in sortOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
        <select
          :value="direction"
          aria-label="Sort direction"
          @change="$emit('update:direction', $event.target.value)"
        >
          <option value="asc">Ascending</option>
          <option value="desc">Descending</option>
        </select>
      </div>
    </details>
    <slot name="actions"></slot>
  </div>
</template>
