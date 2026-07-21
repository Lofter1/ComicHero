<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'

import { assetURL, listComics } from '@/api/client.js'
import { comicLabel } from '@/features/comics/model.js'
import {
  readingOrderEditorPage,
  readingOrderEditorPageSize,
  reorderReadingOrderEntry,
} from '@/features/reading-orders/model.js'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import EmptyState from '@/shared/components/feedback/EmptyState.vue'
import StatusPill from '@/shared/components/feedback/StatusPill.vue'

const props = defineProps({
  form: { type: Object, required: true },
  selectedOrder: { type: Object, default: null },
  comics: { type: Array, required: true },
  readingOrders: { type: Array, default: () => [] },
  saving: { type: Boolean, default: false },
  formId: { type: String, default: 'reading-order-editor-form' },
  itemLabel: { type: String, default: 'reading order' },
  emptyEntryMessage: {
    type: String,
    default: 'No comics in this reading order yet.',
  },
})

const emit = defineEmits(['update:form', 'save', 'delete'])

const draggedIndex = ref(null)
const dragOverIndex = ref(null)
const comicSearch = ref('')
const comicSearchResults = ref([])
const comicSearchLoading = ref(false)
const comicSearchError = ref('')
const readingOrderSearch = ref('')
const sectionTitle = ref('')
const sectionDescription = ref('')
const activeAddType = ref('comic')
const expandedEntryKeys = ref(new Set())
const entryPage = ref(0)

let comicSearchTimer = null
let comicSearchRequestId = 0

const orderEntries = computed(() => props.form.entries || props.form.comics || [])
const entryPageState = computed(() => readingOrderEditorPage(orderEntries.value, entryPage.value))
const coverPreview = computed(() => {
  if (props.form.coverImageData) return props.form.coverImageData
  if (props.form.image) return assetURL(props.form.image)
  return ''
})

watch(
  () => props.form.id,
  () => {
    expandedEntryKeys.value = new Set()
    entryPage.value = 0
  },
)

watch(
  () => orderEntries.value.length,
  () => {
    entryPage.value = entryPageState.value.page
  },
)

watch(
  comicSearch,
  () => {
    queueComicSearch()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  clearTimeout(comicSearchTimer)
  comicSearchRequestId += 1
})

function queueComicSearch() {
  clearTimeout(comicSearchTimer)
  comicSearchTimer = setTimeout(searchComicsForReadingOrder, 250)
}

async function searchComicsForReadingOrder() {
  const requestId = ++comicSearchRequestId
  const q = comicSearch.value.trim()

  comicSearchLoading.value = true
  comicSearchError.value = ''

  try {
    const page = await listComics({
      q,
      limit: 8,
      offset: 0,
    })

    if (requestId === comicSearchRequestId) {
      comicSearchResults.value = page.items
    }
  } catch (error) {
    if (requestId === comicSearchRequestId) {
      comicSearchResults.value = []
      comicSearchError.value = error?.message || 'Could not search comics.'
    }
  } finally {
    if (requestId === comicSearchRequestId) {
      comicSearchLoading.value = false
    }
  }
}

const childOrderChoices = computed(() => {
  const selected = new Set(
    orderEntries.value
      .filter((entry) => entry.type === 'readingOrder')
      .map((entry) => Number(entry.readingOrderId)),
  )

  const term = readingOrderSearch.value.trim().toLowerCase()

  return props.readingOrders
    .filter((order) => order.id !== props.form.id && !selected.has(order.id))
    .filter((order) => {
      if (!term) return true

      return [order.name, order.description]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(term))
    })
    .slice(0, 6)
})

function updateForm(patch) {
  emit('update:form', {
    ...props.form,
    ...patch,
  })
}

function chooseCoverImage(event) {
  const file = event.target.files?.[0]
  if (!file) return

  const reader = new FileReader()
  reader.addEventListener('load', () => {
    updateForm({
      coverImageData: typeof reader.result === 'string' ? reader.result : '',
    })
  })
  reader.readAsDataURL(file)
}

