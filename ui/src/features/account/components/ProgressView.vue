<script setup>
import LoadingState from '@/shared/components/feedback/LoadingState.vue'
import EmptyState from '@/shared/components/feedback/EmptyState.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'

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
    <EmptyState v-else-if="error">
      {{ error }}
    </EmptyState>
    <template v-else-if="statisticsView">
      <article class="progress-summary-panel">
        <div class="progress-section-heading">
          <div>
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Progress</p>
            <h3>Reading progress</h3>
          </div>
          <BaseButton size="compact" :disabled="loading" @click="emit('refresh')">
            {{ loading ? 'Refreshing...' : 'Refresh' }}
          </BaseButton>
        </div>

        <div class="metadata-grid progress-stat-grid">
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

      <article class="progress-section-panel">
        <div class="metadata-grid progress-stat-grid">
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

      <article class="progress-section-panel">
        <div class="progress-section-heading">
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Achievements</p>
          <h3>Milestones</h3>
        </div>

        <div
          v-if="statisticsView.achievements?.length"
          class="achievement-grid grid grid-cols-[repeat(auto-fit,minmax(220px,1fr))] gap-2.5"
        >
          <article
            v-for="achievement in statisticsView.achievements"
            :key="achievement.id"
            class="achievement-card"
            :class="{ earned: achievement.earned }"
          >
            <div class="achievement-card-heading">
              <span class="achievement-badge" aria-hidden="true">{{
                achievement.earned ? 'OK' : '--'
              }}</span>
              <div>
                <strong>{{ achievement.name }}</strong>
                <small>{{ achievement.category }}</small>
              </div>
            </div>
            <p>{{ achievement.description }}</p>
            <div class="achievement-progress">
              <span
                >{{ Math.min(achievement.progress, achievement.target) }} /
                {{ achievement.target }}</span
              >
              <div class="progress-track" aria-hidden="true">
                <span :style="{ width: percentLabel(achievement.percent) }"></span>
              </div>
              <small>{{ achievementTimestampLabel(achievement) }}</small>
            </div>
          </article>
        </div>
        <EmptyState v-else> No achievements yet. </EmptyState>
      </article>
    </template>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.progress-summary-panel {
  @apply grid-cols-1 grid gap-4 border border-line rounded bg-surface-soft p-4 down-tablet:grid-cols-1;
}

.progress-section-heading {
  @apply flex items-center justify-between gap-3 min-w-0;
}

.metadata-grid.progress-stat-grid {
  @apply grid-cols-[repeat(auto-fit,minmax(150px,1fr))] grid gap-2.5 [&_span]:border [&_span]:border-line [&_span]:rounded [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_small]:block [&_small]:text-muted [&_small]:mt-1 down-tablet:grid-cols-1;
}

.achievement-card {
  @apply grid gap-2.5 content-start border border-line rounded bg-surface p-3 text-muted [&.earned]:border-[color-mix(in_srgb,var(--accent)_45%,var(--line))] [&.earned]:[background:color-mix(in_srgb,var(--accent)_12%,var(--surface))] [&.earned]:text-ink [&.earned_.achievement-badge]:border-(--accent) [&.earned_.achievement-badge]:[background:var(--accent)] [&.earned_.achievement-badge]:text-(--surface) [&_p]:m-0;
}

.achievement-card-heading {
  @apply grid grid-cols-[auto_minmax(0,1fr)] gap-2.5 items-center [&_strong]:block [&_small]:block [&_small]:text-muted [&_small]:mt-0.5;
}

.achievement-badge {
  @apply grid place-items-center w-8 h-8 border border-line rounded-full bg-surface-soft text-muted text-xs font-black;
}

.achievement-progress {
  @apply grid gap-1.5 text-sm font-extrabold [&_small]:block [&_small]:text-muted [&_small]:mt-1;
}

.progress-track {
  @apply overflow-hidden h-2 rounded-full bg-surface-muted [&_span]:block [&_span]:w-0 [&_span]:h-full [&_span]:rounded-[inherit] [&_span]:[background:var(--accent)] [&_span]:[transition:width_0.2s_ease];
}

.progress-section-heading h3 {
  overflow-wrap: anywhere;
}

.metadata-grid.progress-stat-grid strong {
  overflow-wrap: anywhere;
}

.progress-section-panel {
  @apply grid gap-4 rounded border border-line bg-surface-soft p-4;
}
</style>
