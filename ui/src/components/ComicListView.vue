<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

import { listComics } from '@/api/client.js'
import IssueListItem from '@/components/IssueListItem.vue'

const props = defineProps({
  comics: {
    type: Array,
    default: () => [],
  },
  totalCount: {
    type: Number,
    default: null,
  },
  search: {
    type: String,
    default: null,
  },
  serverSearch: {
    type: Boolean,
    default: false,
  },
  serverSource: {
    type: Boolean,
    default: false,
  },
  sourceParams: {
    type: Object,
    default: () => ({}),
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
  showReadingOrderSort: {
    type: Boolean,
    default: false,
  },
  initialSort: {
    type: String,
    default: 'series',
  },
  status: {
    type: String,
    default: null,
  },
  sort: {
    type: String,
    default: null,
  },
  direction: {
    type: String,
    default: null,
  },
  paginateLocal: {
    type: Boolean,
    default: false,
  },
  readOnly: {
    type: Boolean,
    default: false,
  },
  localPageSize: {
    type: Number,
    default: 50,
  },
})

const emit = defineEmits([
  'open-comic',
  'toggle-read',
  'update:search',
  'update:status',
  'update:sort',
  'update:direction',
  'new-comic',
])

const localSearch = ref('')
const localStatus = ref('all')
const tag = ref('all')
const localSort = ref(
  props.showReadingOrderSort ? props.initialSort : normalizeSort(props.initialSort),
)
const localDirection = ref('asc')
const visibleLimit = ref(props.localPageSize)
const loadMoreSentinel = ref(null)
const autoLoadSupported = ref(false)

let loadMoreObserver = null

const searchText = computed({
  get() {
    return props.search === null ? localSearch.value : props.search
  },
  set(value) {
    if (props.search === null) {
      localSearch.value = value
      return
    }

    emit('update:search', value)
  },
})

const searchTerm = computed(() => searchText.value.trim().toLowerCase())

const statusModel = computed({
  get: () => (props.status == null ? localStatus.value : props.status),
  set(value) {
    if (props.status == null) {
      localStatus.value = value
      return
    }

    emit('update:status', value)
  },
})

const sortModel = computed({
  get: () => normalizeSort(props.sort == null ? localSort.value : props.sort),
  set(value) {
    const normalized = normalizeSort(value)

    if (props.sort == null) {
      localSort.value = normalized
      return
    }

    emit('update:sort', normalized)
  },
})

const directionModel = computed({
  get: () => (props.direction == null ? localDirection.value : props.direction),
  set(value) {
    if (props.direction == null) {
      localDirection.value = value
      return
    }

    emit('update:direction', value)
  },
})

const serverComics = ref([])
const serverTotal = ref(0)
const serverOffset = ref(0)
const serverHasMore = ref(false)
const serverLoading = ref(false)

const effectiveServerMode = computed(() => props.serverSearch || props.serverSource)
const sourceComics = computed(() => (props.serverSource ? serverComics.value : props.comics))

// In server-source mode, serverComics is fetched independently of the
// `comics` prop. The parent already updates the read status of the
// matching comic inside `comics` when a toggle-read completes (see
// applyComicReadState in useComics.js) -- reconcile that back into
// serverComics so the visible list actually reflects it instead of only
// the parent's own copy of the data.
watch(
  () =>
    props.serverSource
      ? props.comics.map((comic) => `${comic.id}:${comic.read ? 1 : 0}`).join('|')
      : '',
  () => {
    if (!props.serverSource) return
    const readById = new Map(props.comics.map((comic) => [comic.id, Boolean(comic.read)]))
    serverComics.value = serverComics.value.map((comic) => {
      const read = readById.get(comic.id)
      return read !== undefined && read !== comic.read ? { ...comic, read } : comic
    })
  },
)
const sourceParamsKey = computed(() => JSON.stringify(props.sourceParams || {}))

const filteredComics = computed(() => {
  const filtered = sourceComics.value.filter((comic) => {
    if (!effectiveServerMode.value) {
      if (statusModel.value === 'read' && !comic.read) return false
      if (statusModel.value === 'unread' && comic.read) return false
      if (tag.value !== 'all' && !comicTags(comic).some((item) => item.toLowerCase() === tag.value))
        return false
    }

    if (effectiveServerMode.value || !searchTerm.value) return true

    return [
      comic.title,
      comic.series,
      comic.issue,
      comic.publisher,
      comic.coverDate,
      comic.comment,
      comic.tags,
    ]
      .filter((value) => value !== undefined && value !== null && value !== '')
      .some((value) => String(value).toLowerCase().includes(searchTerm.value))
  })

  if (effectiveServerMode.value) return filtered

  if (sortModel.value === 'readingOrder') {
    return directionModel.value === 'desc' ? [...filtered].reverse() : filtered
  }

  return [...filtered].sort((a, b) => {
    const result = compareComics(a, b, sortModel.value)
    return directionModel.value === 'desc' ? -result : result
  })
})

const visibleComics = computed(() => {
  if (props.serverSource) return filteredComics.value
  if (!props.paginateLocal) return filteredComics.value

  return filteredComics.value.slice(0, visibleLimit.value)
})

const tagOptions = computed(() => {
  const seen = new Set()

  sourceComics.value.forEach((comic) => {
    comicTags(comic).forEach((item) => {
      const value = item.trim()
      if (value) seen.add(value)
    })
  })

  return [...seen].sort((a, b) => compareText(a, b))
})

const readCount = computed(() => sourceComics.value.filter((comic) => comic.read).length)
const hasFilters = computed(
  () =>
    searchTerm.value ||
    statusModel.value !== 'all' ||
    (!effectiveServerMode.value && tag.value !== 'all'),
)
const totalComics = computed(() =>
  props.serverSource ? serverTotal.value : (props.totalCount ?? props.comics.length),
)

const summaryText = computed(() => {
  const loaded = sourceComics.value.length
  const total = totalComics.value

  if (hasFilters.value) {
    return `${filteredComics.value.length} matching loaded · ${loaded} of ${total} loaded · ${readCount.value} loaded read`
  }

  return `${loaded} of ${total} loaded · ${readCount.value} loaded read`
})

const canLoadMoreServer = computed(
  () => props.serverSource && serverHasMore.value && !serverLoading.value,
)
const canLoadMoreLocal = computed(
  () =>
    !props.serverSource &&
    props.paginateLocal &&
    visibleComics.value.length < filteredComics.value.length,
)
const showManualLoadMore = computed(
  () => (canLoadMoreLocal.value || canLoadMoreServer.value) && !autoLoadSupported.value,
)

function normalizeSort(sort) {
  if (sort === 'readingOrder' && !props.showReadingOrderSort) return 'series'
  return sort || 'series'
}

function compareComics(a, b, mode) {
  if (mode === 'readingOrder') return 0
  if (mode === 'date') return compareText(a.coverDate, b.coverDate) || compareSeries(a, b)
  if (mode === 'publisher') return compareText(a.publisher, b.publisher) || compareSeries(a, b)
  if (mode === 'read') return Number(a.read) - Number(b.read) || compareSeries(a, b)
  if (mode === 'title') return compareText(a.title, b.title) || compareSeries(a, b)

  return compareSeries(a, b)
}

function compareSeries(a, b) {
  return (
    compareText(a.series, b.series) ||
    (a.seriesYear ?? 0) - (b.seriesYear ?? 0) ||
    compareText(a.issue, b.issue) ||
    compareText(a.title, b.title)
  )
}

function compareText(a, b) {
  return String(a || '').localeCompare(String(b || ''), undefined, {
    numeric: true,
    sensitivity: 'base',
  })
}

function comicTags(comic) {
  return String(comic.tags || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function clearFilters() {
  searchText.value = ''
  statusModel.value = 'all'
  tag.value = 'all'
}

function loadMoreLocal() {
  if (props.serverSource) {
    fetchServerComics({ append: true })
    return
  }

  if (!canLoadMoreLocal.value) return
  visibleLimit.value += props.localPageSize
}

async function fetchServerComics({ append = false } = {}) {
  if (!props.serverSource || serverLoading.value) return

  const offset = append ? serverOffset.value : 0
  serverLoading.value = true

  try {
    const params = {
      ...props.sourceParams,
      limit: props.localPageSize,
      offset,
      sort: sortModel.value,
      direction: directionModel.value,
    }

    if (searchTerm.value) params.q = searchTerm.value
    if (statusModel.value === 'read') params.read = true
    if (statusModel.value === 'unread') params.read = false

    const page = await listComics(params)

    serverComics.value = append ? [...serverComics.value, ...page.items] : page.items
    serverTotal.value = page.total
    serverHasMore.value = page.hasMore
    serverOffset.value = offset + page.items.length
  } finally {
    serverLoading.value = false
  }
}

function setupLoadMoreObserver() {
  if (typeof IntersectionObserver === 'undefined') return

  autoLoadSupported.value = true
  loadMoreObserver = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        loadMoreLocal()
      }
    },
    { rootMargin: '320px 0px' },
  )

  observeLoadMoreSentinel()
}

