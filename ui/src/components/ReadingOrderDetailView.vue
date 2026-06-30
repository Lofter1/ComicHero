<script setup>
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

defineEmits(['back', 'edit', 'open-comic', 'toggle-read'])
</script>

<template>
  <div class="detail-view">
    <header class="detail-nav sticky-toolbar">
      <button class="secondary-button" type="button" @click="$emit('back')">Back</button>
      <div class="detail-nav-actions">
        <button v-if="selectedOrder" class="primary-button" type="button" @click="$emit('edit')">Edit</button>
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

        <p class="detail-description">{{ selectedOrder.description || 'No description' }}</p>
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
        </div>

        <ComicListView
          class="preview-list"
          title="Comics"
          :comics="selectedOrder.comics"
          :selected-comic-id="selectedComicId"
          :quick-saving-comic-id="quickSavingComicId"
          show-comment
          empty-message="No comics in this reading order yet."
          filtered-empty-message="No comics match these filters."
          @open-comic="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
        />
      </div>
      <p v-else class="empty-state">Select a reading order to view it.</p>
    </article>
  </div>
</template>
