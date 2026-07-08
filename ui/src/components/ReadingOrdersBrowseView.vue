<script setup>
import { computed, ref } from 'vue'
import { assetURL } from '@/api/client.js'
import BrowseListTools from '@/components/BrowseListTools.vue'
import { formatProgress } from '@/domain/readingOrders.js'

const props = defineProps({
  orders: {
    type: Array,
    default: () => [],
  },
  sections: {
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

const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'progress', label: 'Progress' },
]

const visibleOrders = computed(() => props.orders)
const visibleSections = computed(() => {
  if (props.filter === 'favorites') return sectionList('Favorites', visibleOrders.value)
  if (props.filter === 'other') return sectionList('Other Orders', visibleOrders.value)
  return sectionList('All Orders', visibleOrders.value)
})
const hasFilters = computed(() => props.searchTerm || props.filter !== 'all')

function sectionList(title, orders) {
  return orders.length ? [{ key: props.filter, title, orders }] : []
}

function chooseCBLFile() {
  cblFileInput.value?.click()
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
            @update:search="$emit('update:search', $event)"
            @update:filter="$emit('update:filter', $event)"
            @update:sort="$emit('update:sort', $event)"
            @update:direction="$emit('update:direction', $event)"
          />
          <div v-if="!readOnly" class="browse-header-actions">
            <input
              ref="cblFileInput"
              hidden
              type="file"
              accept=".cbl,application/xml,text/xml"
              @change="handleCBLFile"
            />
            <button
              class="secondary-button icon-text-button cbl-import-button"
              type="button"
              aria-label="Import CBL"
              title="Import CBL"
              @click="chooseCBLFile"
            >
              Import CBL
            </button>
            <button
              class="primary-button icon-text-button"
              type="button"
              aria-label="New order"
              title="New order"
              @click="$emit('new-order')"
            >
              <span aria-hidden="true" class="button-icon">+</span>
            </button>
          </div>
        </div>
      </div>
      <div v-if="visibleOrders.length" class="sectioned-list">
        <section v-for="section in visibleSections" :key="section.key" class="list-section">
          <div class="list-section-header">
            <p class="eyebrow">{{ section.title }}</p>
            <small>{{ section.orders.length }}</small>
          </div>
          <div class="list">
            <div
              v-for="order in section.orders"
              :key="order.id"
              class="row order-row"
              :class="{ selected: selectedOrderId === order.id }"
            >
              <span class="order-row-content">
                <button
                  class="row-main arc-row-main"
                  type="button"
                  @click="$emit('open-order', order)"
                >
                  <span v-if="order.image" class="issue-list-cover" aria-hidden="true">
                    <img :src="assetURL(order.image)" alt="" loading="lazy" />
                  </span>
                  <span>
                    <strong>{{ order.name }}</strong>
                    <small>{{ order.description || 'No description' }}</small>
                    <span v-if="order.authorName" class="author-pill"
                      >Author: {{ order.authorName }}</span
                    >
                  </span>
                </button>
                <button
                  v-if="order.canEdit"
                  type="button"
                  class="favorite-toggle"
                  :class="{ active: order.favorite }"
                  :disabled="quickSavingOrderId === order.id"
                  :aria-label="order.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  :title="order.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  @click="$emit('toggle-favorite', order)"
                >
                  <span aria-hidden="true">{{ order.favorite ? '★' : '☆' }}</span>
                </button>
              </span>
              <span class="row-progress" aria-label="Reading order progress">
                <span :style="{ width: formatProgress(order.progress) }"></span>
              </span>
            </div>
          </div>
        </section>
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
