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
    job.type === 'readingLists'
      ? 'Reading lists'
      : job.type === 'readingList'
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
  <div
    v-if="jobs.length"
    class="import-monitor fixed right-6 bottom-6 z-45 grid gap-2.5 [width:min(440px,_calc(100vw_-_32px))] border border-line-strong rounded bg-panel shadow-monitor p-3 down-tablet:right-4 down-tablet:bottom-4 down-mobile:right-2.5 down-mobile:bottom-2.5 down-mobile:[width:calc(100vw_-_20px)] down-mobile:[max-height:42vh] down-mobile:overflow-auto [&.collapsed]:w-auto [&.collapsed]:[max-width:min(300px,_calc(100vw_-_32px))] [&.collapsed]:py-2 [&.collapsed]:px-2.5 [&_header]:flex [&_header]:items-baseline [&_header]:justify-between [&_header]:gap-3 [&_header_small]:text-muted [&_header_small]:font-bold down-mobile:[&.collapsed]:w-auto down-mobile:[&.collapsed]:[max-width:calc(100vw_-_20px)]"
    :class="{ collapsed: !open }"
    aria-live="polite"
  >
    <header>
      <button
        class="import-monitor-toggle flex items-baseline gap-2.5 min-h-0 border-0 bg-transparent text-inherit p-0 text-left [&_strong]:overflow-hidden [&_strong]:text-ellipsis [&_strong]:whitespace-nowrap [&_small]:overflow-hidden [&_small]:text-ellipsis [&_small]:whitespace-nowrap"
        type="button"
        :aria-expanded="open"
        @click="updateOpen(!open)"
      >
        <strong>Metron imports</strong>
        <small>{{ summary }}</small>
      </button>
      <small v-if="inProgress && open">Running in background</small>
    </header>
    <div
      v-if="open"
      class="metron-jobs grid gap-2 [max-height:min(54vh,_420px)] overflow-auto pr-0.5"
    >
      <div
        v-for="job in jobs"
        :key="job.id"
        class="metron-job flex items-start justify-between gap-3 border border-line-strong rounded bg-surface-soft py-2.5 px-3 [&.failed]:border-danger-border [&.failed]:bg-danger-soft [&.canceled]:[border-color:#d8c38a] [&.canceled]:bg-warning-soft [&.succeeded]:[border-color:color-mix(in_srgb,_var(--primary)_35%,_var(--line-strong))] [&_span:first-child]:min-w-0 [&_strong]:block [&_small]:block [&_small]:text-muted [&_small]:mt-0.75 down-mobile:items-stretch down-mobile:flex-col"
        :class="job.status"
      >
        <span>
          <strong>{{ jobTitle(job) }}</strong>
          <small>{{ jobMessage(job) }}</small>
          <small>{{ progressLabel(job) }}</small>
          <span
            class="job-progress block [width:min(260px,_100%)] h-2 rounded-full bg-read-progress overflow-hidden mt-2 [&_span]:block [&_span]:h-full [&_span]:[border-radius:inherit] [&_span]:bg-primary [&_span]:[transition:width_180ms_ease] [&.indeterminate_span]:![width:42%] [&.indeterminate_span]:[min-width:42%] [&.indeterminate_span]:animate-job-progress-sweep"
            :class="{ indeterminate: progressIndeterminate(job) }"
            aria-hidden="true"
          >
            <span :style="{ width: `${progressPercent(job)}%` }"></span>
          </span>
        </span>
        <span class="job-actions flex items-center gap-2 flex-none down-mobile:flex-wrap">
          <span
            class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-compact flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
            >{{ job.status }}</span
          >
          <button
            v-if="job.status === 'failed'"
            class="icon-button compact-icon-button self-center w-8.5 min-w-8.5 min-h-8.5 p-0 min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 self-end py-0 px-3 down-mobile:self-stretch down-mobile:w-full"
            type="button"
            aria-label="Retry import"
            title="Retry import"
            @click="$emit('retry', job)"
          >
            <span
              aria-hidden="true"
              class="button-icon inline-flex items-center justify-center w-em h-em text-xl font-extrabold leading-none"
              >↻</span
            >
          </button>
          <button
            v-if="canContinue(job)"
            class="ghost-button min-h-8.5 border-0 rounded-[7px] bg-transparent text-accent py-1.5 px-2 font-bold"
            type="button"
            @click="$emit('continue', job)"
          >
            Continue
          </button>
          <button
            v-if="canCancel(job)"
            class="ghost-button min-h-8.5 border-0 rounded-[7px] bg-transparent text-accent py-1.5 px-2 font-bold"
            type="button"
            @click="$emit('cancel', job.id)"
          >
            Cancel
          </button>
          <button
            v-if="canDismiss(job)"
            class="icon-button compact-icon-button self-center w-8.5 min-w-8.5 min-h-8.5 p-0 min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 self-end py-0 px-3 down-mobile:self-stretch down-mobile:w-full"
            type="button"
            aria-label="Dismiss import"
            title="Dismiss import"
            @click="$emit('dismiss', job.id)"
          >
            <span
              aria-hidden="true"
              class="button-icon inline-flex items-center justify-center w-em h-em text-xl font-extrabold leading-none"
              >×</span
            >
          </button>
        </span>
      </div>
    </div>
  </div>
</template>
