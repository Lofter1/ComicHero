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
    class="comic-list-tools browse-list-tools flex flex-auto flex-wrap items-center gap-2 w-full [&_>_*]:min-w-0 [&_>_input]:[flex:1_1_280px] [&_>_input]:[min-width:min(280px,_100%)] [&_>_.inline-filter-tabs]:[flex:1_1_360px] [&_select]:flex-none [&_select]:w-auto [&_select]:max-w-56 [&_select]:justify-self-start [&_>_.browse-header-actions]:flex-none [&_>_.browse-header-actions]:w-auto [&_>_.browse-header-actions]:ml-auto [&_.list-direction-select]:min-w-36 [&_.list-direction-select]:max-w-44 [&_input]:h-11 [&_input]:min-h-11 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:py-2 [&_input]:px-2.5 [&_select]:h-11 [&_select]:min-h-11 [&_select]:border [&_select]:border-line-strong [&_select]:rounded [&_select]:bg-surface [&_select]:py-2 [&_select]:px-2.5 [&_select]:[flex:1_1_140px] [&_select]:min-w-32 [&_select]:max-w-44 [&_select]:pr-8 [&_.list-sort-select]:min-w-44 [&_.inline-filter-tabs]:[flex:1_1_230px] [&_.inline-filter-tabs]:[min-width:min(230px,_100%)] [&_.issue-status-tabs]:[flex-basis:320px] [&_.issue-status-tabs]:[min-width:min(320px,_100%)] [&_.four-filter-tabs]:min-w-96 down-mobile:[&_>_.inline-filter-tabs]:hidden down-mobile:[&_>_.list-sort-select]:hidden down-mobile:[&_>_.list-direction-select]:hidden down-mobile:[&:has(>_.comic-filter-controls)]:relative down-mobile:[&:has(>_.comic-filter-controls)]:flex down-mobile:[&:has(>_.comic-filter-controls)]:flex-wrap down-mobile:[&:has(>_.comic-filter-controls)]:items-center down-mobile:[&:has(>_.comic-filter-controls)]:gap-2 down-mobile:[&:has(>_.comic-filter-controls)_>_input]:[flex:1_1_280px] down-mobile:[&_.issue-status-tabs]:min-w-0 down-mobile:w-full"
  >
    <input v-model="searchModel" type="search" :placeholder="searchPlaceholder" />
    <div
      class="inline-filter-tabs inline-grid grid-cols-3 gap-1 border border-line rounded bg-panel-soft p-1 [&_button]:min-h-8 [&_button]:border-0 [&_button]:rounded-[6px] [&_button]:bg-transparent [&_button]:text-label [&_button]:py-1.5 [&_button]:px-2 [&_button]:text-sm [&_button]:font-bold [&_button.active]:bg-primary [&_button.active]:text-white down-mobile:w-full down-phone:[&_button]:px-1.5 down-phone:[&_button]:text-xs"
      :class="{ 'grid-cols-4': filterOptions.length === 4 }"
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
      class="mobile-list-options down-mobile:relative down-mobile:block down-mobile:flex-auto down-mobile:min-w-36 hidden down-mobile:[&_summary]:min-h-11 down-mobile:[&_summary]:border down-mobile:[&_summary]:border-line-strong down-mobile:[&_summary]:rounded down-mobile:[&_summary]:bg-surface down-mobile:[&_summary]:text-control down-mobile:[&_summary]:pt-3 down-mobile:[&_summary]:pr-9 down-mobile:[&_summary]:pb-3 down-mobile:[&_summary]:pl-3 down-mobile:[&_summary]:font-bold down-mobile:[&_summary]:[list-style:none] down-mobile:[&_summary]:cursor-pointer down-mobile:[&_summary::-webkit-details-marker]:hidden down-mobile:[&_summary::after]:[content:'⌄'] down-mobile:[&_summary::after]:absolute down-mobile:[&_summary::after]:top-2.5 down-mobile:[&_summary::after]:right-3 down-mobile:[&_summary::after]:text-muted down-mobile:[&[open]_summary::after]:[content:'⌃'] down-mobile:[&_>_div]:absolute down-mobile:[&_>_div]:z-25 down-mobile:[&_>_div]:[top:calc(100%_+_8px)] down-mobile:[&_>_div]:right-0 down-mobile:[&_>_div]:[left:auto] down-mobile:[&_>_div]:gap-2.5 down-mobile:[&_>_div]:border down-mobile:[&_>_div]:border-line-strong down-mobile:[&_>_div]:rounded-lg down-mobile:[&_>_div]:bg-surface down-mobile:[&_>_div]:p-3 down-mobile:[&_>_div]:[box-shadow:0_18px_40px_var(--shadow-panel)] down-mobile:[&_>_div]:grid down-mobile:[&_>_div]:[width:min(320px,_calc(100vw_-_28px))] down-mobile:[&_.inline-filter-tabs]:grid down-mobile:[&_.inline-filter-tabs]:w-full down-mobile:[&_.inline-filter-tabs]:min-w-0 down-mobile:[&_select]:w-full down-mobile:[&_select]:max-w-none"
    >
      <summary>Filter &amp; sort</summary>
      <div>
        <div
          class="inline-filter-tabs inline-grid grid-cols-3 gap-1 border border-line rounded bg-panel-soft p-1 [&_button]:min-h-8 [&_button]:border-0 [&_button]:rounded-[6px] [&_button]:bg-transparent [&_button]:text-label [&_button]:py-1.5 [&_button]:px-2 [&_button]:text-sm [&_button]:font-bold [&_button.active]:bg-primary [&_button.active]:text-white down-mobile:w-full down-phone:[&_button]:px-1.5 down-phone:[&_button]:text-xs"
          :class="{ 'grid-cols-4': filterOptions.length === 4 }"
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
