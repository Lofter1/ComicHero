<script setup>
import { computed } from 'vue'
import BaseSelect from '@/shared/components/form/BaseSelect.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'

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
    class="comic-list-tools browse-list-tools flex w-full flex-auto flex-wrap items-center gap-2 down-mobile:w-full"
  >
    <BaseTextInput
      v-model="searchModel"
      size="toolbar"
      type="search"
      :placeholder="searchPlaceholder"
    />
    <div
      class="inline-filter-tabs inline-grid grid-cols-3 gap-1 rounded border border-line bg-panel-soft p-1 down-mobile:w-full"
      :class="{ 'grid-cols-4': filterOptions.length === 4 }"
      role="tablist"
      aria-label="List filter"
    >
      <!-- Native buttons: list filters are a stateful tablist. -->
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
    <BaseSelect
      class="list-sort-select"
      :model-value="sort"
      size="toolbar"
      variant="trailing"
      aria-label="Sort list"
      @update:model-value="$emit('update:sort', $event)"
    >
      <option v-for="option in sortOptions" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </BaseSelect>
    <BaseSelect
      class="list-direction-select"
      :model-value="direction"
      size="toolbar"
      variant="trailing"
      aria-label="Sort direction"
      @update:model-value="$emit('update:direction', $event)"
    >
      <option value="asc">Ascending</option>
      <option value="desc">Descending</option>
    </BaseSelect>
    <details class="mobile-list-options hidden">
      <summary>Filter &amp; sort</summary>
      <div>
        <div
          class="inline-filter-tabs inline-grid grid-cols-3 gap-1 rounded border border-line bg-panel-soft p-1 down-mobile:w-full"
          :class="{ 'grid-cols-4': filterOptions.length === 4 }"
          role="tablist"
          aria-label="List filter"
        >
          <!-- Native buttons: mobile list filters mirror the stateful tablist above. -->
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
        <BaseSelect
          :model-value="sort"
          aria-label="Sort list"
          @update:model-value="$emit('update:sort', $event)"
        >
          <option v-for="option in sortOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </BaseSelect>
        <BaseSelect
          :model-value="direction"
          aria-label="Sort direction"
          @update:model-value="$emit('update:direction', $event)"
        >
          <option value="asc">Ascending</option>
          <option value="desc">Descending</option>
        </BaseSelect>
      </div>
    </details>
    <slot name="actions"></slot>
  </div>
</template>

<style scoped>
.browse-list-tools > * {
  @apply min-w-0;
}

.browse-list-tools > input {
  flex: 1 1 280px;
  min-width: min(280px, 100%);
}

.browse-list-tools > .inline-filter-tabs {
  flex: 1 1 360px;
}

.browse-list-tools select {
  @apply w-auto min-w-32 max-w-44 flex-none justify-self-start;
  flex: 1 1 140px;
}

.browse-list-tools .list-sort-select {
  @apply min-w-44;
}

.browse-list-tools .list-direction-select {
  @apply min-w-36 max-w-44;
}

.browse-list-tools :slotted(.browse-header-actions) {
  @apply ml-auto w-auto flex-none;
}

.inline-filter-tabs button {
  @apply min-h-8 rounded-ui-sm border-0 bg-transparent px-2 py-1.5 text-sm font-bold text-label;
}

.inline-filter-tabs button.active {
  @apply bg-primary text-white;
}

@screen down-mobile {
  .browse-list-tools > :is(.inline-filter-tabs, .list-sort-select, .list-direction-select) {
    @apply hidden;
  }

  .mobile-list-options {
    @apply relative block min-w-36 flex-auto;
  }

  .mobile-list-options summary {
    @apply min-h-11 cursor-pointer rounded border border-line-strong bg-surface pt-3 pr-9 pb-3 pl-3 font-bold text-control;
    list-style: none;
  }

  .mobile-list-options summary::-webkit-details-marker {
    @apply hidden;
  }

  .mobile-list-options summary::after {
    @apply absolute right-3 top-2.5 text-muted;
    content: '⌄';
  }

  .mobile-list-options[open] summary::after {
    content: '⌃';
  }

  .mobile-list-options > div {
    @apply absolute right-0 z-25 grid gap-2.5 rounded-lg border border-line-strong bg-surface p-3 shadow-monitor;
    top: calc(100% + 8px);
    left: auto;
    width: min(320px, calc(100vw - 28px));
  }

  .mobile-list-options .inline-filter-tabs {
    @apply grid w-full min-w-0;
  }

  .mobile-list-options select {
    @apply w-full max-w-none;
  }
}

@screen down-phone {
  .inline-filter-tabs button {
    @apply px-1.5 text-xs;
  }
}
</style>
