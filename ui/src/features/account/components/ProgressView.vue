<script setup>
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
  <section class="browse-view progress-view">
    <div v-if="loading && !statisticsView" class="empty-state">Loading progress...</div>
    <div v-else-if="error" class="empty-state">
      {{ error }}
    </div>
    <template v-else-if="statisticsView">
      <article class="progress-summary-panel">
        <div class="progress-section-heading">
          <div>
            <p class="eyebrow">Progress</p>
            <h3>Reading progress</h3>
          </div>
          <button
            class="secondary-button compact-button"
            type="button"
            :disabled="loading"
            @click="emit('refresh')"
          >
            {{ loading ? 'Refreshing...' : 'Refresh' }}
          </button>
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
          <p class="eyebrow">Achievements</p>
          <h3>Milestones</h3>
        </div>

        <div v-if="statisticsView.achievements?.length" class="achievement-grid">
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
        <div v-else class="empty-state">No achievements yet.</div>
      </article>
    </template>
  </section>
</template>
