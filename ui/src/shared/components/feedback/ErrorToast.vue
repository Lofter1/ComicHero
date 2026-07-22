<script setup>
defineProps({
  message: {
    type: String,
    default: '',
  },
  actionLabel: {
    type: String,
    default: '',
  },
  actionBusy: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['action', 'dismiss'])
</script>

<template>
  <div v-if="message" class="error-toast" role="alert" aria-live="assertive">
    <span>{{ message }}</span>
    <span class="error-toast-actions">
      <button
        v-if="actionLabel"
        class="action-button"
        type="button"
        :disabled="actionBusy"
        @click="$emit('action')"
      >
        {{ actionBusy ? 'Merging...' : actionLabel }}
      </button>
      <!-- Native button: toast dismissal inherits the toast's compact contextual styling. -->
      <button
        class="dismiss-button"
        type="button"
        aria-label="Dismiss error"
        :disabled="actionBusy"
        @click="$emit('dismiss')"
      >
        Dismiss
      </button>
    </span>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.error-toast {
  @apply fixed right-6 bottom-6 z-50 flex w-[min(420px,calc(100vw-32px))] flex-col items-stretch gap-3 rounded border border-danger-border bg-danger-soft px-3.5 py-3 text-danger shadow-monitor;
}

.error-toast > span {
  @apply min-w-0 flex-auto;
  overflow-wrap: anywhere;
}

.error-toast-actions {
  @apply flex flex-none items-center justify-end gap-3;
}

.action-button,
.dismiss-button {
  @apply min-h-0 flex-none border-0 bg-transparent p-0 text-sm font-extrabold text-inherit;
}

.action-button {
  @apply underline underline-offset-2;
}
</style>
