<script setup>
defineProps({
  tone: {
    type: String,
    default: 'primary',
    validator: (value) => ['primary', 'neutral', 'warning', 'danger'].includes(value),
  },
})
</script>

<template>
  <span class="status-pill" :class="`status-pill--${tone}`">
    <slot />
  </span>
</template>

<style scoped>
@reference '../../../styles.css';

.status-pill {
  @apply flex-none rounded-full border-0 px-2 py-1 text-xs font-bold;
}

.status-pill--primary {
  @apply bg-primary-soft text-primary;
}

.status-pill--neutral {
  @apply bg-surface-muted text-muted;
}

.status-pill--warning {
  @apply bg-warning-soft text-warning;
}

.status-pill--danger {
  @apply bg-danger-soft text-danger;
}

@media (width <= 720px) {
  .status-pill {
    @apply ml-auto;
  }
}

@media (width <= 420px) {
  .status-pill {
    @apply ml-0 justify-self-start;
  }
}
</style>
