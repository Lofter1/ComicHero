<script setup>
import { assetURL } from '@/api/client.js'
import FavoriteToggle from '@/shared/components/feedback/FavoriteToggle.vue'

defineProps({
  title: {
    type: String,
    required: true,
  },
  subtitle: {
    type: String,
    default: '',
  },
  image: {
    type: String,
    default: '',
  },
  mainClass: {
    type: String,
    default: '',
  },
  selected: {
    type: Boolean,
    default: false,
  },
  favorite: {
    type: Boolean,
    default: false,
  },
  canFavorite: {
    type: Boolean,
    default: false,
  },
  favoriteSaving: {
    type: Boolean,
    default: false,
  },
  progress: {
    type: String,
    default: '0%',
  },
  progressLabel: {
    type: String,
    required: true,
  },
})

defineEmits(['open', 'toggle-favorite'])
</script>

<template>
  <div
    class="row order-row flex-col min-h-10.5 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-13 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
    :class="{ selected }"
  >
    <span
      class="order-row-content flex items-start justify-between gap-3 w-full min-w-0 [&_>_span:first-child]:min-w-0"
    >
      <button
        class="row-main flex-auto min-w-0 border-0 bg-transparent text-inherit p-0 text-left [&:hover:not(:disabled)]:[border-color:transparent] [&:hover:not(:disabled)]:shadow-none [&:hover:not(:disabled)]:transform-none [&_span]:break-anywhere [&_>_*]:min-w-0"
        :class="mainClass"
        type="button"
        @click="$emit('open')"
      >
        <span
          v-if="image"
          class="issue-list-cover flex-none w-11 h-15 overflow-hidden border border-line rounded-[6px] bg-surface-muted [&_img]:block [&_img]:w-full [&_img]:h-full [&_img]:object-cover down-phone:w-9.5 down-phone:h-13"
          aria-hidden="true"
        >
          <img :src="assetURL(image)" alt="" loading="lazy" />
        </span>
        <span>
          <strong>{{ title }}</strong>
          <small>{{ subtitle }}</small>
          <span
            v-if="$slots.byline"
            class="row-byline flex items-center flex-wrap gapy-1.5 gapx-2.5 mt-2 [&_.author-pill]:mt-0 [&_.started-pill]:mt-0"
          >
            <slot name="byline" />
          </span>
        </span>
      </button>
      <FavoriteToggle
        v-if="canFavorite"
        :favorite="favorite"
        :disabled="favoriteSaving"
        @toggle="$emit('toggle-favorite')"
      />
      <slot name="actions" />
    </span>
    <span
      class="row-progress block flex-none w-full h-2 overflow-hidden rounded-full bg-read-progress [&_span]:block [&_span]:h-full [&_span]:min-w-0.5 [&_span]:[border-radius:inherit] [&_span]:bg-progress"
      :aria-label="progressLabel"
    >
      <span :style="{ width: progress }"></span>
    </span>
  </div>
</template>
