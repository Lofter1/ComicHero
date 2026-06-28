<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  assetURL,
  cancelMetronImportJob,
  continueMetronImportJob,
  createComic,
  createReadingOrder,
  deleteComic as removeComic,
  deleteReadingOrder as removeReadingOrder,
  getCharacter,
  getComic,
  getMetronImportJob,
  getReadingOrder,
  importMetronCharacterAppearances,
  importMetronComic,
  importMetronReadingList,
  importMetronSeries,
  listCharacters,
  listComics,
  listReadingOrders,
  searchMetronComics,
  setReadingOrderComics,
  updateComic,
  updateComicFromMetron,
  updateComicReadStatus,
  updateCharacterFavorite,
  updateReadingOrder,
} from '@/api/client.js'
import AppSidebar from '@/components/AppSidebar.vue'
import AppToolbar from '@/components/AppToolbar.vue'
import ComicListView from '@/components/ComicListView.vue'
import ComicEditor from '@/components/ComicEditor.vue'
import MetronImport from '@/components/MetronImport.vue'
import ReadingOrderEditor from '@/components/ReadingOrderEditor.vue'
import { comicPayload, emptyComic } from '@/domain/comics.js'
import {
  emptyReadingOrder,
  formatProgress,
  readingOrderComicsPayload,
  readingOrderFormFromDetail,
  readingOrderMatchesSearch,
  readingOrderPayload,
} from '@/domain/readingOrders.js'

const activeView = ref('readingOrders')
const viewMode = ref('browse')
const comics = ref([])
const characters = ref([])
const readingOrders = ref([])
const selectedComic = ref(null)
const selectedCharacter = ref(null)
const selectedOrder = ref(null)
const loading = ref(false)
const saving = ref(false)
const quickSavingComicID = ref(null)
const quickSavingCharacterID = ref(null)
const quickSavingOrderID = ref(null)
const comicReturnTarget = ref(null)
const error = ref('')
const search = ref('')
const characterFilter = ref('all')
const readingOrderFilter = ref('all')
const metronMetadataOpen = ref(false)
const metronMetadataSearching = ref(false)
const metronMetadataApplyingID = ref(null)
const metronMetadataStatus = ref('')
const metronMetadataResults = ref([])
const metronImportJobs = ref([])
const metronImportMonitorOpen = ref(false)
const metronImportPollTimers = new Map()

const comicForm = ref(emptyComic())
const orderForm = ref(emptyReadingOrder())

const searchTerm = computed(() => search.value.trim().toLowerCase())
const isEditing = computed(() => viewMode.value === 'edit')
const isDetail = computed(() => viewMode.value === 'detail')
const filteredCharacters = computed(() => {
  return characters.value.filter(character => {
    if (!searchTerm.value) return true
    return [character.name, ...(character.aliases || [])]
      .filter(value => value !== undefined && value !== null && value !== '')
      .some(value => String(value).toLowerCase().includes(searchTerm.value))
  })
})
const visibleCharacters = computed(() => {
  if (characterFilter.value === 'favorites') {
    return filteredCharacters.value.filter(character => character.favorite)
  }
  return filteredCharacters.value
})
const favoriteVisibleCharacters = computed(() => filteredCharacters.value.filter(character => character.favorite))
const remainingVisibleCharacters = computed(() => filteredCharacters.value.filter(character => !character.favorite))
const characterBrowseSections = computed(() => {
  if (characterFilter.value === 'favorites') {
    return [{ key: 'favorites', title: 'Favorites', characters: favoriteVisibleCharacters.value }]
  }
  if (!favoriteVisibleCharacters.value.length) {
    return [{ key: 'all', title: 'All Characters', characters: filteredCharacters.value }]
  }
  return [
    { key: 'favorites', title: 'Favorites', characters: favoriteVisibleCharacters.value },
    { key: 'other', title: 'Other Characters', characters: remainingVisibleCharacters.value },
  ].filter(section => section.characters.length)
})
const filteredOrders = computed(() => {
  return readingOrders.value.filter(order => readingOrderMatchesSearch(order, searchTerm.value))
})
const visibleOrders = computed(() => {
  if (readingOrderFilter.value === 'favorites') {
    return filteredOrders.value.filter(order => order.favorite)
  }
  return filteredOrders.value
})
const unreadComicCount = computed(() => comics.value.filter(comic => !comic.read).length)
const favoriteCharacterCount = computed(() => characters.value.filter(character => character.favorite).length)
const favoriteOrderCount = computed(() => readingOrders.value.filter(order => order.favorite).length)
const toolbarResultCount = computed(() => {
  if (activeView.value === 'readingOrders') return visibleOrders.value.length
  if (activeView.value === 'comics') return comics.value.length
  if (activeView.value === 'characters') return visibleCharacters.value.length
  return 0
})
const toolbarTotalCount = computed(() => {
  if (activeView.value === 'readingOrders') return readingOrders.value.length
  if (activeView.value === 'comics') return comics.value.length
  if (activeView.value === 'characters') return characters.value.length
  return 0
})
const currentOrderIndex = computed(() => {
  return visibleOrders.value.findIndex(order => order.id === selectedOrder.value?.id)
})
const currentComicIndex = computed(() => {
  return comics.value.findIndex(comic => comic.id === selectedComic.value?.id)
})
const currentCharacterIndex = computed(() => {
  return visibleCharacters.value.findIndex(character => character.id === selectedCharacter.value?.id)
})
const metronImportInProgress = computed(() => {
  return metronImportJobs.value.some(job => job.status === 'queued' || job.status === 'running')
})
const metronImportSummary = computed(() => {
  const running = metronImportJobs.value.filter(job => job.status === 'queued' || job.status === 'running').length
  if (running > 0) return `${running} running`
  const latest = metronImportJobs.value[0]
  return latest ? latest.status : ''
})

