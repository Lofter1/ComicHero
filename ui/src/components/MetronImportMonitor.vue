<script setup>
import { computed } from 'vue'

const props = defineProps({
  jobs: {
    type: Array,
    default: () => [],
  },
  open: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:open', 'retry', 'continue', 'cancel', 'dismiss'])

const inProgress = computed(() => props.jobs.some(isActiveJob))
const summary = computed(() => {
  const running = props.jobs.filter(isActiveJob).length
  if (running > 0) return `${running} running`
  const latest = props.jobs[0]
  return latest ? latest.status : ''
})

function updateOpen(value) {
  emit('update:open', value)
}

function canCancel(job) {
  return job.status === 'queued' || job.status === 'running'
}

function canDismiss(job) {
  return job.status === 'succeeded' || job.status === 'failed' || job.status === 'canceled'
}

function canContinue(job) {
  return job.status === 'canceled'
}

function progressLabel(job) {
  if (!job.total) {
    if (job.status === 'queued') return 'Queued'
    if (job.status === 'canceling') return 'Canceling...'
    if (job.status === 'running') return 'Preparing...'
    return job.status
  }
  if (job.status === 'canceling') return `Canceling at ${job.completed} of ${job.total}`
  return `${job.completed} of ${job.total}`
}

function progressPercent(job) {
  if (!job.total) return job.status === 'succeeded' ? 100 : 0
  return Math.min(100, Math.round((job.completed / job.total) * 100))
}

function progressIndeterminate(job) {
  return isActiveJob(job) && !job.total
}

function isActiveJob(job) {
  return job.status === 'queued' || job.status === 'running' || job.status === 'canceling'
}

function jobTitle(job) {
  const type =
    job.type === 'readingList'
      ? 'Reading list'
      : job.type === 'character'
        ? 'Character'
        : job.type === 'arc'
          ? 'Arc'
          : job.type
  return job.displayName ? `${type} import for ${job.displayName}` : `${type} import`
}

function jobMessage(job) {
  if (job.status === 'canceled') {
    return `${jobTitle(job)} was canceled.`
  }
  return job.message
}
</script>

<template>
  <div v-if="jobs.length" class="import-monitor" :class="{ collapsed: !open }" aria-live="polite">
    <header>
      <button
        class="import-monitor-toggle"
        type="button"
        :aria-expanded="open"
        @click="updateOpen(!open)"
      >
        <strong>Metron imports</strong>
        <small>{{ summary }}</small>
      </button>
      <small v-if="inProgress && open">Running in background</small>
    </header>
    <div v-if="open" class="metron-jobs">
      <div v-for="job in jobs" :key="job.id" class="metron-job" :class="job.status">
        <span>
          <strong>{{ jobTitle(job) }}</strong>
          <small>{{ jobMessage(job) }}</small>
          <small>{{ progressLabel(job) }}</small>
          <span
            class="job-progress"
            :class="{ indeterminate: progressIndeterminate(job) }"
            aria-hidden="true"
          >
            <span :style="{ width: `${progressPercent(job)}%` }"></span>
          </span>
        </span>
        <span class="job-actions">
          <span class="status-pill">{{ job.status }}</span>
          <button
            v-if="job.status === 'failed'"
            class="icon-button compact-icon-button"
            type="button"
            aria-label="Retry import"
            title="Retry import"
            @click="$emit('retry', job)"
          >
            <span aria-hidden="true" class="button-icon">↻</span>
          </button>
          <button
            v-if="canContinue(job)"
            class="ghost-button"
            type="button"
            @click="$emit('continue', job)"
          >
            Continue
          </button>
          <button
            v-if="canCancel(job)"
            class="ghost-button"
            type="button"
            @click="$emit('cancel', job.id)"
          >
            Cancel
          </button>
          <button
            v-if="canDismiss(job)"
            class="icon-button compact-icon-button"
            type="button"
            aria-label="Dismiss import"
            title="Dismiss import"
            @click="$emit('dismiss', job.id)"
          >
            <span aria-hidden="true" class="button-icon">×</span>
          </button>
        </span>
      </div>
    </div>
  </div>
</template>
