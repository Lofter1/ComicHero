<script setup>
import { computed } from 'vue'
import { assetURL } from '@/api/client.js'

const props = defineProps({
  selectedComic: {
    type: Object,
    default: null,
  },
  quickSavingComicId: {
    type: Number,
    default: null,
  },
  metronMetadataOpen: {
    type: Boolean,
    default: false,
  },
  metronMetadataSearching: {
    type: Boolean,
    default: false,
  },
  metronMetadataApplyingId: {
    type: Number,
    default: null,
  },
  metronMetadataStatus: {
    type: String,
    default: '',
  },
  metronMetadataResults: {
    type: Array,
    default: () => [],
  },
  readOnly: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits([
  'back',
  'search-metron',
  'apply-metron',
  'reset-metron',
  'toggle-read',
  'toggle-skipped',
  'open-character',
  'open-series',
])

const metronActionDisabled = computed(
  () => props.metronMetadataSearching || props.metronMetadataApplyingId !== null,
)
const metronActionLabel = computed(() =>
  props.selectedComic?.metronIssueId ? 'Refresh Metron' : 'Get Metron metadata',
)

function runMetronAction() {
  if (!props.selectedComic) return
  if (props.selectedComic.metronIssueId) {
    emit('apply-metron', props.selectedComic.metronIssueId, { force: false })
    return
  }
  emit('search-metron')
}

function forceMetronRefresh() {
  if (!props.selectedComic?.metronIssueId) return
  emit('apply-metron', props.selectedComic.metronIssueId, { force: true })
}

function seriesLabel(comic) {
  if (!comic) return ''
  return `${comic.series || 'Unknown series'}${comic.seriesYear ? ` (${comic.seriesYear})` : ''}`
}
</script>

<template>
  <div class="detail-view">
    <header class="detail-nav sticky-toolbar">
      <button class="secondary-button" type="button" @click="$emit('back')">Back</button>
      <div class="detail-nav-actions">
        <button
          v-if="selectedComic && !readOnly"
          class="secondary-button"
          type="button"
          :disabled="metronActionDisabled"
          @click="runMetronAction"
        >
          {{ metronActionLabel }}
        </button>
        <button
          v-if="selectedComic?.metronIssueId && !readOnly"
          class="ghost-button"
          type="button"
          :disabled="metronActionDisabled"
          @click="forceMetronRefresh"
        >
          Force refresh
        </button>
        <button
          v-if="selectedComic && !readOnly"
          class="read-toggle-button large"
          type="button"
          :disabled="quickSavingComicId === selectedComic.id"
          @click="$emit('toggle-read', selectedComic)"
        >
          {{ selectedComic.read ? 'Mark unread' : 'Mark read' }}
        </button>
        <button
          v-if="selectedComic && !readOnly"
          class="read-toggle-button large"
          :class="{ skipped: selectedComic.skipped }"
          type="button"
          :disabled="quickSavingComicId === selectedComic.id"
          @click="$emit('toggle-skipped', selectedComic)"
        >
          {{ selectedComic.skipped ? 'Unskip' : 'Skip' }}
        </button>
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
          <img
            :src="assetURL(selectedComic.coverImage)"
            :alt="`${selectedComic.title} cover`"
            loading="lazy"
          />
        </div>

        <div class="metadata-grid">
          <span>
            <strong>
              {{ selectedComic.skipped ? 'Skipped' : selectedComic.read ? 'Read' : 'Unread' }}
            </strong>
            <small>Status</small>
          </span>
          <span>
            <button
              v-if="selectedComic.seriesId"
              class="metadata-link-button"
              type="button"
              @click="$emit('open-series', { id: selectedComic.seriesId })"
            >
              {{ seriesLabel(selectedComic) }}
            </button>
            <strong v-else>{{ seriesLabel(selectedComic) }}</strong>
            <small>Series</small>
          </span>
          <span>
            <strong>{{ selectedComic.publisher || 'Unknown' }}</strong>
            <small>Publisher</small>
          </span>
          <span>
            <strong>{{ selectedComic.coverDate || 'Unknown' }}</strong>
            <small>Cover Date</small>
          </span>
        </div>

        <div
          v-if="!readOnly && (metronMetadataOpen || metronMetadataStatus)"
          class="metron-metadata-panel"
        >
          <header class="section-title">
            <div>
              <p class="eyebrow">Metron</p>
              <h4>Metadata matches</h4>
            </div>
            <button
              v-if="metronMetadataOpen || metronMetadataStatus"
              class="ghost-button"
              type="button"
              @click="$emit('reset-metron')"
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
              :disabled="metronMetadataApplyingId !== null"
              @click="$emit('apply-metron', issue.id)"
            >
              <span>
                <strong
                  >{{ issue.series }} #{{ issue.number || issue.issue }}:
                  {{ issue.title || 'Untitled issue' }}</strong
                >
                <small
                  >{{ issue.publisher || 'Unknown publisher' }} ·
                  {{ issue.coverDate || 'Unknown date' }}</small
                >
              </span>
              <span class="status-pill">
                {{ metronMetadataApplyingId === issue.id ? 'Applying...' : 'Apply' }}
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
              @click="$emit('open-character', character)"
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
</template>
