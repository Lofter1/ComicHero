<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import ArcDetailView from '@/components/ArcDetailView.vue'
import ArcsBrowseView from '@/components/ArcsBrowseView.vue'
import AppSidebar from '@/components/AppSidebar.vue'
import AppToolbar from '@/components/AppToolbar.vue'
import CharactersBrowseView from '@/components/CharactersBrowseView.vue'
import CharacterDetailView from '@/components/CharacterDetailView.vue'
import ComicDetailView from '@/components/ComicDetailView.vue'
import ComicListView from '@/components/ComicListView.vue'
import MetronImport from '@/components/MetronImport.vue'
import MetronImportMonitor from '@/components/MetronImportMonitor.vue'
import ReadingOrderDetailView from '@/components/ReadingOrderDetailView.vue'
import ReadingOrderEditView from '@/components/ReadingOrderEditView.vue'
import ReadingOrdersBrowseView from '@/components/ReadingOrdersBrowseView.vue'
import SeriesBrowseView from '@/components/SeriesBrowseView.vue'
import SeriesDetailView from '@/components/SeriesDetailView.vue'
import { useArcs } from '@/composables/useArcs.js'
import { useCharacters } from '@/composables/useCharacters.js'
import { useComics } from '@/composables/useComics.js'
import { useMetronJobs } from '@/composables/useMetronJobs.js'
import { usePagination } from '@/composables/usePagination.js'
import { useReadingOrders } from '@/composables/useReadingOrders.js'
import { useSeries } from '@/composables/useSeries.js'

const activeView = ref('readingOrders')
const viewMode = ref('browse')
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const search = ref('')
const listOptions = ref({
  readingOrders: { filter: 'all', sort: 'name', direction: 'asc' },
  arcs: { filter: 'all', sort: 'name', direction: 'asc' },
  comics: { status: 'all', sort: 'series', direction: 'asc' },
  series: { filter: 'all', sort: 'name', direction: 'asc' },
  characters: { filter: 'all', sort: 'name', direction: 'asc' },
})
const loadMoreSentinel = ref(null)
const themePreference = ref(getInitialThemePreference())
const systemTheme = ref(getSystemTheme())
let loadMoreObserver = null
let searchDebounceTimer = null
let themeMediaQuery = null
let routeSyncPaused = false

const searchTerm = computed(() => search.value.trim().toLowerCase())
const activeListParams = computed(() => {
  const options = listOptions.value[activeView.value] || {}
  const params = {}
  if (options.filter === 'favorites') params.favorite = true
  if (options.filter === 'other') params.favorite = false
  if (options.status === 'read') params.read = true
  if (options.status === 'unread') params.read = false
  if (options.sort) params.sort = options.sort
  if (options.direction) params.direction = options.direction
  return params
})
const isEditing = computed(() => viewMode.value === 'edit')
const isDetail = computed(() => viewMode.value === 'detail')
const resolvedTheme = computed(() => themePreference.value === 'system' ? systemTheme.value : themePreference.value)
const currentRoutePath = computed(() => routeForCurrentState())

const {
  pageState,
  showInfiniteScrollSentinel,
  activeListLoadingMore,
  listTotal,
  loadPagedList,
} = usePagination({ activeView, loading, isEditing, isDetail, searchTerm, listParams: activeListParams })

const {
  metronImportJobs,
  metronImportMonitorOpen,
  metronQuota,
  loadMetronImportJobs,
  loadMetronQuota,
  updateMetronQuota,
  trackMetronImportJob,
  dismissMetronJob,
  retryMetronJob,
  continueMetronJob,
  cancelMetronJob,
  closeMetronImportEvents,
} = useMetronJobs({ activeView, error, handleImported: handleMetronImported })

const {
  series,
  selectedSeries,
  visibleSeries,
  seriesBrowseSections,
  openSeries,
  toggleSeriesFavorite,
  importSelectedSeriesFromMetron,
  seriesImportRunning,
  refreshSelectedSeriesDetail,
  loadSeries,
} = useSeries({ activeView, viewMode, error, loadPagedList, metronImportJobs, trackMetronImportJob })

