<script setup>
import { computed, ref } from 'vue'
import BrowseListTools from '@/shared/components/browse/BrowseListTools.vue'
import BrowseEntityRow from '@/shared/components/browse/BrowseEntityRow.vue'
import BrowseListSection from '@/shared/components/browse/BrowseListSection.vue'
import BrowseRowStats from '@/shared/components/browse/BrowseRowStats.vue'
import { ENGAGEMENT_FILTER_OPTIONS } from '@/shared/browseOptions.js'
import { formatProgress, formatRating } from '@/features/reading-orders/model.js'

const props = defineProps({
  orders: {
    type: Array,
    default: () => [],
  },
  selectedOrderId: {
    type: Number,
    default: null,
  },
  quickSavingOrderId: {
    type: Number,
    default: null,
  },
  cblImporting: {
    type: Boolean,
    default: false,
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

const emit = defineEmits([
  'update:search',
  'update:filter',
  'update:sort',
  'update:direction',
  'new-order',
  'open-order',
  'toggle-favorite',
  'import-cbl',
])

const cblFileInput = ref(null)
const orderActionsOpen = ref(false)

const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'rating', label: 'Rating' },
  { value: 'progress', label: 'Progress' },
  { value: 'favoriteCount', label: 'Favorites' },
  { value: 'startedCount', label: 'Currently reading' },
]

const sectionTitle = computed(
  () =>
    ({ favorites: 'Favorites', other: 'Other Orders', started: 'Started Orders' })[props.filter] ||
    'All Orders',
)
const hasFilters = computed(() => props.searchTerm || props.filter !== 'all')

function chooseCBLFile() {
  if (props.cblImporting) return
  orderActionsOpen.value = false
  cblFileInput.value?.click()
}

function createReadingOrder() {
  orderActionsOpen.value = false
  emit('new-order')
}

function handleCBLFile(event) {
  const file = event.target.files?.[0]
  if (file) emit('import-cbl', file)
  event.target.value = ''
}
</script>

<template>
  <div class="browse-view">
    <div class="list-pane">
      <div class="browse-list-sticky">
        <div class="comic-list-header">
          <BrowseListTools
            :search="search"
            search-placeholder="Search orders"
            :filter="filter"
            :sort="sort"
            :direction="direction"
            :sort-options="sortOptions"
            :filter-options="ENGAGEMENT_FILTER_OPTIONS"
            @update:search="$emit('update:search', $event)"
            @update:filter="$emit('update:filter', $event)"
            @update:sort="$emit('update:sort', $event)"
            @update:direction="$emit('update:direction', $event)"
          >
            <template #actions>
              <div v-if="!readOnly" class="browse-header-actions order-actions">
                <button
                  class="secondary-button icon-text-button mobile-order-actions-trigger"
                  type="button"
                  :aria-expanded="orderActionsOpen"
                  aria-label="Actions"
                  title="Actions"
                  @click="orderActionsOpen = !orderActionsOpen"
                >
                  <span aria-hidden="true" class="vertical-ellipsis">⋮</span>
                </button>
                <input
                  ref="cblFileInput"
                  hidden
                  type="file"
                  accept=".cbl,application/xml,text/xml"
                  @change="handleCBLFile"
                />
                <div class="order-actions-panel" :class="{ open: orderActionsOpen }">
                  <button
                    class="secondary-button icon-text-button cbl-import-button"
                    :class="{ loading: cblImporting }"
                    type="button"
                    :disabled="cblImporting"
                    :aria-busy="cblImporting"
                    aria-label="Import CBL"
                    title="Import CBL"
                    @click="chooseCBLFile"
                  >
                    <span v-if="cblImporting" class="button-spinner" aria-hidden="true"></span>
                    {{ cblImporting ? 'Importing CBL...' : 'Import CBL' }}
                  </button>
                  <span v-if="cblImporting" class="sr-only" aria-live="polite">Importing CBL</span>
                  <button
                    class="primary-button icon-text-button new-order-action"
                    type="button"
                    aria-label="New reading order"
                    title="New reading order"
                    @click="createReadingOrder"
                  >
                    <span aria-hidden="true" class="button-icon">+</span>
                    <span class="order-action-label">New reading order</span>
                  </button>
                </div>
              </div>
            </template>
          </BrowseListTools>
        </div>
      </div>
      <div v-if="orders.length" class="sectioned-list">
        <BrowseListSection :title="sectionTitle" :items="orders">
          <template #item="{ item: order }">
            <BrowseEntityRow
              :title="order.name"
              :subtitle="order.description || 'No description'"
              :image="order.image"
              main-class="arc-row-main"
              :selected="selectedOrderId === order.id"
              :favorite="order.favorite"
              :can-favorite="order.canEdit"
              :favorite-saving="quickSavingOrderId === order.id"
              :progress="formatProgress(order.progress)"
              progress-label="Reading order progress"
              @open="$emit('open-order', order)"
              @toggle-favorite="$emit('toggle-favorite', order)"
            >
              <template #byline>
                <span v-if="order.authorName" class="author-pill">
                  Author: {{ order.authorName }}
                </span>
                <span v-if="order.rating" class="author-pill">
                  Rating: {{ formatRating(order.rating) }}
                  <template v-if="order.ratingCount">({{ order.ratingCount }})</template>
                </span>
                <span v-if="order.startedAt" class="started-pill">Started</span>
                <BrowseRowStats
                  :items="[`${order.favoriteCount} favorites`, `${order.startedCount} reading`]"
                />
              </template>
            </BrowseEntityRow>
          </template>
        </BrowseListSection>
      </div>
      <div v-else class="empty-state">
        {{ hasFilters ? 'No reading orders match these filters.' : 'No reading orders yet.' }}
        <button
          v-if="!hasFilters && !readOnly"
          class="secondary-button"
          type="button"
          @click="$emit('new-order')"
        >
          <span aria-hidden="true" class="button-icon">+</span>
          Create the first order
        </button>
      </div>
    </div>
  </div>
</template>
