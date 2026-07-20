<script setup>
import { computed } from 'vue'
import { assetURL } from '@/api/client.js'
import { formatProgress } from '@/features/reading-orders/model.js'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'

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

defineEmits(['open-comic', 'mark-read', 'mark-skipped'])

const items = computed(() => props.dashboard?.items || [])
const recentAchievement = computed(() => props.dashboard?.achievements?.recent || null)
const nextAchievement = computed(() => props.dashboard?.achievements?.next || null)

function itemTypeLabel(type) {
  if (type === 'readingOrder') return 'Reading order'
  if (type === 'arc') return 'Arc'
  if (type === 'character') return 'Character'
  if (type === 'characterCollection') return 'Character collection'
  if (type === 'series') return 'Series'
  return 'Started'
}

function achievementProgress(achievement) {
  if (!achievement) return ''
  return `${achievement.progress} / ${achievement.target}`
}
</script>

<template>
  <section class="dashboard-view grid gap-4 pt-4">
    <header class="dashboard-header flex items-start justify-between gap-4">
      <div>
        <h2>Continue reading</h2>
      </div>
    </header>

    <LoadingState v-if="loading && !dashboard" />

    <div
      v-else-if="items.length"
      class="dashboard-grid grid grid-cols-3 gap-4 down-laptop:grid-cols-2 down-phone:grid-cols-1! down-mobile:grid-cols-1!"
    >
      <article
        v-for="item in items"
        :key="`${item.type}:${item.id}`"
        class="dashboard-card grid gap-3.5 border border-line rounded bg-surface p-4 [&_h3]:my-1 [&_h3]:mx-2 [&_p]:m-0"
      >
        <div
          class="dashboard-card-header flex items-start justify-between gap-4 [&_strong]:text-accent [&_strong]:whitespace-nowrap"
        >
          <div>
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">
              {{ itemTypeLabel(item.type) }}
            </p>
            <h3>{{ item.name }}</h3>
          </div>
          <strong>{{ formatProgress(item.progress) }}</strong>
        </div>

        <template v-if="item.nextComic">
          <!-- Native button: the comic preview is a full-card navigation target. -->
          <button
            class="dashboard-comic grid grid-cols-[56px_minmax(0,1fr)] gap-3 items-center w-full min-h-20 border border-line rounded [background:var(--surface-strong)] text-inherit p-2.5 text-left [&_img]:w-14 [&_img]:h-20 [&_img]:rounded-[6px] [&_img]:object-cover [&_img]:bg-surface-muted [&_strong]:block [&_small]:block [&_strong]:break-anywhere"
            type="button"
            @click="$emit('open-comic', item.nextComic)"
          >
            <img
              v-if="item.nextComic.coverImage"
              :src="assetURL(item.nextComic.coverImage)"
              :alt="`${item.nextComic.title} cover`"
              loading="lazy"
            />
            <span
              v-else
              class="dashboard-cover-placeholder w-14 h-20 rounded-[6px] object-cover bg-surface-muted"
              aria-hidden="true"
            ></span>
            <span>
              <strong>{{ item.nextComic.title }}</strong>
              <small>{{ item.nextComic.publisher || 'No publisher saved' }}</small>
            </span>
          </button>

          <div v-if="!readOnly" class="dashboard-card-actions grid grid-cols-2 gap-2.5">
            <BaseButton
              variant="primary"
              :disabled="quickSavingComicId === item.nextComic.id"
              @click="$emit('mark-read', item.nextComic)"
            >
              Read
            </BaseButton>
            <BaseButton
              :disabled="quickSavingComicId === item.nextComic.id"
              @click="$emit('mark-skipped', item.nextComic)"
            >
              Skipped
            </BaseButton>
          </div>
        </template>

        <p v-else class="dashboard-complete-copy text-muted">No unread comics left here.</p>
      </article>
    </div>

    <section
      v-else
      class="empty-panel border border-dashed border-line-strong rounded bg-surface-soft text-muted p-5 font-extrabold"
    >
      <h2>No started reading yet</h2>
      <p>
        Start a reading order, arc, character, character collection, or series and it will appear
        here.
      </p>
    </section>

    <div
      class="dashboard-achievements grid grid-cols-[repeat(auto-fit,minmax(260px,1fr))] gap-3.5"
    >
      <article
        class="achievement-summary-card border border-line rounded bg-surface p-4 [&_h3]:my-1 [&_h3]:mx-2 [&_p]:m-0"
      >
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Recently earned</p>
        <template v-if="recentAchievement">
          <h3>{{ recentAchievement.name }}</h3>
          <p>{{ recentAchievement.description }}</p>
        </template>
        <p v-else class="muted block text-muted">No achievements earned yet.</p>
      </article>
      <article
        class="achievement-summary-card border border-line rounded bg-surface p-4 [&_h3]:my-1 [&_h3]:mx-2 [&_p]:m-0"
      >
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Next achievement</p>
        <template v-if="nextAchievement">
          <h3>{{ nextAchievement.name }}</h3>
          <p>{{ nextAchievement.description }}</p>
          <div
            class="progress-meter h-2.5 overflow-hidden rounded-full bg-read-progress [&_span]:block [&_span]:h-full [&_span]:min-w-0.5 [&_span]:rounded-[inherit] [&_span]:bg-progress"
            :aria-label="`${nextAchievement.name} progress`"
          >
            <span :style="{ width: formatProgress(nextAchievement.percent) }"></span>
          </div>
          <small>{{ achievementProgress(nextAchievement) }}</small>
        </template>
        <p v-else class="muted block text-muted">All achievements earned.</p>
      </article>
    </div>
  </section>
</template>
