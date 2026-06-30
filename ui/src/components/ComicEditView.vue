<script setup>
import { computed } from 'vue'
import ComicEditor from '@/components/ComicEditor.vue'

const props = defineProps({
  form: {
    type: Object,
    required: true,
  },
  selectedComic: {
    type: Object,
    default: null,
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
        <p class="eyebrow">Comic</p>
        <h2>{{ form.id ? 'Edit comic' : 'New comic' }}</h2>
      </div>
      <div class="editor-actions">
        <button v-if="form.id" type="button" class="danger-button" :disabled="saving" @click="$emit('delete')">
          Delete
        </button>
        <button class="primary-button" type="submit" form="comic-editor-form" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save Comic' }}
        </button>
      </div>
    </header>

    <article class="detail-panel">
      <ComicEditor
        v-model:form="formModel"
        form-id="comic-editor-form"
        :selected-comic="selectedComic"
        :saving="saving"
        @save="$emit('save')"
      />
    </article>
  </div>
</template>
