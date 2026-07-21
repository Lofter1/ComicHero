<script setup>
defineProps({
  message: {
    type: String,
    default: '',
  },
})

defineEmits(['dismiss'])
</script>

<template>
  <div v-if="message" class="error-toast" role="alert" aria-live="assertive">
    <span>{{ message }}</span>
    <!-- Native button: toast dismissal inherits the toast's compact contextual styling. -->
    <button
      class="dismiss-button"
      type="button"
      aria-label="Dismiss error"
      @click="$emit('dismiss')"
    >
      Dismiss
    </button>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.error-toast {
  @apply fixed right-6 bottom-6 z-50 flex w-[min(420px,calc(100vw-32px))] items-start gap-3 rounded border border-danger-border bg-danger-soft px-3.5 py-3 text-danger shadow-monitor;
}

.error-toast > span {
  @apply min-w-0 flex-auto;
  overflow-wrap: anywhere;
}

.dismiss-button {
  @apply min-h-0 flex-none border-0 bg-transparent p-0 text-sm font-extrabold text-inherit;
}
</style>
