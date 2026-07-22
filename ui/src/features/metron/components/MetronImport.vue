<script setup>
import { computed, ref } from 'vue'
import {
  getMetronReadingList,
  importAllMetronReadingLists,
  importMetronCharacterAppearances,
  importMetronComic,
  importMetronReadingList,
  importMetronSeries,
  importMetronArc,
  searchMetronCharacters,
  searchMetronComics,
  searchMetronReadingLists,
  searchMetronSeries,
  searchMetronArcs,
} from '@/api/client.js'
import MetronImportOptions from '@/features/metron/components/MetronImportOptions.vue'
import MetronReadingListDialog from '@/features/metron/components/MetronReadingListDialog.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import MetronSearchResults from '@/features/metron/components/MetronSearchResults.vue'

const props = defineProps({
  importJobs: {
    type: Array,
    default: () => [],
  },
  metronQuota: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['imported', 'error', 'job-started', 'quota-updated'])

const activeSearch = ref('comics')
const query = ref('')
const series = ref('')
const issue = ref('')
const seriesYearBegan = ref('')
const seriesVolume = ref('')
const searching = ref(false)
const importingKey = ref('')
const importStatus = ref('')
const importMode = ref('quick')
const fullImportData = ref(['comics', 'series', 'arcs', 'characters'])
const comicResults = ref([])
const characterResults = ref([])
const readingListResults = ref([])
const seriesResults = ref([])
const arcResults = ref([])
const selectedReadingList = ref(null)
const readingListDetailOpen = ref(false)
const readingListDetailLoading = ref(false)
const readingListDetailStatus = ref('')
const importingAllReadingLists = ref(false)

const busy = computed(() => searching.value)
const searchLabel = computed(() => {
  if (searching.value) return 'Searching...'
  if (activeSearch.value === 'characters') return 'Search Characters'
  if (activeSearch.value === 'readingLists') return 'Search Reading Lists'
  if (activeSearch.value === 'arcs') return 'Search Arcs'
  return `Search ${activeSearch.value}`
})
const quotaKnown = computed(() => Boolean(props.metronQuota?.known))
const quotaSummary = computed(() => {
  if (!quotaKnown.value) return 'Quota appears after the first Metron response'
  const burst = quotaText('Burst', props.metronQuota.burstUsed, props.metronQuota.burstLimit)
  const sustained = quotaText(
    'Sustained',
    props.metronQuota.sustainedUsed,
    props.metronQuota.sustainedLimit,
  )
  return [burst, sustained].filter(Boolean).join(' · ')
})
const quotaReset = computed(() => {
  if (!quotaKnown.value) return ''
  const resets = [props.metronQuota.burstReset, props.metronQuota.sustainedReset].filter(Boolean)
  if (resets.length === 0) return ''
  const nextReset = Math.max(...resets)
  return new Date(nextReset * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
})
const importOptions = computed(() => {
  if (importMode.value === 'full') {
    return { mode: 'full', fullData: normalizedFullImportData.value }
  }
  return { mode: 'quick' }
})
const selectedReadingListImporting = computed(() =>
  Boolean(selectedReadingList.value && rowImporting('readingList', selectedReadingList.value.id)),
)
const selectedReadingListSummary = computed(() =>
  selectedReadingList.value ? readingListSummary(selectedReadingList.value) : '',
)
const normalizedFullImportData = computed(() => {
  if (importMode.value !== 'full') return []
  const selected = new Set(fullImportData.value)
  if (selected.has('series') || selected.has('arcs') || selected.has('characters')) {
    selected.add('comics')
  }
  return ['comics', 'series', 'arcs', 'characters'].filter((item) => selected.has(item))
})

async function search() {
  searching.value = true
  importStatus.value = ''
  try {
    if (activeSearch.value === 'comics') {
      const { data, rateLimit: nextRateLimit } = await searchMetronComics({
        q: query.value,
        series: series.value,
        issue: issue.value,
      })
      updateRateLimit(nextRateLimit)
      comicResults.value = Array.isArray(data) ? data : []
      return
    }

    if (activeSearch.value === 'readingLists') {
      const { data, rateLimit: nextRateLimit } = await searchMetronReadingLists({ q: query.value })
      updateRateLimit(nextRateLimit)
      readingListResults.value = Array.isArray(data) ? data : []
      return
    }

    if (activeSearch.value === 'characters') {
      const { data, rateLimit: nextRateLimit } = await searchMetronCharacters({ q: query.value })
      updateRateLimit(nextRateLimit)
      characterResults.value = Array.isArray(data) ? data : []
      return
    }

    if (activeSearch.value === 'arcs') {
      const { data, rateLimit: nextRateLimit } = await searchMetronArcs({ q: query.value })
      updateRateLimit(nextRateLimit)
      arcResults.value = Array.isArray(data) ? data : []
      return
    }

    const { data, rateLimit: nextRateLimit } = await searchMetronSeries({
      q: query.value || series.value,
      year_began: seriesYearBegan.value,
      volume: seriesVolume.value,
    })
    updateRateLimit(nextRateLimit)
    seriesResults.value = Array.isArray(data) ? data : []
  } catch (err) {
    updateRateLimit(err.rateLimit)
    emit('error', err.message)
  } finally {
    searching.value = false
  }
}

async function importComic(comic) {
  const id = comic.id
  importingKey.value = `comic:${id}`
  importStatus.value = 'Comic import started in the background.'
  try {
    const { data: job, rateLimit: nextRateLimit } = await importMetronComic(id, importOptions.value)
    updateRateLimit(nextRateLimit)
    trackJob(job, comic.title || `${comic.series} #${comic.number || comic.issue}`)
  } catch (err) {
    updateRateLimit(err.rateLimit)
    emit('error', err.message)
  } finally {
    importingKey.value = ''
  }
}

async function importReadingList(list) {
  const id = list.id
  importingKey.value = `readingList:${id}`
  importStatus.value = 'Reading list import started in the background.'
  try {
    const { data: job, rateLimit: nextRateLimit } = await importMetronReadingList(
      id,
      importOptions.value,
    )
    updateRateLimit(nextRateLimit)
    trackJob(job, list.name || 'Untitled reading list')
  } catch (err) {
    updateRateLimit(err.rateLimit)
    emit('error', err.message)
  } finally {
    importingKey.value = ''
  }
}

async function importAllReadingLists() {
  const confirmed = window.confirm(
    'Pulling every reading list from Metron may take a long time and make many API requests. Continue?',
  )
  if (!confirmed) return

  importingAllReadingLists.value = true
  importStatus.value = 'Importing all Metron reading lists in the background.'
  try {
    const { data: job, rateLimit: nextRateLimit } = await importAllMetronReadingLists(
      importOptions.value,
    )
    updateRateLimit(nextRateLimit)
    trackJob(job, 'All reading lists')
  } catch (err) {
    updateRateLimit(err.rateLimit)
    emit('error', err.message)
  } finally {
    importingAllReadingLists.value = false
  }
}

async function openReadingList(list) {
  selectedReadingList.value = list
  readingListDetailOpen.value = true
  readingListDetailLoading.value = true
  readingListDetailStatus.value = ''
  try {
    const { data, rateLimit: nextRateLimit } = await getMetronReadingList(list.id)
    updateRateLimit(nextRateLimit)
    selectedReadingList.value = { ...list, ...(data || {}) }
  } catch (err) {
    updateRateLimit(err.rateLimit)
    readingListDetailStatus.value = err.message
  } finally {
    readingListDetailLoading.value = false
  }
}

function closeReadingListDetail() {
  readingListDetailOpen.value = false
  readingListDetailLoading.value = false
  readingListDetailStatus.value = ''
  selectedReadingList.value = null
}

function importSelectedReadingList() {
  const list = selectedReadingList.value
  if (!list) return

  closeReadingListDetail()
  importReadingList(list)
}

async function importSeries(item) {
  const id = item.id
  importingKey.value = `series:${id}`
  importStatus.value = 'Series import started in the background.'
  try {
    const { data: job, rateLimit: nextRateLimit } = await importMetronSeries(
      id,
      importOptions.value,
    )
    updateRateLimit(nextRateLimit)
    trackJob(job, item.name || 'Untitled series')
  } catch (err) {
    updateRateLimit(err.rateLimit)
    emit('error', err.message)
  } finally {
    importingKey.value = ''
  }
}

async function importCharacter(character) {
  const id = character.id
  importingKey.value = `character:${id}`
  importStatus.value = 'Character import started in the background.'
  try {
    const { data: job, rateLimit: nextRateLimit } = await importMetronCharacterAppearances(
      id,
      importOptions.value,
    )
    updateRateLimit(nextRateLimit)
    trackJob(job, character.name || 'Untitled character')
  } catch (err) {
    updateRateLimit(err.rateLimit)
    emit('error', err.message)
  } finally {
    importingKey.value = ''
  }
}

async function importArc(arc) {
  const id = arc.id
  importingKey.value = `arc:${id}`
  importStatus.value = 'Arc import started in the background.'
  try {
    const { data: job, rateLimit: nextRateLimit } = await importMetronArc(id, importOptions.value)
    updateRateLimit(nextRateLimit)
    trackJob(job, arc.name || 'Untitled arc')
  } catch (err) {
    updateRateLimit(err.rateLimit)
    emit('error', err.message)
  } finally {
    importingKey.value = ''
  }
}

function setSearchMode(mode) {
  activeSearch.value = mode
  if (mode !== 'comics') {
    series.value = ''
    issue.value = ''
  }
  if (mode !== 'series') {
    seriesYearBegan.value = ''
    seriesVolume.value = ''
  }
}

function updateRateLimit(nextRateLimit) {
  if (nextRateLimit) {
    emit('quota-updated', quotaFromRateLimit(nextRateLimit))
  }
}

function trackJob(job, displayName) {
  if (!job?.id) return
  emit('job-started', { ...job, displayName })
}

function quotaText(label, used, limit) {
  if (limit === null || limit === undefined) return ''
  return `${label} ${used ?? 0}/${limit} used`
}

function quotaFromRateLimit(nextRateLimit) {
  const quota = {
    burstLimit: nextRateLimit.burstLimit,
    burstRemaining: nextRateLimit.burstRemaining,
    burstUsed: usedQuota(nextRateLimit.burstLimit, nextRateLimit.burstRemaining),
    burstReset: nextRateLimit.burstReset,
    sustainedLimit: nextRateLimit.sustainedLimit,
    sustainedRemaining: nextRateLimit.sustainedRemaining,
    sustainedUsed: usedQuota(nextRateLimit.sustainedLimit, nextRateLimit.sustainedRemaining),
    sustainedReset: nextRateLimit.sustainedReset,
    known: true,
  }
  return quota
}

function usedQuota(limit, remaining) {
  if (limit === null || limit === undefined || remaining === null || remaining === undefined)
    return 0
  return Math.max(0, limit - remaining)
}

function rowImporting(type, id) {
  return (
    importingKey.value === `${type}:${id}` ||
    props.importJobs.some((job) => {
      return job.type === type && job.metronId === id && isActiveJob(job)
    })
  )
}

function isActiveJob(job) {
  return job.status === 'queued' || job.status === 'running' || job.status === 'canceling'
}

function setImportMode(mode) {
  importMode.value = mode
}

function toggleFullImportData(value, checked) {
  const selected = new Set(fullImportData.value)
  if (checked) {
    selected.add(value)
  } else {
    selected.delete(value)
  }
  if (selected.has('series') || selected.has('arcs') || selected.has('characters')) {
    selected.add('comics')
  }
  if (selected.size === 0) {
    selected.add('comics')
  }
  fullImportData.value = ['comics', 'series', 'arcs', 'characters'].filter((item) =>
    selected.has(item),
  )
}

function readingListSummary(list) {
  return [
    list.listType || 'Reading list',
    list.user?.username ? `by ${list.user.username}` : '',
    list.attributionSource ? `via ${list.attributionSource}` : '',
    list.ratingCount ? `${list.averageRating || 0} avg from ${list.ratingCount} ratings` : '',
    list.modified ? `Modified ${formatDate(list.modified)}` : '',
  ]
    .filter(Boolean)
    .join(' · ')
}

function formatDate(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString([], { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<template>
  <div class="metron-view grid gap-5 pbs-[16px] down-mobile:gap-3.5">
    <div class="metron-modes" role="tablist" aria-label="Metron search type">
      <!-- Native buttons: search modes are a stateful tablist rather than form actions. -->
      <button
        type="button"
        class="search-mode-button"
        :class="{ active: activeSearch === 'comics' }"
        role="tab"
        :aria-selected="activeSearch === 'comics'"
        @click="setSearchMode('comics')"
      >
        Comics
      </button>
      <button
        type="button"
        class="search-mode-button"
        :class="{ active: activeSearch === 'readingLists' }"
        role="tab"
        :aria-selected="activeSearch === 'readingLists'"
        @click="setSearchMode('readingLists')"
      >
        Reading Lists
      </button>
      <button
        type="button"
        class="search-mode-button"
        :class="{ active: activeSearch === 'series' }"
        role="tab"
        :aria-selected="activeSearch === 'series'"
        @click="setSearchMode('series')"
      >
        Series
      </button>
      <button
        type="button"
        class="search-mode-button"
        :class="{ active: activeSearch === 'characters' }"
        role="tab"
        :aria-selected="activeSearch === 'characters'"
        @click="setSearchMode('characters')"
      >
        Characters
      </button>
      <button
        type="button"
        class="search-mode-button"
        :class="{ active: activeSearch === 'arcs' }"
        role="tab"
        :aria-selected="activeSearch === 'arcs'"
        @click="setSearchMode('arcs')"
      >
        Arcs
      </button>
    </div>

    <div class="metron-quota-strip">
      <span>
        <strong>Metron quota</strong>
        <small>{{ quotaSummary }}</small>
      </span>
      <small v-if="quotaReset">Resets around {{ quotaReset }}</small>
    </div>

    <MetronImportOptions
      :import-mode="importMode"
      :selected-data="fullImportData"
      @update:import-mode="setImportMode"
      @toggle-data="toggleFullImportData"
    />

    <form class="metron-search" @submit.prevent="search">
      <label>
        {{
          activeSearch === 'comics'
            ? 'Search'
            : activeSearch === 'readingLists'
              ? 'Reading List'
              : activeSearch === 'characters'
                ? 'Character'
                : activeSearch === 'series'
                  ? 'Series'
                  : 'Arc'
        }}
        <BaseTextInput v-model="query" placeholder="Batman, X-Men, Civil War" />
      </label>
      <label v-if="activeSearch === 'comics'">
        Series
        <BaseTextInput v-model="series" placeholder="Optional series filter" />
      </label>
      <label v-if="activeSearch === 'comics'">
        Issue
        <BaseTextInput v-model="issue" />
      </label>
      <label v-if="activeSearch === 'series'">
        Year began
        <BaseTextInput v-model="seriesYearBegan" inputmode="numeric" placeholder="Optional year" />
      </label>
      <label v-if="activeSearch === 'series'">
        Volume
        <BaseTextInput v-model="seriesVolume" inputmode="numeric" placeholder="Optional volume" />
      </label>
      <BaseButton variant="primary" type="submit" :disabled="busy">
        {{ searchLabel }}
      </BaseButton>
    </form>

    <div v-if="importStatus" class="metron-status">
      <span>{{ importStatus }}</span>
    </div>

    <MetronSearchResults
      :active-search="activeSearch"
      :searching="searching"
      :comic-results="comicResults"
      :reading-list-results="readingListResults"
      :series-results="seriesResults"
      :arc-results="arcResults"
      :character-results="characterResults"
      :importing-all-reading-lists="importingAllReadingLists"
      :is-importing="rowImporting"
      @import-comic="importComic"
      @open-reading-list="openReadingList"
      @import-all-reading-lists="importAllReadingLists"
      @import-series="importSeries"
      @import-arc="importArc"
      @import-character="importCharacter"
    />

    <MetronReadingListDialog
      v-if="readingListDetailOpen"
      :reading-list="selectedReadingList"
      :loading="readingListDetailLoading"
      :error="readingListDetailStatus"
      :importing="selectedReadingListImporting"
      :summary="selectedReadingListSummary"
      @close="closeReadingListDetail"
      @import="importSelectedReadingList"
    />
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.metron-modes {
  @apply inline-grid grid-cols-5 gap-1.5 w-[min(680px,100%)] down-mobile:grid-cols-2 down-mobile:w-full;
}

.metron-quota-strip {
  @apply flex items-baseline justify-between gap-3 border border-line-strong rounded bg-surface-soft text-label py-2.5 px-3 text-sm font-bold [&_span]:flex [&_span]:items-baseline [&_span]:gap-2.5 [&_span]:min-w-0 [&_small]:text-muted [&_small]:overflow-hidden [&_small]:text-ellipsis [&_small]:whitespace-nowrap down-mobile:items-stretch down-mobile:flex-col down-mobile:[&_span]:flex-wrap;
}

.metron-search {
  @apply grid grid-cols-[repeat(auto-fit,minmax(160px,1fr))] gap-3.5 items-end [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:text-sm [&_label]:font-bold down-tablet:grid-cols-1 down-mobile:[&_button]:w-full;
}

.metron-status {
  @apply flex items-center flex-wrap gap-y-2 gap-x-3 border border-line-strong rounded bg-surface-soft text-label py-2.5 px-3 text-sm font-bold;
}

.search-mode-button {
  @apply min-h-10 rounded border border-line-strong bg-surface px-3 py-2.5 text-control;
}

.search-mode-button.active {
  @apply border-primary bg-primary text-white;
}
</style>
