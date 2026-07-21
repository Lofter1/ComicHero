<script setup>
defineProps({
  favorite: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['toggle'])
</script>

<template>
  <!-- Native button: this reusable control has a unique circular pressed-state design. -->
  <button
    type="button"
    class="favorite-toggle"
    :class="{ active: favorite }"
    :disabled="disabled"
    :aria-label="favorite ? 'Remove from favorites' : 'Add to favorites'"
    :title="favorite ? 'Remove from favorites' : 'Add to favorites'"
    @click="$emit('toggle')"
  >
    <span aria-hidden="true">{{ favorite ? '★' : '☆' }}</span>
  </button>
</template>

<style scoped>
@reference '../../../styles.css';

.favorite-toggle {
  @apply inline-flex min-h-8 w-8 min-w-8 flex-none items-center justify-center rounded-full border border-warning-border bg-warning-soft p-0 text-lg leading-none font-bold text-warning;
}

.favorite-toggle.active {
  @apply border-warning-border text-warning;
  background: color-mix(in srgb, var(--warning-soft) 40%, #f6d56b);
}
</style>
