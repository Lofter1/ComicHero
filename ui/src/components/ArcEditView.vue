<script setup>
import { computed } from 'vue'
import ReadingOrderEditor from '@/components/ReadingOrderEditor.vue'

const props = defineProps({
  form: {
    type: Object,
    required: true,
  },
  selectedArc: {
    type: Object,
    default: null,
  },
  comics: {
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
  set: value => emit('update:form', value),
})
</script>

<template>
  <div class="editor-view">
    <header class="editor-header sticky-toolbar">
      <button class="secondary-button" type="button" @click="$emit('back')">Back</button>
      <div>
        <p class="eyebrow">Arc</p>
        <h2>{{ form.id ? 'Edit arc' : 'New arc' }}</h2>
      </div>
      <div class="editor-actions">
        <button v-if="form.id" type="button" class="danger-button" :disabled="saving" @click="$emit('delete')">
          Delete
        </button>
        <button class="primary-button" type="submit" form="arc-editor-form" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save Arc' }}
        </button>
      </div>
    </header>

    <article class="detail-panel">
      <ReadingOrderEditor
        v-model:form="formModel"
        form-id="arc-editor-form"
        item-label="arc"
        empty-entry-message="No comics in this arc yet."
        :selected-order="selectedArc"
        :comics="comics"
        :saving="saving"
        @save="$emit('save')"
      />
    </article>
  </div>
</template>
