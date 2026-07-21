<script setup>
defineProps({
  title: {
    type: String,
    required: true,
  },
  items: {
    type: Array,
    default: () => [],
  },
})
</script>

<template>
  <section class="grid gap-2">
    <div class="browse-section-header">
      <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">
        {{ title }}
      </p>
      <small>{{ items.length }}</small>
    </div>
    <div class="list grid gap-2.5 down-mobile:gap-2">
      <slot v-for="item in items" :key="item.id" name="item" :item="item" />
    </div>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.browse-section-header {
  @apply flex items-center justify-between gap-2.5 px-0.5 py-0;
}

.browse-section-header .eyebrow {
  @apply m-0;
}

.browse-section-header small {
  @apply font-bold text-muted;
}
</style>
