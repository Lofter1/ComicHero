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
