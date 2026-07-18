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
    class="loading-state grid place-items-center w-full [min-height:min(520px,_calc(100dvh_-_180px))] text-label"
    :class="{ compact }"
    role="status"
    aria-live="polite"
    aria-label="Loading"
  >
    <span class="loading-state-content flex flex-col items-center gap-3">
      <span class="loading-spinner" aria-hidden="true"></span>
      <strong>Loading</strong>
    </span>
  </div>
</template>
