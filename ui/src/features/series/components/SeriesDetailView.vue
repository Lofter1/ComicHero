<script setup>
import ComicListView from '@/features/comics/components/ComicListView.vue'
import { formatProgress } from '@/features/reading-orders/model.js'
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import EmptyState from '@/shared/components/feedback/EmptyState.vue'
import FavoriteToggle from '@/shared/components/feedback/FavoriteToggle.vue'
import ProgressBar from '@/shared/components/feedback/ProgressBar.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import DetailPanel from '@/shared/components/layout/DetailPanel.vue'
import MetadataGrid from '@/shared/components/layout/MetadataGrid.vue'
import PanelHeader from '@/shared/components/layout/PanelHeader.vue'

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
      <BaseButton
        v-if="selectedSeries && canDelete"
        variant="danger"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? 'Deleting...' : 'Delete series' }}
      </BaseButton>
      <FavoriteToggle
        v-if="selectedSeries && !readOnly"
        class="detail-favorite-toggle self-center"
        :favorite="selectedSeries.favorite"
        @toggle="$emit('toggle-favorite', selectedSeries)"
      />
      <BaseButton
        v-if="selectedSeries && !readOnly"
        :variant="selectedSeries.startedAt ? 'secondary' : 'primary'"
        :disabled="startSaving"
        @click="$emit('toggle-started')"
      >
        {{
          startSaving ? 'Saving...' : selectedSeries.startedAt ? 'Stop reading' : 'Start reading'
        }}
      </BaseButton>
      <BaseButton
        v-if="selectedSeries?.metronSeriesId && !readOnly"
        variant="primary"
        :disabled="importRunning"
        @click="$emit('import-series')"
      >
        {{ importRunning ? 'Importing...' : 'Import from Metron' }}
      </BaseButton>
    </DetailNavigation>

    <DetailPanel>
      <div v-if="selectedSeries" class="read-only-detail grid gap-4">
        <PanelHeader
          eyebrow="Series"
          :title="`${selectedSeries.name}${seriesYearLabel(selectedSeries)}`"
        />

        <ProgressBar
          :value="formatProgress(selectedSeries.progress)"
          label="Series read progress"
        />
        <MetadataGrid>
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
        </MetadataGrid>

        <ComicListView
          embedded
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
      <EmptyState v-else tag="p"> Select a series to view entries. </EmptyState>
    </DetailPanel>
  </div>
</template>
