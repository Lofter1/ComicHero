<script setup>
import ComicListView from '@/components/ComicListView.vue'
import { formatProgress } from '@/domain/readingOrders.js'

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
})

defineEmits(['back', 'toggle-favorite', 'import-series', 'open-comic', 'toggle-read'])

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
    <header class="detail-nav sticky-toolbar">
      <button class="secondary-button" type="button" @click="$emit('back')">Back</button>
      <div class="detail-nav-actions">
        <button
          v-if="selectedSeries"
          type="button"
          class="favorite-toggle detail-favorite-toggle"
          :class="{ active: selectedSeries.favorite }"
          :aria-label="selectedSeries.favorite ? 'Remove from favorites' : 'Add to favorites'"
          :title="selectedSeries.favorite ? 'Remove from favorites' : 'Add to favorites'"
          @click="$emit('toggle-favorite', selectedSeries)"
        >
          <span aria-hidden="true">{{ selectedSeries.favorite ? '★' : '☆' }}</span>
        </button>
        <button
          v-if="selectedSeries"
          class="primary-button"
          type="button"
          :disabled="importRunning"
          @click="$emit('import-series')"
        >
          {{ importRunning ? 'Importing...' : 'Import from Metron' }}
        </button>
      </div>
    </header>

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
            <strong>{{ seriesPublisherLabel(selectedSeries) }}</strong>
            <small>Publisher</small>
          </span>
        </div>

        <ComicListView
          class="preview-list"
          title="Entries"
          :comics="selectedSeries.comics || []"
          :selected-comic-id="selectedComicId"
          :quick-saving-comic-id="quickSavingComicId"
          show-cover
          empty-message="No comics in this series yet."
          filtered-empty-message="No series entries match these filters."
          @open-comic="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
        />
      </div>
      <p v-else class="empty-state">Select a series to view entries.</p>
    </article>
  </div>
</template>
