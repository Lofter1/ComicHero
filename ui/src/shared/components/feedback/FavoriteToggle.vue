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
    class="favorite-toggle flex-none inline-flex items-center justify-center w-8 min-w-8 min-h-8 border border-warning-border rounded-full bg-warning-soft text-warning p-0 text-lg font-bold leading-none [&.active]:border-warning-border [&.active]:[background:color-mix(in_srgb,var(--warning-soft)_40%,#f6d56b)] [&.active]:text-warning"
    :class="{ active: favorite }"
    :disabled="disabled"
    :aria-label="favorite ? 'Remove from favorites' : 'Add to favorites'"
    :title="favorite ? 'Remove from favorites' : 'Add to favorites'"
    @click="$emit('toggle')"
  >
    <span aria-hidden="true">{{ favorite ? '★' : '☆' }}</span>
  </button>
</template>
