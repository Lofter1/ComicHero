<script setup>
import { useOnlineStatus } from '@/shared/composables/useOnlineStatus.js'

const { isOnline } = useOnlineStatus()
</script>

<template>
  <div v-if="!isOnline" class="offline-banner" role="status" aria-live="polite">
    <span>You're offline. Showing saved data — changes will sync once you're back online.</span>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.offline-banner {
  @apply fixed left-6 bottom-6 z-50 w-[min(420px,calc(100vw-32px))] rounded border border-line-strong bg-panel px-3.5 py-3 text-sm text-muted shadow-monitor;
}
</style>
