<script setup>
import { computed } from 'vue'

const props = defineProps({
  items: {
    type: Array,
    default: () => [],
  },
})

const visibleItems = computed(() =>
  props.items.filter((item) => item !== null && item !== undefined && item !== ''),
)
</script>

<template>
  <span
    class="engagement-stats inline-flex items-center text-muted text-xs font-bold whitespace-nowrap"
    >{{ visibleItems.join(' · ') }}</span
  >
</template>
