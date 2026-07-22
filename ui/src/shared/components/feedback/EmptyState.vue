<script setup>
defineProps({
  tag: { type: String, default: 'div' },
  roomy: { type: Boolean, default: false },
})
</script>

<template>
  <component :is="tag" class="empty-state" :class="{ 'empty-state--roomy': roomy }">
    <slot />
  </component>
</template>

<style scoped>
@reference '../../../styles.css';

.empty-state {
  @apply grid justify-items-start gap-3 rounded-xl border border-dashed border-line-strong bg-panel-soft p-5 text-muted;
}

.empty-state--roomy {
  @apply bg-surface-soft p-5 font-extrabold;
}
</style>
