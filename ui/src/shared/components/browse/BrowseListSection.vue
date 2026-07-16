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
  <section class="list-section">
    <div class="list-section-header">
      <p class="eyebrow">{{ title }}</p>
      <small>{{ items.length }}</small>
    </div>
    <div class="list">
      <slot v-for="item in items" :key="item.id" name="item" :item="item" />
    </div>
  </section>
</template>
