<script setup>
import { ref } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import PanelHeader from '@/shared/components/layout/PanelHeader.vue'
import ModalShell from '@/shared/components/overlay/ModalShell.vue'
import StatusPill from '@/shared/components/feedback/StatusPill.vue'

defineProps({
  target: { type: Object, required: true },
  candidates: { type: Array, default: () => [] },
  searching: { type: Boolean, default: false },
  saving: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'search', 'merge'])
const query = ref('')

function search() {
  emit('search', query.value)
}
</script>

<template>
  <ModalShell v-slot="{ titleId }" size="large" @close="$emit('close')">
    <PanelHeader
      eyebrow="Admin"
      :title="`Merge a duplicate into ${target.title}`"
      :title-id="titleId"
      closable
      close-label="Close comic merge"
      @close="$emit('close')"
    />

    <p class="muted block text-muted">
      The selected duplicate will be deleted after its reading progress, list positions, arcs,
      characters, and missing metadata are moved to this comic.
    </p>

    <form
      class="comic-merge-search grid grid-cols-[minmax(0,1fr)_max-content] gap-2.5 my-4 mx-0"
      @submit.prevent="search"
    >
      <BaseTextInput
        v-model="query"
        type="search"
        placeholder="Search duplicate comics"
        autofocus
      />
      <BaseButton variant="primary" type="submit" :disabled="searching || saving">
        {{ searching ? 'Searching...' : 'Search' }}
      </BaseButton>
    </form>

    <div v-if="candidates.length" class="comic-merge-results grid gap-2">
      <!-- Native buttons: merge candidates are full-card selection targets. -->
      <button
        v-for="comic in candidates"
        :key="comic.id"
        class="row"
        type="button"
        :disabled="saving"
        @click="emit('merge', comic)"
      >
        <span>
          <strong>{{ comic.title }}</strong>
          <small>{{ comic.publisher || 'Unknown publisher' }}</small>
        </span>
        <StatusPill>Merge</StatusPill>
      </button>
    </div>
    <p v-else-if="!searching" class="muted block text-muted">
      Search for the duplicate comic to merge.
    </p>
  </ModalShell>
</template>

<style scoped>
@reference '../../../styles.css';

.row {
  @apply min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1;
}

.row strong {
  overflow-wrap: anywhere;
}

.row small {
  overflow-wrap: anywhere;
}
</style>
