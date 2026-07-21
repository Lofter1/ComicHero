<script setup>
import { ref } from 'vue'

defineProps({ loading: { type: Boolean, default: false } })

const element = ref(null)
defineExpose({ element })
</script>

<template>
  <div ref="element" class="infinite-scroll-status" aria-live="polite">
    <span v-if="loading" class="loading-spinner" aria-hidden="true"></span>
    <span>{{ loading ? 'Loading more...' : 'Scroll for more' }}</span>
  </div>
</template>

<style scoped>
@reference '../../styles.css';

.infinite-scroll-status {
  @apply flex items-center gap-2 w-full min-h-10 text-muted text-sm font-bold;
}

.loading-spinner {
  @apply w-3.5 h-3.5 flex-none border-2 border-spinner-track border-t-primary rounded-full animate-loading-spin;
}
</style>
