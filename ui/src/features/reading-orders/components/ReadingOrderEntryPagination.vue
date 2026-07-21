<script setup>
import BaseButton from '@/shared/components/form/BaseButton.vue'

defineProps({
  page: { type: Number, required: true },
  pageCount: { type: Number, required: true },
  start: { type: Number, default: 0 },
  end: { type: Number, default: 0 },
  total: { type: Number, default: 0 },
  compact: { type: Boolean, default: false },
})

defineEmits(['go'])
</script>

<template>
  <nav
    class="entry-pagination"
    :class="{ 'entry-pagination--compact': compact }"
    aria-label="Reading order entry pages"
  >
    <span v-if="compact">Page {{ page + 1 }} of {{ pageCount }}</span>
    <span v-else>Entries {{ start + 1 }}–{{ end }} of {{ total }}</span>
    <div>
      <BaseButton
        v-if="!compact"
        type="button"
        variant="secondary"
        size="compact"
        :disabled="page === 0"
        @click="$emit('go', 0)"
      >
        First
      </BaseButton>
      <BaseButton
        type="button"
        variant="secondary"
        size="compact"
        :disabled="page === 0"
        @click="$emit('go', page - 1)"
      >
        Previous
      </BaseButton>
      <strong v-if="!compact">Page {{ page + 1 }} of {{ pageCount }}</strong>
      <BaseButton
        type="button"
        variant="secondary"
        size="compact"
        :disabled="page === pageCount - 1"
        @click="$emit('go', page + 1)"
      >
        Next
      </BaseButton>
      <BaseButton
        v-if="!compact"
        type="button"
        variant="secondary"
        size="compact"
        :disabled="page === pageCount - 1"
        @click="$emit('go', pageCount - 1)"
      >
        Last
      </BaseButton>
    </div>
  </nav>
</template>

<style scoped>
@reference '../../../styles.css';

.entry-pagination {
  @apply flex items-center justify-between gap-3 mb-2.5 border border-line rounded bg-panel-soft py-2.5 px-3 down-mobile:items-stretch down-mobile:flex-col [&_>_span]:text-muted [&_>_span]:text-sm [&_>_span]:font-ui-bold [&_>_div]:flex [&_>_div]:items-center [&_>_div]:justify-end [&_>_div]:gap-2 [&_strong]:min-w-24 [&_strong]:text-ink [&_strong]:text-center down-mobile:[&_>_div]:grid down-mobile:[&_>_div]:grid-cols-2 down-mobile:[&_strong]:col-span-full down-mobile:[&_strong]:row-start-1;
}

.entry-pagination :deep(button) {
  @apply down-mobile:w-full;
}

.entry-pagination--compact {
  @apply mt-2.5 mb-0;
}
</style>
