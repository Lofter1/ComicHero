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
  <div
    class="modal-backdrop fixed inset-0 z-60 grid place-items-center bg-backdrop p-4"
    @click.self="$emit('close')"
  >
    <section
      class="metron-detail-dialog [width:min(760px,_100%)] [max-height:min(760px,_calc(100vh_-_36px))] grid [grid-template-rows:auto_minmax(0,_1fr)_auto] border border-line-strong rounded bg-panel shadow-overlay overflow-hidden"
      role="dialog"
      aria-modal="true"
      aria-labelledby="reading-list-detail-title"
    >
      <header
        class="metron-detail-header border-b border-line flex items-start justify-between gap-3 py-3.5 px-4 [&_span]:min-w-0 [&_strong]:block [&_strong]:break-anywhere [&_small]:block [&_small]:break-anywhere [&_small]:text-muted [&_small]:mt-1"
      >
        <span>
          <strong id="reading-list-detail-title">{{ readingList?.name || 'Reading list' }}</strong>
          <small>{{ summary }}</small>
        </span>
        <button
          class="icon-button min-h-10 border border-line-strong rounded bg-surface text-control self-end py-0 px-3 down-mobile:self-stretch down-mobile:w-full"
          type="button"
          aria-label="Close reading list detail"
          @click="$emit('close')"
        >
          ×
        </button>
      </header>
      <div
        class="metron-detail-body min-h-0 overflow-auto grid [grid-template-columns:minmax(140px,_210px)_minmax(0,_1fr)] gap-4 p-4 down-mobile:grid-cols-1"
      >
        <img
          v-if="readingList?.image"
          class="metron-detail-image w-full aspect-portrait border border-line rounded object-cover bg-surface-muted down-mobile:max-w-56"
          :src="assetURL(readingList.image)"
          :alt="readingList.name || 'Reading list image'"
        />
        <div class="metron-detail-copy min-w-0 text-body [&_p]:break-anywhere">
          <LoadingState v-if="loading" compact />
          <p v-else-if="error" class="error-text text-danger font-bold">{{ error }}</p>
          <p v-else>{{ readingList?.description || 'No description from Metron.' }}</p>
          <dl
            class="metron-detail-facts grid grid-cols-2 gap-y-2.5 gap-x-3.5 mt-3.5 mx-0 mb-0 [&_div]:min-w-0 [&_dt]:text-muted [&_dt]:text-xs [&_dt]:font-extrabold [&_dt]:uppercase [&_dd]:mt-1 [&_dd]:mx-0 [&_dd]:mb-0 [&_dd]:break-anywhere [&_dd]:font-bold down-mobile:grid-cols-1"
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
      <footer
        class="metron-detail-actions justify-end border-t border-line flex items-start gap-3 py-3.5 px-4"
      >
        <button
          class="secondary-button min-h-10 border rounded text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]"
          type="button"
          @click="$emit('close')"
        >
          Close
        </button>
        <button
          class="primary-button min-h-10 border rounded py-2.5 px-3.5 border-primary bg-primary text-white"
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
