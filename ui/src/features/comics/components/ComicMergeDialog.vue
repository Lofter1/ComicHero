<script setup>
import { ref } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'

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
  <div
    class="modal-backdrop fixed inset-0 z-60 grid place-items-center bg-backdrop p-4"
    @click.self="$emit('close')"
  >
    <section
      class="comic-merge-dialog [width:min(680px,_calc(100%_-_28px))] [max-height:min(720px,_calc(100dvh_-_28px))] overflow-auto border border-line-strong rounded-lg bg-panel p-5 shadow-elevated"
      role="dialog"
      aria-modal="true"
      aria-labelledby="merge-title"
    >
      <header
        class="panel-header justify-between mb-4 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5"
      >
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Admin</p>
          <h3 id="merge-title">Merge a duplicate into {{ target.title }}</h3>
        </div>
        <button
          class="icon-button min-h-10 border border-line-strong rounded bg-surface text-control self-end py-0 px-3 down-mobile:self-stretch down-mobile:w-full"
          type="button"
          aria-label="Close comic merge"
          @click="$emit('close')"
        >
          ×
        </button>
      </header>

      <p class="muted block text-muted">
        The selected duplicate will be deleted after its reading progress, list positions, arcs,
        characters, and missing metadata are moved to this comic.
      </p>

      <form
        class="comic-merge-search grid [grid-template-columns:minmax(0,_1fr)_max-content] gap-2.5 my-4 mx-0"
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
        <button
          v-for="comic in candidates"
          :key="comic.id"
          class="row min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
          type="button"
          :disabled="saving"
          @click="emit('merge', comic)"
        >
          <span>
            <strong>{{ comic.title }}</strong>
            <small>{{ comic.publisher || 'Unknown publisher' }}</small>
          </span>
          <span
            class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-xs flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
            >Merge</span
          >
        </button>
      </div>
      <p v-else-if="!searching" class="muted block text-muted">
        Search for the duplicate comic to merge.
      </p>
    </section>
  </div>
</template>
