<script setup>
import { assetURL } from '@/api/client.js'
import ComicListView from '@/features/comics/components/ComicListView.vue'
import { formatProgress } from '@/features/reading-orders/model.js'
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import FavoriteToggle from '@/shared/components/feedback/FavoriteToggle.vue'

defineProps({
  selectedArc: {
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
  quickSavingArcId: {
    type: Number,
    default: null,
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
  'open-comic',
  'toggle-read',
  'toggle-skipped',
  'delete',
])
</script>

<template>
  <div class="detail-view grid gap-4 w-full">
    <DetailNavigation @back="$emit('back')">
      <button
        v-if="selectedArc && canDelete"
        class="danger-button min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 [border-color:color-mix(in_srgb,_var(--danger)_42%,_var(--line-strong))] bg-danger-soft text-danger"
        type="button"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? 'Deleting...' : 'Delete arc' }}
      </button>
      <FavoriteToggle
        v-if="selectedArc && !readOnly"
        class="detail-favorite-toggle self-center"
        :favorite="selectedArc.favorite"
        :disabled="quickSavingArcId === selectedArc.id"
        @toggle="$emit('toggle-favorite', selectedArc)"
      />
      <button
        v-if="selectedArc && !readOnly"
        :class="selectedArc.startedAt ? 'secondary-button' : 'primary-button'"
        type="button"
        :disabled="startSaving"
        @click="$emit('toggle-started')"
      >
        {{ startSaving ? 'Saving...' : selectedArc.startedAt ? 'Stop reading' : 'Start reading' }}
      </button>
    </DetailNavigation>

    <article
      class="detail-panel min-h-90 border border-line rounded bg-panel p-5 shadow-detail down-mobile:min-h-0 down-mobile:p-3.5"
    >
      <div v-if="selectedArc" class="read-only-detail grid gap-4.5">
        <header
          class="panel-header justify-between mb-4.5 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5"
        >
          <div>
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Arc</p>
            <h3>{{ selectedArc.name }}</h3>
          </div>
        </header>

        <div
          v-if="selectedArc.image"
          class="character-portrait overflow-hidden border border-line rounded bg-surface-muted max-w-45 [&_img]:block [&_img]:w-full [&_img]:aspect-square [&_img]:object-cover"
        >
          <img
            :src="assetURL(selectedArc.image)"
            :alt="`${selectedArc.name} arc artwork`"
            loading="lazy"
          />
        </div>

        <p class="detail-description text-body leading-normal">
          {{ selectedArc.description || 'No description' }}
        </p>
        <div
          class="progress-meter h-2.5 overflow-hidden rounded-full bg-read-progress [&_span]:block [&_span]:h-full [&_span]:min-w-0.5 [&_span]:[border-radius:inherit] [&_span]:bg-progress"
          aria-label="Arc progress"
        >
          <span :style="{ width: formatProgress(selectedArc.progress) }"></span>
        </div>
        <div
          class="metadata-grid grid grid-cols-3 gap-2.5 [&_span]:border [&_span]:border-line [&_span]:rounded [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_strong]:break-anywhere [&_small]:block [&_small]:text-muted [&_small]:mt-1 down-tablet:grid-cols-1"
        >
          <span>
            <strong>{{ formatProgress(selectedArc.progress) }}</strong>
            <small>Progress</small>
          </span>
          <span>
            <strong>{{ selectedArc.comics.length }}</strong>
            <small>Comics</small>
          </span>
          <span>
            <strong>{{ selectedArc.favoriteCount }}</strong>
            <small>Favorites</small>
          </span>
          <span>
            <strong>{{ selectedArc.startedCount }}</strong>
            <small>Currently reading</small>
          </span>
          <span v-if="selectedArc.startedAt"
            ><strong>Started</strong
            ><small>{{ new Date(selectedArc.startedAt).toLocaleDateString() }}</small></span
          >
        </div>

        <ComicListView
          class="preview-list [&_small]:block [&_small]:text-muted border-t border-line pt-3.5 [&_ol]:mb-0 [&_ol]:pl-5.5 [&_ul]:mb-0 [&_ul]:pl-5.5 [&_li]:mb-2.5"
          title="Comics"
          :comics="selectedArc.comics"
          :source-params="{ arcId: selectedArc.id }"
          :selected-comic-id="selectedComicId"
          :quick-saving-comic-id="quickSavingComicId"
          show-comment
          paginate-local
          server-source
          :read-only="readOnly"
          empty-message="No comics in this arc yet."
          filtered-empty-message="No comics match these filters."
          @open-comic="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
          @toggle-skipped="$emit('toggle-skipped', $event)"
        />
      </div>
      <p
        v-else
        class="empty-state grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4"
      >
        Select an arc to view it.
      </p>
    </article>
  </div>
</template>
