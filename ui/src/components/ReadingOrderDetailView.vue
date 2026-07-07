<script setup>
import { assetURL } from '@/api/client.js'
import ComicListView from '@/components/ComicListView.vue'
import { formatProgress } from '@/domain/readingOrders.js'

defineProps({
  selectedOrder: {
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
})

defineEmits(['back', 'edit', 'export-cbl', 'open-comic', 'toggle-read'])
</script>

<template>
  <div class="detail-view">
    <header class="detail-nav sticky-toolbar">
      <button class="secondary-button" type="button" @click="$emit('back')">Back</button>

      <div class="detail-nav-actions">
        <button v-if="selectedOrder" class="secondary-button" type="button" @click="$emit('export-cbl')">Export CBL</button>
        <button v-if="selectedOrder?.canEdit" class="primary-button" type="button" @click="$emit('edit')">Edit</button>
      </div>
    </header>

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
            <img :src="assetURL(selectedOrder.image)" :alt="`${selectedOrder.name} thumbnail`" loading="lazy" />
          </div>

          <p class="detail-description">{{ selectedOrder.description || 'No description' }}</p>
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
          <span v-if="selectedOrder.authorName">
            <strong>{{ selectedOrder.authorName }}</strong>
            <small>Author</small>
          </span>
        </div>

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
          paginate-local
          empty-message="No comics in this reading order yet."
          filtered-empty-message="No comics match these filters."
          @open-comic="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
        />
      </div>

      <div v-else class="empty-state">Select a reading order to view details.</div>
    </article>
  </div>
</template>