const {
  characters,
  selectedCharacter,
  quickSavingCharacterID,
  visibleCharacters,
  characterBrowseSections,
  openCharacter,
  toggleCharacterFavorite,
  importSelectedCharacterAppearances,
  characterImportRunning,
  refreshSelectedCharacterDetail,
  loadCharacters,
} = useCharacters({ activeView, viewMode, error, loadPagedList, metronImportJobs, trackMetronImportJob })

const {
  arcs,
  selectedArc,
  quickSavingArcID,
  arcForm,
  visibleArcs,
  arcBrowseSections,
  arcProgress,
  openArc,
  toggleArcFavorite,
  newArc,
  saveArc,
  deleteArc,
  editArc,
  loadArcs,
} = useArcs({
  activeView,
  viewMode,
  error,
  saving,
  loadComics: (...args) => loadComics(...args),
  loadPagedList,
})

const {
  readingOrders,
  selectedOrder,
  quickSavingOrderID,
  orderForm,
  visibleOrders,
  readingOrderBrowseSections,
  readingOrderProgress,
  openReadingOrder,
  toggleReadingOrderFavorite,
  newReadingOrder,
  saveReadingOrder,
  deleteReadingOrder,
  editReadingOrder,
  importReadingOrderCBLFile,
  exportSelectedReadingOrderCBL,
  loadReadingOrders,
} = useReadingOrders({
  activeView,
  viewMode,
  error,
  saving,
  loadComics: (...args) => loadComics(...args),
  loadPagedList,
})

const {
  comics,
  selectedComic,
  quickSavingComicID,
  comicReturnTarget,
  comicForm,
  metronMetadataOpen,
  metronMetadataSearching,
  metronMetadataApplyingID,
  metronMetadataStatus,
  metronMetadataResults,
  loadComics,
  openComic,
  openOrderComic,
  newComic,
  editComic,
  saveComic,
  deleteComic,
  toggleComicRead,
  resetMetronMetadata,
  searchSelectedComicMetron,
  applyMetronMetadata,
} = useComics({
  activeView,
  viewMode,
  error,
  saving,
  loadPagedList,
  refreshActiveLibraryData,
  readingOrderProgress,
  selectedOrder,
  selectedArc,
  selectedCharacter,
  selectedSeries,
  collectionProgress: arcProgress,
})

const toolbarResultCount = computed(() => {
  if (activeView.value === 'readingOrders') return visibleOrders.value.length
  if (activeView.value === 'arcs') return visibleArcs.value.length
  if (activeView.value === 'comics') return comics.value.length
  if (activeView.value === 'series') return visibleSeries.value.length
  if (activeView.value === 'characters') return visibleCharacters.value.length
  return 0
})
const toolbarTotalCount = computed(() => {
  if (activeView.value === 'readingOrders') return listTotal('readingOrders')
  if (activeView.value === 'arcs') return listTotal('arcs')
  if (activeView.value === 'comics') return listTotal('comics')
  if (activeView.value === 'series') return listTotal('series')
  if (activeView.value === 'characters') return listTotal('characters')
  return 0
})
const loadingLabel = computed(() => {
  if (activeView.value === 'readingOrders') return 'Loading orders...'
  if (activeView.value === 'arcs') return 'Loading arcs...'
  if (activeView.value === 'comics') return 'Loading comics...'
  if (activeView.value === 'series') return 'Loading series...'
  if (activeView.value === 'characters') return 'Loading characters...'
  if (activeView.value === 'metron') return 'Loading Metron...'
  return 'Loading...'
})
const showBlockingLoading = computed(() => loading.value && activeView.value !== 'series')
const seriesListLoading = computed(() => Boolean(pageState.value.series?.refreshing))

function getInitialThemePreference() {
  if (typeof window === 'undefined') return 'system'
  const savedTheme = window.localStorage.getItem('comichero-theme')
  if (savedTheme === 'dark' || savedTheme === 'light' || savedTheme === 'system') return savedTheme
  return 'system'
}

