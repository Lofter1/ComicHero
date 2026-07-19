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
    class="loading-state grid place-items-center w-full [min-height:min(520px,_calc(100dvh_-_180px))] text-label [&.compact]:min-h-40"
    :class="{ compact }"
    role="status"
    aria-live="polite"
    aria-label="Loading"
  >
    <span class="loading-state-content flex flex-col items-center gap-3 [&_strong]:text-base">
      <span
        class="loading-spinner w-4 h-4 border-3 border-spinner-track border-t-primary rounded-full animate-loading-spin flex-none [&.small]:w-3.5 [&.small]:h-3.5 [&.small]:[border-width:2px]"
        aria-hidden="true"
      ></span>
      <strong>Loading</strong>
    </span>
  </div>
</template>
