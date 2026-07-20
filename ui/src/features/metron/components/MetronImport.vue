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

function comicTitle(comic) {
  if (comic.title) return comic.title
  const seriesName = comic.series || 'Unknown series'
  const number = comic.number || comic.issue
  return number ? `${seriesName} #${number}` : seriesName
}

function comicMeta(comic) {
  return [
    comic.series && comic.title ? comic.series : '',
    comic.seriesVolume ? `Vol. ${comic.seriesVolume}` : '',
    comic.seriesYear || '',
    comic.publisher || '',
    comic.storeDate ? `Store ${formatDate(comic.storeDate)}` : '',
    comic.coverDate ? `Cover ${formatDate(comic.coverDate)}` : '',
  ]
    .filter(Boolean)
    .join(' · ')
}

function comicStoryLine(comic) {
  if (!Array.isArray(comic.storyNames) || comic.storyNames.length === 0) return ''
  return comic.storyNames.join(', ')
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

function arcSummary(item) {
  return [
    item.modified ? `Modified ${formatDate(item.modified)}` : '',
    item.id ? `Metron ID ${item.id}` : '',
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
  <div class="metron-view grid gap-5 [padding-block-start:16px] down-mobile:gap-3.5">
    <div
      class="metron-modes inline-grid grid-cols-5 gap-1.5 [width:min(680px,_100%)] [&_button]:min-h-10 [&_button]:border [&_button]:border-line-strong [&_button]:rounded [&_button]:bg-surface [&_button]:text-control [&_button]:py-2.5 [&_button]:px-3 [&_button.active]:border-primary [&_button.active]:bg-primary [&_button.active]:text-white [&.compact]:grid-cols-2 [&.compact]:[width:min(260px,_100%)] [&.compact_button]:min-h-8 [&.compact_button]:py-2 [&.compact_button]:px-2.5 down-mobile:grid-cols-2 down-mobile:w-full"
      role="tablist"
      aria-label="Metron search type"
    >
      <button
        type="button"
        :class="{ active: activeSearch === 'comics' }"
        role="tab"
        :aria-selected="activeSearch === 'comics'"
        @click="setSearchMode('comics')"
      >
        Comics
      </button>
      <button
        type="button"
        :class="{ active: activeSearch === 'readingLists' }"
        role="tab"
        :aria-selected="activeSearch === 'readingLists'"
        @click="setSearchMode('readingLists')"
      >
        Reading Lists
      </button>
      <button
        type="button"
        :class="{ active: activeSearch === 'series' }"
        role="tab"
        :aria-selected="activeSearch === 'series'"
        @click="setSearchMode('series')"
      >
        Series
      </button>
      <button
        type="button"
        :class="{ active: activeSearch === 'characters' }"
        role="tab"
        :aria-selected="activeSearch === 'characters'"
        @click="setSearchMode('characters')"
      >
        Characters
      </button>
      <button
        type="button"
        :class="{ active: activeSearch === 'arcs' }"
        role="tab"
        :aria-selected="activeSearch === 'arcs'"
        @click="setSearchMode('arcs')"
      >
        Arcs
      </button>
    </div>

    <div
      class="metron-quota-strip flex items-baseline justify-between gap-3 border border-line-strong rounded bg-surface-soft text-label py-2.5 px-3 text-sm font-bold [&_span]:flex [&_span]:items-baseline [&_span]:gap-2.5 [&_span]:min-w-0 [&_small]:text-muted [&_small]:overflow-hidden [&_small]:text-ellipsis [&_small]:whitespace-nowrap down-mobile:items-stretch down-mobile:flex-col down-mobile:[&_span]:flex-wrap"
    >
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

    <form
      class="metron-search grid [grid-template-columns:repeat(auto-fit,_minmax(160px,_1fr))] gap-3.5 items-end [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:text-sm [&_label]:font-bold down-tablet:grid-cols-1 down-mobile:[&_button]:w-full"
      @submit.prevent="search"
    >
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

    <div
      v-if="importStatus"
      class="metron-status flex items-center flex-wrap gap-y-2 gap-x-3 border border-line-strong rounded bg-surface-soft text-label py-2.5 px-3 text-sm font-bold"
    >
      <span>{{ importStatus }}</span>
    </div>

    <section
      class="metron-results single grid grid-cols-3 gap-5 items-start [&.single]:[grid-template-columns:minmax(0,_1fr)] [&_.row:disabled]:cursor-wait [&_.row:disabled]:[opacity:0.72] down-mobile:gap-3.5 down-mobile:[&_.detail-panel]:py-3.5 down-mobile:[&_.detail-panel]:px-3 down-tablet:grid-cols-1"
    >
      <article
        class="detail-panel min-h-panel border border-line rounded bg-panel p-5 shadow-detail down-mobile:min-h-0 down-mobile:p-3.5"
      >
        <template v-if="activeSearch === 'comics'">
          <h3>Comics</h3>
          <p v-if="searching" class="muted block text-muted">Searching Metron comics...</p>
          <p v-else-if="comicResults.length === 0" class="muted block text-muted">
            No Metron comic results yet.
          </p>
          <button
            v-for="comic in comicResults"
            :key="comic.id"
            class="row min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
            :disabled="rowImporting('comic', comic.id)"
            @click="importComic(comic)"
          >
            <span>
              <strong>{{ comicTitle(comic) }}</strong>
              <small v-if="comicMeta(comic)">{{ comicMeta(comic) }}</small>
              <small v-if="comicStoryLine(comic)">{{ comicStoryLine(comic) }}</small>
            </span>
            <span
              class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-xs flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
              >{{ rowImporting('comic', comic.id) ? 'Importing...' : 'Import' }}</span
            >
          </button>
        </template>

        <template v-else-if="activeSearch === 'readingLists'">
          <div
            class="section-title justify-between mb-2.5 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5"
          >
            <h3>Reading Lists</h3>
            <BaseButton
              variant="neutral"
              :disabled="importingAllReadingLists || rowImporting('readingLists', 0)"
              @click="importAllReadingLists"
            >
              {{
                importingAllReadingLists || rowImporting('readingLists', 0)
                  ? 'Pulling all...'
                  : 'Pull all'
              }}
            </BaseButton>
          </div>
          <p v-if="searching" class="muted block text-muted">Searching Metron reading lists...</p>
          <p v-else-if="readingListResults.length === 0" class="muted block text-muted">
            No Metron reading-list results yet.
          </p>
          <button
            v-for="list in readingListResults"
            :key="list.id"
            class="row min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
            :disabled="rowImporting('readingList', list.id)"
            @click="openReadingList(list)"
          >
            <span>
              <strong>{{ list.name || 'Untitled reading list' }}</strong>
              <small>{{ readingListSummary(list) }}</small>
            </span>
            <span
              class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-xs flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
              >{{ rowImporting('readingList', list.id) ? 'Importing...' : 'Details' }}</span
            >
          </button>
        </template>

        <template v-else-if="activeSearch === 'series'">
          <h3>Series</h3>
          <p v-if="searching" class="muted block text-muted">Searching Metron series...</p>
          <p v-else-if="seriesResults.length === 0" class="muted block text-muted">
            No Metron series results yet.
          </p>
          <button
            v-for="item in seriesResults"
            :key="item.id"
            class="row min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
            :disabled="rowImporting('series', item.id)"
            @click="importSeries(item)"
          >
            <span>
              <strong>{{ item.name || 'Untitled series' }}</strong>
              <small>
                Vol. {{ item.volume }} · {{ item.yearBegan || 'Unknown year' }} ·
                {{ item.issueCount }} issues
              </small>
            </span>
            <span
              class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-xs flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
              >{{ rowImporting('series', item.id) ? 'Importing...' : 'Import' }}</span
            >
          </button>
        </template>

        <template v-else-if="activeSearch === 'arcs'">
          <h3>Arcs</h3>
          <p v-if="searching" class="muted block text-muted">Searching Metron arcs...</p>
          <p v-else-if="arcResults.length === 0" class="muted block text-muted">
            No Metron arc results yet.
          </p>
          <button
            v-for="item in arcResults"
            :key="item.id"
            class="row min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
            :disabled="rowImporting('arc', item.id)"
            @click="importArc(item)"
          >
            <span>
              <strong>{{ item.name || 'Untitled arc' }}</strong>
              <small>{{ arcSummary(item) }}</small>
            </span>
            <span
              class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-xs flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
              >{{ rowImporting('arc', item.id) ? 'Importing...' : 'Import' }}</span
            >
          </button>
        </template>

        <template v-else>
          <h3>Characters</h3>
          <p v-if="searching" class="muted block text-muted">Searching Metron characters...</p>
          <p v-else-if="characterResults.length === 0" class="muted block text-muted">
            No Metron character results yet.
          </p>
          <button
            v-for="character in characterResults"
            :key="character.id"
            class="row min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
            :disabled="rowImporting('character', character.id)"
            @click="importCharacter(character)"
          >
            <span>
              <strong>{{ character.name }}</strong>
              <small>Metron ID {{ character.id }}</small>
            </span>
            <span
              class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-xs flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
            >
              {{ rowImporting('character', character.id) ? 'Importing...' : 'Import' }}
            </span>
          </button>
        </template>
      </article>
    </section>

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
