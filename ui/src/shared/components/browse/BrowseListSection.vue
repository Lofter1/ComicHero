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
    <div
      class="flex items-center justify-between gap-2.5 py-0 px-0.5 [&_.eyebrow]:m-0 [&_small]:text-muted [&_small]:font-bold"
    >
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
