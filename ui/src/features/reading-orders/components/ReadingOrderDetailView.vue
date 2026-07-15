<script setup>
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import { assetURL } from '@/api/client.js'
import ComicListView from '@/features/comics/components/ComicListView.vue'
import { formatProgress, formatRating } from '@/features/reading-orders/model.js'

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

const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
})

const renderedDescription = computed(() => {
  const description = props.selectedOrder?.description?.trim()
  return description ? markdown.render(description) : '<p>No description</p>'
})
</script>

<template>
  <div class="detail-view">
    <DetailNavigation @back="$emit('back')">
      <button
        v-if="selectedOrder && !readOnly && !selectedOrder.startedAt"
        class="primary-button"
        type="button"
        :disabled="startSaving"
        @click="$emit('start')"
      >
        {{ startSaving ? 'Starting...' : 'Start reading order' }}
      </button>
      <button
        v-if="selectedOrder && !readOnly && selectedOrder.startedAt"
        class="secondary-button"
        type="button"
        :disabled="startSaving"
        @click="$emit('stop')"
      >
        {{ startSaving ? 'Stopping...' : 'Stop reading' }}
      </button>
      <button
        v-if="selectedOrder?.canEdit"
        class="primary-button"
        type="button"
        @click="$emit('edit')"
      >
        Edit
      </button>
      <button
        v-if="
          selectedOrder &&
          !readOnly &&
          selectedOrder.authorUserId &&
          selectedOrder.authorUserId !== currentUserId
        "
        class="secondary-button"
        type="button"
        :disabled="saving"
        @click="$emit('copy')"
      >
        {{ saving ? 'Copying...' : 'Copy' }}
      </button>
      <button
        v-if="selectedOrder"
        class="secondary-button"
        type="button"
        @click="$emit('export-cbl')"
      >
        Export CBL
      </button>
    </DetailNavigation>

    <article class="detail-panel">
      <div v-if="selectedOrder" class="read-only-detail">
        <header class="panel-header">
          <div>
            <p class="eyebrow">Reading Order</p>
            <h3>{{ selectedOrder.name }}</h3>
          </div>
        </header>

        <div class="reading-order-summary">
          <div v-if="selectedOrder.image" class="reading-order-thumbnail">
            <img
              :src="assetURL(selectedOrder.image)"
              :alt="`${selectedOrder.name} thumbnail`"
              loading="lazy"
            />
          </div>

          <!-- eslint-disable-next-line vue/no-v-html -- markdown-it renders with raw HTML disabled. -->
          <div class="detail-description markdown-content" v-html="renderedDescription"></div>
        </div>

        <div class="progress-meter" aria-label="Reading order progress">
          <span :style="{ width: formatProgress(selectedOrder.progress) }"></span>
        </div>

        <div class="metadata-grid">
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
        </div>

        <section v-if="!readOnly" class="reading-order-rating-panel">
          <div>
            <p class="eyebrow">Your rating</p>
            <strong>{{
              selectedOrder.myRating ? `${selectedOrder.myRating.toFixed(1)} / 5` : 'Not rated'
            }}</strong>
          </div>
          <div class="rating-button-group" role="group" aria-label="Rate reading order">
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
            <button
              type="button"
              class="secondary-button compact-rating-clear"
              :disabled="ratingSaving || !selectedOrder.myRating"
              @click="emit('rate', 0)"
            >
              Clear
            </button>
          </div>
        </section>

        <ComicListView
          class="preview-list"
          title="Comics"
          :comics="selectedOrder.comics"
          :selected-comic-id="selectedComicId"
          :quick-saving-comic-id="quickSavingComicId"
          initial-sort="readingOrder"
          show-reading-order-sort
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

      <div v-else class="empty-state">Select a reading order to view details.</div>
    </article>
  </div>
</template>
