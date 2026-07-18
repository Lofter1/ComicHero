<script setup>
import { nextTick, ref } from 'vue'
import { formatProgress } from '@/features/reading-orders/model.js'

const props = defineProps({
  collections: { type: Array, default: () => [] },
  saving: { type: Boolean, default: false },
})
const emit = defineEmits(['create', 'open'])
const name = ref('')
const nameInput = ref(null)
const createOpen = ref(!props.collections.length)

function create() {
  const value = name.value.trim()
  if (!value) return
  emit('create', value)
}

async function openCreate() {
  createOpen.value = true
  await nextTick()
  nameInput.value?.focus()
}

function closeCreate() {
  if (props.saving) return
  createOpen.value = false
  name.value = ''
}

function collectionMonogram(collection) {
  return (collection?.name || '?').trim().slice(0, 1).toUpperCase() || '?'
}

function countLabel(count, singular) {
  return `${count} ${count === 1 ? singular : `${singular}s`}`
}
</script>

<template>
  <section class="collections-view">
    <header class="collection-page-intro">
      <div>
        <p>
          Group related characters and follow all of their appearances in one release-date order.
        </p>
      </div>
      <button
        v-if="!createOpen"
        class="primary-button icon-text-button"
        type="button"
        @click="openCreate"
      >
        <span class="button-icon" aria-hidden="true">+</span>
        New collection
      </button>
    </header>

    <form v-if="createOpen" class="collection-create-panel" @submit.prevent="create">
      <div class="collection-create-heading">
        <div>
          <strong>Create a collection</strong>
          <small>Give this character group a memorable name.</small>
        </div>
        <button
          v-if="collections.length"
          class="icon-button collection-create-close"
          type="button"
          aria-label="Cancel creating collection"
          @click="closeCreate"
        >
          ×
        </button>
      </div>
      <div class="collection-create-controls">
        <label class="sr-only" for="collection-name">Collection name</label>
        <input id="collection-name" ref="nameInput" v-model="name" maxlength="120" />
        <button class="primary-button" type="submit" :disabled="saving || !name.trim()">
          {{ saving ? 'Creating...' : 'Create' }}
        </button>
      </div>
    </form>

    <section
      v-if="collections.length"
      class="collection-library"
      aria-labelledby="collection-list-title"
    >
      <header class="collection-library-heading">
        <div>
          <h3 id="collection-list-title">Your collections</h3>
          <p>{{ countLabel(collections.length, 'collection') }}</p>
        </div>
      </header>
      <div class="collection-grid">
        <button
          v-for="collection in collections"
          :key="collection.id"
          class="collection-card"
          type="button"
          @click="emit('open', collection)"
        >
          <span class="collection-card-monogram" aria-hidden="true">
            {{ collectionMonogram(collection) }}
          </span>
          <span class="collection-card-body">
            <span class="collection-card-title-row">
              <strong>{{ collection.name }}</strong>
              <span class="collection-card-chevron" aria-hidden="true">›</span>
            </span>
            <span class="collection-card-meta">
              <span>{{ countLabel(collection.characterCount, 'character') }}</span>
              <span>{{ countLabel(collection.appearanceCount, 'appearance') }}</span>
            </span>
            <span class="collection-card-progress-copy">
              <span v-if="collection.startedAt" class="collection-card-status">Reading</span>
              <span v-else>Not started</span>
              <strong>{{ formatProgress(collection.progress) }}</strong>
            </span>
            <span class="row-progress" :aria-label="`${collection.name} progress`">
              <span :style="{ width: formatProgress(collection.progress) }"></span>
            </span>
          </span>
        </button>
      </div>
    </section>
    <section v-else class="collection-empty-state">
      <span class="collection-empty-icon" aria-hidden="true">◇</span>
      <h3>Build your first character collection</h3>
      <p>Characters you add will become one combined, release-date reading queue.</p>
    </section>
  </section>
</template>
