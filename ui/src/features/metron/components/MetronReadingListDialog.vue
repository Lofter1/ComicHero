<script setup>
import { assetURL } from '@/api/client.js'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'

defineProps({
  readingList: {
    type: Object,
    default: null,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  error: {
    type: String,
    default: '',
  },
  importing: {
    type: Boolean,
    default: false,
  },
  summary: {
    type: String,
    default: '',
  },
})

defineEmits(['close', 'import'])

function formatDate(value) {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <section
      class="metron-detail-dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="reading-list-detail-title"
    >
      <header class="metron-detail-header">
        <span>
          <strong id="reading-list-detail-title">{{ readingList?.name || 'Reading list' }}</strong>
          <small>{{ summary }}</small>
        </span>
        <button
          class="icon-button"
          type="button"
          aria-label="Close reading list detail"
          @click="$emit('close')"
        >
          ×
        </button>
      </header>
      <div class="metron-detail-body">
        <img
          v-if="readingList?.image"
          class="metron-detail-image"
          :src="assetURL(readingList.image)"
          :alt="readingList.name || 'Reading list image'"
        />
        <div class="metron-detail-copy">
          <LoadingState v-if="loading" compact />
          <p v-else-if="error" class="error-text">{{ error }}</p>
          <p v-else>{{ readingList?.description || 'No description from Metron.' }}</p>
          <dl class="metron-detail-facts">
            <div v-if="readingList?.user?.username">
              <dt>User</dt>
              <dd>{{ readingList.user.username }}</dd>
            </div>
            <div v-if="readingList?.listType">
              <dt>Type</dt>
              <dd>{{ readingList.listType }}</dd>
            </div>
            <div v-if="readingList?.attributionSource">
              <dt>Source</dt>
              <dd>{{ readingList.attributionSource }}</dd>
            </div>
            <div v-if="readingList?.ratingCount">
              <dt>Rating</dt>
              <dd>{{ readingList.averageRating || 0 }} from {{ readingList.ratingCount }}</dd>
            </div>
            <div v-if="readingList?.modified">
              <dt>Modified</dt>
              <dd>{{ formatDate(readingList.modified) }}</dd>
            </div>
            <div v-if="readingList?.issues?.length">
              <dt>Issues</dt>
              <dd>{{ readingList.issues.length }}</dd>
            </div>
          </dl>
        </div>
      </div>
      <footer class="metron-detail-actions">
        <button class="secondary-button" type="button" @click="$emit('close')">Close</button>
        <button
          class="primary-button"
          type="button"
          :disabled="!readingList || importing"
          @click="$emit('import')"
        >
          {{ importing ? 'Importing...' : 'Import' }}
        </button>
      </footer>
    </section>
  </div>
</template>
