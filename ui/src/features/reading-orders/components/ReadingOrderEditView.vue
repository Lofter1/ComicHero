<script setup>
import { computed } from 'vue'
import ReadingOrderEditor from '@/features/reading-orders/components/ReadingOrderEditor.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import DetailPanel from '@/shared/components/layout/DetailPanel.vue'

const props = defineProps({
  form: {
    type: Object,
    required: true,
  },
  selectedOrder: {
    type: Object,
    default: null,
  },
  comics: {
    type: Array,
    default: () => [],
  },
  readingOrders: {
    type: Array,
    default: () => [],
  },
  saving: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:form', 'back', 'delete', 'save'])

const formModel = computed({
  get: () => props.form,
  set: (value) => emit('update:form', value),
})
</script>

<template>
  <div class="editor-view grid gap-4 w-full">
    <header class="editor-header sticky-toolbar">
      <BaseButton @click="$emit('back')"> Back </BaseButton>
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Reading Order</p>
        <h2>{{ form.id ? 'Edit reading order' : 'New reading order' }}</h2>
      </div>
      <div class="editor-actions">
        <BaseButton v-if="form.id" variant="danger" :disabled="saving" @click="$emit('delete')">
          Delete
        </BaseButton>
        <BaseButton
          variant="primary"
          type="submit"
          form="reading-order-editor-form"
          :disabled="saving"
        >
          {{ saving ? 'Saving...' : 'Save Reading Order' }}
        </BaseButton>
      </div>
    </header>

    <DetailPanel>
      <ReadingOrderEditor
        v-model:form="formModel"
        form-id="reading-order-editor-form"
        :selected-order="selectedOrder"
        :comics="comics"
        :reading-orders="readingOrders"
        :saving="saving"
        @save="$emit('save')"
      />
    </DetailPanel>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.editor-header.sticky-toolbar {
  @apply flex items-center gap-3.5 justify-between flex-wrap sticky top-(--sticky-toolbar-top) z-20 mx-[calc(var(--sticky-toolbar-inline-offset)*-1)] p-[14px_var(--sticky-toolbar-inline-offset)] border-b border-sticky-border bg-sticky-bg shadow-sticky backdrop-blur-ui [&.sticky-toolbar]:mt-[calc(var(--content-padding)*-1)] max-w-none [&_>_div:not(.editor-actions)]:min-w-0 [&_h2]:m-0 down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:pb-3 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none down-mobile:[&_button]:w-full;
}

.editor-actions {
  @apply flex items-center gap-2.5 flex-wrap justify-end ml-auto down-tablet:items-stretch down-tablet:flex-col down-mobile:w-full;
}
</style>
