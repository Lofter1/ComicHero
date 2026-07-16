<script setup>
import { computed } from 'vue'
import { assetURL } from '@/api/client.js'
import { formatProgress } from '@/features/reading-orders/model.js'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'

const props = defineProps({
  dashboard: {
    type: Object,
    default: null,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  quickSavingComicId: {
    type: Number,
    default: null,
  },
  readOnly: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['refresh', 'open-comic', 'mark-read', 'mark-skipped'])

const items = computed(() => props.dashboard?.items || [])
const recentAchievement = computed(() => props.dashboard?.achievements?.recent || null)
const nextAchievement = computed(() => props.dashboard?.achievements?.next || null)

function itemTypeLabel(type) {
  if (type === 'readingOrder') return 'Reading order'
  if (type === 'arc') return 'Arc'
  if (type === 'character') return 'Character'
  if (type === 'series') return 'Series'
  return 'Started'
}

function achievementProgress(achievement) {
  if (!achievement) return ''
  return `${achievement.progress} / ${achievement.target}`
}
</script>

<template>
  <section class="dashboard-view">
    <header class="dashboard-header">
      <div>
        <h2>Continue reading</h2>
      </div>
      <button class="secondary-button" type="button" :disabled="loading" @click="$emit('refresh')">
        {{ loading ? 'Refreshing...' : 'Refresh' }}
      </button>
    </header>

    <div class="dashboard-achievements">
      <article class="achievement-summary-card">
        <p class="eyebrow">Recently earned</p>
        <template v-if="recentAchievement">
          <h3>{{ recentAchievement.name }}</h3>
          <p>{{ recentAchievement.description }}</p>
        </template>
        <p v-else class="muted">No achievements earned yet.</p>
      </article>
      <article class="achievement-summary-card">
        <p class="eyebrow">Next achievement</p>
        <template v-if="nextAchievement">
          <h3>{{ nextAchievement.name }}</h3>
          <p>{{ nextAchievement.description }}</p>
          <div class="progress-meter" :aria-label="`${nextAchievement.name} progress`">
            <span :style="{ width: formatProgress(nextAchievement.percent) }"></span>
          </div>
          <small>{{ achievementProgress(nextAchievement) }}</small>
        </template>
        <p v-else class="muted">All achievements earned.</p>
      </article>
    </div>

    <LoadingState v-if="loading && !dashboard" />

    <div v-else-if="items.length" class="dashboard-grid">
      <article v-for="item in items" :key="`${item.type}:${item.id}`" class="dashboard-card">
        <div class="dashboard-card-header">
          <div>
            <p class="eyebrow">{{ itemTypeLabel(item.type) }}</p>
            <h3>{{ item.name }}</h3>
          </div>
          <strong>{{ formatProgress(item.progress) }}</strong>
        </div>

        <template v-if="item.nextComic">
          <button
            class="dashboard-comic"
            type="button"
            @click="$emit('open-comic', item.nextComic)"
          >
            <img
              v-if="item.nextComic.coverImage"
              :src="assetURL(item.nextComic.coverImage)"
              :alt="`${item.nextComic.title} cover`"
              loading="lazy"
            />
            <span v-else class="dashboard-cover-placeholder" aria-hidden="true"></span>
            <span>
              <strong>{{ item.nextComic.title }}</strong>
              <small>{{ item.nextComic.publisher || 'No publisher saved' }}</small>
            </span>
          </button>

          <div v-if="!readOnly" class="dashboard-card-actions">
            <button
              class="primary-button"
              type="button"
              :disabled="quickSavingComicId === item.nextComic.id"
              @click="$emit('mark-read', item.nextComic)"
            >
              Read
            </button>
            <button
              class="secondary-button"
              type="button"
              :disabled="quickSavingComicId === item.nextComic.id"
              @click="$emit('mark-skipped', item.nextComic)"
            >
              Skipped
            </button>
          </div>
        </template>

        <p v-else class="dashboard-complete-copy">No unread comics left here.</p>
      </article>
    </div>

    <section v-else class="empty-panel">
      <h2>No started reading yet</h2>
      <p>Start a reading order, arc, character, or series and it will appear here.</p>
    </section>
  </section>
</template>
