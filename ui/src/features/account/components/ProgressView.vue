<script setup>
import LoadingState from '@/shared/components/feedback/LoadingState.vue'

defineProps({
  statisticsView: {
    type: Object,
    default: null,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  error: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['refresh'])

function percentLabel(value) {
  const numeric = Number(value || 0)
  return `${Math.round(numeric * 100)}%`
}

function formatTimestamp(value) {
  if (!value) return 'Not yet'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString([], {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function achievementTimestampLabel(achievement) {
  if (!achievement.earned) return 'Not earned yet'
  if (!achievement.earnedAt) return 'Earned, timestamp unavailable'
  return `Earned ${formatTimestamp(achievement.earnedAt)}`
}
</script>

<template>
  <section class="browse-view progress-view grid gap-4 max-w-content min-w-0 w-full">
    <LoadingState v-if="loading && !statisticsView" />
    <div
      v-else-if="error"
      class="empty-state grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4"
    >
      {{ error }}
    </div>
    <template v-else-if="statisticsView">
      <article
        class="progress-summary-panel grid-cols-1 grid gap-4 border border-line rounded bg-surface-soft p-4 down-tablet:grid-cols-1"
      >
        <div
          class="progress-section-heading flex items-center justify-between gap-3 min-w-0 [&_h3]:break-anywhere"
        >
          <div>
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Progress</p>
            <h3>Reading progress</h3>
          </div>
          <button
            class="secondary-button compact-button min-h-9 py-2 px-2.5 border rounded text-control bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]"
            type="button"
            :disabled="loading"
            @click="emit('refresh')"
          >
            {{ loading ? 'Refreshing...' : 'Refresh' }}
          </button>
        </div>

        <div
          class="metadata-grid progress-stat-grid [grid-template-columns:repeat(auto-fit,_minmax(150px,_1fr))] grid gap-2.5 [&_span]:border [&_span]:border-line [&_span]:rounded [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_strong]:break-anywhere [&_small]:block [&_small]:text-muted [&_small]:mt-1 down-tablet:grid-cols-1"
        >
          <span>
            <strong>{{ statisticsView.statistics.readComics }}</strong>
            <small>Read comics</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.skippedComics }}</strong>
            <small>Skipped comics</small>
          </span>
          <span>
            <strong>{{ formatTimestamp(statisticsView.statistics.firstReadAt) }}</strong>
            <small>First read timestamp</small>
          </span>
          <span>
            <strong>{{ formatTimestamp(statisticsView.statistics.lastReadAt) }}</strong>
            <small>Latest read timestamp</small>
          </span>
        </div>
      </article>

      <article
        class="progress-section-panel grid gap-4 border border-line rounded bg-surface-soft p-4"
      >
        <div
          class="metadata-grid progress-stat-grid [grid-template-columns:repeat(auto-fit,_minmax(150px,_1fr))] grid gap-2.5 [&_span]:border [&_span]:border-line [&_span]:rounded [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_strong]:break-anywhere [&_small]:block [&_small]:text-muted [&_small]:mt-1 down-tablet:grid-cols-1"
        >
          <span>
            <strong>{{ statisticsView.statistics.distinctReadSeries }}</strong>
            <small>Series read</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.completedSeries }}</strong>
            <small>Series completed</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.distinctReadPublishers }}</strong>
            <small>Publishers read</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.completedReadingOrders }}</strong>
            <small>Reading orders completed</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.startedReadingOrders }}</strong>
            <small>Reading orders started</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.completedArcs }}</strong>
            <small>Story arcs completed</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.startedArcs }}</strong>
            <small>Story arcs started</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.startedSeries }}</strong>
            <small>Series started</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.startedCharacters }}</strong>
            <small>Character paths started</small>
          </span>
          <span>
            <strong>{{ statisticsView.statistics.charactersMet }}</strong>
            <small>Characters met</small>
          </span>
        </div>
      </article>

      <article
        class="progress-section-panel grid gap-4 border border-line rounded bg-surface-soft p-4"
      >
        <div
          class="progress-section-heading flex items-center justify-between gap-3 min-w-0 [&_h3]:break-anywhere"
        >
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Achievements</p>
          <h3>Milestones</h3>
        </div>

        <div
          v-if="statisticsView.achievements?.length"
          class="achievement-grid grid [grid-template-columns:repeat(auto-fit,_minmax(220px,_1fr))] gap-2.5"
        >
          <article
            v-for="achievement in statisticsView.achievements"
            :key="achievement.id"
            class="achievement-card grid gap-2.5 content-start border border-line rounded bg-surface p-3 text-muted [&.earned]:[border-color:color-mix(in_srgb,_var(--accent)_45%,_var(--line))] [&.earned]:[background:color-mix(in_srgb,_var(--accent)_12%,_var(--surface))] [&.earned]:text-ink [&.earned_.achievement-badge]:[border-color:var(--accent)] [&.earned_.achievement-badge]:[background:var(--accent)] [&.earned_.achievement-badge]:[color:var(--surface)] [&_p]:m-0"
            :class="{ earned: achievement.earned }"
          >
            <div
              class="achievement-card-heading grid [grid-template-columns:auto_minmax(0,_1fr)] gap-2.5 items-center [&_strong]:block [&_small]:block [&_small]:text-muted [&_small]:mt-0.5"
            >
              <span
                class="achievement-badge grid place-items-center w-8 h-8 border border-line rounded-full bg-surface-soft text-muted text-xs font-black"
                aria-hidden="true"
                >{{ achievement.earned ? 'OK' : '--' }}</span
              >
              <div>
                <strong>{{ achievement.name }}</strong>
                <small>{{ achievement.category }}</small>
              </div>
            </div>
            <p>{{ achievement.description }}</p>
            <div
              class="achievement-progress grid gap-1.5 text-sm font-extrabold [&_small]:block [&_small]:text-muted [&_small]:mt-1"
            >
              <span
                >{{ Math.min(achievement.progress, achievement.target) }} /
                {{ achievement.target }}</span
              >
              <div
                class="progress-track overflow-hidden h-2 rounded-full bg-surface-muted [&_span]:block [&_span]:[width:0] [&_span]:h-full [&_span]:[border-radius:inherit] [&_span]:[background:var(--accent)] [&_span]:[transition:width_0.2s_ease]"
                aria-hidden="true"
              >
                <span :style="{ width: percentLabel(achievement.percent) }"></span>
              </div>
              <small>{{ achievementTimestampLabel(achievement) }}</small>
            </div>
          </article>
        </div>
        <div
          v-else
          class="empty-state grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4"
        >
          No achievements yet.
        </div>
      </article>
    </template>
  </section>
</template>