function observeLoadMoreSentinel() {
  if (!loadMoreObserver) return

  loadMoreObserver.disconnect()

  if (loadMoreSentinel.value && (canLoadMoreLocal.value || canLoadMoreServer.value)) {
    loadMoreObserver.observe(loadMoreSentinel.value)
  }
}

onMounted(() => {
  setupLoadMoreObserver()

  if (props.serverSource) {
    fetchServerComics()
  }
})

onUnmounted(() => {
  if (loadMoreObserver) {
    loadMoreObserver.disconnect()
    loadMoreObserver = null
  }
})

watch([searchTerm, statusModel, sortModel, directionModel, sourceParamsKey], () => {
  visibleLimit.value = props.localPageSize

  if (props.serverSource) {
    fetchServerComics()
  }
})

watch([visibleComics, canLoadMoreLocal, canLoadMoreServer], () => {
  nextTick(observeLoadMoreSentinel)
})
</script>

<template>
  <section class="comic-list-view">
    <div class="comic-list-sticky">
      <header class="comic-list-header">
        <div>
          <p class="eyebrow">{{ title }}</p>
          <small>{{ summaryText }}</small>
        </div>

        <button
          v-if="showNewButton && !readOnly"
          class="primary-button"
          type="button"
          @click="$emit('new-comic')"
        >
          New Comic
        </button>
      </header>

      <div v-if="sourceComics.length || serverSource || hasFilters" class="comic-list-tools">
        <input v-model="searchText" type="search" placeholder="Search issues" />

        <div class="inline-filter-tabs" role="tablist" aria-label="Issue read status filter">
          <button
            type="button"
            :class="{ active: statusModel === 'all' }"
            role="tab"
            :aria-selected="statusModel === 'all'"
            @click="statusModel = 'all'"
          >
            All
          </button>
          <button
            type="button"
            :class="{ active: statusModel === 'unread' }"
            role="tab"
            :aria-selected="statusModel === 'unread'"
            @click="statusModel = 'unread'"
          >
            Unread
          </button>
          <button
            type="button"
            :class="{ active: statusModel === 'read' }"
            role="tab"
            :aria-selected="statusModel === 'read'"
            @click="statusModel = 'read'"
          >
            Read
          </button>
        </div>

        <select
          v-if="!effectiveServerMode && tagOptions.length"
          v-model="tag"
          aria-label="Filter by tag"
        >
          <option value="all">All Tags</option>
          <option v-for="option in tagOptions" :key="option" :value="option.toLowerCase()">
            {{ option }}
          </option>
        </select>

        <select v-model="sortModel" aria-label="Sort issues">
          <option v-if="showReadingOrderSort" value="readingOrder">Reading Order</option>
          <option value="series">Series</option>
          <option value="title">Title</option>
          <option value="date">Date</option>
          <option value="publisher">Publisher</option>
          <option value="read">Read Status</option>
        </select>

        <select v-model="directionModel" aria-label="Sort direction">
          <option value="asc">Ascending</option>
          <option value="desc">Descending</option>
        </select>

        <button v-if="hasFilters" class="ghost-button" type="button" @click="clearFilters">
          Clear
        </button>
      </div>
    </div>

    <template v-if="visibleComics.length">
      <div class="issue-list">
        <IssueListItem
          v-for="(comic, index) in visibleComics"
          :key="`${comic.id}-${index}`"
          :comic="comic"
          :selected="selectedComicId === comic.id"
          :quick-saving="quickSavingComicId === comic.id"
          :show-cover="showCover"
          :show-comment="showComment"
          :read-only="readOnly"
          @open="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
        />
      </div>

      <div
        v-if="canLoadMoreLocal || canLoadMoreServer"
        ref="loadMoreSentinel"
        class="issue-list-sentinel"
        aria-hidden="true"
      ></div>

      <button
        v-if="showManualLoadMore"
        class="ghost-button load-more-button"
        type="button"
        @click="loadMoreLocal"
      >
        Load more
      </button>
    </template>

    <div v-else class="empty-state">
      {{ hasFilters ? filteredEmptyMessage : emptyMessage }}
    </div>
  </section>
</template>
