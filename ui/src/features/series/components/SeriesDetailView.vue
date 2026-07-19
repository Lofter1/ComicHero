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
  <div class="detail-view grid gap-4 w-full">
    <DetailNavigation @back="$emit('back')">
      <button
        v-if="selectedSeries && canDelete"
        class="danger-button min-h-10 border rounded py-2.5 px-3.5 [border-color:color-mix(in_srgb,_var(--danger)_42%,_var(--line-strong))] bg-danger-soft text-danger"
        type="button"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? 'Deleting...' : 'Delete series' }}
      </button>
      <FavoriteToggle
        v-if="selectedSeries && !readOnly"
        class="detail-favorite-toggle self-center"
        :favorite="selectedSeries.favorite"
        @toggle="$emit('toggle-favorite', selectedSeries)"
      />
      <button
        v-if="selectedSeries && !readOnly"
        class="min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5"
        :class="
          selectedSeries.startedAt
            ? 'secondary-button bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]'
            : 'primary-button border-primary bg-primary text-white'
        "
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
        class="primary-button min-h-10 border rounded py-2.5 px-3.5 border-primary bg-primary text-white"
        type="button"
        :disabled="importRunning"
        @click="$emit('import-series')"
      >
        {{ importRunning ? 'Importing...' : 'Import from Metron' }}
      </button>
    </DetailNavigation>

    <article
      class="detail-panel min-h-panel border border-line rounded bg-panel p-5 shadow-detail down-mobile:min-h-0 down-mobile:p-3.5"
    >
      <div v-if="selectedSeries" class="read-only-detail grid gap-4">
        <header
          class="panel-header justify-between mb-4 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5"
        >
          <div>
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Series</p>
            <h3>{{ selectedSeries.name }}{{ seriesYearLabel(selectedSeries) }}</h3>
          </div>
        </header>

        <div
          class="progress-meter h-2.5 overflow-hidden rounded-full bg-read-progress [&_span]:block [&_span]:h-full [&_span]:min-w-0.5 [&_span]:[border-radius:inherit] [&_span]:bg-progress"
          aria-label="Series read progress"
        >
          <span :style="{ width: formatProgress(selectedSeries.progress) }"></span>
        </div>
        <div
          class="metadata-grid grid grid-cols-3 gap-2.5 [&_span]:border [&_span]:border-line [&_span]:rounded [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_strong]:break-anywhere [&_small]:block [&_small]:text-muted [&_small]:mt-1 down-tablet:grid-cols-1"
        >
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
          class="[&_small]:block [&_small]:text-muted border-t border-line pt-3.5 [&_ol]:mb-0 [&_ol]:pl-6 [&_ul]:mb-0 [&_ul]:pl-6 [&_li]:mb-2.5"
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
      <p
        v-else
        class="empty-state grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4"
      >
        Select a series to view entries.
      </p>
    </article>
  </div>
</template>
