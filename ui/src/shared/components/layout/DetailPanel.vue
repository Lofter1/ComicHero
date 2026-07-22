<template>
  <article class="detail-panel">
    <slot />
  </article>
</template>

<style scoped>
@reference '../../../styles.css';

.detail-panel {
  @apply min-h-panel rounded-xl border border-line bg-panel p-5 shadow-detail;
}

@media (width <= 720px) {
  .detail-panel {
    @apply min-h-0 p-3.5;
  }
}
</style>
