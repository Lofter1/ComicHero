<script setup>
import { ref } from 'vue'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import PanelHeader from '@/shared/components/layout/PanelHeader.vue'
import ModalShell from '@/shared/components/overlay/ModalShell.vue'
import StatusPill from '@/shared/components/feedback/StatusPill.vue'

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
  <ModalShell v-slot="{ titleId }" @close="$emit('close')">
    <PanelHeader
      eyebrow="My collections"
      :title="`Add ${character.name} to a collection`"
      :title-id="titleId"
      divided
      closable
      @close="$emit('close')"
    >
      <template #description>
        <p class="collection-dialog-description mt-1 mx-0 mb-0 text-muted text-sm leading-snug">
          Choose an existing collection or create one below.
        </p>
      </template>
    </PanelHeader>

    <LoadingState v-if="loading" compact />
    <div v-else-if="collections.length" class="collection-dialog-list">
      <!-- Native button: collection rows are full-card selection targets. -->
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
        <StatusPill>
          {{ collection.containsCharacter ? 'Added' : 'Add' }}
        </StatusPill>
      </button>
    </div>
    <p v-else-if="!loading" class="muted block text-muted">You do not have any collections yet.</p>

    <form class="collection-create-form" @submit.prevent="createCollection">
      <label for="new-character-collection">Create a new collection</label>
      <div>
        <BaseTextInput id="new-character-collection" v-model="newName" maxlength="120" />
        <BaseButton variant="primary" type="submit" :disabled="saving || !newName.trim()">
          {{ saving ? 'Saving...' : 'Create and add' }}
        </BaseButton>
      </div>
    </form>
  </ModalShell>
</template>

<style scoped>
@reference '../../../styles.css';

.collection-dialog-list {
  @apply grid gap-2 my-3.5 mx-4 [&_.row]:items-center [&_.row]:rounded-lg [&_.row]:py-3 [&_.row]:px-3 [&_.row_strong]:block [&_.row_small]:block [&_.row_small]:mt-1 [&_.row:disabled_.status-pill]:bg-surface-muted [&_.row:disabled_.status-pill]:text-muted;
}

.row {
  @apply min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1;
}

.collection-dialog-item-main {
  @apply grid grid-cols-[38px_minmax(0,1fr)] items-center gap-2.5 min-w-0;
}

.collection-dialog-monogram {
  @apply bg-primary-soft grid place-items-center w-10 h-10 overflow-hidden border border-line-strong rounded-md text-primary-strong font-black;
}

.collection-create-form {
  @apply grid gap-2 mt-4 pt-4 border-t border-line [&_>_label]:text-label [&_>_label]:text-sm [&_>_label]:font-extrabold [&_>_div]:grid [&_>_div]:grid-cols-[minmax(0,1fr)_max-content] [&_>_div]:gap-2.5 down-compact:[&_>_div]:grid-cols-1;
}

.row strong {
  overflow-wrap: anywhere;
}

.row small {
  overflow-wrap: anywhere;
}
</style>
