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
