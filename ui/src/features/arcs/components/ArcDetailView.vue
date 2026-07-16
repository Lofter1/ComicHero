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
  <div class="detail-view">
    <DetailNavigation @back="$emit('back')">
      <button
        v-if="selectedArc && canDelete"
        class="danger-button"
        type="button"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? 'Deleting...' : 'Delete arc' }}
      </button>
      <FavoriteToggle
        v-if="selectedArc && !readOnly"
        class="detail-favorite-toggle"
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

    <article class="detail-panel">
      <div v-if="selectedArc" class="read-only-detail">
        <header class="panel-header">
          <div>
            <p class="eyebrow">Arc</p>
            <h3>{{ selectedArc.name }}</h3>
          </div>
        </header>

        <div v-if="selectedArc.image" class="character-portrait">
          <img
            :src="assetURL(selectedArc.image)"
            :alt="`${selectedArc.name} arc artwork`"
            loading="lazy"
          />
        </div>

        <p class="detail-description">{{ selectedArc.description || 'No description' }}</p>
        <div class="progress-meter" aria-label="Arc progress">
          <span :style="{ width: formatProgress(selectedArc.progress) }"></span>
        </div>
        <div class="metadata-grid">
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
          class="preview-list"
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
      <p v-else class="empty-state">Select an arc to view it.</p>
    </article>
  </div>
</template>
