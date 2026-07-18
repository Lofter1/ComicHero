<script setup>
import { ref } from 'vue'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'

defineProps({
  character: { type: Object, required: true },
  collections: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  saving: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'add', 'create'])
const newName = ref('')

function createCollection() {
  const name = newName.value.trim()
  if (!name) return
  emit('create', name)
}

function monogram(name) {
  return (name || '?').trim().slice(0, 1).toUpperCase() || '?'
}
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <section
      class="collection-dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="add-to-collection-title"
    >
      <header class="panel-header">
        <div>
          <p class="eyebrow">My collections</p>
          <h3 id="add-to-collection-title">Add {{ character.name }} to a collection</h3>
          <p class="collection-dialog-description">
            Choose an existing collection or create one below.
          </p>
        </div>
        <button class="icon-button" type="button" aria-label="Close" @click="$emit('close')">
          ×
        </button>
      </header>

      <LoadingState v-if="loading" compact />
      <div v-else-if="collections.length" class="collection-dialog-list">
        <button
          v-for="collection in collections"
          :key="collection.id"
          class="row"
          type="button"
          :disabled="saving || collection.containsCharacter"
          @click="emit('add', collection)"
        >
          <span class="collection-dialog-item-main">
            <span class="collection-dialog-monogram" aria-hidden="true">
              {{ monogram(collection.name) }}
            </span>
            <span>
              <strong>{{ collection.name }}</strong>
              <small>{{ collection.characterCount }} characters</small>
            </span>
          </span>
          <span class="status-pill">
            {{ collection.containsCharacter ? 'Added' : 'Add' }}
          </span>
        </button>
      </div>
      <p v-else-if="!loading" class="muted">You do not have any collections yet.</p>

      <form class="collection-create-form" @submit.prevent="createCollection">
        <label for="new-character-collection">Create a new collection</label>
        <div>
          <input id="new-character-collection" v-model="newName" maxlength="120" />
          <button class="primary-button" type="submit" :disabled="saving || !newName.trim()">
            {{ saving ? 'Saving...' : 'Create and add' }}
          </button>
        </div>
      </form>
    </section>
  </div>
</template>
