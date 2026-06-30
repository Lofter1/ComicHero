<script setup>
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
  formId: {
    type: String,
    default: 'comic-editor-form',
  },
})

const emit = defineEmits(['update:form', 'save', 'delete'])

function updateField(field, value) {
  emit('update:form', {
    ...props.form,
    [field]: value,
  })
}
</script>

<template>
  <form :id="formId" class="edit-form" @submit.prevent="$emit('save')">
    <div class="form-grid">
      <label>
        Series
        <input :value="form.series" required @input="updateField('series', $event.target.value)" />
      </label>
      <label>
        Series Year
        <input
          :value="form.seriesYear"
          min="0"
          type="number"
          @input="updateField('seriesYear', Number($event.target.value))"
        />
      </label>
      <label>
        Issue
        <input :value="form.issue" @input="updateField('issue', $event.target.value)" />
      </label>
      <label>
        Publisher
        <input :value="form.publisher" @input="updateField('publisher', $event.target.value)" />
      </label>
      <label>
        Cover Date
        <input :value="form.coverDate" @input="updateField('coverDate', $event.target.value)" />
      </label>
      <label>
        Cover Image URL
        <input :value="form.coverImage" @input="updateField('coverImage', $event.target.value)" />
      </label>
    </div>

    <label>
      Description
      <textarea :value="form.description" rows="4" @input="updateField('description', $event.target.value)" />
    </label>
    <label class="check-row">
      <input :checked="form.read" type="checkbox" @change="updateField('read', $event.target.checked)" />
      Read
    </label>

    <div v-if="selectedComic?.readingOrders?.length" class="preview-list">
      <p class="eyebrow">Reading Orders</p>
      <ul>
        <li v-for="order in selectedComic.readingOrders" :key="order.id">
          {{ order.name }}
        </li>
      </ul>
    </div>

  </form>
</template>
