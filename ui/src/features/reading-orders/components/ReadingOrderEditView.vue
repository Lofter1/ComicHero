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
    <header class="editor-header sticky-toolbar">
      <button class="secondary-button" type="button" @click="$emit('back')">Back</button>
      <div>
        <p class="eyebrow">Reading Order</p>
        <h2>{{ form.id ? 'Edit reading order' : 'New reading order' }}</h2>
      </div>
      <div class="editor-actions">
        <button
          v-if="form.id"
          type="button"
          class="danger-button"
          :disabled="saving"
          @click="$emit('delete')"
        >
          Delete
        </button>
        <button
          class="primary-button"
          type="submit"
          form="reading-order-editor-form"
          :disabled="saving"
        >
          {{ saving ? 'Saving...' : 'Save Reading Order' }}
        </button>
      </div>
    </header>

    <article class="detail-panel">
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
