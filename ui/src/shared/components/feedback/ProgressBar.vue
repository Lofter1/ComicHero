<script setup>
import { computed } from 'vue'

const props = defineProps({
  value: { type: [Number, String], default: 0 },
  label: { type: String, required: true },
  compact: { type: Boolean, default: false },
  tag: { type: String, default: 'div' },
})

const width = computed(() => {
  if (typeof props.value === 'string') return props.value
  return `${Math.min(100, Math.max(0, props.value))}%`
})

const numericValue = computed(() => {
  const value = Number.parseFloat(props.value)
  return Number.isFinite(value) ? Math.min(100, Math.max(0, value)) : undefined
})
</script>

<template>
  <component
    :is="tag"
    class="progress-bar"
    :class="{ 'progress-bar--compact': compact }"
    role="progressbar"
    :aria-label="label"
    aria-valuemin="0"
    aria-valuemax="100"
    :aria-valuenow="numericValue"
  >
    <span :style="{ width }"></span>
  </component>
</template>

<style scoped>
@reference '../../../styles.css';

.progress-bar {
  @apply h-2.5 overflow-hidden rounded-full bg-read-progress;
}

.progress-bar--compact {
  @apply h-2;
}

.progress-bar > span {
  @apply block h-full min-w-0.5 bg-progress;
  border-radius: inherit;
}
</style>
