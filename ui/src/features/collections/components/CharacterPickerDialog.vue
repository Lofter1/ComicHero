<script setup>
import { ref } from 'vue'
import { assetURL, listCharacters } from '@/api/client.js'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'

defineProps({ saving: { type: Boolean, default: false } })
const emit = defineEmits(['close', 'add'])
const query = ref('')
const results = ref([])
const searching = ref(false)
const searched = ref(false)
const searchError = ref('')

async function search() {
  if (!query.value.trim() || searching.value) return
  searching.value = true
  searchError.value = ''
  try {
    const page = await listCharacters({ q: query.value.trim(), limit: 50 })
    results.value = page.items
    searched.value = true
  } catch (error) {
    results.value = []
    searched.value = true
    searchError.value = error.message
  } finally {
    searching.value = false
  }
}

function add(character) {
  emit('add', character)
  emit('close')
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
      aria-labelledby="add-character-title"
    >
      <header class="panel-header">
        <div>
          <p class="eyebrow">Collection members</p>
          <h3 id="add-character-title">Add a character</h3>
          <p class="collection-dialog-description">
            Search names and aliases in your character library.
          </p>
        </div>
        <button class="icon-button" type="button" aria-label="Close" @click="$emit('close')">
          ×
        </button>
      </header>
      <form class="collection-search-form" @submit.prevent="search">
        <input v-model="query" type="search" placeholder="Search characters or aliases" autofocus />
        <button class="primary-button" type="submit" :disabled="searching || !query.trim()">
          {{ searching ? 'Searching...' : 'Search' }}
        </button>
      </form>
      <LoadingState v-if="searching" compact />
      <p v-else-if="searchError" class="error-text">{{ searchError }}</p>
      <div v-else-if="results.length" class="collection-dialog-list">
        <button
          v-for="character in results"
          :key="character.id"
          class="row"
          type="button"
          :disabled="saving"
          @click="add(character)"
        >
          <span class="collection-dialog-item-main">
            <span class="collection-dialog-avatar" aria-hidden="true">
              <img v-if="character.image" :src="assetURL(character.image)" alt="" loading="lazy" />
              <span v-else>{{ monogram(character.name) }}</span>
            </span>
            <span>
              <strong>{{ character.name }}</strong>
              <small>{{ character.appearanceCount }} appearances</small>
            </span>
          </span>
          <span class="status-pill">Add</span>
        </button>
      </div>
      <p v-else-if="searched" class="muted">No characters match this search.</p>
      <p v-else class="muted">Search your imported characters by name or alias.</p>
    </section>
  </div>
</template>
