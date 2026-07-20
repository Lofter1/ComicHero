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
    class="row flex min-h-10 w-full flex-col items-start justify-between gap-3 rounded border border-line-strong bg-surface p-3.5 text-left text-control hover:bg-surface-soft down-mobile:min-h-12 down-mobile:flex-wrap down-mobile:p-3 down-phone:grid down-phone:grid-cols-1"
    :class="{ selected }"
  >
    <span class="row-heading flex w-full min-w-0 items-start justify-between gap-3">
      <!-- Native button: the entity body is a full-row navigation target. -->
      <button
        class="row-main min-w-0 flex-auto border-0 bg-transparent p-0 text-left text-inherit"
        :class="mainClass"
        type="button"
        @click="$emit('open')"
      >
        <span
          v-if="image"
          class="row-cover h-16 w-11 flex-none overflow-hidden rounded-ui-sm border border-line bg-surface-muted down-phone:h-12 down-phone:w-10"
          aria-hidden="true"
        >
          <img :src="assetURL(image)" alt="" loading="lazy" />
        </span>
        <span>
          <strong>{{ title }}</strong>
          <small>{{ subtitle }}</small>
          <span
            v-if="$slots.byline"
            class="row-byline mt-2 flex flex-wrap items-center gap-x-2.5 gap-y-1.5"
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
      class="row-progress block h-2 w-full flex-none overflow-hidden rounded-full bg-read-progress"
      :aria-label="progressLabel"
    >
      <span :style="{ width: progress }"></span>
    </span>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.row > span:first-child,
.row-heading > span:first-child,
.row-main > * {
  @apply min-w-0;
}

.row strong,
.row small,
.row-main span {
  overflow-wrap: anywhere;
}

.row small {
  @apply block text-muted;
}

.row.selected {
  @apply border-primary shadow-selected;
}

.row-main:hover:not(:disabled) {
  @apply border-transparent shadow-none transform-none;
}

.row-cover img {
  @apply block h-full w-full object-cover;
}

.row-byline :is(.author-pill, .started-pill) {
  @apply mt-0;
}

.row-progress span {
  @apply block h-full min-w-0.5 bg-progress;
  border-radius: inherit;
}
</style>
