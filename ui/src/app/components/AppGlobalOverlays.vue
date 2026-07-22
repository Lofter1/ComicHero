<script setup>
import ErrorToast from '@/shared/components/feedback/ErrorToast.vue'
import MetronImportMonitor from '@/features/metron/components/MetronImportMonitor.vue'

defineProps({
  error: { type: String, default: '' },
  isAdmin: { type: Boolean, default: false },
  conflictMessage: { type: String, default: '' },
  mergeSaving: { type: Boolean, default: false },
  jobs: { type: Array, default: () => [] },
})

const open = defineModel('open', { type: Boolean, default: false })

defineEmits(['merge', 'dismiss-error', 'retry', 'continue', 'cancel', 'dismiss-job'])
</script>

<template>
  <ErrorToast
    :message="error"
    :action-label="isAdmin && conflictMessage === error ? 'Merge now' : ''"
    :action-busy="mergeSaving"
    @action="$emit('merge')"
    @dismiss="$emit('dismiss-error')"
  />

  <MetronImportMonitor
    v-model:open="open"
    :jobs="jobs"
    @retry="$emit('retry', $event)"
    @continue="$emit('continue', $event)"
    @cancel="$emit('cancel', $event)"
    @dismiss="$emit('dismiss-job', $event)"
  />
</template>
