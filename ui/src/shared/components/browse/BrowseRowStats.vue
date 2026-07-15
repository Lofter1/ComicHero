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
  <span class="engagement-stats">{{ visibleItems.join(' · ') }}</span>
</template>
