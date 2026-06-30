<script setup>
import { computed, ref } from 'vue'
import { comicLabel } from '@/domain/comics.js'

const props = defineProps({
  form: {
    type: Object,
    required: true,
  },
  selectedOrder: {
    type: Object,
    default: null,
  },
  comics: {
    type: Array,
    required: true,
  },
  saving: {
    type: Boolean,
    default: false,
  },
  formId: {
    type: String,
    default: 'reading-order-editor-form',
  },
  itemLabel: {
    type: String,
    default: 'reading order',
  },
  emptyEntryMessage: {
    type: String,
    default: 'No comics in this reading order yet.',
  },
})

const emit = defineEmits(['update:form', 'save', 'delete'])

const draggedIndex = ref(null)
const dragOverIndex = ref(null)
const comicSearch = ref('')

const comicSearchResults = computed(() => {
  const term = comicSearch.value.trim().toLowerCase()
  const matches = term ? props.comics.filter(comicMatchesAddSearch) : props.comics
  return matches.slice(0, 8)
})

function updateForm(patch) {
  emit('update:form', {
    ...props.form,
    ...patch,
  })
}

function updateEntry(index, patch) {
  const entries = props.form.comics.map((entry, entryIndex) => {
    return entryIndex === index ? { ...entry, ...patch } : entry
  })
  updateForm({ comics: entries })
}

function comicMatchesAddSearch(comic) {
  const term = comicSearch.value.trim().toLowerCase()
  return [comic.title, comic.series, comic.seriesYear, comic.publisher, comic.issue, comic.coverDate, comic.read ? 'read' : 'unread']
    .filter(value => value !== undefined && value !== null)
    .some(value => String(value).toLowerCase().includes(term))
}

function comicSeriesLine(comic) {
  const series = comic.series || 'Unknown series'
  const year = comic.seriesYear ? ` (${comic.seriesYear})` : ''
  const issue = comic.issue ? ` #${comic.issue}` : ''
  return `${series}${year}${issue}`
}

function comicMetaLine(comic) {
  return [
    comic.publisher || 'Unknown publisher',
    comic.coverDate || 'Unknown date',
    comic.read ? 'Read' : 'Unread',
  ].join(' · ')
}

function addEntry(comicId) {
  if (!comicId) return
  updateForm({
    comics: [
      ...props.form.comics,
      {
        comicId,
        comment: '',
        tags: '',
      },
    ],
  })
}

function clearComicSearch() {
  comicSearch.value = ''
}

function removeEntry(index) {
  updateForm({
    comics: props.form.comics.filter((_, entryIndex) => entryIndex !== index),
  })
}

function moveEntry(index, offset) {
  reorderEntry(index, index + offset)
}

function reorderEntry(fromIndex, toIndex) {
  if (fromIndex === toIndex) return
  if (fromIndex < 0 || toIndex < 0) return
  if (fromIndex >= props.form.comics.length || toIndex >= props.form.comics.length) return

  const entries = [...props.form.comics]
  const [entry] = entries.splice(fromIndex, 1)
  entries.splice(toIndex, 0, entry)
  updateForm({ comics: entries })
}

function startDrag(event, index) {
  draggedIndex.value = index
  dragOverIndex.value = index
  event.dataTransfer.effectAllowed = 'move'
  event.dataTransfer.setData('text/plain', String(index))
}

function overDrag(event, index) {
  event.preventDefault()
  dragOverIndex.value = index
  event.dataTransfer.dropEffect = 'move'
}

function dropEntry(event, index) {
  event.preventDefault()
  const fromIndex = draggedIndex.value ?? Number(event.dataTransfer.getData('text/plain'))
  reorderEntry(fromIndex, index)
  endDrag()
}

function endDrag() {
  draggedIndex.value = null
  dragOverIndex.value = null
}
</script>

<template>
  <form :id="formId" class="edit-form" @submit.prevent="$emit('save')">
    <label>
      Name
      <input :value="form.name" required @input="updateForm({ name: $event.target.value })" />
    </label>
    <label>
      Description
      <textarea :value="form.description" rows="3" @input="updateForm({ description: $event.target.value })" />
    </label>

    <section class="entry-section">
      <div class="section-title">
        <h4>Comics</h4>
      </div>

      <div v-if="comics.length" class="comic-add-panel">
        <div class="comic-add-search">
          <input
            v-model="comicSearch"
            type="search"
            placeholder="Search title, series, issue, publisher, status"
            @keydown.enter.prevent
          />
          <button v-if="comicSearch" class="ghost-button" type="button" @click="clearComicSearch">Clear</button>
        </div>
        
        <div v-if="comicSearchResults.length" class="comic-add-results">
          <button
            v-for="comic in comicSearchResults"
            :key="comic.id"
            type="button"
            class="comic-add-result"
            @click="addEntry(comic.id)"
          >
            <span>
              <strong>{{ comic.title }}</strong>
              <small>{{ comicSeriesLine(comic) }}</small>
              <small>{{ comicMetaLine(comic) }}</small>
            </span>
            <span class="status-pill">Add</span>
          </button>
        </div>
        <p v-else class="muted">No comics match that search.</p>
      </div>
      <div v-else class="empty-state">Add comics before building a reading order.</div>

      <div v-if="comics.length && form.comics.length === 0" class="empty-state">
        {{ emptyEntryMessage }}
      </div>

      <div
        v-for="(entry, index) in form.comics"
        :key="index"
        class="order-entry"
        :class="{
          dragging: draggedIndex === index,
          'drag-over': dragOverIndex === index && draggedIndex !== index,
        }"
        @dragover="overDrag($event, index)"
        @drop="dropEntry($event, index)"
      >
        <div class="entry-position">
          <div class="mobile-reorder">
            <button type="button" :disabled="index === 0" @click="moveEntry(index, -1)">Up</button>
            <button type="button" :disabled="index === form.comics.length - 1" @click="moveEntry(index, 1)">Down</button>
          </div>
          <span
            class="drag-handle"
            draggable="true"
            role="img"
            aria-label="Drag to reorder"
            title="Drag to reorder"
            @dragstart="startDrag($event, index)"
            @dragend="endDrag"
          >
            <span aria-hidden="true" class="drag-icon">⋮⋮</span>
          </span>
        </div>
        <div class="selected-order-comic">
          <strong>{{ comicLabel(comics, entry.comicId) }}</strong>
        </div>
        <label class="comment-input-label">
          <input
            :value="entry.comment"
            aria-label="Entry comment"
            placeholder="Optional note for this spot"
            @input="updateEntry(index, { comment: $event.target.value })"
          />
        </label>
        <label class="comment-input-label">
          <input
            :value="entry.tags"
            aria-label="Entry tags"
            placeholder="Tags"
            @input="updateEntry(index, { tags: $event.target.value })"
          />
        </label>
        <button type="button" class="remove-entry-button" :aria-label="`Remove comic from ${itemLabel}`" title="Remove" @click="removeEntry(index)">
          <span aria-hidden="true">×</span>
        </button>
      </div>
    </section>

  </form>
</template>
