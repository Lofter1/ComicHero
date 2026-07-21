<script setup>
import { ref } from 'vue'
import { assetURL, listCharacters } from '@/api/client.js'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import PanelHeader from '@/shared/components/layout/PanelHeader.vue'
import ModalShell from '@/shared/components/overlay/ModalShell.vue'
import StatusPill from '@/shared/components/feedback/StatusPill.vue'

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
  <ModalShell v-slot="{ titleId }" @close="$emit('close')">
    <PanelHeader
      eyebrow="Collection members"
      title="Add a character"
      :title-id="titleId"
      divided
      closable
      @close="$emit('close')"
    >
      <template #description>
        <p class="collection-dialog-description mt-1 mx-0 mb-0 text-muted text-sm leading-snug">
          Search names and aliases in your character library.
        </p>
      </template>
    </PanelHeader>
    <form class="collection-search-form" @submit.prevent="search">
      <BaseTextInput
        v-model="query"
        type="search"
        placeholder="Search characters or aliases"
        autofocus
      />
      <BaseButton variant="primary" type="submit" :disabled="searching || !query.trim()">
        {{ searching ? 'Searching...' : 'Search' }}
      </BaseButton>
    </form>
    <LoadingState v-if="searching" compact />
    <p v-else-if="searchError" class="error-text text-danger font-bold">{{ searchError }}</p>
    <div v-else-if="results.length" class="collection-dialog-list">
      <!-- Native button: search results are full-card selection targets. -->
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
        <StatusPill>Add</StatusPill>
      </button>
    </div>
    <p v-else-if="searched" class="muted block text-muted">No characters match this search.</p>
    <p v-else class="muted block text-muted">Search your imported characters by name or alias.</p>
  </ModalShell>
</template>

<style scoped>
@reference '../../../styles.css';

.collection-search-form {
  @apply my-4 mx-0 grid grid-cols-[minmax(0,1fr)_max-content] gap-2.5 down-compact:grid-cols-1;
}

.collection-dialog-list {
  @apply grid gap-2 my-3.5 mx-4 [&_.row]:items-center [&_.row]:rounded-lg [&_.row]:py-3 [&_.row]:px-3 [&_.row_strong]:block [&_.row_small]:block [&_.row_small]:mt-1 [&_.row:disabled_.status-pill]:bg-surface-muted [&_.row:disabled_.status-pill]:text-muted;
}

.row {
  @apply min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1;
}

.collection-dialog-item-main {
  @apply grid grid-cols-[38px_minmax(0,1fr)] items-center gap-2.5 min-w-0;
}

.collection-dialog-avatar {
  @apply grid place-items-center w-10 h-10 overflow-hidden border border-line-strong rounded-md bg-surface-muted text-primary-strong font-black [&_img]:w-full [&_img]:h-full [&_img]:object-cover;
}

.row strong {
  overflow-wrap: anywhere;
}

.row small {
  overflow-wrap: anywhere;
}
</style>
