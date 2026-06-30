<script setup>
import { computed, ref } from 'vue'
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
})

defineEmits(['update:search', 'new-order', 'open-order', 'toggle-favorite'])

const filter = ref('all')
const sort = ref('name')
const direction = ref('asc')
const sortOptions = [
  { value: 'name', label: 'Name' },
  { value: 'progress', label: 'Progress' },
]

const visibleOrders = computed(() => {
  return [...props.orders]
    .filter(order => {
      if (filter.value === 'favorites') return order.favorite
      if (filter.value === 'other') return !order.favorite
      return true
    })
    .sort((a, b) => {
      const result = compareOrders(a, b)
      return direction.value === 'desc' ? -result : result
    })
})
const visibleSections = computed(() => {
  if (filter.value === 'favorites') return sectionList('Favorites', visibleOrders.value)
  if (filter.value === 'other') return sectionList('Other Orders', visibleOrders.value)

  const favorites = visibleOrders.value.filter(order => order.favorite)
  if (!favorites.length) return sectionList('All Orders', visibleOrders.value)
  return [
    { key: 'favorites', title: 'Favorites', orders: favorites },
    { key: 'other', title: 'Other Orders', orders: visibleOrders.value.filter(order => !order.favorite) },
  ].filter(section => section.orders.length)
})
const hasFilters = computed(() => props.searchTerm || filter.value !== 'all')

function sectionList(title, orders) {
  return orders.length ? [{ key: filter.value, title, orders }] : []
}

function compareOrders(a, b) {
  if (sort.value === 'progress') return (a.progress ?? 0) - (b.progress ?? 0) || compareText(a.name, b.name)
  return compareText(a.name, b.name)
}

function compareText(a, b) {
  return String(a || '').localeCompare(String(b || ''), undefined, { numeric: true, sensitivity: 'base' })
}
</script>

<template>
  <div class="browse-view">
    <div class="list-pane">
      <div class="browse-list-sticky">
        <div class="overview-strip">
          <span>
            <strong>{{ totalCount }}</strong>
            <small>Orders</small>
          </span>
          <span>
            <strong>{{ favoriteCount }}</strong>
            <small>Favorites</small>
          </span>
        </div>
        <div class="comic-list-header">
          <BrowseListTools
            :search="search"
            search-placeholder="Search orders"
            :filter="filter"
            :sort="sort"
            :direction="direction"
            :sort-options="sortOptions"
            @update:search="$emit('update:search', $event)"
            @update:filter="filter = $event"
            @update:sort="sort = $event"
            @update:direction="direction = $event"
          />
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
                <button class="row-main" type="button" @click="$emit('open-order', order)">
                  <strong>{{ order.name }}</strong>
                  <small>{{ order.description || 'No description' }}</small>
                </button>
                <button
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
        <button v-if="!hasFilters" class="secondary-button" type="button" @click="$emit('new-order')">
          <span aria-hidden="true" class="button-icon">+</span>
          Create the first order
        </button>
      </div>
    </div>
  </div>
</template>
