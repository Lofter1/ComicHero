<script setup>
import { computed, ref } from 'vue'
import {
  assetURL,
  getMetronReadingList,
  importMetronCharacterAppearances,
  importMetronComic,
  importMetronReadingList,
  importMetronSeries,
  importMetronArc,
  listMetronRequests,
  searchMetronCharacters,
  searchMetronComics,
  searchMetronReadingLists,
  searchMetronSeries,
  searchMetronArcs
} from '@/api/client.js'

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
const arc = ref('')
const searching = ref(false)
const importingKey = ref('')
const importStatus = ref('')
const importMode = ref('quick')
const importForce = ref(false)
const fullImportData = ref(['comics', 'series', 'arcs', 'characters'])
const rateLimit = ref(null)
const comicResults = ref([])
const characterResults = ref([])
const readingListResults = ref([])
const seriesResults = ref([])
const arcResults = ref([])
const selectedReadingList = ref(null)
const readingListDetailOpen = ref(false)
const readingListDetailLoading = ref(false)
const readingListDetailStatus = ref('')
const metronRequests = ref([])

const busy = computed(() => searching.value)
const searchLabel = computed(() => {
  if (searching.value) return 'Searching...'
  if (activeSearch.value === 'characters') return 'Search Characters'
  if (activeSearch.value === 'readingLists') return 'Search Reading Lists'
  if (activeSearch.value === 'arcs') return 'Search Arcs'
  return `Search ${activeSearch.value}`
})
const rateLimitSummary = computed(() => {
  if (!rateLimit.value) return ''

  const burst = limitText('Burst', rateLimit.value.burstRemaining, rateLimit.value.burstLimit)
  const sustained = limitText('Sustained', rateLimit.value.sustainedRemaining, rateLimit.value.sustainedLimit)
  return [burst, sustained].filter(Boolean).join(' · ')
})
const rateLimitReset = computed(() => {
  if (!rateLimit.value) return ''
  const resets = [rateLimit.value.burstReset, rateLimit.value.sustainedReset].filter(Boolean)
  if (resets.length === 0) return ''
  const nextReset = Math.max(...resets)
  return new Date(nextReset * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
})
const rateLimitLow = computed(() => {
  if (!rateLimit.value) return false
  return rateLimit.value.burstRemaining === 0 || rateLimit.value.sustainedRemaining === 0
})
const quotaKnown = computed(() => Boolean(props.metronQuota?.known))
const quotaSummary = computed(() => {
  if (!quotaKnown.value) return 'Quota appears after the first Metron response'
  const burst = quotaText('Burst', props.metronQuota.burstUsed, props.metronQuota.burstLimit)
  const sustained = quotaText('Sustained', props.metronQuota.sustainedUsed, props.metronQuota.sustainedLimit)
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
  const force = importForce.value
  if (importMode.value === 'full') {
    return { mode: 'full', fullData: normalizedFullImportData.value, force }
  }
  return { mode: 'quick', force }
})
const normalizedFullImportData = computed(() => {
  if (importMode.value !== 'full') return []
  const selected = new Set(fullImportData.value)
  if (selected.has('series') || selected.has('arcs') || selected.has('characters')) {
    selected.add('comics')
  }
  return ['comics', 'series', 'arcs', 'characters'].filter(item => selected.has(item))
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
      await refreshMetronRequests()
      return
    }

    if (activeSearch.value === 'readingLists') {
      const { data, rateLimit: nextRateLimit } = await searchMetronReadingLists({ q: query.value })
      updateRateLimit(nextRateLimit)
      readingListResults.value = Array.isArray(data) ? data : []
      await refreshMetronRequests()
      return
    }

    if (activeSearch.value === 'characters') {
      const { data, rateLimit: nextRateLimit } = await searchMetronCharacters({ q: query.value })
      updateRateLimit(nextRateLimit)
      characterResults.value = Array.isArray(data) ? data : []
      await refreshMetronRequests()
      return
    }

    if (activeSearch.value === 'arcs') {
      const { data, rateLimit: nextRateLimit } = await searchMetronArcs({ q: query.value })
      updateRateLimit(nextRateLimit)
      arcResults.value = Array.isArray(data) ? data : []
      await refreshMetronRequests()
      return
    }

    const { data, rateLimit: nextRateLimit } = await searchMetronSeries({ q: query.value || series.value })
    updateRateLimit(nextRateLimit)
    seriesResults.value = Array.isArray(data) ? data : []
    await refreshMetronRequests()
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
    await refreshMetronRequests()
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
    const { data: job, rateLimit: nextRateLimit } = await importMetronReadingList(id, importOptions.value)
    updateRateLimit(nextRateLimit)
    trackJob(job, list.name || 'Untitled reading list')
    await refreshMetronRequests()
  } catch (err) {
    updateRateLimit(err.rateLimit)
    emit('error', err.message)
  } finally {
    importingKey.value = ''
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
    await refreshMetronRequests()
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

async function importSeries(item) {
  const id = item.id
  importingKey.value = `series:${id}`
  importStatus.value = 'Series import started in the background.'
  try {
    const { data: job, rateLimit: nextRateLimit } = await importMetronSeries(id, importOptions.value)
    updateRateLimit(nextRateLimit)
    trackJob(job, item.name || 'Untitled series')
    await refreshMetronRequests()
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
    const { data: job, rateLimit: nextRateLimit } = await importMetronCharacterAppearances(id, importOptions.value)
    updateRateLimit(nextRateLimit)
    trackJob(job, character.name || 'Untitled character')
    await refreshMetronRequests()
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
    await refreshMetronRequests()
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
}

function updateRateLimit(nextRateLimit) {
  if (nextRateLimit) {
    rateLimit.value = nextRateLimit
    emit('quota-updated', quotaFromRateLimit(nextRateLimit))
  }
}

function trackJob(job, displayName) {
  if (!job?.id) return
  emit('job-started', { ...job, displayName })
}

async function refreshMetronRequests() {
  try {
    metronRequests.value = (await listMetronRequests()).slice(0, 8)
  } catch {
    metronRequests.value = []
  }
}

function limitText(label, remaining, limit) {
  if (remaining === null || remaining === undefined) return ''
  if (limit === null || limit === undefined) return `${label}: ${remaining} left`
  return `${label}: ${remaining}/${limit}`
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
  if (limit === null || limit === undefined || remaining === null || remaining === undefined) return 0
  return Math.max(0, limit - remaining)
}

function rowImporting(type, id) {
  return importingKey.value === `${type}:${id}` || props.importJobs.some(job => {
    return job.type === type && job.metronId === id && isActiveJob(job)
  })
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
  fullImportData.value = ['comics', 'series', 'arcs', 'characters'].filter(item => selected.has(item))
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
  ].filter(Boolean).join(' · ')
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
  ].filter(Boolean).join(' · ')
}

function arcSummary(item) {
  return [
    item.modified ? `Modified ${formatDate(item.modified)}` : '',
    item.id ? `Metron ID ${item.id}` : '',
  ].filter(Boolean).join(' · ')
}

function formatDate(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString([], { year: 'numeric', month: 'short', day: 'numeric' })
}

function requestLine(request) {
  const query = request.query ? `?${request.query}` : ''
  return `${request.method || 'GET'} ${request.path || request.url || ''}${query}`
}

</script>

<template>
  <div class="metron-view">
    <div class="metron-modes" role="tablist" aria-label="Metron search type">
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

    <div class="metron-quota-strip">
      <span>
        <strong>Metron quota</strong>
        <small>{{ quotaSummary }}</small>
      </span>
      <small v-if="quotaReset">Resets around {{ quotaReset }}</small>
    </div>

    <div class="metron-import-options">
      <div class="metron-modes compact" role="tablist" aria-label="Metron import depth">
        <button
          type="button"
          :class="{ active: importMode === 'quick' }"
          role="tab"
          :aria-selected="importMode === 'quick'"
          @click="setImportMode('quick')"
        >
          Quick
        </button>
        <button
          type="button"
          :class="{ active: importMode === 'full' }"
          role="tab"
          :aria-selected="importMode === 'full'"
          @click="setImportMode('full')"
        >
          Full
        </button>
      </div>
      <label class="inline-toggle">
        <input v-model="importForce" type="checkbox" />
        Force refresh
      </label>
      <fieldset v-if="importMode === 'full'" class="metron-data-options">
        <label class="inline-toggle">
          <input
            type="checkbox"
            :checked="fullImportData.includes('comics')"
            @change="toggleFullImportData('comics', $event.target.checked)"
          />
          Comics
        </label>
        <label class="inline-toggle">
          <input
            type="checkbox"
            :checked="fullImportData.includes('series')"
            @change="toggleFullImportData('series', $event.target.checked)"
          />
          Series
        </label>
        <label class="inline-toggle">
          <input
            type="checkbox"
            :checked="fullImportData.includes('arcs')"
            @change="toggleFullImportData('arcs', $event.target.checked)"
          />
          Arcs
        </label>
        <label class="inline-toggle">
          <input
            type="checkbox"
            :checked="fullImportData.includes('characters')"
            @change="toggleFullImportData('characters', $event.target.checked)"
          />
          Characters
        </label>
      </fieldset>
    </div>

    <form class="metron-search" @submit.prevent="search">
      <label>
        {{
          activeSearch === 'comics'
            ? 'Search'
            : activeSearch === 'readingLists'
              ? 'Reading List'
              : activeSearch === 'characters'
                ? 'Character'
                : activeSearch === 'series' ?
                  'Series' : 'Arc'
        }}
        <input v-model="query" placeholder="Batman, X-Men, Civil War" />
      </label>
      <label v-if="activeSearch === 'comics'">
        Series
        <input v-model="series" placeholder="Optional series filter" />
      </label>
      <label v-if="activeSearch === 'comics'">
        Issue
        <input v-model="issue" />
      </label>
      <button class="primary-button" type="submit" :disabled="busy">
        {{ searchLabel }}
      </button>
    </form>

    <div v-if="rateLimitSummary || importStatus" class="metron-status" :class="{ warning: rateLimitLow }">
      <span v-if="rateLimitSummary">{{ rateLimitSummary }}</span>
      <span v-if="rateLimitReset">Resets around {{ rateLimitReset }}</span>
      <span v-if="importStatus">{{ importStatus }}</span>
    </div>

    <section v-if="metronRequests.length" class="metron-request-log">
      <header>
        <strong>Recent Metron calls</strong>
        <small>{{ metronRequests.length }} shown</small>
      </header>
      <div class="metron-request-list">
        <span v-for="request in metronRequests" :key="`${request.startedAt}-${request.path}-${request.query}`">
          <code>{{ requestLine(request) }}</code>
          <small>
            {{ request.status || 'error' }} · {{ request.durationMillis }}ms<span v-if="request.conditional"> · conditional</span>
          </small>
        </span>
      </div>
    </section>

    <section class="metron-results single">
      <article class="detail-panel">
        <template v-if="activeSearch === 'comics'">
          <h3>Comics</h3>
          <p v-if="searching" class="muted">Searching Metron comics...</p>
          <p v-else-if="comicResults.length === 0" class="muted">No Metron comic results yet.</p>
          <button
            v-for="comic in comicResults"
            :key="comic.id"
            class="row"
            :disabled="rowImporting('comic', comic.id)"
            @click="importComic(comic)"
          >
            <span>
              <strong>{{ comicTitle(comic) }}</strong>
              <small v-if="comicMeta(comic)">{{ comicMeta(comic) }}</small>
              <small v-if="comicStoryLine(comic)">{{ comicStoryLine(comic) }}</small>
            </span>
            <span class="status-pill">{{ rowImporting('comic', comic.id) ? 'Importing...' : 'Import' }}</span>
          </button>
        </template>

        <template v-else-if="activeSearch === 'readingLists'">
          <h3>Reading Lists</h3>
          <p v-if="searching" class="muted">Searching Metron reading lists...</p>
          <p v-else-if="readingListResults.length === 0" class="muted">No Metron reading-list results yet.</p>
          <button
            v-for="list in readingListResults"
            :key="list.id"
            class="row"
            :disabled="rowImporting('readingList', list.id)"
            @click="openReadingList(list)"
          >
            <span>
              <strong>{{ list.name || 'Untitled reading list' }}</strong>
              <small>{{ readingListSummary(list) }}</small>
            </span>
            <span class="status-pill">{{ rowImporting('readingList', list.id) ? 'Importing...' : 'Details' }}</span>
          </button>
        </template>

        <template v-else-if="activeSearch === 'series'">
          <h3>Series</h3>
          <p v-if="searching" class="muted">Searching Metron series...</p>
          <p v-else-if="seriesResults.length === 0" class="muted">No Metron series results yet.</p>
          <button
            v-for="item in seriesResults"
            :key="item.id"
            class="row"
            :disabled="rowImporting('series', item.id)"
            @click="importSeries(item)"
          >
            <span>
              <strong>{{ item.name || 'Untitled series' }}</strong>
              <small>
                Vol. {{ item.volume }} · {{ item.yearBegan || 'Unknown year' }} · {{ item.issueCount }} issues
              </small>
            </span>
            <span class="status-pill">{{ rowImporting('series', item.id) ? 'Importing...' : 'Import' }}</span>
          </button>
        </template>

        <template v-else-if="activeSearch === 'arcs'">
          <h3>Arcs</h3>
          <p v-if="searching" class="muted">Searching Metron arcs...</p>
          <p v-else-if="arcResults.length === 0" class="muted">No Metron arc results yet.</p>
          <button
            v-for="item in arcResults"
            :key="item.id"
            class="row"
            :disabled="rowImporting('arc', item.id)"
            @click="importArc(item)"
          >
            <span>
              <strong>{{ item.name || 'Untitled arc' }}</strong>
              <small>{{ arcSummary(item) }}</small>
            </span>
            <span class="status-pill">{{ rowImporting('arc', item.id) ? 'Importing...' : 'Import' }}</span>
          </button>
        </template>

        <template v-else>
          <h3>Characters</h3>
          <p v-if="searching" class="muted">Searching Metron characters...</p>
          <p v-else-if="characterResults.length === 0" class="muted">No Metron character results yet.</p>
          <button
            v-for="character in characterResults"
            :key="character.id"
            class="row"
            :disabled="rowImporting('character', character.id)"
            @click="importCharacter(character)"
          >
            <span>
              <strong>{{ character.name }}</strong>
              <small>Metron ID {{ character.id }}</small>
            </span>
            <span class="status-pill">
              {{ rowImporting('character', character.id) ? 'Importing...' : 'Import' }}
            </span>
          </button>
        </template>
      </article>
    </section>

    <div v-if="readingListDetailOpen" class="modal-backdrop" @click.self="closeReadingListDetail">
      <section class="metron-detail-dialog" role="dialog" aria-modal="true" aria-labelledby="reading-list-detail-title">
        <header class="metron-detail-header">
          <span>
            <strong id="reading-list-detail-title">{{ selectedReadingList?.name || 'Reading list' }}</strong>
            <small>{{ selectedReadingList ? readingListSummary(selectedReadingList) : '' }}</small>
          </span>
          <button class="icon-button" type="button" aria-label="Close reading list detail" @click="closeReadingListDetail">×</button>
        </header>
        <div class="metron-detail-body">
          <img
            v-if="selectedReadingList?.image"
            class="metron-detail-image"
            :src="assetURL(selectedReadingList.image)"
            :alt="selectedReadingList.name || 'Reading list image'"
          />
          <div class="metron-detail-copy">
            <p v-if="readingListDetailLoading" class="muted">Loading reading-list details...</p>
            <p v-else-if="readingListDetailStatus" class="error-text">{{ readingListDetailStatus }}</p>
            <p v-else>{{ selectedReadingList?.description || 'No description from Metron.' }}</p>
            <dl class="metron-detail-facts">
              <div v-if="selectedReadingList?.user?.username">
                <dt>User</dt>
                <dd>{{ selectedReadingList.user.username }}</dd>
              </div>
              <div v-if="selectedReadingList?.listType">
                <dt>Type</dt>
                <dd>{{ selectedReadingList.listType }}</dd>
              </div>
              <div v-if="selectedReadingList?.attributionSource">
                <dt>Source</dt>
                <dd>{{ selectedReadingList.attributionSource }}</dd>
              </div>
              <div v-if="selectedReadingList?.ratingCount">
                <dt>Rating</dt>
                <dd>{{ selectedReadingList.averageRating || 0 }} from {{ selectedReadingList.ratingCount }}</dd>
              </div>
              <div v-if="selectedReadingList?.modified">
                <dt>Modified</dt>
                <dd>{{ formatDate(selectedReadingList.modified) }}</dd>
              </div>
              <div v-if="selectedReadingList?.issues?.length">
                <dt>Issues</dt>
                <dd>{{ selectedReadingList.issues.length }}</dd>
              </div>
            </dl>
          </div>
        </div>
        <footer class="metron-detail-actions">
          <button class="secondary-button" type="button" @click="closeReadingListDetail">Close</button>
          <button
            class="primary-button"
            type="button"
            :disabled="!selectedReadingList || rowImporting('readingList', selectedReadingList.id)"
            @click="importReadingList(selectedReadingList)"
          >
            {{ selectedReadingList && rowImporting('readingList', selectedReadingList.id) ? 'Importing...' : 'Import' }}
          </button>
        </footer>
      </section>
    </div>
  </div>
</template>
