<script setup>
import ComicListView from '@/features/comics/components/ComicListView.vue'
import { formatProgress } from '@/features/reading-orders/model.js'
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import FavoriteToggle from '@/shared/components/feedback/FavoriteToggle.vue'

defineProps({
  selectedSeries: {
    type: Object,
    default: null,
  },
  selectedComicId: {
    type: Number,
    default: null,
  },
  quickSavingComicId: {
    type: Number,
    default: null,
  },
  importRunning: {
    type: Boolean,
    default: false,
  },
  readOnly: {
    type: Boolean,
    default: false,
  },
  canDelete: { type: Boolean, default: false },
  deleting: { type: Boolean, default: false },
  startSaving: { type: Boolean, default: false },
})

defineEmits([
  'back',
  'toggle-favorite',
  'toggle-started',
  'import-series',
  'open-comic',
  'toggle-read',
  'toggle-skipped',
  'delete',
])

function seriesYearLabel(series) {
  return series?.seriesYear ? ` (${series.seriesYear})` : ''
}

function seriesPublisherLabel(series) {
  if (!series?.publishers?.length) return 'No publisher saved'
  return series.publishers.join(', ')
}
</script>

<template>
  <div class="detail-view">
    <DetailNavigation @back="$emit('back')">
      <button
        v-if="selectedSeries && canDelete"
        class="danger-button"
        type="button"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? 'Deleting...' : 'Delete series' }}
      </button>
      <FavoriteToggle
        v-if="selectedSeries && !readOnly"
        class="detail-favorite-toggle"
        :favorite="selectedSeries.favorite"
        @toggle="$emit('toggle-favorite', selectedSeries)"
      />
      <button
        v-if="selectedSeries && !readOnly"
        :class="selectedSeries.startedAt ? 'secondary-button' : 'primary-button'"
        type="button"
        :disabled="startSaving"
        @click="$emit('toggle-started')"
      >
        {{
          startSaving ? 'Saving...' : selectedSeries.startedAt ? 'Stop reading' : 'Start reading'
        }}
      </button>
      <button
        v-if="selectedSeries?.metronSeriesId && !readOnly"
        class="primary-button"
        type="button"
        :disabled="importRunning"
        @click="$emit('import-series')"
      >
        {{ importRunning ? 'Importing...' : 'Import from Metron' }}
      </button>
    </DetailNavigation>

    <article class="detail-panel">
      <div v-if="selectedSeries" class="read-only-detail">
        <header class="panel-header">
          <div>
            <p class="eyebrow">Series</p>
            <h3>{{ selectedSeries.name }}{{ seriesYearLabel(selectedSeries) }}</h3>
          </div>
        </header>

        <div class="progress-meter" aria-label="Series read progress">
          <span :style="{ width: formatProgress(selectedSeries.progress) }"></span>
        </div>
        <div class="metadata-grid">
          <span>
            <strong>{{ formatProgress(selectedSeries.progress) }}</strong>
            <small>Progress</small>
          </span>
          <span>
            <strong>{{ selectedSeries.readCount }} / {{ selectedSeries.entryCount }}</strong>
            <small>Read</small>
          </span>
          <span>
            <strong>{{ selectedSeries.entryCount }}</strong>
            <small>Entries</small>
          </span>
          <span>
            <strong>{{ selectedSeries.favoriteCount }}</strong>
            <small>Favorites</small>
          </span>
          <span>
            <strong>{{ selectedSeries.startedCount }}</strong>
            <small>Currently reading</small>
          </span>
          <span>
            <strong>{{ seriesPublisherLabel(selectedSeries) }}</strong>
            <small>Publisher</small>
          </span>
          <span v-if="selectedSeries.startedAt"
            ><strong>Started</strong
            ><small>{{ new Date(selectedSeries.startedAt).toLocaleDateString() }}</small></span
          >
        </div>

        <ComicListView
          class="preview-list"
          title="Entries"
          :comics="selectedSeries.comics || []"
          :source-params="{ seriesId: selectedSeries.id }"
          :selected-comic-id="selectedComicId"
          :quick-saving-comic-id="quickSavingComicId"
          show-cover
          paginate-local
          server-source
          :read-only="readOnly"
          empty-message="No comics in this series yet."
          filtered-empty-message="No series entries match these filters."
          @open-comic="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
          @toggle-skipped="$emit('toggle-skipped', $event)"
        />
      </div>
      <p v-else class="empty-state">Select a series to view entries.</p>
    </article>
  </div>
</template>
