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
  <div class="row order-row flex-col" :class="{ selected }">
    <span
      class="order-row-content flex [align-items:flex-start] justify-between gap-3 w-full min-w-0"
    >
      <button class="row-main" :class="mainClass" type="button" @click="$emit('open')">
        <span v-if="image" class="issue-list-cover" aria-hidden="true">
          <img :src="assetURL(image)" alt="" loading="lazy" />
        </span>
        <span>
          <strong>{{ title }}</strong>
          <small>{{ subtitle }}</small>
          <span
            v-if="$slots.byline"
            class="row-byline flex items-center flex-wrap [gap:6px_10px] mt-2"
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
    <span class="row-progress" :aria-label="progressLabel">
      <span :style="{ width: progress }"></span>
    </span>
  </div>
</template>
