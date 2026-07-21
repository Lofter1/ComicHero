<script setup>
defineProps({
  eyebrow: { type: String, default: '' },
  title: { type: String, default: '' },
  titleId: { type: String, default: undefined },
  divided: { type: Boolean, default: false },
})
</script>

<template>
  <header class="panel-header" :class="{ 'panel-header--divided': divided }">
    <div class="panel-heading">
      <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
      <slot name="title">
        <h3 :id="titleId">{{ title }}</h3>
      </slot>
      <slot name="description" />
    </div>
    <div v-if="$slots.actions" class="panel-actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<style scoped>
@reference '../../../styles.css';

.panel-header {
  @apply mb-4 flex items-center justify-between gap-3.5;
}

.panel-header--divided {
  @apply items-start border-b border-line pb-3.5;
}

.panel-heading {
  @apply min-w-0;
}

.eyebrow {
  @apply mt-0 mb-1.5 text-xs font-bold text-eyebrow uppercase;
}

.panel-actions {
  @apply flex-none;
}

@media (width <= 720px) {
  .panel-header {
    @apply flex-col items-stretch gap-2.5;
  }

  .panel-actions :deep(button) {
    @apply w-full;
  }
}
</style>
