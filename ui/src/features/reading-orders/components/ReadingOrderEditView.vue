<script setup>
import { computed } from 'vue'
import ReadingOrderEditor from '@/features/reading-orders/components/ReadingOrderEditor.vue'

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
  <div class="editor-view grid gap-4.5 w-full">
    <header
      class="editor-header sticky-toolbar flex items-center gap-3.5 justify-between flex-wrap sticky [top:var(--sticky-toolbar-top)] z-20 [margin-inline:calc(var(--sticky-toolbar-inline-offset)_*_-1)] [padding:14px_var(--sticky-toolbar-inline-offset)] border-b border-sticky-border bg-sticky-bg shadow-sticky backdrop-blur-ui [&.sticky-toolbar]:[margin-top:calc(var(--content-padding)_*_-1)] max-w-none [&_>_div:not(.editor-actions)]:min-w-0 [&_h2]:m-0 down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:pb-3 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none down-mobile:[&_button]:w-full"
    >
      <button
        class="secondary-button min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]"
        type="button"
        @click="$emit('back')"
      >
        Back
      </button>
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Reading Order</p>
        <h2>{{ form.id ? 'Edit reading order' : 'New reading order' }}</h2>
      </div>
      <div
        class="editor-actions flex items-center gap-2.5 flex-wrap justify-end ml-auto down-tablet:items-stretch down-tablet:flex-col down-mobile:w-full"
      >
        <button
          v-if="form.id"
          type="button"
          class="danger-button min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 [border-color:color-mix(in_srgb,_var(--danger)_42%,_var(--line-strong))] bg-danger-soft text-danger"
          :disabled="saving"
          @click="$emit('delete')"
        >
          Delete
        </button>
        <button
          class="primary-button min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 border-primary bg-primary text-white"
          type="submit"
          form="reading-order-editor-form"
          :disabled="saving"
        >
          {{ saving ? 'Saving...' : 'Save Reading Order' }}
        </button>
      </div>
    </header>

    <article
      class="detail-panel min-h-90 border border-line rounded bg-panel p-5 shadow-detail down-mobile:min-h-0 down-mobile:p-3.5"
    >
      <ReadingOrderEditor
        v-model:form="formModel"
        form-id="reading-order-editor-form"
        :selected-order="selectedOrder"
        :comics="comics"
        :reading-orders="readingOrders"
        :saving="saving"
        @save="$emit('save')"
      />
    </article>
  </div>
</template>
