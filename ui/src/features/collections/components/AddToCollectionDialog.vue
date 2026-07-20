<script setup>
import { ref } from 'vue'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'

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
  <div
    class="modal-backdrop fixed inset-0 z-60 grid place-items-center bg-backdrop p-4"
    @click.self="$emit('close')"
  >
    <section
      class="collection-dialog w-[min(620px,calc(100%-28px))] max-h-[min(720px,calc(100dvh-28px))] overflow-auto border border-line-strong rounded-xl bg-panel p-5 shadow-elevated [&_>_.panel-header]:items-start [&_>_.panel-header]:mb-4 [&_>_.panel-header]:pb-3.5 [&_>_.panel-header]:border-b [&_>_.panel-header]:border-line [&_>_.panel-header_h3]:mt-1 [&_>_.panel-header_h3]:mx-0 [&_>_.panel-header_h3]:mb-0"
      role="dialog"
      aria-modal="true"
      aria-labelledby="add-to-collection-title"
    >
      <header
        class="panel-header justify-between mb-4 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5"
      >
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">My collections</p>
          <h3 id="add-to-collection-title">Add {{ character.name }} to a collection</h3>
          <p class="collection-dialog-description mt-1 mx-0 mb-0 text-muted text-sm leading-snug">
            Choose an existing collection or create one below.
          </p>
        </div>
        <BaseButton
          class="self-end down-mobile:self-stretch down-mobile:w-full"
          variant="neutral"
          size="icon"
          aria-label="Close"
          @click="$emit('close')"
        >
          ×
        </BaseButton>
      </header>

      <LoadingState v-if="loading" compact />
      <div
        v-else-if="collections.length"
        class="collection-dialog-list grid gap-2 my-3.5 mx-4 [&_.row]:items-center [&_.row]:rounded-lg [&_.row]:py-3 [&_.row]:px-3 [&_.row_strong]:block [&_.row_small]:block [&_.row_small]:mt-1 [&_.row:disabled_.status-pill]:bg-surface-muted [&_.row:disabled_.status-pill]:text-muted"
      >
        <!-- Native button: collection rows are full-card selection targets. -->
        <button
          v-for="collection in collections"
          :key="collection.id"
          class="row min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
          type="button"
          :disabled="saving || collection.containsCharacter"
          @click="emit('add', collection)"
        >
          <span
            class="collection-dialog-item-main grid grid-cols-[38px_minmax(0,1fr)] items-center gap-2.5 min-w-0"
          >
            <span
              class="collection-dialog-monogram bg-primary-soft grid place-items-center w-10 h-10 overflow-hidden border border-line-strong rounded-md text-primary-strong font-black"
              aria-hidden="true"
            >
              {{ monogram(collection.name) }}
            </span>
            <span>
              <strong>{{ collection.name }}</strong>
              <small>{{ collection.characterCount }} characters</small>
            </span>
          </span>
          <span
            class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-xs flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
          >
            {{ collection.containsCharacter ? 'Added' : 'Add' }}
          </span>
        </button>
      </div>
      <p v-else-if="!loading" class="muted block text-muted">
        You do not have any collections yet.
      </p>

      <form
        class="collection-create-form grid gap-2 mt-4 pt-4 border-t border-line [&_>_label]:text-label [&_>_label]:text-sm [&_>_label]:font-extrabold [&_>_div]:grid [&_>_div]:grid-cols-[minmax(0,1fr)_max-content] [&_>_div]:gap-2.5 down-compact:[&_>_div]:grid-cols-1"
        @submit.prevent="createCollection"
      >
        <label for="new-character-collection">Create a new collection</label>
        <div>
          <BaseTextInput id="new-character-collection" v-model="newName" maxlength="120" />
          <BaseButton variant="primary" type="submit" :disabled="saving || !newName.trim()">
            {{ saving ? 'Saving...' : 'Create and add' }}
          </BaseButton>
        </div>
      </form>
    </section>
  </div>
</template>
