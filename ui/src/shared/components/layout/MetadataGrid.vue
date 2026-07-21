<script setup>
defineProps({
  columns: { type: Number, default: 3 },
})
</script>

<template>
  <div class="metadata-grid" :style="{ '--metadata-columns': columns }">
    <slot />
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.metadata-grid {
  @apply grid gap-2.5;
  grid-template-columns: repeat(var(--metadata-columns), minmax(0, 1fr));
}

.metadata-grid :deep(> *) {
  @apply rounded border border-line bg-surface-soft p-3;
}

.metadata-grid :deep(strong),
.metadata-grid :deep(small) {
  @apply block;
}

.metadata-grid :deep(strong) {
  overflow-wrap: anywhere;
}

.metadata-grid :deep(small) {
  @apply mt-1 text-muted;
}

@media (width <= 960px) {
  .metadata-grid {
    @apply grid-cols-1;
  }
}
</style>