function updateEntry(index, patch) {
  const entries = orderEntries.value.map((entry, entryIndex) => {
    return entryIndex === index ? { ...entry, ...patch } : entry
  })

  updateForm({ entries })
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

function insertEntryAt(entry, index) {
  const entries = [...orderEntries.value]
  const safeIndex = Math.max(0, Math.min(index, entries.length))

  entries.splice(safeIndex, 0, entry)
  updateForm({ entries })
}

function entryKey(entry, index) {
  if (entry.type === 'section') return `section:${index}`
  const id = entry.type === 'readingOrder' ? entry.readingOrderId : entry.comicId
  return `${entry.type}:${id}:${index}`
}

function isEntryExpanded(entry, index) {
  return expandedEntryKeys.value.has(entryKey(entry, index))
}

function toggleEntry(entry, index) {
  const key = entryKey(entry, index)
  const expanded = new Set(expandedEntryKeys.value)

  if (expanded.has(key)) {
    expanded.delete(key)
  } else {
    expanded.add(key)
  }

  expandedEntryKeys.value = expanded
}

function collapseRemovedEntry(index) {
  const expanded = new Set()

  orderEntries.value.forEach((entry, entryIndex) => {
    if (entryIndex !== index && expandedEntryKeys.value.has(entryKey(entry, entryIndex))) {
      const nextIndex = entryIndex > index ? entryIndex - 1 : entryIndex
      expanded.add(entryKey(entry, nextIndex))
    }
  })

  expandedEntryKeys.value = expanded
}

function startAddDrag(event, entry) {
  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData('application/x-comichero-entry', JSON.stringify(entry))
}

function newComicEntry(comic) {
  return {
    type: 'comic',
    comicId: comic.id,
    title: comic.title || '',
    comment: '',
    tags: '',
  }
}

function newChildOrderEntry(order) {
  return {
    type: 'readingOrder',
    readingOrderId: order.id,
    title: order.name || '',
    description: order.description || '',
    comment: '',
  }
}

function addSection() {
  const title = sectionTitle.value.trim()
  if (!title) return

  addNewEntryToEnd({
    type: 'section',
    title,
    description: sectionDescription.value.trim(),
  })
  sectionTitle.value = ''
  sectionDescription.value = ''
}

function addNewEntryToEnd(entry) {
  const index = orderEntries.value.length
  insertEntryAt(entry, index)
  entryPage.value = Math.floor(index / readingOrderEditorPageSize)
}

function goToEntryPage(page) {
  endDrag()
  entryPage.value = Math.min(Math.max(0, page), entryPageState.value.pageCount - 1)
}

function dropAt(event, index) {
  event.preventDefault()

  const newEntry = event.dataTransfer.getData('application/x-comichero-entry')

  if (newEntry) {
    insertEntryAt(JSON.parse(newEntry), index)
    endDrag()
    return
  }

  const fromIndex = Number(
    event.dataTransfer.getData('application/x-comichero-move') ||
      event.dataTransfer.getData('text/plain'),
  )

  if (Number.isFinite(fromIndex)) {
    moveEntryTo(fromIndex, index)
  }

  endDrag()
}

function overDropZone(event, index) {
  event.preventDefault()
  dragOverIndex.value = index
  event.dataTransfer.dropEffect = Array.from(event.dataTransfer.types).includes(
    'application/x-comichero-entry',
  )
    ? 'copy'
    : 'move'
}

function moveEntryTo(fromIndex, insertIndex) {
  if (fromIndex < 0 || fromIndex >= orderEntries.value.length) return

  const entries = [...orderEntries.value]
  const [entry] = entries.splice(fromIndex, 1)
  const targetIndex = Math.max(
    0,
    Math.min(fromIndex < insertIndex ? insertIndex - 1 : insertIndex, entries.length),
  )

  entries.splice(targetIndex, 0, entry)
  updateForm({ entries })
}

function clearComicSearch() {
  comicSearch.value = ''
}

function removeEntry(index) {
  collapseRemovedEntry(index)

  updateForm({
    entries: orderEntries.value.filter((_, entryIndex) => entryIndex !== index),
  })
}

function entryLabel(entry) {
  if (entry.type === 'section') return entry.title || 'Untitled section'
  if (entry.type === 'readingOrder') return entry.title || 'Unknown reading order'

  return entry.title || comicLabel(props.comics, entry.comicId)
}

function entryTypeLabel(entry) {
  if (entry.type === 'section') return 'Section'
  return entry.type === 'readingOrder' ? 'Reading order' : 'Issue'
}

function moveEntry(index, offset) {
  reorderEntry(index, index + offset)
}

function reorderEntry(fromIndex, toIndex) {
  const entries = reorderReadingOrderEntry(orderEntries.value, fromIndex, toIndex)
  if (entries === orderEntries.value) return
  updateForm({ entries })
}

function moveEntryAcrossPage(index, direction) {
  const targetPage = entryPageState.value.page + direction
  reorderEntry(index, index + direction)
  entryPage.value = targetPage
}

function startDrag(event, index) {
  draggedIndex.value = index
  dragOverIndex.value = index

  event.dataTransfer.effectAllowed = 'move'
  event.dataTransfer.setData('application/x-comichero-move', String(index))
  event.dataTransfer.setData('text/plain', String(index))
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
      <BaseTextInput
        :model-value="form.name"
        required
        @update:model-value="updateForm({ name: $event })"
      />
    </label>

    <label>
      Description
      <textarea
        :value="form.description"
        rows="3"
        @input="updateForm({ description: $event.target.value })"
      />
    </label>

    <fieldset class="visibility-field">
      <legend>Visibility</legend>
      <label>
        <input
          type="radio"
          name="reading-order-visibility"
          :checked="form.isPublic"
          @change="updateForm({ isPublic: true })"
        />
        Public
      </label>
      <p class="muted block text-muted">
        Anyone with access to ComicHero can view this reading order.
      </p>
      <label>
        <input
          type="radio"
          name="reading-order-visibility"
          :checked="!form.isPublic"
          @change="updateForm({ isPublic: false })"
        />
        Private
      </label>
      <p class="muted block text-muted">Only you and administrators can view this reading order.</p>
    </fieldset>

    <label class="cover-image-field">
      Cover image
      <span class="reading-order-cover-editor flex items-center gap-3 min-w-0">
        <span v-if="coverPreview" class="reading-order-cover-preview" aria-hidden="true">
          <img :src="coverPreview" alt="" />
        </span>
        <input class="min-w-0" type="file" accept="image/*" @change="chooseCoverImage" />
      </span>
    </label>

    <div class="reading-order-editor-layout">
      <div class="reading-order-search-column">
        <section class="entry-section add-entry-panel grid gap-2.5 border-t border-line pt-3.5">
          <div class="section-title">
            <h4>Add Entries</h4>
          </div>

          <div
            class="add-entry-tabs grid grid-cols-3 gap-2"
            role="tablist"
            aria-label="Entry source"
          >
            <!-- Native buttons: entry sources are a stateful tablist. -->
            <button
              type="button"
              class="entry-source-button"
              :class="{ active: activeAddType === 'comic' }"
              @click="activeAddType = 'comic'"
            >
              Issues
            </button>
            <button
              type="button"
              class="entry-source-button"
              :class="{ active: activeAddType === 'readingOrder' }"
              @click="activeAddType = 'readingOrder'"
            >
              Reading Orders
            </button>
            <button
              type="button"
              class="entry-source-button"
              :class="{ active: activeAddType === 'section' }"
              @click="activeAddType = 'section'"
            >
              Sections
            </button>
          </div>

          <div v-if="activeAddType === 'comic'" class="comic-add-panel">
            <div class="comic-add-search">
              <BaseTextInput
                v-model="comicSearch"
                variant="embedded"
                size="compact"
                type="search"
                placeholder="Search title, series, issue, publisher, status"
                @keydown.enter.prevent
              />
              <!-- Native button: Clear is embedded inside the compound search field. -->
              <button
                v-if="comicSearch"
                class="ghost-button"
                type="button"
                @click="clearComicSearch"
              >
                Clear
              </button>
            </div>

            <p v-if="comicSearchLoading" class="muted block text-muted">Searching comics...</p>
            <p v-else-if="comicSearchError" class="muted block text-muted">
              {{ comicSearchError }}
            </p>

            <div v-else-if="comicSearchResults.length" class="comic-add-results">
              <!-- Native buttons: search results are draggable full-card targets. -->
              <button
                v-for="comic in comicSearchResults"
                :key="comic.id"
                type="button"
                class="comic-add-result"
                draggable="true"
                @click="addNewEntryToEnd(newComicEntry(comic))"
                @dragstart="startAddDrag($event, newComicEntry(comic))"
              >
                <span>
                  <span class="entry-type-pill">Issue</span>
                  <strong>{{ comic.title }}</strong>
                  <small>{{ comicSeriesLine(comic) }}</small>
                  <small>{{ comicMetaLine(comic) }}</small>
                </span>

                <StatusPill>Add</StatusPill>
              </button>
            </div>

            <p v-else class="muted block text-muted">No comics match that search.</p>
          </div>

          <div
            v-if="activeAddType === 'readingOrder' && readingOrders.length"
            class="comic-add-panel"
          >
            <div class="comic-add-search">
              <BaseTextInput
                v-model="readingOrderSearch"
                variant="embedded"
                size="compact"
                type="search"
                placeholder="Search reading orders"
                @keydown.enter.prevent
              />
              <!-- Native button: Clear is embedded inside the compound search field. -->
              <button
                v-if="readingOrderSearch"
                class="ghost-button"
                type="button"
                @click="readingOrderSearch = ''"
              >
                Clear
              </button>
            </div>

            <div v-if="childOrderChoices.length" class="comic-add-results">
              <!-- Native buttons: reading-order results are draggable full-card targets. -->
              <button
                v-for="order in childOrderChoices"
                :key="order.id"
                type="button"
                class="comic-add-result reading-order-add-result"
                draggable="true"
                @click="addNewEntryToEnd(newChildOrderEntry(order))"
                @dragstart="startAddDrag($event, newChildOrderEntry(order))"
              >
                <span>
                  <span class="entry-type-pill">Reading order</span>
                  <strong>{{ order.name }}</strong>
                  <small>{{ order.description || 'No description' }}</small>
                </span>

                <StatusPill>Add</StatusPill>
              </button>
            </div>

            <p v-else class="muted block text-muted">No reading orders match that search.</p>
          </div>

          <EmptyState v-else-if="activeAddType === 'readingOrder'">
            No reading orders available to include.
          </EmptyState>

          <div v-if="activeAddType === 'section'" class="section-add-panel">
            <label>
              Section title
              <BaseTextInput
                v-model="sectionTitle"
                placeholder="Main story"
                @keydown.enter.prevent="addSection"
              />
            </label>
            <label>
              Description
              <textarea
                v-model="sectionDescription"
                rows="3"
                placeholder="Optional context for this section"
              />
            </label>
            <BaseButton
              class="justify-self-start"
              type="button"
              variant="primary"
              :disabled="!sectionTitle.trim()"
              @click="addSection"
            >
              Add Section
            </BaseButton>
          </div>
        </section>
      </div>

      <section class="entry-section reading-order-list-edit">
        <div class="section-title">
          <h4>List Order</h4>
          <span>{{ orderEntries.length }} entries</span>
        </div>

        <nav
          v-if="entryPageState.pageCount > 1"
          class="reading-order-entry-pages"
          aria-label="Reading order entry pages"
        >
          <span>
            Entries {{ entryPageState.start + 1 }}–{{ entryPageState.end }} of
            {{ orderEntries.length }}
          </span>
          <div>
            <BaseButton
              class="down-mobile:w-full"
              type="button"
              variant="secondary"
              size="compact"
              :disabled="entryPageState.page === 0"
              @click="goToEntryPage(0)"
            >
              First
            </BaseButton>
            <BaseButton
              class="down-mobile:w-full"
              type="button"
              variant="secondary"
              size="compact"
              :disabled="entryPageState.page === 0"
              @click="goToEntryPage(entryPageState.page - 1)"
            >
              Previous
            </BaseButton>
            <strong>Page {{ entryPageState.page + 1 }} of {{ entryPageState.pageCount }}</strong>
            <BaseButton
              class="down-mobile:w-full"
              type="button"
              variant="secondary"
              size="compact"
              :disabled="entryPageState.page === entryPageState.pageCount - 1"
              @click="goToEntryPage(entryPageState.page + 1)"
            >
              Next
            </BaseButton>
            <BaseButton
              class="down-mobile:w-full"
              type="button"
              variant="secondary"
              size="compact"
              :disabled="entryPageState.page === entryPageState.pageCount - 1"
              @click="goToEntryPage(entryPageState.pageCount - 1)"
            >
              Last
            </BaseButton>
          </div>
        </nav>

        <div
          v-if="orderEntries.length === 0"
          class="empty-state empty-entry-drop-zone"
          :class="{ active: dragOverIndex === 0 }"
          @dragover="overDropZone($event, 0)"
          @dragleave="dragOverIndex = null"
          @drop="dropAt($event, 0)"
        >
          {{ emptyEntryMessage }}
        </div>

        <div v-else class="order-entry-list grid gap-1.5">
          <template
            v-for="{ entry, index } in entryPageState.entries"
            :key="entryKey(entry, index)"
          >
            <div
              class="entry-drop-zone"
              :class="{ active: dragOverIndex === index }"
              @dragover="overDropZone($event, index)"
              @dragleave="dragOverIndex = null"
              @drop="dropAt($event, index)"
            />

            <div
              class="order-entry sortable-order-entry"
              :class="{
                dragging: draggedIndex === index,
                'nested-order-entry': entry.type === 'readingOrder',
                'section-order-entry': entry.type === 'section',
                expanded: isEntryExpanded(entry, index),
              }"
            >
              <!-- Native button: entry rows use a bespoke expand/collapse control. -->
              <button
                type="button"
                class="selected-order-comic entry-summary-button"
                @click="toggleEntry(entry, index)"
              >
                <span class="entry-drag-cell">
                  <span
                    class="drag-handle"
                    draggable="true"
                    role="img"
                    aria-label="Drag to reorder"
                    title="Drag to reorder"
                    @click.stop
                    @dragstart="startDrag($event, index)"
                    @dragend="endDrag"
                  >
                    <span
                      aria-hidden="true"
                      class="drag-icon tracking-[-4px] transform-[rotate(90deg)]"
                      >⋮⋮</span
                    >
                  </span>
                </span>

                <span class="entry-summary-copy">
                  <span class="entry-type-pill">{{ entryTypeLabel(entry) }}</span>
                  <strong>{{ entryLabel(entry) }}</strong>
                </span>

                <span
                  aria-hidden="true"
                  class="button-icon entry-expand-icon"
                  :title="isEntryExpanded(entry, index) ? 'Collapse' : 'Expand'"
                >
                  {{ isEntryExpanded(entry, index) ? '▴' : '▾' }}
                </span>

                <span class="mobile-reorder" @click.stop>
                  <!-- Native buttons: compact reorder arrows use editor-specific styling. -->
                  <button
                    class="min-h-10 border border-line-strong rounded bg-surface"
                    type="button"
                    :disabled="index === 0"
                    @click="moveEntry(index, -1)"
                  >
                    Up
                  </button>
                  <button
                    class="min-h-10 border border-line-strong rounded bg-surface"
                    type="button"
                    :disabled="index === orderEntries.length - 1"
                    @click="moveEntry(index, 1)"
                  >
                    Down
                  </button>
                </span>
              </button>

              <!-- Native button: removal spans the full row height and has bespoke danger styling. -->
              <button
                type="button"
                class="remove-entry-button"
                :aria-label="`Remove ${entryLabel(entry)} from ${itemLabel}`"
                title="Remove"
                @click="removeEntry(index)"
              >
                <span aria-hidden="true">×</span>
              </button>

              <div
                v-if="
                  (index === entryPageState.start && entryPageState.page > 0) ||
                  (index === entryPageState.end - 1 &&
                    entryPageState.page < entryPageState.pageCount - 1)
                "
                class="entry-page-move flex col-span-full justify-end gap-2"
              >
                <BaseButton
                  v-if="index === entryPageState.start && entryPageState.page > 0"
                  type="button"
                  variant="secondary"
                  size="compact-wide"
                  @click="moveEntryAcrossPage(index, -1)"
                >
                  Move to previous page
                </BaseButton>
                <BaseButton
                  v-if="
                    index === entryPageState.end - 1 &&
                    entryPageState.page < entryPageState.pageCount - 1
                  "
                  type="button"
                  variant="secondary"
                  size="compact-wide"
                  @click="moveEntryAcrossPage(index, 1)"
                >
                  Move to next page
                </BaseButton>
              </div>

              <div v-if="isEntryExpanded(entry, index)" class="entry-edit-panel">
                <template v-if="entry.type === 'section'">
                  <label
                    class="comment-input-label self-stretch [&_textarea]:h-full [&_textarea]:min-h-0"
                  >
                    Section title
                    <BaseTextInput
                      :model-value="entry.title"
                      size="fill"
                      required
                      aria-label="Section title"
                      @update:model-value="updateEntry(index, { title: $event })"
                    />
                  </label>
                  <label
                    class="comment-input-label self-stretch [&_textarea]:h-full [&_textarea]:min-h-0"
                  >
                    Description
                    <textarea
                      :value="entry.description"
                      rows="3"
                      aria-label="Section description"
                      placeholder="Optional context for this section"
                      @input="updateEntry(index, { description: $event.target.value })"
                    />
                  </label>
                </template>

                <template v-else>
                  <label
                    class="comment-input-label self-stretch [&_textarea]:h-full [&_textarea]:min-h-0"
                  >
                    Note
                    <textarea
                      :value="entry.comment"
                      rows="3"
                      aria-label="Entry comment"
                      :placeholder="
                        entry.type === 'readingOrder'
                          ? 'Optional note for this reading order'
                          : 'Optional note for this spot'
                      "
                      @input="updateEntry(index, { comment: $event.target.value })"
                    />
                  </label>

                  <label
                    v-if="entry.type !== 'readingOrder'"
                    class="comment-input-label self-stretch [&_textarea]:h-full [&_textarea]:min-h-0"
                  >
                    Tags
                    <BaseTextInput
                      :model-value="entry.tags"
                      size="fill"
                      aria-label="Entry tags"
                      placeholder="Tags"
                      @update:model-value="updateEntry(index, { tags: $event })"
                    />
                  </label>
                </template>
              </div>
            </div>
          </template>

          <div
            class="entry-drop-zone end-zone"
            :class="{ active: dragOverIndex === entryPageState.end }"
            @dragover="overDropZone($event, entryPageState.end)"
            @dragleave="dragOverIndex = null"
            @drop="dropAt($event, entryPageState.end)"
          />
        </div>

        <nav
          v-if="entryPageState.pageCount > 1"
          class="reading-order-entry-pages reading-order-entry-pages-bottom"
          aria-label="Reading order entry pages"
        >
          <span>Page {{ entryPageState.page + 1 }} of {{ entryPageState.pageCount }}</span>
          <div>
            <BaseButton
              class="down-mobile:w-full"
              type="button"
              variant="secondary"
              size="compact"
              :disabled="entryPageState.page === 0"
              @click="goToEntryPage(entryPageState.page - 1)"
            >
              Previous
            </BaseButton>
            <BaseButton
              class="down-mobile:w-full"
              type="button"
              variant="secondary"
              size="compact"
              :disabled="entryPageState.page === entryPageState.pageCount - 1"
              @click="goToEntryPage(entryPageState.page + 1)"
            >
              Next
            </BaseButton>
          </div>
        </nav>
      </section>
    </div>
  </form>
</template>

<style scoped>
@reference '../../../styles.css';

.edit-form {
  @apply grid gap-3.5 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:text-sm [&_label]:font-bold [&_.visibility-field_label]:flex [&_.visibility-field_label]:items-center [&_.visibility-field_label]:gap-2;
}

.visibility-field {
  @apply grid grid-cols-[max-content_1fr] gap-y-1.5 gap-x-3 m-0 p-3 border border-line rounded [&_legend]:py-0 [&_legend]:px-1 [&_legend]:text-label [&_legend]:text-sm [&_legend]:font-bold [&_.muted]:m-0 [&_.muted]:self-center;
}

.reading-order-cover-preview {
  @apply w-[76px] min-w-[76px] h-[76px] overflow-hidden border border-line rounded bg-surface-muted [&_img]:block [&_img]:w-full [&_img]:h-full [&_img]:object-cover;
}

.reading-order-editor-layout {
  @apply grid grid-cols-[minmax(320px,420px)_minmax(620px,1fr)] items-start gap-4 border-t border-line pt-3.5 down-laptop:grid-cols-1 [&_.entry-section]:border-t-0 [&_.entry-section]:pt-0;
}

.reading-order-search-column {
  @apply grid self-start gap-4 min-w-0 sticky top-[calc(var(--sticky-toolbar-top,0px)+14px)] down-laptop:static;
}

.section-title {
  @apply justify-between mb-2.5 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5;
}

.comic-add-panel {
  @apply grid gap-2.5 border border-line-strong rounded bg-surface-soft p-2.5 mb-2.5;
}

.comic-add-search {
  @apply flex items-center gap-2 border border-line-strong rounded bg-surface pt-1 pr-1.5 pb-1 pl-0;
}

.ghost-button {
  @apply min-h-8 border-0 rounded-[7px] bg-transparent text-accent py-1.5 px-2 font-bold;
}

.comic-add-results {
  @apply grid grid-cols-[repeat(auto-fit,minmax(min(100%,280px),1fr))] gap-2;
}

.comic-add-result {
  @apply flex items-start justify-between gap-3 min-h-20 border border-line rounded bg-surface text-ink py-2.5 px-3 text-left [[draggable='true']]:cursor-grab [&[draggable='true']:active]:cursor-grabbing hover:border-[color-mix(in_srgb,var(--primary)_46%,var(--line-strong))] hover:bg-primary-soft [&_span:first-child]:min-w-0 [&_strong]:[display:-webkit-box] [&_strong]:overflow-hidden [&_strong]:text-ellipsis [&_strong]:whitespace-normal [&_strong]:[-webkit-box-orient:vertical] [&_small]:[display:-webkit-box] [&_small]:overflow-hidden [&_small]:text-ellipsis [&_small]:whitespace-normal [&_small]:[-webkit-box-orient:vertical] [&_strong]:[line-clamp:2] [&_strong]:[-webkit-line-clamp:2] [&_small]:[line-clamp:2] [&_small]:[-webkit-line-clamp:1] [&_small]:text-muted [&_small]:mt-1 [&_.status-pill]:flex-none [&_.status-pill]:mt-0.5;
}

.entry-type-pill {
  @apply inline-grid self-start mb-1 [border:1px_solid_color-mix(in_srgb,var(--primary)_42%,var(--line))] rounded-full bg-primary-soft text-eyebrow py-0.5 px-2 text-xs font-black leading-tight uppercase;
}

.comic-add-result.reading-order-add-result {
  @apply flex items-start justify-between gap-3 min-h-20 border border-line rounded bg-surface text-ink py-2.5 px-3 text-left [[draggable='true']]:cursor-grab [&[draggable='true']:active]:cursor-grabbing hover:border-[color-mix(in_srgb,var(--primary)_46%,var(--line-strong))] hover:bg-primary-soft [&_span:first-child]:min-w-0 [&_strong]:[display:-webkit-box] [&_strong]:overflow-hidden [&_strong]:text-ellipsis [&_strong]:whitespace-normal [&_strong]:[-webkit-box-orient:vertical] [&_small]:[display:-webkit-box] [&_small]:overflow-hidden [&_small]:text-ellipsis [&_small]:whitespace-normal [&_small]:[-webkit-box-orient:vertical] [&_strong]:[line-clamp:2] [&_strong]:[-webkit-line-clamp:2] [&_small]:[line-clamp:2] [&_small]:[-webkit-line-clamp:1] [&_small]:text-muted [&_small]:mt-1 [&_.status-pill]:flex-none [&_.status-pill]:mt-0.5;
}

.section-add-panel {
  @apply grid gap-2.5 border border-line-strong rounded bg-surface-soft p-3;
}

.entry-section.reading-order-list-edit {
  @apply min-w-0 border-t border-line pt-3.5 [&_.section-title_>_span]:text-muted [&_.section-title_>_span]:text-sm [&_.section-title_>_span]:font-ui-bold;
}

.reading-order-entry-pages {
  @apply flex items-center justify-between gap-3 mb-2.5 border border-line rounded bg-panel-soft py-2.5 px-3 down-mobile:items-stretch down-mobile:flex-col [&_>_span]:text-muted [&_>_span]:text-sm [&_>_span]:font-ui-bold [&_>_div]:flex [&_>_div]:items-center [&_>_div]:justify-end [&_>_div]:gap-2 [&_strong]:min-w-24 [&_strong]:text-ink [&_strong]:text-center down-mobile:[&_>_div]:grid down-mobile:[&_>_div]:grid-cols-2 down-mobile:[&_strong]:col-span-full down-mobile:[&_strong]:row-start-1;
}

.empty-state.empty-entry-drop-zone {
  @apply [transition:background-color_120ms_ease,border-color_120ms_ease,color_120ms_ease] [&.active]:border-primary [&.active]:bg-primary-soft [&.active]:text-ink grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4;
}

.entry-drop-zone {
  @apply min-h-3 border border-dashed border-transparent rounded [transition:background-color_120ms_ease,border-color_120ms_ease,min-height_120ms_ease] [&.active]:min-h-8 [&.active]:border-primary [&.active]:bg-primary-soft [&.end-zone]:min-h-10 [&.end-zone]:border-line [&.end-zone]:bg-panel-soft;
}

.selected-order-comic.entry-summary-button {
  @apply justify-center min-w-0 border border-line rounded bg-surface-soft py-2 px-3 grid grid-cols-[34px_minmax(0,1fr)_34px] items-center gap-3 w-full min-h-20 h-20 text-ink text-left down-mobile:grid-cols-[30px_minmax(0,1fr)_30px] down-mobile:h-auto down-mobile:min-h-20 down-mobile:py-2 down-mobile:px-2.5 hover:border-[color-mix(in_srgb,var(--primary)_42%,var(--line))] hover:bg-primary-soft [&:hover_.entry-expand-icon]:bg-surface [&:hover_.entry-expand-icon]:text-accent [&_strong]:[display:-webkit-box] [&_strong]:overflow-hidden [&_strong]:text-ellipsis [&_strong]:whitespace-normal [&_strong]:[-webkit-box-orient:vertical] [&_strong]:[line-clamp:2] [&_strong]:[-webkit-line-clamp:2] [&_small]:[display:-webkit-box] [&_small]:overflow-hidden [&_small]:text-ellipsis [&_small]:whitespace-normal [&_small]:[-webkit-box-orient:vertical] [&_small]:[line-clamp:2] [&_small]:[-webkit-line-clamp:2] [&_small]:text-muted [&_small]:mt-1;
}

.entry-drag-cell {
  @apply grid place-items-center self-stretch [&_.drag-handle]:grid [&_.drag-handle]:place-items-center [&_.drag-handle]:w-7 [&_.drag-handle]:h-11 [&_.drag-handle]:rounded [&_.drag-handle]:cursor-grab [&_.drag-handle]:text-muted [&_.drag-handle]:text-lg [&_.drag-handle]:font-black [&_.drag-handle]:leading-none [&_.drag-handle:hover]:bg-surface [&_.drag-handle:hover]:text-primary [&_.drag-handle:active]:cursor-grabbing;
}

.entry-summary-copy {
  @apply grid min-w-0 justify-items-start [&_strong]:[line-clamp:1] [&_strong]:[-webkit-line-clamp:1];
}

.button-icon.entry-expand-icon {
  @apply grid place-items-center justify-self-end w-8 h-8 rounded-full text-control text-lg [transition:background-color_120ms_ease,color_120ms_ease] font-extrabold leading-none;
}

.mobile-reorder {
  @apply hidden down-mobile:grid down-mobile:col-span-full down-mobile:grid-cols-2 down-mobile:gap-2 down-mobile:w-full;
}

.remove-entry-button {
  @apply grid place-items-center self-stretch w-10 h-full min-h-0 border border-danger-border rounded bg-danger-soft text-danger p-0 text-2xl font-extrabold leading-none hover:border-(--danger) hover:[background:color-mix(in_srgb,var(--danger-soft)_72%,var(--danger))] hover:text-danger;
}

.entry-edit-panel {
  @apply grid col-span-full grid-cols-[minmax(0,1fr)_minmax(180px,0.55fr)] gap-3 border border-line rounded bg-surface-soft p-3 down-mobile:col-start-1 down-mobile:grid-cols-1;
}

.entry-drop-zone.end-zone {
  @apply min-h-3 border border-dashed border-transparent rounded [transition:background-color_120ms_ease,border-color_120ms_ease,min-height_120ms_ease] [&.active]:min-h-8 [&.active]:border-primary [&.active]:bg-primary-soft [&.end-zone]:min-h-10 [&.end-zone]:border-line [&.end-zone]:bg-panel-soft;
}

.reading-order-entry-pages.reading-order-entry-pages-bottom {
  @apply mt-2.5 mb-0 flex items-center justify-between gap-3 border border-line rounded bg-panel-soft py-2.5 px-3 down-mobile:items-stretch down-mobile:flex-col [&_>_span]:text-muted [&_>_span]:text-sm [&_>_span]:font-ui-bold [&_>_div]:flex [&_>_div]:items-center [&_>_div]:justify-end [&_>_div]:gap-2 [&_strong]:min-w-24 [&_strong]:text-ink [&_strong]:text-center down-mobile:[&_>_div]:grid down-mobile:[&_>_div]:grid-cols-2 down-mobile:[&_strong]:col-span-full down-mobile:[&_strong]:row-start-1;
}

.entry-source-button {
  @apply min-h-10 rounded border border-line bg-surface font-black text-muted;
}

.entry-source-button.active {
  @apply border-primary bg-primary text-white;
}

.sortable-order-entry {
  @apply grid grid-cols-[minmax(280px,1fr)_44px] items-stretch gap-3 rounded p-0 [transition:background-color_120ms_ease,box-shadow_120ms_ease,opacity_120ms_ease] down-tablet:grid-cols-1 down-mobile:grid-cols-1 down-mobile:border down-mobile:border-line down-mobile:bg-surface-soft down-mobile:p-2.5;
}

.sortable-order-entry.dragging {
  @apply opacity-55;
}

.sortable-order-entry.drag-over {
  @apply bg-primary-soft shadow-[inset_0_0_0_2px_var(--primary)];
}

.sortable-order-entry.nested-order-entry {
  @apply grid-cols-[minmax(280px,1fr)_44px];
}

.sortable-order-entry.nested-order-entry .selected-order-comic {
  border-color: color-mix(in srgb, var(--accent) 42%, var(--line));
  background: color-mix(in srgb, var(--surface-soft) 82%, var(--accent));
}

.sortable-order-entry.section-order-entry .selected-order-comic {
  border-color: color-mix(in srgb, var(--primary) 54%, var(--line));
  background: color-mix(in srgb, var(--surface-soft) 78%, var(--primary));
}

.sortable-order-entry.nested-order-entry .entry-edit-panel {
  @apply grid-cols-1;
}

.sortable-order-entry.section-order-entry .entry-edit-panel {
  @apply grid-cols-[minmax(180px,0.6fr)_minmax(0,1fr)];
}

.sortable-order-entry.nested-order-entry .entry-type-pill {
  @apply text-accent;
  border-color: color-mix(in srgb, var(--accent) 48%, var(--line));
  background: color-mix(in srgb, var(--surface-soft) 76%, var(--accent));
}

.sortable-order-entry.section-order-entry .entry-type-pill {
  @apply bg-primary text-white;
  border-color: color-mix(in srgb, var(--primary) 58%, var(--line));
}

@media (width <= 720px) {
  .sortable-order-entry.nested-order-entry,
  .sortable-order-entry.section-order-entry .entry-edit-panel {
    @apply grid-cols-1;
  }
}
</style>
