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
    class="favorite-toggle [flex:0_0_auto] inline-flex items-center justify-center [width:34px] [min-width:34px] [min-height:34px] border border-warning-border rounded-full bg-warning-soft text-warning [padding:0] [font-size:1.15rem] font-bold leading-none"
    :class="{ active: favorite }"
    :disabled="disabled"
    :aria-label="favorite ? 'Remove from favorites' : 'Add to favorites'"
    :title="favorite ? 'Remove from favorites' : 'Add to favorites'"
    @click="$emit('toggle')"
  >
    <span aria-hidden="true">{{ favorite ? '★' : '☆' }}</span>
  </button>
</template>
