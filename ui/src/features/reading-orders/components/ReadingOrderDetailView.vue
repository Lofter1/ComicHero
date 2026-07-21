<script setup>
import { computed } from 'vue'
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import EmptyState from '@/shared/components/feedback/EmptyState.vue'
import ProgressBar from '@/shared/components/feedback/ProgressBar.vue'
import SafeMarkdown from '@/shared/components/content/SafeMarkdown.vue'
import { assetURL } from '@/api/client.js'
import ComicListView from '@/features/comics/components/ComicListView.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import DetailPanel from '@/shared/components/layout/DetailPanel.vue'
import MetadataGrid from '@/shared/components/layout/MetadataGrid.vue'
import PanelHeader from '@/shared/components/layout/PanelHeader.vue'
import {
  formatProgress,
  formatRating,
  readingOrderCover,
  readingOrderDisplayComics,
} from '@/features/reading-orders/model.js'

const props = defineProps({
  selectedOrder: {
    type: Object,
    default: null,
  },
  currentUserId: {
    type: Number,
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
  readOnly: {
    type: Boolean,
    default: false,
  },
  saving: {
    type: Boolean,
    default: false,
  },
  ratingSaving: {
    type: Boolean,
    default: false,
  },
  startSaving: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits([
  'back',
  'copy',
  'edit',
  'export-cbl',
  'open-comic',
  'rate',
  'start',
  'stop',
  'toggle-read',
  'toggle-skipped',
])

const ratingValues = [1, 2, 3, 4, 5]

const displayComics = computed(() => readingOrderDisplayComics(props.selectedOrder))
</script>

<template>
  <div class="detail-view grid gap-4 w-full">
    <DetailNavigation @back="$emit('back')">
      <BaseButton
        v-if="selectedOrder && !readOnly && !selectedOrder.startedAt"
        variant="primary"
        :disabled="startSaving"
        @click="$emit('start')"
      >
        {{ startSaving ? 'Starting...' : 'Start reading order' }}
      </BaseButton>
      <BaseButton
        v-if="selectedOrder && !readOnly && selectedOrder.startedAt"
        :disabled="startSaving"
        @click="$emit('stop')"
      >
        {{ startSaving ? 'Stopping...' : 'Stop reading' }}
      </BaseButton>
      <BaseButton v-if="selectedOrder?.canEdit" variant="primary" @click="$emit('edit')">
        Edit
      </BaseButton>
      <BaseButton
        v-if="
          selectedOrder &&
          !readOnly &&
          selectedOrder.authorUserId &&
          selectedOrder.authorUserId !== currentUserId
        "
        :disabled="saving"
        @click="$emit('copy')"
      >
        {{ saving ? 'Copying...' : 'Copy' }}
      </BaseButton>
      <BaseButton v-if="selectedOrder" @click="$emit('export-cbl')"> Export CBL </BaseButton>
    </DetailNavigation>

    <DetailPanel>
      <div v-if="selectedOrder" class="read-only-detail grid gap-4">
        <PanelHeader eyebrow="Reading order" :title="selectedOrder.name" />

        <div class="reading-order-summary">
          <div v-if="readingOrderCover(selectedOrder)" class="reading-order-thumbnail">
            <img
              :src="assetURL(readingOrderCover(selectedOrder))"
              :alt="`${selectedOrder.name} thumbnail`"
              loading="lazy"
            />
          </div>

          <SafeMarkdown
            class="detail-description"
            :source="selectedOrder.description"
            fallback="No description"
          />
        </div>

        <ProgressBar
          :value="formatProgress(selectedOrder.progress)"
          label="Reading order progress"
        />

        <MetadataGrid>
          <span>
            <strong>{{ formatProgress(selectedOrder.progress) }}</strong>
            <small>Progress</small>
          </span>
          <span>
            <strong>{{ selectedOrder.comics.length }}</strong>
            <small>Comics</small>
          </span>
          <span>
            <strong>{{ formatRating(selectedOrder.rating) }}</strong>
            <small>
              Rating<template v-if="selectedOrder.ratingCount">
                · {{ selectedOrder.ratingCount }}</template
              >
            </small>
          </span>
          <span v-if="selectedOrder.authorName">
            <strong>{{ selectedOrder.authorName }}</strong>
            <small>Author</small>
          </span>
          <span>
            <strong>{{ selectedOrder.favoriteCount }}</strong>
            <small>Favorites</small>
          </span>
          <span>
            <strong>{{ selectedOrder.startedCount }}</strong>
            <small>Currently reading</small>
          </span>
          <span v-if="selectedOrder.startedAt">
            <strong>Started</strong>
            <small>{{ new Date(selectedOrder.startedAt).toLocaleDateString() }}</small>
          </span>
        </MetadataGrid>

        <section v-if="!readOnly" class="reading-order-rating-panel">
          <div>
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Your rating</p>
            <strong>{{
              selectedOrder.myRating ? `${selectedOrder.myRating.toFixed(1)} / 5` : 'Not rated'
            }}</strong>
          </div>
          <div
            class="rating-button-group flex items-center flex-wrap justify-end gap-2"
            role="group"
            aria-label="Rate reading order"
          >
            <!-- Native buttons: rating choices are a stateful numeric segmented control. -->
            <button
              v-for="value in ratingValues"
              :key="value"
              type="button"
              class="rating-button"
              :class="{ active: selectedOrder.myRating === value }"
              :aria-pressed="selectedOrder.myRating === value"
              :disabled="ratingSaving"
              @click="emit('rate', value)"
            >
              {{ value }}
            </button>
            <BaseButton
              type="button"
              class="w-auto"
              variant="secondary"
              size="compact"
              :disabled="ratingSaving || !selectedOrder.myRating"
              @click="emit('rate', 0)"
            >
              Clear
            </BaseButton>
          </div>
        </section>

        <ComicListView
          embedded
          title="Comics"
          :comics="displayComics"
          :selected-comic-id="selectedComicId"
          :quick-saving-comic-id="quickSavingComicId"
          initial-sort="readingOrder"
          show-reading-order-sort
          show-sections
          show-comment
          show-cover
          :read-only="readOnly"
          paginate-local
          empty-message="No comics in this reading order yet."
          filtered-empty-message="No comics match these filters."
          @open-comic="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
          @toggle-skipped="$emit('toggle-skipped', $event)"
        />
      </div>

      <EmptyState v-else> Select a reading order to view details. </EmptyState>
    </DetailPanel>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.reading-order-summary {
  @apply grid grid-cols-[minmax(150px,220px)_minmax(0,1fr)] items-start gap-4 down-mobile:grid-cols-1 [&_.detail-description]:m-0;
}

.reading-order-thumbnail {
  @apply w-full overflow-hidden border border-line rounded bg-surface-muted down-mobile:max-w-56 [&_img]:block [&_img]:w-full [&_img]:aspect-square [&_img]:object-cover;
}

.reading-order-rating-panel {
  @apply flex items-center justify-between gap-3 border border-line rounded bg-surface-soft p-3 [&_strong]:block;
}

.rating-button {
  @apply inline-flex items-center justify-center w-10 h-10 border border-line-strong rounded bg-surface text-(--text) font-extrabold cursor-pointer [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:bg-primary [&:hover:not(:disabled)]:text-(--primary-contrast) [&.active]:border-primary [&.active]:bg-primary [&.active]:text-(--primary-contrast);
}
</style>
