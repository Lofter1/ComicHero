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
      class="metron-detail-dialog [width:min(760px,_100%)] [max-height:min(760px,_calc(100vh_-_36px))] grid [grid-template-rows:auto_minmax(0,_1fr)_auto] border border-line-strong rounded [background:var(--panel-bg)] [box-shadow:0_22px_56px_var(--shadow-panel)] overflow-hidden"
      role="dialog"
      aria-modal="true"
      aria-labelledby="reading-list-detail-title"
    >
      <header class="metron-detail-header [border-bottom:1px_solid_var(--line)]">
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
      <div
        class="metron-detail-body [min-height:0] overflow-auto grid [grid-template-columns:minmax(140px,_210px)_minmax(0,_1fr)] gap-4 p-4"
      >
        <img
          v-if="readingList?.image"
          class="metron-detail-image w-full [aspect-ratio:2_/_3] border border-line rounded object-cover bg-surface-muted down-mobile:[max-width:220px]"
          :src="assetURL(readingList.image)"
          :alt="readingList.name || 'Reading list image'"
        />
        <div class="metron-detail-copy min-w-0 text-body">
          <LoadingState v-if="loading" compact />
          <p v-else-if="error" class="error-text">{{ error }}</p>
          <p v-else>{{ readingList?.description || 'No description from Metron.' }}</p>
          <dl
            class="metron-detail-facts grid [grid-template-columns:repeat(2,_minmax(0,_1fr))] [gap:10px_14px] [margin:14px_0_0]"
          >
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
      <footer class="metron-detail-actions justify-end [border-top:1px_solid_var(--line)]">
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