function setView(view) {
  error.value = ''
  comicReturnTarget.value = null
  activeView.value = view
  viewMode.value = 'browse'
}

async function loadData() {
  loading.value = true
  error.value = ''

  try {
    await refreshLists()
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

async function refreshLists() {
  const [comicList, characterList, orderList] = await Promise.all([listComics(), listCharacters(), listReadingOrders()])
  comics.value = comicList
  characters.value = characterList
  readingOrders.value = orderList
}

async function openComic(comic, options = {}) {
  error.value = ''
  if (!options.preserveReturnTarget) {
    comicReturnTarget.value = {
      activeView: activeView.value,
      viewMode: viewMode.value,
    }
  }
  activeView.value = 'comics'
  viewMode.value = 'detail'
  resetMetronMetadata()
  const detail = await getComic(comic.id)
  selectedComic.value = detail
  comicForm.value = { ...detail }
}

async function openOrderComic(comic) {
  if (!comic?.id) return
  await openComic(comic)
}

async function openAdjacentComic(offset) {
  const nextComic = comics.value[currentComicIndex.value + offset]
  if (nextComic) {
    await openComic(nextComic, { preserveReturnTarget: true })
  }
}

async function openCharacter(character) {
  error.value = ''
  activeView.value = 'characters'
  viewMode.value = 'detail'
  const detail = await getCharacter(character.id)
  selectedCharacter.value = detail
}

async function openAdjacentCharacter(offset) {
  const nextCharacter = visibleCharacters.value[currentCharacterIndex.value + offset]
  if (nextCharacter) {
    await openCharacter(nextCharacter)
  }
}

async function toggleCharacterFavorite(character) {
  if (!character?.id || quickSavingCharacterID.value) return

  quickSavingCharacterID.value = character.id
  error.value = ''

  try {
    const detail = await updateCharacterFavorite(character.id, !character.favorite)
    applyCharacterFavoriteState(detail)
  } catch (err) {
    error.value = err.message
  } finally {
    quickSavingCharacterID.value = null
  }
}

function applyCharacterFavoriteState(detail) {
  characters.value = characters.value.map(character => {
    return character.id === detail.id ? { ...character, favorite: detail.favorite } : character
  })

  if (selectedCharacter.value?.id === detail.id) {
    selectedCharacter.value = { ...selectedCharacter.value, favorite: detail.favorite }
  }
}

function characterProgress(character) {
  return formatProgress(character?.progress ?? 0)
}

async function importSelectedCharacterAppearances() {
  if (!selectedCharacter.value?.metronCharacterId || characterImportRunning(selectedCharacter.value)) return

  error.value = ''
  try {
    const { data: job } = await importMetronCharacterAppearances(selectedCharacter.value.metronCharacterId)
    trackMetronImportJob({ ...job, displayName: selectedCharacter.value.name })
  } catch (err) {
    error.value = err.message
  }
}

function characterImportRunning(character) {
  if (!character?.metronCharacterId) return false
  return metronImportJobs.value.some(job => {
    return job.type === 'character'
      && job.metronId === character.metronCharacterId
      && (job.status === 'queued' || job.status === 'running')
  })
}

function newComic() {
  error.value = ''
  comicReturnTarget.value = null
  activeView.value = 'comics'
  viewMode.value = 'edit'
  selectedComic.value = null
  comicForm.value = emptyComic()
}

function editComic() {
  if (!selectedComic.value) return
  error.value = ''
  comicForm.value = { ...selectedComic.value }
  viewMode.value = 'edit'
}

async function saveComic() {
  saving.value = true
  error.value = ''

  try {
    const payload = comicPayload(comicForm.value)
    const detail = comicForm.value.id
      ? await updateComic(comicForm.value.id, payload)
      : await createComic(payload)

    selectedComic.value = detail
    comicForm.value = { ...detail }
    await loadData()
    viewMode.value = 'detail'
  } catch (err) {
    error.value = err.message
  } finally {
    saving.value = false
  }
}

async function deleteComic() {
  if (!comicForm.value.id || !confirm(`Delete ${comicForm.value.title}?`)) return

  saving.value = true
  error.value = ''

  try {
    await removeComic(comicForm.value.id)
    selectedComic.value = null
    comicForm.value = emptyComic()
    await loadData()
    viewMode.value = 'browse'
  } catch (err) {
    error.value = err.message
  } finally {
    saving.value = false
  }
}

async function toggleComicRead(comic) {
  if (!comic?.id || quickSavingComicID.value) return

  quickSavingComicID.value = comic.id
  error.value = ''

  try {
    const detail = await updateComicReadStatus(comic.id, !comic.read)
    applyComicReadState(detail)
    await refreshLists()
  } catch (err) {
    error.value = err.message
  } finally {
    quickSavingComicID.value = null
  }
}

function applyComicReadState(detail) {
  comics.value = comics.value.map(comic => (comic.id === detail.id ? { ...comic, read: detail.read } : comic))

  if (selectedComic.value?.id === detail.id) {
    selectedComic.value = { ...selectedComic.value, ...detail }
  }
  if (comicForm.value?.id === detail.id) {
    comicForm.value = { ...comicForm.value, read: detail.read }
  }
  if (selectedOrder.value) {
    const orderComics = selectedOrder.value.comics.map(comic => {
      return comic.id === detail.id ? { ...comic, read: detail.read } : comic
    })
    selectedOrder.value = {
      ...selectedOrder.value,
      comics: orderComics,
      progress: readingOrderProgress(orderComics),
    }
  }
  if (selectedCharacter.value) {
    const characterComics = selectedCharacter.value.comics.map(comic => {
      return comic.id === detail.id ? { ...comic, read: detail.read } : comic
    })
    selectedCharacter.value = {
      ...selectedCharacter.value,
      comics: characterComics,
      progress: readingOrderProgress(characterComics),
    }
  }
}

function applyComicDetailState(detail) {
  comics.value = comics.value.map(comic => (comic.id === detail.id ? { ...comic, ...detail } : comic))
  selectedComic.value = detail
  comicForm.value = { ...detail }

  if (selectedOrder.value) {
    const orderComics = selectedOrder.value.comics.map(comic => {
      return comic.id === detail.id ? { ...comic, ...detail, comment: comic.comment } : comic
    })
    selectedOrder.value = {
      ...selectedOrder.value,
      comics: orderComics,
      progress: readingOrderProgress(orderComics),
    }
  }
  if (selectedCharacter.value) {
    const characterComics = selectedCharacter.value.comics.map(comic => {
      return comic.id === detail.id ? { ...comic, ...detail } : comic
    })
    selectedCharacter.value = {
      ...selectedCharacter.value,
      comics: characterComics,
      progress: readingOrderProgress(characterComics),
    }
  }
}

function resetMetronMetadata() {
  metronMetadataOpen.value = false
  metronMetadataSearching.value = false
  metronMetadataApplyingID.value = null
  metronMetadataStatus.value = ''
  metronMetadataResults.value = []
}

async function searchSelectedComicMetron() {
  if (!selectedComic.value) return

  metronMetadataOpen.value = true
  metronMetadataSearching.value = true
  metronMetadataStatus.value = ''
  error.value = ''

  try {
    const { data } = await searchMetronComics({
      q: selectedComic.value.title,
      series: selectedComic.value.series,
      issue: selectedComic.value.issue,
    })
    metronMetadataResults.value = Array.isArray(data) ? data : []
    metronMetadataStatus.value = metronMetadataResults.value.length
      ? 'Choose the Metron issue that matches this comic.'
      : 'No Metron matches found.'
  } catch (err) {
    error.value = err.message
  } finally {
    metronMetadataSearching.value = false
  }
}

async function applyMetronMetadata(metronIssueID) {
  if (!selectedComic.value?.id || !metronIssueID) return

  metronMetadataApplyingID.value = metronIssueID
  metronMetadataStatus.value = 'Updating comic metadata from Metron...'
  error.value = ''

  try {
    const { data } = await updateComicFromMetron(selectedComic.value.id, metronIssueID)
    applyComicDetailState(data)
    await refreshLists()
    metronMetadataStatus.value = 'Metadata updated from Metron.'
    metronMetadataOpen.value = false
    metronMetadataResults.value = []
  } catch (err) {
    error.value = err.message
  } finally {
    metronMetadataApplyingID.value = null
  }
}

function readingOrderProgress(orderComics) {
  if (orderComics.length === 0) return 0
  const readCount = orderComics.filter(comic => comic.read).length
  return readCount / orderComics.length
}

async function openReadingOrder(order) {
  error.value = ''
  activeView.value = 'readingOrders'
  viewMode.value = 'detail'
  const detail = await getReadingOrder(order.id)
  selectedOrder.value = detail
  orderForm.value = readingOrderFormFromDetail(detail)
}

async function openAdjacentReadingOrder(offset) {
  const nextOrder = visibleOrders.value[currentOrderIndex.value + offset]
  if (nextOrder) {
    await openReadingOrder(nextOrder)
  }
}

async function toggleReadingOrderFavorite(order) {
  if (!order?.id || quickSavingOrderID.value) return

  quickSavingOrderID.value = order.id
  error.value = ''

  try {
    const detail = await updateReadingOrder(order.id, {
      name: order.name,
      description: order.description,
      favorite: !order.favorite,
    })
    applyReadingOrderFavoriteState(detail)
  } catch (err) {
    error.value = err.message
  } finally {
    quickSavingOrderID.value = null
  }
}

function applyReadingOrderFavoriteState(detail) {
  readingOrders.value = readingOrders.value.map(order => {
    return order.id === detail.id ? { ...order, favorite: detail.favorite } : order
  })

  if (selectedOrder.value?.id === detail.id) {
    selectedOrder.value = { ...selectedOrder.value, favorite: detail.favorite }
  }
  if (orderForm.value?.id === detail.id) {
    orderForm.value = { ...orderForm.value, favorite: detail.favorite }
  }
}

function newReadingOrder() {
  error.value = ''
  activeView.value = 'readingOrders'
  viewMode.value = 'edit'
  selectedOrder.value = null
  orderForm.value = emptyReadingOrder()
}

function editReadingOrder() {
  if (!selectedOrder.value) return
  error.value = ''
  orderForm.value = readingOrderFormFromDetail(selectedOrder.value)
  viewMode.value = 'edit'
}

async function saveReadingOrder() {
  saving.value = true
  error.value = ''

  try {
    const payload = readingOrderPayload(orderForm.value)
    const detail = orderForm.value.id
      ? await updateReadingOrder(orderForm.value.id, payload)
      : await createReadingOrder(payload)

    selectedOrder.value = await setReadingOrderComics(detail.id, readingOrderComicsPayload(orderForm.value))
    orderForm.value = readingOrderFormFromDetail(selectedOrder.value)
    await loadData()
    viewMode.value = 'detail'
  } catch (err) {
    error.value = err.message
  } finally {
    saving.value = false
  }
}

async function deleteReadingOrder() {
  if (!orderForm.value.id || !confirm(`Delete ${orderForm.value.name}?`)) return

  saving.value = true
  error.value = ''

  try {
    await removeReadingOrder(orderForm.value.id)
    selectedOrder.value = null
    orderForm.value = emptyReadingOrder()
    await loadData()
    viewMode.value = 'browse'
  } catch (err) {
    error.value = err.message
  } finally {
    saving.value = false
  }
}

function cancelEdit() {
  error.value = ''
  if (activeView.value === 'readingOrders' && selectedOrder.value) {
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
  await refreshLists()
  if (activeView.value === 'characters' && viewMode.value === 'detail' && selectedCharacter.value?.id) {
    selectedCharacter.value = await getCharacter(selectedCharacter.value.id)
  }
}

function trackMetronImportJob(job) {
	if (!job?.id) return
	metronImportMonitorOpen.value = true
	upsertMetronImportJob(job)
	pollMetronImportJob(job.id)
}

function upsertMetronImportJob(job) {
	const normalizedJob = normalizeMetronImportJob(job)
	const index = metronImportJobs.value.findIndex(item => item.id === job.id)
	if (index === -1) {
		metronImportJobs.value = [normalizedJob, ...metronImportJobs.value]
		return
	}

	metronImportJobs.value = metronImportJobs.value.map(item => {
		return item.id === job.id ? { ...item, ...normalizedJob, displayName: normalizedJob.displayName || item.displayName } : item
	})
}

function normalizeMetronImportJob(job) {
	if (job.status === 'failed' && String(job.message || '').toLowerCase().includes('context canceled')) {
		return { ...job, status: 'canceled', message: 'Import canceled.' }
	}
	return job
}

async function pollMetronImportJob(id) {
	clearMetronImportPoll(id)
	try {
		const job = normalizeMetronImportJob(await getMetronImportJob(id))
		upsertMetronImportJob(job)

    if (job.status === 'succeeded') {
      await handleMetronImported()
      return
    }
    if (job.status === 'failed') {
      showError(job.message || 'Metron import failed.')
      return
    }
    if (job.status === 'canceled') {
      return
    }

    metronImportPollTimers.set(id, window.setTimeout(() => pollMetronImportJob(id), 1500))
  } catch (err) {
    showError(err.message)
  }
}

function dismissMetronJob(id) {
  clearMetronImportPoll(id)
  metronImportJobs.value = metronImportJobs.value.filter(job => job.id !== id)
}

async function retryMetronJob(job) {
  try {
    const { data: nextJob } = await startMetronRetry(job)
    dismissMetronJob(job.id)
    trackMetronImportJob({ ...nextJob, displayName: job.displayName })
  } catch (err) {
    showError(err.message)
  }
}

function startMetronRetry(job) {
  if (job.type === 'comic') return importMetronComic(job.metronId)
  if (job.type === 'readingList') return importMetronReadingList(job.metronId)
  if (job.type === 'character') return importMetronCharacterAppearances(job.metronId)
  return importMetronSeries(job.metronId)
}

async function continueMetronJob(job) {
  try {
    const nextJob = await continueMetronImportJob(job.id)
    dismissMetronJob(job.id)
    trackMetronImportJob({ ...nextJob, displayName: job.displayName })
  } catch (err) {
    showError(err.message)
  }
}

async function cancelMetronJob(id) {
  clearMetronImportPoll(id)
  try {
    const job = await cancelMetronImportJob(id)
    upsertMetronImportJob(job)
    if (job.status === 'queued' || job.status === 'running') {
      metronImportPollTimers.set(id, window.setTimeout(() => pollMetronImportJob(id), 500))
    }
  } catch (err) {
    showError(err.message)
  }
}

function clearMetronImportPoll(id) {
  const timer = metronImportPollTimers.get(id)
  if (timer) {
    window.clearTimeout(timer)
    metronImportPollTimers.delete(id)
  }
}

function metronJobCanCancel(job) {
  return job.status === 'queued' || job.status === 'running'
}

function metronJobCanDismiss(job) {
  return job.status === 'succeeded' || job.status === 'failed' || job.status === 'canceled'
}

function metronJobCanContinue(job) {
  return job.status === 'canceled'
}

function metronJobProgressLabel(job) {
  if (!job.total) {
    if (job.status === 'queued') return 'Queued'
    if (job.status === 'running') return 'Preparing...'
    return job.status
  }
  return `${job.completed} of ${job.total}`
}

function metronJobProgressPercent(job) {
  if (!job.total) return job.status === 'succeeded' ? 100 : 0
  return Math.min(100, Math.round((job.completed / job.total) * 100))
}

function metronJobProgressIndeterminate(job) {
  return (job.status === 'queued' || job.status === 'running') && !job.total
}

function metronJobTitle(job) {
  const type = job.type === 'readingList' ? 'Reading list' : job.type === 'character' ? 'Character' : job.type
  return job.displayName ? `${type} import for ${job.displayName}` : `${type} import`
}

function metronJobMessage(job) {
  if (job.status === 'canceled') {
    return `${metronJobTitle(job)} was canceled.`
  }
  return job.message
}

onMounted(loadData)

onUnmounted(() => {
  metronImportPollTimers.forEach(timer => window.clearTimeout(timer))
  metronImportPollTimers.clear()
})
</script>

<template>
  <main class="app-shell">
    <AppSidebar
      :active-view="activeView"
      :comic-count="comics.length"
      :character-count="characters.length"
      :order-count="readingOrders.length"
      :loading="loading"
      @change-view="setView"
      @refresh="loadData"
    />

    <section class="content">
      <AppToolbar
        v-if="!isEditing && !isDetail && activeView !== 'comics'"
        :active-view="activeView"
        :search="search"
        :result-count="toolbarResultCount"
        :total-count="toolbarTotalCount"
        @update:search="search = $event"
      />

      <div v-if="error" class="toast error-toast" role="alert" aria-live="assertive">
        <span>{{ error }}</span>
        <button type="button" aria-label="Dismiss error" @click="clearError">Dismiss</button>
      </div>

      <div
        v-if="metronImportJobs.length"
        class="import-monitor"
        :class="{ collapsed: !metronImportMonitorOpen }"
        aria-live="polite"
      >
        <header>
          <button
            class="import-monitor-toggle"
            type="button"
            :aria-expanded="metronImportMonitorOpen"
            @click="metronImportMonitorOpen = !metronImportMonitorOpen"
          >
            <strong>Metron imports</strong>
            <small>{{ metronImportSummary }}</small>
          </button>
          <small v-if="metronImportInProgress && metronImportMonitorOpen">Running in background</small>
        </header>
        <div v-if="metronImportMonitorOpen" class="metron-jobs">
          <div v-for="job in metronImportJobs" :key="job.id" class="metron-job" :class="job.status">
            <span>
              <strong>{{ metronJobTitle(job) }}</strong>
              <small>{{ metronJobMessage(job) }}</small>
              <small>{{ metronJobProgressLabel(job) }}</small>
              <span class="job-progress" :class="{ indeterminate: metronJobProgressIndeterminate(job) }" aria-hidden="true">
                <span :style="{ width: `${metronJobProgressPercent(job)}%` }"></span>
              </span>
            </span>
            <span class="job-actions">
              <span class="status-pill">{{ job.status }}</span>
              <button
                v-if="job.status === 'failed'"
                class="icon-button compact-icon-button"
                type="button"
                aria-label="Retry import"
                title="Retry import"
                @click="retryMetronJob(job)"
              >
                <span aria-hidden="true" class="button-icon">↻</span>
              </button>
              <button v-if="metronJobCanContinue(job)" class="ghost-button" type="button" @click="continueMetronJob(job)">
                Continue
              </button>
              <button v-if="metronJobCanCancel(job)" class="ghost-button" type="button" @click="cancelMetronJob(job.id)">
                Cancel
              </button>
              <button
                v-if="metronJobCanDismiss(job)"
                class="icon-button compact-icon-button"
                type="button"
                aria-label="Dismiss import"
                title="Dismiss import"
                @click="dismissMetronJob(job.id)"
              >
                <span aria-hidden="true" class="button-icon">×</span>
              </button>
            </span>
          </div>
        </div>
      </div>

      <p v-if="loading" class="muted">Loading...</p>

      <MetronImport
        v-if="!loading && activeView === 'metron'"
        :import-jobs="metronImportJobs"
        @imported="handleMetronImported"
        @error="showError"
        @job-started="trackMetronImportJob"
      />

      <div v-else-if="!loading && activeView === 'readingOrders' && isEditing" class="editor-view">
        <header class="editor-header">
          <button class="secondary-button" type="button" @click="cancelEdit">Back</button>
          <div>
            <p class="eyebrow">Reading Order</p>
            <h2>{{ orderForm.id ? 'Edit reading order' : 'New reading order' }}</h2>
          </div>
          <div class="editor-actions">
            <button v-if="orderForm.id" type="button" class="danger-button" :disabled="saving" @click="deleteReadingOrder">
              Delete
            </button>
            <button class="primary-button" type="submit" form="reading-order-editor-form" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save Reading Order' }}
            </button>
          </div>
        </header>

        <article class="detail-panel">
          <ReadingOrderEditor
            v-model:form="orderForm"
            form-id="reading-order-editor-form"
            :selected-order="selectedOrder"
            :comics="comics"
            :saving="saving"
            @save="saveReadingOrder"
          />
        </article>
      </div>

      <div v-else-if="activeView === 'readingOrders' && isDetail" class="detail-view">
        <header class="detail-nav">
          <button class="secondary-button" type="button" @click="backToBrowse">Back</button>
        </header>

        <article class="detail-panel">
          <div v-if="selectedOrder" class="read-only-detail">
            <header class="panel-header">
              <div>
                <p class="eyebrow">Reading Order</p>
                <h3>{{ selectedOrder.name }}</h3>
              </div>
            </header>

            <p class="detail-description">{{ selectedOrder.description || 'No description' }}</p>
            <div class="progress-meter" aria-label="Reading order progress">
              <span :style="{ width: formatProgress(selectedOrder.progress) }"></span>
            </div>
            <div class="metadata-grid">
              <span>
                <strong>{{ formatProgress(selectedOrder.progress) }}</strong>
                <small>Progress</small>
              </span>
              <span>
                <strong>{{ selectedOrder.favorite ? 'Yes' : 'No' }}</strong>
                <small>Favorite</small>
              </span>
              <span>
                <strong>{{ selectedOrder.comics.length }}</strong>
                <small>Comics</small>
              </span>
            </div>

            <ComicListView
              class="preview-list"
              title="Comics"
              :comics="selectedOrder.comics"
              :selected-comic-id="selectedComic?.id"
              :quick-saving-comic-id="quickSavingComicID"
              show-comment
              empty-message="No comics in this reading order yet."
              filtered-empty-message="No comics match these filters."
              @open-comic="openOrderComic"
              @toggle-read="toggleComicRead"
            />
          </div>
          <p v-else class="empty-state">Select a reading order to view it.</p>
        </article>
      </div>

      <div v-else-if="activeView === 'readingOrders'" class="browse-view">
        <div class="list-pane">
          <div class="overview-strip">
            <span>
              <strong>{{ readingOrders.length }}</strong>
              <small>Orders</small>
            </span>
            <span>
              <strong>{{ favoriteOrderCount }}</strong>
              <small>Favorites</small>
            </span>
            <span>
              <strong>{{ comics.length }}</strong>
              <small>Comics</small>
            </span>
          </div>
          <div class="browse-controls">
            <div class="filter-tabs" role="tablist" aria-label="Reading order filter">
              <button
                type="button"
                :class="{ active: readingOrderFilter === 'all' }"
                role="tab"
                :aria-selected="readingOrderFilter === 'all'"
                @click="readingOrderFilter = 'all'"
              >
                All
              </button>
              <button
                type="button"
                :class="{ active: readingOrderFilter === 'favorites' }"
                role="tab"
                :aria-selected="readingOrderFilter === 'favorites'"
                @click="readingOrderFilter = 'favorites'"
              >
                Favorites
              </button>
            </div>
            <button
              class="primary-button icon-text-button"
              type="button"
              aria-label="New order"
              title="New order"
              @click="newReadingOrder"
            >
              <span aria-hidden="true" class="button-icon">+</span>
            </button>
          </div>
          <div v-if="visibleOrders.length" class="list">
            <div
              v-for="order in visibleOrders"
              :key="order.id"
              class="row order-row"
              :class="{ selected: selectedOrder?.id === order.id }"
            >
              <span class="order-row-content">
                <button class="row-main" type="button" @click="openReadingOrder(order)">
                  <strong>{{ order.name }}</strong>
                  <small>{{ order.description || 'No description' }}</small>
                </button>
                <button
                  type="button"
                  class="favorite-toggle"
                  :class="{ active: order.favorite }"
                  :disabled="quickSavingOrderID === order.id"
                  :aria-label="order.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  :title="order.favorite ? 'Remove from favorites' : 'Add to favorites'"
                  @click="toggleReadingOrderFavorite(order)"
                >
                  <span aria-hidden="true">{{ order.favorite ? '★' : '☆' }}</span>
                </button>
              </span>
              <span class="row-progress" aria-label="Reading order progress">
                <span :style="{ width: formatProgress(order.progress) }"></span>
              </span>
            </div>
          </div>
          <div v-else class="empty-state">
            {{
              searchTerm
                ? 'No reading orders match your search.'
                : readingOrderFilter === 'favorites'
                  ? 'No favorite reading orders yet.'
                  : 'No reading orders yet.'
            }}
            <button
              v-if="!searchTerm && readingOrderFilter === 'favorites' && readingOrders.length"
              class="secondary-button"
              type="button"
              @click="readingOrderFilter = 'all'"
            >
              Show all orders
            </button>
            <button v-else-if="!searchTerm" class="secondary-button" type="button" @click="newReadingOrder">
              <span aria-hidden="true" class="button-icon">+</span>
              Create the first order
            </button>
          </div>
        </div>
      </div>

      <div v-else-if="activeView === 'characters' && isDetail" class="detail-view">
        <header class="detail-nav">
          <button class="secondary-button" type="button" @click="backToBrowse">Back</button>
          <div class="detail-nav-actions">
            <button
              v-if="selectedCharacter"
              type="button"
              class="favorite-toggle detail-favorite-toggle"
              :class="{ active: selectedCharacter.favorite }"
              :disabled="quickSavingCharacterID === selectedCharacter.id"
              :aria-label="selectedCharacter.favorite ? 'Remove from favorites' : 'Add to favorites'"
              :title="selectedCharacter.favorite ? 'Remove from favorites' : 'Add to favorites'"
              @click="toggleCharacterFavorite(selectedCharacter)"
            >
              <span aria-hidden="true">{{ selectedCharacter.favorite ? '★' : '☆' }}</span>
            </button>
            <button
              v-if="selectedCharacter?.metronCharacterId"
              class="primary-button"
              type="button"
              :disabled="characterImportRunning(selectedCharacter)"
              @click="importSelectedCharacterAppearances"
            >
              {{ characterImportRunning(selectedCharacter) ? 'Importing...' : 'Import from Metron' }}
            </button>
          </div>
        </header>

        <article class="detail-panel">
          <div v-if="selectedCharacter" class="read-only-detail">
            <header class="panel-header">
              <div>
                <p class="eyebrow">Character</p>
                <h3>{{ selectedCharacter.name }}</h3>
              </div>
            </header>

            <div v-if="selectedCharacter.image" class="character-portrait">
              <img :src="assetURL(selectedCharacter.image)" :alt="`${selectedCharacter.name} portrait`" loading="lazy" />
            </div>

            <div class="metadata-grid">
              <span>
                <strong>{{ characterProgress(selectedCharacter) }}</strong>
                <small>Progress</small>
              </span>
              <span>
                <strong>{{ selectedCharacter.appearanceCount }}</strong>
                <small>Appearances</small>
              </span>
              <span>
                <strong>{{ selectedCharacter.aliases?.length || 0 }}</strong>
                <small>Aliases</small>
              </span>
              <span>
                <strong>{{ selectedCharacter.favorite ? 'Yes' : 'No' }}</strong>
                <small>Favorite</small>
              </span>
            </div>
            <div class="progress-meter" aria-label="Character read progress">
              <span :style="{ width: characterProgress(selectedCharacter) }"></span>
            </div>

            <div v-if="selectedCharacter.aliases?.length" class="alias-list">
              <span v-for="alias in selectedCharacter.aliases" :key="alias">{{ alias }}</span>
            </div>

            <p class="detail-description">{{ selectedCharacter.description || 'No description' }}</p>

            <ComicListView
              class="preview-list"
              title="Appearances"
              :comics="selectedCharacter.comics || []"
              :selected-comic-id="selectedComic?.id"
              :quick-saving-comic-id="quickSavingComicID"
              empty-message="No appearances saved yet."
              filtered-empty-message="No appearances match these filters."
              @open-comic="openComic"
              @toggle-read="toggleComicRead"
            />
          </div>
          <p v-else class="empty-state">Select a character to view appearances.</p>
        </article>
      </div>

      <div v-else-if="activeView === 'characters'" class="browse-view">
        <div class="list-pane">
          <div class="overview-strip">
            <span>
              <strong>{{ characters.length }}</strong>
              <small>Characters</small>
            </span>
            <span>
              <strong>{{ favoriteCharacterCount }}</strong>
              <small>Favorites</small>
            </span>
            <span>
              <strong>{{ characters.reduce((total, character) => total + (character.appearanceCount || 0), 0) }}</strong>
              <small>Appearances</small>
            </span>
          </div>
          <div v-if="visibleCharacters.length" class="sectioned-list">
            <section v-for="section in characterBrowseSections" :key="section.key" class="list-section">
              <div class="list-section-header">
                <p class="eyebrow">{{ section.title }}</p>
                <small>{{ section.characters.length }}</small>
              </div>
              <div class="list">
                <div
                  v-for="character in section.characters"
                  :key="character.id"
                  class="row character-row"
                  :class="{ selected: selectedCharacter?.id === character.id }"
                >
                  <span class="order-row-content">
                    <button class="row-main character-row-main" type="button" @click="openCharacter(character)">
                      <span v-if="character.image" class="character-list-avatar" aria-hidden="true">
                        <img :src="assetURL(character.image)" alt="" loading="lazy" />
                      </span>
                      <span>
                        <strong>{{ character.name }}</strong>
                        <small v-if="character.aliases?.length">{{ character.aliases.join(', ') }}</small>
                        <small v-else>No aliases saved</small>
                      </span>
                    </button>
                    <button
                      type="button"
                      class="favorite-toggle"
                      :class="{ active: character.favorite }"
                      :disabled="quickSavingCharacterID === character.id"
                      :aria-label="character.favorite ? 'Remove from favorites' : 'Add to favorites'"
                      :title="character.favorite ? 'Remove from favorites' : 'Add to favorites'"
                      @click="toggleCharacterFavorite(character)"
                    >
                      <span aria-hidden="true">{{ character.favorite ? '★' : '☆' }}</span>
                    </button>
                  </span>
                  <span class="row-meta">
                    <span class="status-pill">{{ character.appearanceCount }} appearances</span>
                    <span class="status-pill">{{ characterProgress(character) }}</span>
                  </span>
                  <span class="row-progress" aria-label="Character read progress">
                    <span :style="{ width: characterProgress(character) }"></span>
                  </span>
                </div>
              </div>
            </section>
          </div>
          <div v-else class="empty-state">
            {{
              searchTerm
                ? 'No characters match your search.'
                : characterFilter === 'favorites'
                  ? 'No favorite characters yet.'
                  : 'No characters imported yet.'
            }}
            <button
              v-if="!searchTerm && characterFilter === 'favorites' && characters.length"
              class="secondary-button"
              type="button"
              @click="characterFilter = 'all'"
            >
              Show all characters
            </button>
          </div>
        </div>
      </div>

      <div v-else-if="activeView === 'comics' && isEditing" class="editor-view">
        <header class="editor-header">
          <button class="secondary-button" type="button" @click="cancelEdit">Back</button>
          <div>
            <p class="eyebrow">Comic</p>
            <h2>{{ comicForm.id ? 'Edit comic' : 'New comic' }}</h2>
          </div>
          <div class="editor-actions">
            <button v-if="comicForm.id" type="button" class="danger-button" :disabled="saving" @click="deleteComic">
              Delete
            </button>
            <button class="primary-button" type="submit" form="comic-editor-form" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save Comic' }}
            </button>
          </div>
        </header>

        <article class="detail-panel">
          <ComicEditor
            v-model:form="comicForm"
            form-id="comic-editor-form"
            :selected-comic="selectedComic"
            :saving="saving"
            @save="saveComic"
          />
        </article>
      </div>

      <div v-else-if="activeView === 'comics' && isDetail" class="detail-view">
        <header class="detail-nav">
          <button class="secondary-button" type="button" @click="backToBrowse">Back</button>
          <div class="detail-nav-actions">
            <button
              v-if="selectedComic"
              class="secondary-button"
              type="button"
              :disabled="metronMetadataSearching || metronMetadataApplyingID !== null"
              @click="selectedComic?.metronIssueId ? applyMetronMetadata(selectedComic.metronIssueId) : searchSelectedComicMetron()"
            >
              {{ selectedComic?.metronIssueId ? 'Refresh Metron' : 'Get Metron metadata' }}
            </button>
            <button
              v-if="selectedComic"
              class="read-toggle-button large"
              type="button"
              :disabled="quickSavingComicID === selectedComic.id"
              @click="toggleComicRead(selectedComic)"
            >
              {{ selectedComic.read ? 'Mark unread' : 'Mark read' }}
            </button>
            <button v-if="selectedComic" class="primary-button" type="button" @click="editComic">Edit</button>
          </div>
        </header>

        <article class="detail-panel">
          <div v-if="selectedComic" class="read-only-detail">
            <header class="panel-header">
              <div>
                <p class="eyebrow">Comic</p>
                <h3>{{ selectedComic.title }}</h3>
              </div>
            </header>

            <div v-if="selectedComic.coverImage" class="cover-preview">
              <img :src="assetURL(selectedComic.coverImage)" :alt="`${selectedComic.title} cover`" loading="lazy" />
            </div>

            <div class="metadata-grid">
              <span>
                <strong>{{ selectedComic.series }}{{ selectedComic.seriesYear ? ` (${selectedComic.seriesYear})` : '' }} #{{ selectedComic.issue }}</strong>
                <small>Series</small>
              </span>
              <span>
                <strong>{{ selectedComic.publisher || 'Unknown' }}</strong>
                <small>Publisher</small>
              </span>
              <span>
                <strong>
                  <span class="read-state-pill" :class="{ read: selectedComic.read, unread: !selectedComic.read }">
                    {{ selectedComic.read ? 'Read' : 'Unread' }}
                  </span>
                </strong>
                <small>Status</small>
              </span>
              <span>
                <strong>{{ selectedComic.coverDate || 'Unknown' }}</strong>
                <small>Cover Date</small>
              </span>
            </div>

            <div v-if="metronMetadataOpen || metronMetadataStatus" class="metron-metadata-panel">
              <header class="section-title">
                <div>
                  <p class="eyebrow">Metron</p>
                  <h4>Metadata matches</h4>
                </div>
                <button
                  v-if="metronMetadataOpen || metronMetadataStatus"
                  class="ghost-button"
                  type="button"
                  @click="resetMetronMetadata"
                >
                  Close
                </button>
              </header>
              <p v-if="metronMetadataSearching" class="muted">Searching Metron...</p>
              <p v-else-if="metronMetadataStatus" class="muted">{{ metronMetadataStatus }}</p>
              <div v-if="metronMetadataResults.length" class="list">
                <button
                  v-for="issue in metronMetadataResults"
                  :key="issue.id"
                  class="row"
                  type="button"
                  :disabled="metronMetadataApplyingID !== null"
                  @click="applyMetronMetadata(issue.id)"
                >
                  <span>
                    <strong>{{ issue.series }} #{{ issue.number || issue.issue }}: {{ issue.title || 'Untitled issue' }}</strong>
                    <small>{{ issue.publisher || 'Unknown publisher' }} · {{ issue.coverDate || 'Unknown date' }}</small>
                  </span>
                  <span class="status-pill">
                    {{ metronMetadataApplyingID === issue.id ? 'Applying...' : 'Apply' }}
                  </span>
                </button>
              </div>
            </div>

            <p class="detail-description">{{ selectedComic.description || 'No description' }}</p>

            <div v-if="selectedComic.characters?.length" class="preview-list">
              <p class="eyebrow">Characters</p>
              <div class="alias-list">
                <button
                  v-for="character in selectedComic.characters"
                  :key="character.id"
                  type="button"
                  @click="openCharacter(character)"
                >
                  {{ character.name }}
                </button>
              </div>
            </div>

            <div v-if="selectedComic.readingOrders?.length" class="preview-list">
              <p class="eyebrow">Reading Orders</p>
              <ul>
                <li v-for="order in selectedComic.readingOrders" :key="order.id">
                  {{ order.name }}
                </li>
              </ul>
            </div>
          </div>
          <p v-else class="empty-state">Select a comic to view it.</p>
        </article>
      </div>

      <div v-else class="browse-view">
        <div class="list-pane">
          <div class="overview-strip">
            <span>
              <strong>{{ comics.length }}</strong>
              <small>Comics</small>
            </span>
            <span>
              <strong>{{ unreadComicCount }}</strong>
              <small>Unread</small>
            </span>
            <span>
              <strong>{{ comics.length - unreadComicCount }}</strong>
              <small>Read</small>
            </span>
          </div>
          <ComicListView
            title="Comics"
            :comics="comics"
            :selected-comic-id="selectedComic?.id"
            :quick-saving-comic-id="quickSavingComicID"
            show-new-button
            show-cover
            empty-message="No comics yet."
            filtered-empty-message="No comics match these filters."
            @new-comic="newComic"
            @open-comic="openComic"
            @toggle-read="toggleComicRead"
          />
        </div>
      </div>
    </section>
  </main>
</template>
