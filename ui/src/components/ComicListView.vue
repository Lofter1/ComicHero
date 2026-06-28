<script setup>
import { computed, ref } from 'vue'
import IssueListItem from '@/components/IssueListItem.vue'

const props = defineProps({
  comics: {
    type: Array,
    default: () => [],
  },
  title: {
    type: String,
    default: 'Comics',
  },
  emptyMessage: {
    type: String,
    default: 'No comics yet.',
  },
  filteredEmptyMessage: {
    type: String,
    default: 'No comics match these filters.',
  },
  selectedComicId: {
    type: Number,
    default: null,
  },
  quickSavingComicId: {
    type: Number,
    default: null,
  },
  showNewButton: {
    type: Boolean,
    default: false,
  },
  showCover: {
    type: Boolean,
    default: false,
  },
  showComment: {
    type: Boolean,
    default: false,
  },
  initialSort: {
    type: String,
    default: 'series',
  },
})

defineEmits(['open-comic', 'toggle-read', 'new-comic'])

const localSearch = ref('')
const status = ref('all')
const sort = ref(props.initialSort)
const direction = ref('asc')

const searchTerm = computed(() => localSearch.value.trim().toLowerCase())
const visibleComics = computed(() => {
  return [...props.comics]
    .filter(comic => {
      if (status.value === 'read' && !comic.read) return false
      if (status.value === 'unread' && comic.read) return false
      if (!searchTerm.value) return true

      return [comic.title, comic.series, comic.issue, comic.publisher, comic.coverDate, comic.comment]
        .filter(value => value !== undefined && value !== null && value !== '')
        .some(value => String(value).toLowerCase().includes(searchTerm.value))
    })
    .sort((a, b) => {
      const result = compareComics(a, b, sort.value)
      return direction.value === 'desc' ? -result : result
    })
})

const readCount = computed(() => props.comics.filter(comic => comic.read).length)
const hasFilters = computed(() => searchTerm.value || status.value !== 'all')

function compareComics(a, b, mode) {
  if (mode === 'date') return compareText(a.coverDate, b.coverDate) || compareSeries(a, b)
  if (mode === 'publisher') return compareText(a.publisher, b.publisher) || compareSeries(a, b)
  if (mode === 'read') return Number(a.read) - Number(b.read) || compareSeries(a, b)
  if (mode === 'title') return compareText(a.title, b.title) || compareSeries(a, b)
  return compareSeries(a, b)
}

function compareSeries(a, b) {
  return compareText(a.series, b.series)
    || (a.seriesYear ?? 0) - (b.seriesYear ?? 0)
    || (a.issue ?? 0) - (b.issue ?? 0)
    || compareText(a.title, b.title)
}

function compareText(a, b) {
  return String(a || '').localeCompare(String(b || ''), undefined, { numeric: true, sensitivity: 'base' })
}
</script>

<template>
  <section class="comic-list-view">
    <header class="comic-list-header">
      <div>
        <p class="eyebrow">{{ title }}</p>
        <small>{{ visibleComics.length }} of {{ comics.length }} · {{ readCount }} read</small>
      </div>
      <button
        v-if="showNewButton"
        class="primary-button icon-text-button"
        type="button"
        aria-label="New comic"
        title="New comic"
        @click="$emit('new-comic')"
      >
        <span aria-hidden="true" class="button-icon">+</span>
      </button>
    </header>

    <div v-if="comics.length" class="comic-list-tools">
      <input v-model="localSearch" type="search" placeholder="Search issues" />
      <div class="inline-filter-tabs" role="tablist" aria-label="Issue read status filter">
        <button type="button" :class="{ active: status === 'all' }" role="tab" :aria-selected="status === 'all'" @click="status = 'all'">
          All
        </button>
        <button type="button" :class="{ active: status === 'unread' }" role="tab" :aria-selected="status === 'unread'" @click="status = 'unread'">
          Unread
        </button>
        <button type="button" :class="{ active: status === 'read' }" role="tab" :aria-selected="status === 'read'" @click="status = 'read'">
          Read
        </button>
      </div>
      <select v-model="sort" aria-label="Sort issues">
        <option value="series">Series</option>
        <option value="title">Title</option>
        <option value="date">Date</option>
        <option value="publisher">Publisher</option>
        <option value="read">Read Status</option>
      </select>
      <select v-model="direction" aria-label="Sort direction">
        <option value="asc">Ascending</option>
        <option value="desc">Descending</option>
      </select>
      <button v-if="localSearch" class="ghost-button" type="button" @click="localSearch = ''">
        Clear
      </button>
    </div>

    <div v-if="visibleComics.length" class="issue-list">
      <IssueListItem
        v-for="(comic, index) in visibleComics"
        :key="`${comic.id}-${index}`"
        :comic="comic"
        :selected="selectedComicId === comic.id"
        :quick-saving="quickSavingComicId === comic.id"
        :show-cover="showCover"
        :show-comment="showComment"
        @open="$emit('open-comic', $event)"
        @toggle-read="$emit('toggle-read', $event)"
      />
    </div>

    <div v-else class="empty-state">
      {{ hasFilters ? filteredEmptyMessage : emptyMessage }}
      <button v-if="showNewButton && !hasFilters" class="secondary-button" type="button" @click="$emit('new-comic')">
        <span aria-hidden="true" class="button-icon">+</span>
        Add the first comic
      </button>
    </div>
  </section>
</template>
