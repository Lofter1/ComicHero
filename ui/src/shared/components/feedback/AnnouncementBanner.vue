<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  announcement: { type: Object, default: null },
})

const dismissedVersion = ref('')
const storageKey = 'comichero-dismissed-announcement'

function readDismissedVersion() {
  try {
    dismissedVersion.value = localStorage.getItem(storageKey) || ''
  } catch {
    dismissedVersion.value = ''
  }
}

watch(() => props.announcement?.updatedAt, readDismissedVersion, { immediate: true })

const visible = computed(
  () =>
    Boolean(props.announcement?.message) && dismissedVersion.value !== props.announcement.updatedAt,
)

function dismiss() {
  const version = props.announcement?.updatedAt || ''
  dismissedVersion.value = version
  try {
    localStorage.setItem(storageKey, version)
  } catch {
    // Dismissal still applies for this session when storage is unavailable.
  }
}
</script>

<template>
  <div
    v-if="visible"
    class="announcement-banner"
    :class="`announcement-banner--${announcement.level || 'info'}`"
    role="status"
    aria-live="polite"
  >
    <span class="announcement-banner__message">{{ announcement.message }}</span>
    <button type="button" aria-label="Dismiss announcement" @click="dismiss">×</button>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.announcement-banner {
  @apply fixed top-4 left-1/2 z-50 flex w-[min(760px,calc(100vw-32px))] -translate-x-1/2 items-start justify-between gap-4 rounded-lg border px-4 py-3 text-sm font-semibold shadow-monitor;
}

.announcement-banner--info {
  @apply border-primary bg-primary-soft text-primary-strong;
}

.announcement-banner--warning {
  @apply border-warning-border bg-warning-soft text-warning;
}

.announcement-banner--critical {
  @apply border-danger-border bg-danger-soft text-danger;
}

.announcement-banner__message {
  @apply whitespace-pre-wrap;
}

button {
  @apply flex-none text-xl leading-none opacity-70 hover:opacity-100;
}
</style>
