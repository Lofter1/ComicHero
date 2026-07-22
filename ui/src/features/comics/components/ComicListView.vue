<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

import { listComics } from '@/api/client.js'
import IssueListItem from '@/shared/components/browse/IssueListItem.vue'
import ComicListToolbar from './ComicListToolbar.vue'
import EmptyState from '@/shared/components/feedback/EmptyState.vue'

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
  embedded: {
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
  showSections: {
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
  'toggle-skipped',
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
const collapsedSections = ref(new Set())

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
const selectedStatuses = computed(() => statusValues(statusModel.value))

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
      ? props.comics
          .map((comic) => `${comic.id}:${comic.read ? 1 : 0}:${comic.skipped ? 1 : 0}`)
          .join('|')
      : '',
  () => {
    if (!props.serverSource) return
    const stateById = new Map(
      props.comics.map((comic) => [
        comic.id,
        { read: Boolean(comic.read), skipped: Boolean(comic.skipped) },
      ]),
    )
    serverComics.value = serverComics.value.map((comic) => {
      const state = stateById.get(comic.id)
      if (!state) return comic
      return state.read !== comic.read || state.skipped !== comic.skipped
        ? { ...comic, ...state }
        : comic
    })
  },
)
const sourceParamsKey = computed(() => JSON.stringify(props.sourceParams || {}))

const filteredComics = computed(() => {
  const filtered = sourceComics.value.filter((comic) => {
    if (!effectiveServerMode.value) {
      if (!statusMatches(comic, selectedStatuses.value)) return false
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
const skippedCount = computed(() => sourceComics.value.filter((comic) => comic.skipped).length)
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
    return `${filteredComics.value.length} matching loaded · ${loaded} of ${total} loaded · ${readCount.value} loaded read · ${skippedCount.value} skipped`
  }

  return `${loaded} of ${total} loaded · ${readCount.value} loaded read · ${skippedCount.value} skipped`
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
const sectionHeadingsVisible = computed(
  () => props.showSections && sortModel.value === 'readingOrder',
)

function showSectionBefore(comic, index) {
  if (!sectionHeadingsVisible.value || !comic.section?.title) return false
  return index === 0 || visibleComics.value[index - 1]?.section?.key !== comic.section.key
}

function isSectionCollapsed(comic) {
  return sectionHeadingsVisible.value && collapsedSections.value.has(comic.section?.key)
}

function toggleSection(section) {
  if (!section?.key) return

  const next = new Set(collapsedSections.value)
  if (next.has(section.key)) {
    next.delete(section.key)
  } else {
    next.add(section.key)
  }
  collapsedSections.value = next
}

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

function statusValues(value) {
  if (!value || value === 'all') return ['unread', 'read', 'skipped']
  return String(value)
    .split(',')
    .map((item) => item.trim())
    .filter((item) => ['unread', 'read', 'skipped'].includes(item))
}

function statusMatches(comic, statuses) {
  if (!statuses.length || statuses.length === 3) return true
  return statuses.some((status) => comicStatusCategory(comic, status))
}

function comicStatusCategory(comic, status) {
  if (status === 'skipped') return Boolean(comic.skipped)
  if (status === 'read') return Boolean(comic.read) && !comic.skipped
  return !comic.read && !comic.skipped
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
    if (statusModel.value !== 'all') params.status = statusModel.value

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
  <section
    class="comic-list-view grid w-full min-w-0 max-w-full gap-3"
    :class="{ 'comic-list-view--embedded': embedded }"
  >
    <ComicListToolbar
      v-model:search="searchText"
      v-model:status="statusModel"
      v-model:tag="tag"
      v-model:sort="sortModel"
      v-model:direction="directionModel"
      :title="title"
      :summary-text="summaryText"
      :has-content="Boolean(sourceComics.length)"
      :server-source="serverSource"
      :has-filters="Boolean(hasFilters)"
      :effective-server-mode="effectiveServerMode"
      :tag-options="tagOptions"
      :show-reading-order-sort="showReadingOrderSort"
      :show-new-button="showNewButton"
      :read-only="readOnly"
      @new-comic="$emit('new-comic')"
    />

    <template v-if="visibleComics.length">
      <div class="issue-list grid w-full min-w-0 max-w-full gap-2.5">
        <template v-for="(comic, index) in visibleComics" :key="`${comic.id}-${index}`">
          <!-- Native button: section headings are expandable full-width content rows. -->
          <button
            v-if="showSectionBefore(comic, index)"
            class="reading-order-section-heading"
            :class="{
              'nested-reading-order-heading': comic.section.kind === 'readingOrder',
            }"
            type="button"
            :aria-expanded="!isSectionCollapsed(comic)"
            :aria-label="`${isSectionCollapsed(comic) ? 'Expand' : 'Collapse'} ${comic.section.label || 'section'} ${comic.section.title}`"
            @click="toggleSection(comic.section)"
          >
            <span
              class="reading-order-section-heading-content grid gap-1 min-w-0 [&_strong]:text-lg"
            >
              <span class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">{{
                comic.section.label || 'Section'
              }}</span>
              <strong>{{ comic.section.title }}</strong>
              <span v-if="comic.section.description" class="section-description">
                {{ comic.section.description }}
              </span>
            </span>
            <span class="section-collapse-icon" aria-hidden="true">⌄</span>
          </button>
          <IssueListItem
            v-if="!isSectionCollapsed(comic)"
            :comic="comic"
            :selected="selectedComicId === comic.id"
            :quick-saving="quickSavingComicId === comic.id"
            :show-cover="showCover"
            :show-comment="showComment"
            :read-only="readOnly"
            @open="$emit('open-comic', $event)"
            @toggle-read="$emit('toggle-read', $event)"
            @toggle-skipped="$emit('toggle-skipped', $event)"
          />
        </template>
      </div>

      <div
        v-if="canLoadMoreLocal || canLoadMoreServer"
        ref="loadMoreSentinel"
        class="issue-list-sentinel w-full h-px"
        aria-hidden="true"
      ></div>

      <!-- Native button: this is a borderless inline loading affordance. -->
      <button
        v-if="showManualLoadMore"
        class="ghost-button load-more-button"
        type="button"
        @click="loadMoreLocal"
      >
        Load more
      </button>
    </template>

    <EmptyState v-else>
      {{ hasFilters ? filteredEmptyMessage : emptyMessage }}
    </EmptyState>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.reading-order-section-heading {
  @apply grid grid-cols-[minmax(0,1fr)_auto] items-center gap-1 w-full [border-bottom:2px_solid_color-mix(in_srgb,var(--primary)_52%,var(--line))] border-t-0 border-r-0 border-l-0 rounded-none bg-transparent text-inherit text-left pt-4 px-1 pb-2.5 cursor-pointer first:pt-1 hover:[background:color-mix(in_srgb,var(--primary)_5%,transparent)] [&.nested-reading-order-heading]:border-b-[color-mix(in_srgb,var(--accent)_52%,var(--line))] [&_.section-description]:text-muted [&_.section-description]:font-medium [&[aria-expanded='false']_.section-collapse-icon]:transform-[rotate(-90deg)];
}

.section-collapse-icon {
  @apply me-2 text-xl leading-none [transition:transform_160ms_ease];
}

.ghost-button.load-more-button {
  @apply min-h-8 border-0 rounded-[7px] bg-transparent text-accent py-1.5 px-2 font-bold;
}

.comic-list-view--embedded {
  @apply border-t border-line pt-3.5;
}

.comic-list-view--embedded small {
  @apply block text-muted;
}

.comic-list-view--embedded :is(ol, ul) {
  @apply mb-0 pl-6;
}

.comic-list-view--embedded li {
  @apply mb-2.5;
}
</style>
