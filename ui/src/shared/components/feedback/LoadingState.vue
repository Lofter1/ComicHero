<script setup>
defineProps({
  compact: {
    type: Boolean,
    default: false,
  },
})
</script>

<template>
  <div
    class="loading-state"
    :class="{ compact }"
    role="status"
    aria-live="polite"
    aria-label="Loading"
  >
    <span class="loading-state-content">
      <span class="loading-spinner" aria-hidden="true"></span>
      <strong>Loading</strong>
    </span>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.loading-state {
  @apply grid min-h-[min(520px,calc(100dvh-180px))] w-full place-items-center text-label;
}

.loading-state.compact {
  @apply min-h-40;
}

.loading-state-content {
  @apply flex flex-col items-center gap-3;
}

.loading-state-content strong {
  @apply text-base;
}

.loading-spinner {
  @apply size-4 flex-none animate-loading-spin rounded-full border-3 border-spinner-track border-t-primary;
}
</style>
