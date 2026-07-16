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
  <div class="row order-row" :class="{ selected }">
    <span class="order-row-content">
      <button class="row-main" :class="mainClass" type="button" @click="$emit('open')">
        <span v-if="image" class="issue-list-cover" aria-hidden="true">
          <img :src="assetURL(image)" alt="" loading="lazy" />
        </span>
        <span>
          <strong>{{ title }}</strong>
          <small>{{ subtitle }}</small>
          <span v-if="$slots.byline" class="row-byline">
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
    </span>
    <span class="row-progress" :aria-label="progressLabel">
      <span :style="{ width: progress }"></span>
    </span>
  </div>
</template>
