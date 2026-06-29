<script setup>
import { computed, ref } from 'vue'
import {
  importMetronCharacterAppearances,
  importMetronComic,
  importMetronReadingList,
  importMetronSeries,
  searchMetronCharacters,
  searchMetronComics,
  searchMetronReadingLists,
  searchMetronSeries,
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
const searching = ref(false)
const importingKey = ref('')
const importStatus = ref('')
const rateLimit = ref(null)
const comicResults = ref([])
const characterResults = ref([])
const readingListResults = ref([])
const seriesResults = ref([])

const busy = computed(() => searching.value)
const searchLabel = computed(() => {
  if (searching.value) return 'Searching...'
  if (activeSearch.value === 'characters') return 'Search Characters'
  return `Search ${activeSearch.value === 'readingLists' ? 'Reading Lists' : activeSearch.value}`
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

    const { data, rateLimit: nextRateLimit } = await searchMetronSeries({ q: query.value || series.value })
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
    const { data: job, rateLimit: nextRateLimit } = await importMetronComic(id)
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
    const { data: job, rateLimit: nextRateLimit } = await importMetronReadingList(id)
    updateRateLimit(nextRateLimit)
    trackJob(job, list.name || 'Untitled reading list')
  } catch (err) {
    updateRateLimit(err.rateLimit)
    emit('error', err.message)
  } finally {
    importingKey.value = ''
  }
}

async function importSeries(item) {
  const id = item.id
  importingKey.value = `series:${id}`
  importStatus.value = 'Series import started in the background.'
  try {
    const { data: job, rateLimit: nextRateLimit } = await importMetronSeries(id)
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
    const { data: job, rateLimit: nextRateLimit } = await importMetronCharacterAppearances(id)
    updateRateLimit(nextRateLimit)
    trackJob(job, character.name || 'Untitled character')
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
    return job.type === type && job.metronId === id && (job.status === 'queued' || job.status === 'running')
  })
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
    </div>

    <div class="metron-quota-strip">
      <span>
        <strong>Metron quota</strong>
        <small>{{ quotaSummary }}</small>
      </span>
      <small v-if="quotaReset">Resets around {{ quotaReset }}</small>
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
                : 'Series'
        }}
        <input v-model="query" placeholder="Batman, X-Men, Civil War" />
      </label>
      <label v-if="activeSearch === 'comics'">
        Series
        <input v-model="series" placeholder="Optional series filter" />
      </label>
      <label v-if="activeSearch === 'comics'">
        Issue
        <input v-model="issue" min="0" type="number" />
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
              <strong>{{ comic.title || 'Untitled comic' }}</strong>
              <small>{{ comic.series }} #{{ comic.number || comic.issue }} · {{ comic.publisher }}</small>
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
            @click="importReadingList(list)"
          >
            <span>
              <strong>{{ list.name || 'Untitled reading list' }}</strong>
              <small>{{ list.description || 'No description' }}</small>
            </span>
            <span class="status-pill">{{ rowImporting('readingList', list.id) ? 'Importing...' : 'Import' }}</span>
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

        <template v-else>
          <h3>Characters</h3>
          <p v-if="searching" class="muted">Searching Metron characters...</p>
          <p v-else-if="characterResults.length === 0" class="muted">No Metron character results yet.</p>
          <button
            v-for="character in characterResults"
            :key="character.id"
            class="row"
            :disabled="importingKey === `character:${character.id}`"
            @click="importCharacter(character)"
          >
            <span>
              <strong>{{ character.name }}</strong>
              <small>Metron ID {{ character.id }}</small>
            </span>
            <span class="status-pill">
              {{ importingKey === `character:${character.id}` ? 'Importing...' : 'Import' }}
            </span>
          </button>
        </template>
      </article>
    </section>
  </div>
</template>