function getSystemTheme() {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function applyTheme(value) {
  if (typeof document === 'undefined') return
  document.documentElement.dataset.theme = value
  document.documentElement.style.colorScheme = value
}

function setThemePreference(value) {
  themePreference.value = value
}

function handleSystemThemeChange(event) {
  systemTheme.value = event.matches ? 'dark' : 'light'
}

function normalizeAppPath(pathname) {
  const path = pathname || '/'
  if (path === '/') return '/'
  return path.replace(/\/+$/, '') || '/'
}

function parseRouteID(value) {
  const id = Number(value)
  return Number.isInteger(id) && id > 0 ? id : null
}

function parseAppRoute(pathname) {
  const parts = normalizeAppPath(pathname).split('/').filter(Boolean)
  if (parts.length === 0) return { view: 'readingOrders', mode: 'browse' }

  const [section, rawID, action] = parts
  if (section === 'metron' && parts.length === 1) return { view: 'metron', mode: 'browse' }
  if (section === 'comics') return parseEntityRoute('comics', rawID, action, parts.length)
  if (section === 'arcs') return parseEntityRoute('arcs', rawID, action, parts.length)
  if (section === 'series') return parseEntityRoute('series', rawID, action, parts.length, { canEdit: false })
  if (section === 'characters') return parseEntityRoute('characters', rawID, action, parts.length, { canEdit: false })
  if (section === 'reading-orders' || section === 'readingOrders') {
    return parseEntityRoute('readingOrders', rawID, action, parts.length)
  }
  return { view: 'readingOrders', mode: 'browse', replace: true }
}

function parseEntityRoute(view, rawID, action, partCount, { canEdit = true } = {}) {
  if (partCount === 1) return { view, mode: 'browse' }
  if (rawID === 'new' && canEdit && partCount === 2) return { view, mode: 'edit', isNew: true }

  const id = parseRouteID(rawID)
  if (!id) return { view, mode: 'browse', replace: true }
  if (action === undefined && partCount === 2) return { view, mode: 'detail', id }
  if (action === 'edit' && canEdit && partCount === 3) return { view, mode: 'edit', id }
  return { view, mode: 'detail', id, replace: true }
}

function browseRoutePath(view) {
  if (view === 'readingOrders') return '/reading-orders'
  if (view === 'arcs') return '/arcs'
  if (view === 'comics') return '/comics'
  if (view === 'series') return '/series'
  if (view === 'characters') return '/characters'
  if (view === 'metron') return '/metron'
  return '/reading-orders'
}

function detailRoutePath(view, id) {
  if (!id) return ''
  return `${browseRoutePath(view)}/${id}`
}

function editRoutePath(view, id) {
  if (view === 'readingOrders') return id ? `/reading-orders/${id}/edit` : '/reading-orders/new'
  if (view === 'arcs') return id ? `/arcs/${id}/edit` : '/arcs/new'
  if (view === 'comics') return id ? `/comics/${id}/edit` : '/comics/new'
  return browseRoutePath(view)
}

function routeForCurrentState() {
  if (viewMode.value === 'browse') return browseRoutePath(activeView.value)
  if (viewMode.value === 'detail') {
    if (activeView.value === 'readingOrders') return detailRoutePath(activeView.value, selectedOrder.value?.id)
    if (activeView.value === 'arcs') return detailRoutePath(activeView.value, selectedArc.value?.id)
    if (activeView.value === 'comics') return detailRoutePath(activeView.value, selectedComic.value?.id)
    if (activeView.value === 'series') return detailRoutePath(activeView.value, selectedSeries.value?.id)
    if (activeView.value === 'characters') return detailRoutePath(activeView.value, selectedCharacter.value?.id)
  }
  if (viewMode.value === 'edit') {
    if (activeView.value === 'readingOrders') return editRoutePath(activeView.value, orderForm.value?.id || selectedOrder.value?.id)
    if (activeView.value === 'arcs') return editRoutePath(activeView.value, arcForm.value?.id || selectedArc.value?.id)
    if (activeView.value === 'comics') return editRoutePath(activeView.value, comicForm.value?.id || selectedComic.value?.id)
  }
  return ''
}

function updateBrowserRoute(path, { replace = false } = {}) {
  if (routeSyncPaused || typeof window === 'undefined' || !path) return
  const current = normalizeAppPath(window.location.pathname)
  if (current === path) return

  const method = replace ? 'replaceState' : 'pushState'
  window.history[method]({}, '', path)
}

async function applyCurrentRoute(options = {}) {
  if (typeof window === 'undefined') {
    await loadData()
    return
  }
  await applyRoute(parseAppRoute(window.location.pathname), options)
}

async function applyRoute(route, { replace = false } = {}) {
  routeSyncPaused = true
  error.value = ''

  try {
    await activateRoute(route)
  } catch (err) {
    error.value = err.message
    activeView.value = route.view || 'readingOrders'
    viewMode.value = 'browse'
    await loadActiveViewData({ force: true })
  } finally {
    await nextTick()
    routeSyncPaused = false
  }

  const nextPath = currentRoutePath.value || browseRoutePath(activeView.value)
  updateBrowserRoute(nextPath, { replace: replace || route.replace })
}

async function activateRoute(route) {
  if (route.mode === 'browse') {
    comicReturnTarget.value = null
    activeView.value = route.view
    viewMode.value = 'browse'
    await loadData()
    return
  }

  if (route.view === 'readingOrders') {
    if (route.isNew) {
      newReadingOrder()
      return
    }
    await openReadingOrder({ id: route.id })
    if (route.mode === 'edit') editReadingOrder()
    return
  }

  if (route.view === 'arcs') {
    if (route.isNew) {
      newArc()
      return
    }
    await openArc({ id: route.id })
    if (route.mode === 'edit') editArc()
    return
  }

  if (route.view === 'comics') {
    if (route.isNew) {
      newComic()
      return
    }
    await openComic({ id: route.id }, { skipReturnTarget: true })
    if (route.mode === 'edit') editComic()
    return
  }

  if (route.view === 'series') {
    await openSeries({ id: route.id })
    return
  }

  if (route.view === 'characters') {
    await openCharacter({ id: route.id })
  }
}

function handleRoutePop() {
  applyCurrentRoute()
}

async function setView(view) {
  error.value = ''
  comicReturnTarget.value = null
  activeView.value = view
  viewMode.value = 'browse'
  await loadData()
}

async function loadData(force = false) {
  loading.value = true
  error.value = ''

  try {
    await Promise.all([
      loadActiveViewData({ force }),
      loadMetronImportJobs(),
    ])
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

async function loadActiveViewData(options = {}) {
  if (activeView.value === 'readingOrders') {
    await loadReadingOrders(options)
    return
  }
  if (activeView.value === 'arcs') {
    await loadArcs(options)
    return
  }
  if (activeView.value === 'comics') {
    await loadComics(options)
    return
  }
  if (activeView.value === 'series') {
    await loadSeries(options)
    return
  }
  if (activeView.value === 'characters') {
    await loadCharacters(options)
    return
  }
  if (activeView.value === 'metron') {
    await loadMetronQuota()
  }
}

async function refreshActiveLibraryData() {
  if (activeView.value === 'metron') return
  await loadActiveViewData({ force: true })
}

async function refreshActiveListData() {
  if (activeView.value === 'metron') return
  error.value = ''
  try {
    await loadActiveViewData({ force: true })
  } catch (err) {
    error.value = err.message
  }
}

async function loadMoreActiveViewData() {
  if (loading.value || isEditing.value || isDetail.value) return
  try {
    await loadActiveViewData({ append: true })
  } catch (err) {
    error.value = err.message
  }
}

function cancelEdit() {
  error.value = ''
  if (activeView.value === 'readingOrders' && selectedOrder.value) {
    viewMode.value = 'detail'
    return
  }
  if (activeView.value === 'arcs' && selectedArc.value) {
    viewMode.value = 'detail'
    return
  }
  if (activeView.value === 'comics' && selectedComic.value) {
    viewMode.value = 'detail'
    return
  }
  viewMode.value = 'browse'
}

function backToBrowse() {
  error.value = ''
  if (activeView.value === 'comics' && viewMode.value === 'detail' && comicReturnTarget.value) {
    activeView.value = comicReturnTarget.value.activeView
    viewMode.value = comicReturnTarget.value.viewMode
    comicReturnTarget.value = null
    return
  }
  viewMode.value = 'browse'
}

function showError(message) {
  error.value = message
}

function clearError() {
  error.value = ''
}

async function handleMetronImported() {
  error.value = ''
  await refreshActiveLibraryData()
  if (activeView.value === 'characters' && viewMode.value === 'detail') {
    await refreshSelectedCharacterDetail()
  }
  if (activeView.value === 'series' && viewMode.value === 'detail') {
    await refreshSelectedSeriesDetail()
  }
}

function updateSearch(value) {
  search.value = value
}

function updateListOption(view, key, value) {
  listOptions.value = {
    ...listOptions.value,
    [view]: {
      ...(listOptions.value[view] || {}),
      [key]: value,
    },
  }
  refreshActiveListData()
}

function setupLoadMoreObserver() {
  if (typeof IntersectionObserver === 'undefined') return
  loadMoreObserver = new IntersectionObserver((entries) => {
    if (entries.some(entry => entry.isIntersecting)) {
      loadMoreActiveViewData()
    }
  }, { rootMargin: '360px 0px' })
  observeLoadMoreSentinel()
}

function observeLoadMoreSentinel() {
  if (!loadMoreObserver) return
  loadMoreObserver.disconnect()
  if (loadMoreSentinel.value && showInfiniteScrollSentinel.value) {
    loadMoreObserver.observe(loadMoreSentinel.value)
  }
}

onMounted(() => {
  setupLoadMoreObserver()
  if (typeof window !== 'undefined') {
    window.addEventListener('popstate', handleRoutePop)
    themeMediaQuery = window.matchMedia?.('(prefers-color-scheme: dark)')
    themeMediaQuery?.addEventListener('change', handleSystemThemeChange)
  }
  applyCurrentRoute({ replace: true })
})

watch(showInfiniteScrollSentinel, () => {
  nextTick(observeLoadMoreSentinel)
})

watch(activeView, () => {
  nextTick(observeLoadMoreSentinel)
})

watch(currentRoutePath, (path) => {
  updateBrowserRoute(path)
})

watch(resolvedTheme, (value) => {
  applyTheme(value)
}, { immediate: true })

watch(themePreference, (value) => {
  if (typeof window === 'undefined') return
  window.localStorage.setItem('comichero-theme', value)
}, { immediate: true })

watch(search, () => {
  if (searchDebounceTimer) {
    window.clearTimeout(searchDebounceTimer)
  }
  searchDebounceTimer = window.setTimeout(() => {
    if (!isEditing.value && !isDetail.value && activeView.value !== 'metron') {
      refreshActiveListData()
    }
  }, 250)
})

onUnmounted(() => {
  closeMetronImportEvents()
  if (searchDebounceTimer) {
    window.clearTimeout(searchDebounceTimer)
  }
  if (loadMoreObserver) {
    loadMoreObserver.disconnect()
  }
  window.removeEventListener('popstate', handleRoutePop)
  themeMediaQuery?.removeEventListener('change', handleSystemThemeChange)
})
</script>

<template>
  <main class="app-shell">
    <AppSidebar
      :active-view="activeView"
      :loading="loading"
      :theme-preference="themePreference"
      @change-view="setView"
      @refresh="loadData(true)"
      @set-theme="setThemePreference"
    />

    <section class="content">
      <AppToolbar
        v-if="!isEditing && !isDetail"
        :active-view="activeView"
        :result-count="toolbarResultCount"
        :total-count="toolbarTotalCount"
      />

      <div v-if="error" class="toast error-toast" role="alert" aria-live="assertive">
        <span>{{ error }}</span>
        <button type="button" aria-label="Dismiss error" @click="clearError">Dismiss</button>
      </div>

      <MetronImportMonitor
        v-model:open="metronImportMonitorOpen"
        :jobs="metronImportJobs"
        @retry="retryMetronJob"
        @continue="continueMetronJob"
        @cancel="cancelMetronJob"
        @dismiss="dismissMetronJob"
      />

      <div v-if="showBlockingLoading" class="loading-panel" role="status" aria-live="polite">
        <span class="loading-spinner" aria-hidden="true"></span>
        <strong>{{ loadingLabel }}</strong>
      </div>

      <MetronImport
        v-else-if="activeView === 'metron'"
        :import-jobs="metronImportJobs"
        :metron-quota="metronQuota"
        @imported="handleMetronImported"
        @error="showError"
        @job-started="trackMetronImportJob"
        @quota-updated="updateMetronQuota"
      />

      <ReadingOrderEditView
        v-else-if="activeView === 'readingOrders' && isEditing"
        v-model:form="orderForm"
        :selected-order="selectedOrder"
        :comics="comics"
        :reading-orders="readingOrders"
        :saving="saving"
        @back="cancelEdit"
        @delete="deleteReadingOrder"
        @save="saveReadingOrder"
      />

      <ReadingOrderDetailView
        v-else-if="activeView === 'readingOrders' && isDetail"
        :selected-order="selectedOrder"
        :selected-comic-id="selectedComic?.id"
        :quick-saving-comic-id="quickSavingComicID"
        @back="backToBrowse"
        @edit="editReadingOrder"
        @export-cbl="exportSelectedReadingOrderCBL"
        @open-comic="openOrderComic"
        @toggle-read="toggleComicRead"
      />

      <ReadingOrdersBrowseView
        v-else-if="activeView === 'readingOrders'"
        :orders="visibleOrders"
        :sections="readingOrderBrowseSections"
        :selected-order-id="selectedOrder?.id"
        :quick-saving-order-id="quickSavingOrderID"
        :search="search"
        :search-term="searchTerm"
        :filter="listOptions.readingOrders.filter"
        :sort="listOptions.readingOrders.sort"
        :direction="listOptions.readingOrders.direction"
        @update:search="updateSearch"
        @update:filter="updateListOption('readingOrders', 'filter', $event)"
        @update:sort="updateListOption('readingOrders', 'sort', $event)"
        @update:direction="updateListOption('readingOrders', 'direction', $event)"
        @new-order="newReadingOrder"
        @open-order="openReadingOrder"
        @toggle-favorite="toggleReadingOrderFavorite"
        @import-cbl="importReadingOrderCBLFile"
      />


      <ArcDetailView
        v-else-if="activeView === 'arcs' && isDetail"
        :selected-arc="selectedArc"
        :selected-comic-id="selectedComic?.id"
        :quick-saving-comic-id="quickSavingComicID"
        :quick-saving-arc-id="quickSavingArcID"
        @back="backToBrowse"
        @edit="editArc"
        @toggle-favorite="toggleArcFavorite"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
      />

      <ArcsBrowseView
        v-else-if="activeView === 'arcs'"
        :arcs="visibleArcs"
        :sections="arcBrowseSections"
        :selected-arc-id="selectedArc?.id"
        :quick-saving-arc-id="quickSavingArcID"
        :search="search"
        :search-term="searchTerm"
        :filter="listOptions.arcs.filter"
        :sort="listOptions.arcs.sort"
        :direction="listOptions.arcs.direction"
        @update:search="updateSearch"
        @update:filter="updateListOption('arcs', 'filter', $event)"
        @update:sort="updateListOption('arcs', 'sort', $event)"
        @update:direction="updateListOption('arcs', 'direction', $event)"
        @new-arc="newArc"
        @open-arc="openArc"
        @toggle-favorite="toggleArcFavorite"
      />

      <SeriesDetailView
        v-else-if="activeView === 'series' && isDetail"
        :selected-series="selectedSeries"
        :selected-comic-id="selectedComic?.id"
        :quick-saving-comic-id="quickSavingComicID"
        :import-running="seriesImportRunning(selectedSeries)"
        @back="backToBrowse"
        @toggle-favorite="toggleSeriesFavorite"
        @import-series="importSelectedSeriesFromMetron"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
      />

      <SeriesBrowseView
        v-else-if="activeView === 'series'"
        :loading="seriesListLoading"
        :series="visibleSeries"
        :sections="seriesBrowseSections"
        :selected-series-id="selectedSeries?.id"
        :search="search"
        :search-term="searchTerm"
        :filter="listOptions.series.filter"
        :sort="listOptions.series.sort"
        :direction="listOptions.series.direction"
        @update:search="updateSearch"
        @update:filter="updateListOption('series', 'filter', $event)"
        @update:sort="updateListOption('series', 'sort', $event)"
        @update:direction="updateListOption('series', 'direction', $event)"
        @open-series="openSeries"
        @toggle-favorite="toggleSeriesFavorite"
        @new-comic="newComic"
      />

      <CharacterDetailView
        v-else-if="activeView === 'characters' && isDetail"
        :selected-character="selectedCharacter"
        :selected-comic-id="selectedComic?.id"
        :quick-saving-comic-id="quickSavingComicID"
        :quick-saving-character-id="quickSavingCharacterID"
        :import-running="characterImportRunning(selectedCharacter)"
        @back="backToBrowse"
        @toggle-favorite="toggleCharacterFavorite"
        @import-appearances="importSelectedCharacterAppearances"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
      />

      <CharactersBrowseView
        v-else-if="activeView === 'characters'"
        :characters="visibleCharacters"
        :sections="characterBrowseSections"
        :selected-character-id="selectedCharacter?.id"
        :quick-saving-character-id="quickSavingCharacterID"
        :search="search"
        :search-term="searchTerm"
        :filter="listOptions.characters.filter"
        :sort="listOptions.characters.sort"
        :direction="listOptions.characters.direction"
        @update:search="updateSearch"
        @update:filter="updateListOption('characters', 'filter', $event)"
        @update:sort="updateListOption('characters', 'sort', $event)"
        @update:direction="updateListOption('characters', 'direction', $event)"
        @open-character="openCharacter"
        @toggle-favorite="toggleCharacterFavorite"
      />

      <ComicDetailView
        v-else-if="activeView === 'comics' && isDetail"
        :selected-comic="selectedComic"
        :quick-saving-comic-id="quickSavingComicID"
        :metron-metadata-open="metronMetadataOpen"
        :metron-metadata-searching="metronMetadataSearching"
        :metron-metadata-applying-id="metronMetadataApplyingID"
        :metron-metadata-status="metronMetadataStatus"
        :metron-metadata-results="metronMetadataResults"
        @back="backToBrowse"
        @search-metron="searchSelectedComicMetron"
        @apply-metron="applyMetronMetadata"
        @reset-metron="resetMetronMetadata"
        @toggle-read="toggleComicRead"
        @edit="editComic"
        @open-character="openCharacter"
        @open-series="openSeries"
      />

      <div v-else class="browse-view">
        <ComicListView
          title="Comics"
          :comics="comics"
          :total-count="listTotal('comics')"
          :search="search"
          :status="listOptions.comics.status"
          :sort="listOptions.comics.sort"
          :direction="listOptions.comics.direction"
          server-search
          :selected-comic-id="selectedComic?.id"
          :quick-saving-comic-id="quickSavingComicID"
          show-new-button
          show-cover
          empty-message="No comics yet."
          filtered-empty-message="No comics match these filters."
          @new-comic="newComic"
          @update:search="updateSearch"
          @update:status="updateListOption('comics', 'status', $event)"
          @update:sort="updateListOption('comics', 'sort', $event)"
          @update:direction="updateListOption('comics', 'direction', $event)"
          @open-comic="openComic"
          @toggle-read="toggleComicRead"
        />
      </div>

      <div v-if="showInfiniteScrollSentinel" ref="loadMoreSentinel" class="load-more-sentinel" aria-live="polite">
        <span v-if="activeListLoadingMore" class="loading-spinner small" aria-hidden="true"></span>
        <span>{{ activeListLoadingMore ? 'Loading more...' : 'Scroll for more' }}</span>
      </div>
    </section>
  </main>
</template>
