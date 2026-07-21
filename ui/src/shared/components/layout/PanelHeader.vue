<script setup>
import { computed, useId } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'

const props = defineProps({
  eyebrow: { type: String, default: '' },
  title: { type: String, default: '' },
  titleId: { type: String, default: undefined },
  divided: { type: Boolean, default: false },
  closable: { type: Boolean, default: false },
  closeLabel: { type: String, default: 'Close' },
})

defineEmits(['close'])

const generatedTitleId = useId()
const resolvedTitleId = computed(() => props.titleId || generatedTitleId)
</script>

<template>
  <header class="panel-header" :class="{ 'panel-header--divided': divided }">
    <div class="panel-heading">
      <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
      <slot name="title">
        <h3 :id="resolvedTitleId">{{ title }}</h3>
      </slot>
      <slot name="description" />
    </div>
    <div v-if="$slots.actions || closable" class="panel-actions">
      <slot v-if="$slots.actions" name="actions" />
      <BaseButton
        v-else
        variant="neutral"
        size="icon"
        :aria-label="closeLabel"
        @click="$emit('close')"
      >
        ×
      </BaseButton>
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
