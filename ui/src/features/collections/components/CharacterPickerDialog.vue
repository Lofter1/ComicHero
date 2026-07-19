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
  <div
    class="modal-backdrop fixed inset-0 z-60 grid place-items-center bg-backdrop p-4"
    @click.self="$emit('close')"
  >
    <section
      class="collection-dialog [&_.collection-create-form_input]:w-full [&_.collection-create-form_input]:min-w-0 [&_.collection-create-form_input]:min-h-10 [&_.collection-create-form_input]:border [&_.collection-create-form_input]:border-line-strong [&_.collection-create-form_input]:rounded [&_.collection-create-form_input]:bg-surface [&_.collection-create-form_input]:text-control [&_.collection-create-form_input]:py-2 [&_.collection-create-form_input]:px-3 [&_.collection-create-form_input:focus]:outline-3 [&_.collection-create-form_input:focus]:outline-focus [width:min(620px,_calc(100%_-_28px))] [max-height:min(720px,_calc(100dvh_-_28px))] overflow-auto border border-line-strong rounded-xl bg-panel p-5 shadow-elevated [&_>_.panel-header]:items-start [&_>_.panel-header]:mb-4 [&_>_.panel-header]:pb-3.5 [&_>_.panel-header]:border-b [&_>_.panel-header]:border-line [&_>_.panel-header_h3]:mt-1 [&_>_.panel-header_h3]:mx-0 [&_>_.panel-header_h3]:mb-0"
      role="dialog"
      aria-modal="true"
      aria-labelledby="add-character-title"
    >
      <header
        class="panel-header justify-between mb-4 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5"
      >
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">
            Collection members
          </p>
          <h3 id="add-character-title">Add a character</h3>
          <p class="collection-dialog-description mt-1 mx-0 mb-0 text-muted text-sm leading-snug">
            Search names and aliases in your character library.
          </p>
        </div>
        <button
          class="icon-button min-h-10 border border-line-strong rounded bg-surface text-control self-end py-0 px-3 down-mobile:self-stretch down-mobile:w-full"
          type="button"
          aria-label="Close"
          @click="$emit('close')"
        >
          ×
        </button>
      </header>
      <form
        class="collection-search-form my-4 mx-0 grid [grid-template-columns:minmax(0,_1fr)_max-content] gap-2.5 [&_input]:w-full [&_input]:min-w-0 [&_input]:min-h-10 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-control [&_input]:py-2 [&_input]:px-3 [&_input:focus]:outline-3 [&_input:focus]:outline-focus down-compact:grid-cols-1"
        @submit.prevent="search"
      >
        <input v-model="query" type="search" placeholder="Search characters or aliases" autofocus />
        <button
          class="primary-button min-h-10 border rounded py-2.5 px-3.5 border-primary bg-primary text-white"
          type="submit"
          :disabled="searching || !query.trim()"
        >
          {{ searching ? 'Searching...' : 'Search' }}
        </button>
      </form>
      <LoadingState v-if="searching" compact />
      <p v-else-if="searchError" class="error-text text-danger font-bold">{{ searchError }}</p>
      <div
        v-else-if="results.length"
        class="collection-dialog-list grid gap-2 my-3.5 mx-4 [&_.row]:items-center [&_.row]:rounded-lg [&_.row]:py-3 [&_.row]:px-3 [&_.row_strong]:block [&_.row_small]:block [&_.row_small]:mt-1 [&_.row:disabled_.status-pill]:bg-surface-muted [&_.row:disabled_.status-pill]:text-muted"
      >
        <button
          v-for="character in results"
          :key="character.id"
          class="row min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
          type="button"
          :disabled="saving"
          @click="add(character)"
        >
          <span
            class="collection-dialog-item-main grid [grid-template-columns:38px_minmax(0,_1fr)] items-center gap-2.5 min-w-0"
          >
            <span
              class="collection-dialog-avatar grid place-items-center w-10 h-10 overflow-hidden border border-line-strong rounded-md bg-surface-muted text-primary-strong font-black [&_img]:w-full [&_img]:h-full [&_img]:object-cover"
              aria-hidden="true"
            >
              <img v-if="character.image" :src="assetURL(character.image)" alt="" loading="lazy" />
              <span v-else>{{ monogram(character.name) }}</span>
            </span>
            <span>
              <strong>{{ character.name }}</strong>
              <small>{{ character.appearanceCount }} appearances</small>
            </span>
          </span>
          <span
            class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-xs flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
            >Add</span
          >
        </button>
      </div>
      <p v-else-if="searched" class="muted block text-muted">No characters match this search.</p>
      <p v-else class="muted block text-muted">Search your imported characters by name or alias.</p>
    </section>
  </div>
</template>
