<script setup>
import { ref } from 'vue'

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
  <div class="modal-backdrop" @click.self="$emit('close')">
    <section
      class="comic-merge-dialog [width:min(680px,_calc(100%_-_28px))] [max-height:min(720px,_calc(100dvh_-_28px))] overflow-auto border border-line-strong rounded-lg [background:var(--panel-bg)] p-5 shadow-elevated"
      role="dialog"
      aria-modal="true"
      aria-labelledby="merge-title"
    >
      <header class="panel-header">
        <div>
          <p class="eyebrow">Admin</p>
          <h3 id="merge-title">Merge a duplicate into {{ target.title }}</h3>
        </div>
        <button
          class="icon-button"
          type="button"
          aria-label="Close comic merge"
          @click="$emit('close')"
        >
          ×
        </button>
      </header>

      <p class="muted">
        The selected duplicate will be deleted after its reading progress, list positions, arcs,
        characters, and missing metadata are moved to this comic.
      </p>

      <form
        class="comic-merge-search grid [grid-template-columns:minmax(0,_1fr)_max-content] gap-2.5 [margin:18px_0]"
        @submit.prevent="search"
      >
        <input v-model="query" type="search" placeholder="Search duplicate comics" autofocus />
        <button class="primary-button" type="submit" :disabled="searching || saving">
          {{ searching ? 'Searching...' : 'Search' }}
        </button>
      </form>

      <div v-if="candidates.length" class="comic-merge-results grid gap-2">
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
          <span class="status-pill">Merge</span>
        </button>
      </div>
      <p v-else-if="!searching" class="muted">Search for the duplicate comic to merge.</p>
    </section>
  </div>
</template>
