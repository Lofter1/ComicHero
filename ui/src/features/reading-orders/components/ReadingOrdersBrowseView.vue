<script setup>
import { computed, ref } from 'vue'
import BrowseListTools from '@/shared/components/browse/BrowseListTools.vue'
import BrowseEntityRow from '@/shared/components/browse/BrowseEntityRow.vue'
import BrowseListSection from '@/shared/components/browse/BrowseListSection.vue'
import BrowseRowStats from '@/shared/components/browse/BrowseRowStats.vue'
import { ENGAGEMENT_FILTER_OPTIONS } from '@/shared/browseOptions.js'
import { formatProgress, formatRating, readingOrderCover } from '@/features/reading-orders/model.js'
import { useClickOutside } from '@/shared/composables/useClickOutside.js'

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
const orderActions = ref(null)

useClickOutside(orderActions, () => (orderActionsOpen.value = false), orderActionsOpen)

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
  const files = Array.from(event.target.files || [])
  if (files.length) emit('import-cbl', files)
  event.target.value = ''
}
</script>

<template>
  <div class="browse-view min-w-0 w-full">
    <div class="list-pane grid gap-3">
      <div
        class="browse-list-sticky max-w-none sticky [top:var(--comic-list-sticky-top)] z-18 grid gap-2.5 [margin-inline:calc(var(--sticky-toolbar-inline-offset)_*_-1)] [padding:12px_var(--sticky-toolbar-inline-offset)] border-b border-sticky-border bg-sticky-bg shadow-sticky-soft backdrop-blur-ui down-tablet:[&_.comic-list-header]:items-stretch down-tablet:[&_.comic-list-header]:flex-col down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:pb-3 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none"
      >
        <div
          class="comic-list-header flex items-center justify-between gap-3 [&_>_*]:min-w-0 [&_.eyebrow]:mb-0.5 [&_small]:text-muted desktop-compact:items-stretch desktop-compact:flex-wrap"
        >
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
              <div
                v-if="!readOnly"
                ref="orderActions"
                class="browse-header-actions order-actions relative ml-auto flex flex-none items-center flex-wrap gap-2 down-tablet:justify-start down-tablet:w-full down-mobile:justify-end [&_.icon-text-button]:w-11 [&_.icon-text-button]:min-w-11 [&_.icon-text-button]:h-11 [&_.icon-text-button]:p-0 [&_.cbl-import-button]:w-auto [&_.cbl-import-button]:[min-width:max-content] [&_.cbl-import-button]:py-0 [&_.cbl-import-button]:px-3.5 [&_.cbl-import-button]:whitespace-nowrap down-tablet:[&_.cbl-import-button]:min-w-0"
              >
                <button
                  class="secondary-button icon-text-button mobile-order-actions-trigger inline-flex w-11 min-w-11 h-11 p-0 min-h-10 border rounded text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))] items-center justify-center gap-2"
                  type="button"
                  :aria-expanded="orderActionsOpen"
                  aria-label="Actions"
                  title="Actions"
                  @click="orderActionsOpen = !orderActionsOpen"
                >
                  <span
                    aria-hidden="true"
                    class="vertical-ellipsis text-2xl font-extrabold leading-none"
                    >⋮</span
                  >
                </button>
                <input
                  ref="cblFileInput"
                  hidden
                  type="file"
                  multiple
                  accept=".cbl,application/xml,text/xml"
                  @change="handleCBLFile"
                />
                <div
                  class="order-actions-panel absolute [z-index:26] [top:calc(100%_+_8px)] right-0 hidden items-stretch gap-2 [width:max-content] [min-width:210px] border border-line-strong rounded-lg bg-surface p-2.5 [box-shadow:0_18px_40px_var(--shadow-panel)] [&.open]:grid [&_.icon-text-button]:w-full [&_.icon-text-button]:min-w-0 [&_.icon-text-button]:justify-start [&_.icon-text-button]:px-3"
                  :class="{ open: orderActionsOpen }"
                >
                  <button
                    class="secondary-button icon-text-button cbl-import-button min-h-10 border rounded text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))] inline-flex items-center justify-center gap-2"
                    :class="{ loading: cblImporting }"
                    type="button"
                    :disabled="cblImporting"
                    :aria-busy="cblImporting"
                    aria-label="Import CBL"
                    title="Import CBL"
                    @click="chooseCBLFile"
                  >
                    <span
                      v-if="cblImporting"
                      class="button-spinner w-3.5 min-w-3.5 h-3.5 [border:2px_solid_var(--spinner-track)] [border-top-color:currentColor] rounded-full [animation:loading-spin_780ms_linear_infinite]"
                      aria-hidden="true"
                    ></span>
                    {{ cblImporting ? 'Importing CBL...' : 'Import CBL' }}
                  </button>
                  <span v-if="cblImporting" class="sr-only" aria-live="polite">Importing CBL</span>
                  <button
                    class="primary-button icon-text-button new-order-action min-h-10 border rounded py-2.5 px-3.5 border-primary bg-primary text-white inline-flex items-center justify-center gap-2"
                    type="button"
                    aria-label="New reading order"
                    title="New reading order"
                    @click="createReadingOrder"
                  >
                    <span
                      aria-hidden="true"
                      class="button-icon inline-flex items-center justify-center size-5 text-xl font-extrabold leading-none"
                      >+</span
                    >
                    <span class="order-action-label [display:inline]">New reading order</span>
                  </button>
                </div>
              </div>
            </template>
          </BrowseListTools>
        </div>
      </div>
      <div v-if="orders.length" class="grid gap-4">
        <BrowseListSection :title="sectionTitle" :items="orders">
          <template #item="{ item: order }">
            <BrowseEntityRow
              :title="order.name"
              :subtitle="order.description || 'No description'"
              :image="readingOrderCover(order)"
              main-class="[&_>_span:last-child]:min-w-0 flex items-center gap-2.5"
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
                <span
                  v-if="order.authorName"
                  class="author-pill inline-flex items-center w-fit max-w-full mt-2 border border-line-strong rounded-full bg-surface-muted text-label py-1 px-2 text-xs font-extrabold leading-tight"
                >
                  Author: {{ order.authorName }}
                </span>
                <span
                  v-if="order.rating"
                  class="author-pill inline-flex items-center w-fit max-w-full mt-2 border border-line-strong rounded-full bg-surface-muted text-label py-1 px-2 text-xs font-extrabold leading-tight"
                >
                  Rating: {{ formatRating(order.rating) }}
                  <template v-if="order.ratingCount">({{ order.ratingCount }})</template>
                </span>
                <span
                  v-if="order.startedAt"
                  class="started-pill inline-flex items-center w-fit mt-2 border border-primary rounded-full bg-primary-soft text-primary-strong py-1 px-2 text-xs font-extrabold leading-tight"
                  >Started</span
                >
                <BrowseRowStats
                  :items="[`${order.favoriteCount} favorites`, `${order.startedCount} reading`]"
                />
              </template>
            </BrowseEntityRow>
          </template>
        </BrowseListSection>
      </div>
      <div
        v-else
        class="empty-state grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4"
      >
        {{ hasFilters ? 'No reading orders match these filters.' : 'No reading orders yet.' }}
        <button
          v-if="!hasFilters && !readOnly"
          class="secondary-button min-h-10 border rounded text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]"
          type="button"
          @click="$emit('new-order')"
        >
          <span
            aria-hidden="true"
            class="button-icon inline-flex items-center justify-center size-5 text-xl font-extrabold leading-none"
            >+</span
          >
          Create the first order
        </button>
      </div>
    </div>
  </div>
</template>
