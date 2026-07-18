<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'

import { assetURL, listComics } from '@/api/client.js'
import { comicLabel } from '@/features/comics/model.js'
import {
  readingOrderEditorPage,
  readingOrderEditorPageSize,
  reorderReadingOrderEntry,
} from '@/features/reading-orders/model.js'

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
  <form :id="formId" class="edit-form grid gap-3.5" @submit.prevent="$emit('save')">
    <label>
      Name
      <input :value="form.name" required @input="updateForm({ name: $event.target.value })" />
    </label>

    <label>
      Description
      <textarea
        :value="form.description"
        rows="3"
        @input="updateForm({ description: $event.target.value })"
      />
    </label>

    <fieldset
      class="visibility-field grid [grid-template-columns:max-content_1fr] [gap:6px_12px] [margin:0] p-3 border border-line rounded"
    >
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
      <p class="muted">Anyone with access to ComicHero can view this reading order.</p>
      <label>
        <input
          type="radio"
          name="reading-order-visibility"
          :checked="!form.isPublic"
          @change="updateForm({ isPublic: false })"
        />
        Private
      </label>
      <p class="muted">Only you and administrators can view this reading order.</p>
    </fieldset>

    <label class="cover-image-field">
      Cover image
      <span class="reading-order-cover-editor flex items-center gap-3 min-w-0">
        <span
          v-if="coverPreview"
          class="reading-order-cover-preview [width:76px] [min-width:76px] [height:76px] overflow-hidden border border-line rounded bg-surface-muted"
          aria-hidden="true"
        >
          <img :src="coverPreview" alt="" />
        </span>
        <input type="file" accept="image/*" @change="chooseCoverImage" />
      </span>
    </label>

    <div
      class="reading-order-editor-layout grid [grid-template-columns:minmax(320px,_420px)_minmax(620px,_1fr)] items-start gap-4.5 [border-top:1px_solid_var(--line)] pt-3.5 down-laptop:[grid-template-columns:1fr]"
    >
      <div
        class="reading-order-search-column grid [align-self:start] gap-4.5 min-w-0 sticky [top:calc(var(--sticky-toolbar-top,_0px)_+_14px)] down-laptop:[position:static]"
      >
        <section class="entry-section add-entry-panel grid gap-2.5">
          <div class="section-title">
            <h4>Add Entries</h4>
          </div>

          <div
            class="add-entry-tabs grid [grid-template-columns:repeat(3,_minmax(0,_1fr))] gap-2"
            role="tablist"
            aria-label="Entry source"
          >
            <button
              type="button"
              :class="{ active: activeAddType === 'comic' }"
              @click="activeAddType = 'comic'"
            >
              Issues
            </button>
            <button
              type="button"
              :class="{ active: activeAddType === 'readingOrder' }"
              @click="activeAddType = 'readingOrder'"
            >
              Reading Orders
            </button>
            <button
              type="button"
              :class="{ active: activeAddType === 'section' }"
              @click="activeAddType = 'section'"
            >
              Sections
            </button>
          </div>

          <div
            v-if="activeAddType === 'comic'"
            class="comic-add-panel grid gap-2.5 border border-line-strong rounded bg-surface-soft p-2.5 mb-2.5"
          >
            <div
              class="comic-add-search flex items-center gap-2 border border-line-strong rounded bg-surface [padding:4px_6px_4px_0]"
            >
              <input
                v-model="comicSearch"
                type="search"
                placeholder="Search title, series, issue, publisher, status"
                @keydown.enter.prevent
              />
              <button
                v-if="comicSearch"
                class="ghost-button"
                type="button"
                @click="clearComicSearch"
              >
                Clear
              </button>
            </div>

            <p v-if="comicSearchLoading" class="muted">Searching comics...</p>
            <p v-else-if="comicSearchError" class="muted">{{ comicSearchError }}</p>

            <div
              v-else-if="comicSearchResults.length"
              class="comic-add-results grid [grid-template-columns:repeat(auto-fit,_minmax(min(100%,_280px),_1fr))] gap-2"
            >
              <button
                v-for="comic in comicSearchResults"
                :key="comic.id"
                type="button"
                class="comic-add-result flex [align-items:flex-start] justify-between gap-3 [min-height:82px] border border-line rounded bg-surface text-ink [padding:10px_12px] text-left"
                draggable="true"
                @click="addNewEntryToEnd(newComicEntry(comic))"
                @dragstart="startAddDrag($event, newComicEntry(comic))"
              >
                <span>
                  <span
                    class="entry-type-pill [display:inline-grid] [align-self:flex-start] [margin-bottom:5px] [border:1px_solid_color-mix(in_srgb,_var(--primary)_42%,_var(--line))] rounded-full bg-primary-soft text-eyebrow [padding:2px_7px] [font-size:0.68rem] font-black leading-tight uppercase"
                    >Issue</span
                  >
                  <strong>{{ comic.title }}</strong>
                  <small>{{ comicSeriesLine(comic) }}</small>
                  <small>{{ comicMetaLine(comic) }}</small>
                </span>

                <span class="status-pill">Add</span>
              </button>
            </div>

            <p v-else class="muted">No comics match that search.</p>
          </div>

          <div
            v-if="activeAddType === 'readingOrder' && readingOrders.length"
            class="comic-add-panel grid gap-2.5 border border-line-strong rounded bg-surface-soft p-2.5 mb-2.5"
          >
            <div
              class="comic-add-search flex items-center gap-2 border border-line-strong rounded bg-surface [padding:4px_6px_4px_0]"
            >
              <input
                v-model="readingOrderSearch"
                type="search"
                placeholder="Search reading orders"
                @keydown.enter.prevent
              />
              <button
                v-if="readingOrderSearch"
                class="ghost-button"
                type="button"
                @click="readingOrderSearch = ''"
              >
                Clear
              </button>
            </div>

            <div
              v-if="childOrderChoices.length"
              class="comic-add-results grid [grid-template-columns:repeat(auto-fit,_minmax(min(100%,_280px),_1fr))] gap-2"
            >
              <button
                v-for="order in childOrderChoices"
                :key="order.id"
                type="button"
                class="comic-add-result reading-order-add-result flex [align-items:flex-start] justify-between gap-3 [min-height:82px] border border-line rounded bg-surface text-ink [padding:10px_12px] text-left"
                draggable="true"
                @click="addNewEntryToEnd(newChildOrderEntry(order))"
                @dragstart="startAddDrag($event, newChildOrderEntry(order))"
              >
                <span>
                  <span
                    class="entry-type-pill [display:inline-grid] [align-self:flex-start] [margin-bottom:5px] [border:1px_solid_color-mix(in_srgb,_var(--primary)_42%,_var(--line))] rounded-full bg-primary-soft text-eyebrow [padding:2px_7px] [font-size:0.68rem] font-black leading-tight uppercase"
                    >Reading order</span
                  >
                  <strong>{{ order.name }}</strong>
                  <small>{{ order.description || 'No description' }}</small>
                </span>

                <span class="status-pill">Add</span>
              </button>
            </div>

            <p v-else class="muted">No reading orders match that search.</p>
          </div>

          <div v-else-if="activeAddType === 'readingOrder'" class="empty-state">
            No reading orders available to include.
          </div>

          <div
            v-if="activeAddType === 'section'"
            class="section-add-panel grid gap-2.5 border border-line-strong rounded bg-surface-soft p-3"
          >
            <label>
              Section title
              <input
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
            <button
              type="button"
              class="primary-button"
              :disabled="!sectionTitle.trim()"
              @click="addSection"
            >
              Add Section
            </button>
          </div>
        </section>
      </div>

      <section class="entry-section reading-order-list-edit min-w-0">
        <div class="section-title">
          <h4>List Order</h4>
          <span>{{ orderEntries.length }} entries</span>
        </div>

        <nav
          v-if="entryPageState.pageCount > 1"
          class="reading-order-entry-pages flex items-center justify-between gap-3 mb-2.5 border border-line rounded [background:var(--panel-soft-bg)] [padding:10px_12px] down-mobile:[align-items:stretch] down-mobile:flex-col"
          aria-label="Reading order entry pages"
        >
          <span>
            Entries {{ entryPageState.start + 1 }}–{{ entryPageState.end }} of
            {{ orderEntries.length }}
          </span>
          <div>
            <button
              type="button"
              class="secondary-button"
              :disabled="entryPageState.page === 0"
              @click="goToEntryPage(0)"
            >
              First
            </button>
            <button
              type="button"
              class="secondary-button"
              :disabled="entryPageState.page === 0"
              @click="goToEntryPage(entryPageState.page - 1)"
            >
              Previous
            </button>
            <strong>Page {{ entryPageState.page + 1 }} of {{ entryPageState.pageCount }}</strong>
            <button
              type="button"
              class="secondary-button"
              :disabled="entryPageState.page === entryPageState.pageCount - 1"
              @click="goToEntryPage(entryPageState.page + 1)"
            >
              Next
            </button>
            <button
              type="button"
              class="secondary-button"
              :disabled="entryPageState.page === entryPageState.pageCount - 1"
              @click="goToEntryPage(entryPageState.pageCount - 1)"
            >
              Last
            </button>
          </div>
        </nav>

        <div
          v-if="orderEntries.length === 0"
          class="empty-state empty-entry-drop-zone [transition:background-color_120ms_ease,_border-color_120ms_ease,_color_120ms_ease]"
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
              class="entry-drop-zone min-h-3 [border:1px_dashed_transparent] rounded [transition:background-color_120ms_ease,_border-color_120ms_ease,_min-height_120ms_ease]"
              :class="{ active: dragOverIndex === index }"
              @dragover="overDropZone($event, index)"
              @dragleave="dragOverIndex = null"
              @drop="dropAt($event, index)"
            />

            <div
              class="order-entry grid [grid-template-columns:minmax(280px,_1fr)_44px] [align-items:stretch] gap-3 rounded [padding:0] [transition:background-color_120ms_ease,_box-shadow_120ms_ease,_opacity_120ms_ease] down-mobile:[grid-template-columns:1fr] down-mobile:border down-mobile:border-line down-mobile:bg-surface-soft down-mobile:p-2.5"
              :class="{
                dragging: draggedIndex === index,
                'nested-order-entry': entry.type === 'readingOrder',
                'section-order-entry': entry.type === 'section',
                expanded: isEntryExpanded(entry, index),
              }"
            >
              <button
                type="button"
                class="selected-order-comic entry-summary-button flex flex-col justify-center [align-items:flex-start] min-w-0 border border-line rounded bg-surface-soft [padding:8px_12px] grid [grid-template-columns:34px_minmax(0,_1fr)_34px] items-center gap-3 w-full [min-height:74px] [height:74px] text-ink text-left down-mobile:[grid-template-columns:30px_minmax(0,_1fr)_30px] down-mobile:[height:auto] down-mobile:[min-height:74px] down-mobile:[padding:8px_10px]"
                @click="toggleEntry(entry, index)"
              >
                <span class="entry-drag-cell grid place-items-center [align-self:stretch]">
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
                      class="drag-icon [letter-spacing:-4px] [transform:rotate(90deg)]"
                      >⋮⋮</span
                    >
                  </span>
                </span>

                <span class="entry-summary-copy grid min-w-0 justify-items-start">
                  <span
                    class="entry-type-pill [display:inline-grid] [align-self:flex-start] [margin-bottom:5px] [border:1px_solid_color-mix(in_srgb,_var(--primary)_42%,_var(--line))] rounded-full bg-primary-soft text-eyebrow [padding:2px_7px] [font-size:0.68rem] font-black leading-tight uppercase"
                    >{{ entryTypeLabel(entry) }}</span
                  >
                  <strong>{{ entryLabel(entry) }}</strong>
                </span>

                <span
                  aria-hidden="true"
                  class="button-icon entry-expand-icon grid place-items-center [justify-self:end] [width:30px] [height:30px] rounded-full text-control [font-size:1.15rem] [transition:background-color_120ms_ease,_color_120ms_ease]"
                  :title="isEntryExpanded(entry, index) ? 'Collapse' : 'Expand'"
                >
                  {{ isEntryExpanded(entry, index) ? '▴' : '▾' }}
                </span>

                <span
                  class="mobile-reorder hidden down-mobile:grid down-mobile:[grid-column:1_/_-1] down-mobile:[grid-template-columns:repeat(2,_minmax(0,_1fr))] down-mobile:gap-2 down-mobile:w-full"
                  @click.stop
                >
                  <button type="button" :disabled="index === 0" @click="moveEntry(index, -1)">
                    Up
                  </button>
                  <button
                    type="button"
                    :disabled="index === orderEntries.length - 1"
                    @click="moveEntry(index, 1)"
                  >
                    Down
                  </button>
                </span>
              </button>

              <button
                type="button"
                class="remove-entry-button grid place-items-center [align-self:stretch] w-10 h-full [min-height:0] border border-danger-border rounded bg-danger-soft text-danger [padding:0] [font-size:1.45rem] font-extrabold leading-none"
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
                class="entry-page-move flex [grid-column:1_/_-1] justify-end gap-2"
              >
                <button
                  v-if="index === entryPageState.start && entryPageState.page > 0"
                  type="button"
                  class="secondary-button"
                  @click="moveEntryAcrossPage(index, -1)"
                >
                  Move to previous page
                </button>
                <button
                  v-if="
                    index === entryPageState.end - 1 &&
                    entryPageState.page < entryPageState.pageCount - 1
                  "
                  type="button"
                  class="secondary-button"
                  @click="moveEntryAcrossPage(index, 1)"
                >
                  Move to next page
                </button>
              </div>

              <div
                v-if="isEntryExpanded(entry, index)"
                class="entry-edit-panel grid [grid-column:1_/_-1] [grid-template-columns:minmax(0,_1fr)_minmax(180px,_0.55fr)] gap-3 border border-line rounded bg-surface-soft p-3 down-mobile:[grid-column:1] down-mobile:[grid-template-columns:1fr]"
              >
                <template v-if="entry.type === 'section'">
                  <label class="comment-input-label [align-self:stretch]">
                    Section title
                    <input
                      :value="entry.title"
                      required
                      aria-label="Section title"
                      @input="updateEntry(index, { title: $event.target.value })"
                    />
                  </label>
                  <label class="comment-input-label [align-self:stretch]">
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
                  <label class="comment-input-label [align-self:stretch]">
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
                    class="comment-input-label [align-self:stretch]"
                  >
                    Tags
                    <input
                      :value="entry.tags"
                      aria-label="Entry tags"
                      placeholder="Tags"
                      @input="updateEntry(index, { tags: $event.target.value })"
                    />
                  </label>
                </template>
              </div>
            </div>
          </template>

          <div
            class="entry-drop-zone end-zone min-h-3 [border:1px_dashed_transparent] rounded [transition:background-color_120ms_ease,_border-color_120ms_ease,_min-height_120ms_ease]"
            :class="{ active: dragOverIndex === entryPageState.end }"
            @dragover="overDropZone($event, entryPageState.end)"
            @dragleave="dragOverIndex = null"
            @drop="dropAt($event, entryPageState.end)"
          />
        </div>

        <nav
          v-if="entryPageState.pageCount > 1"
          class="reading-order-entry-pages reading-order-entry-pages-bottom mt-2.5 [margin-bottom:0] flex items-center justify-between gap-3 mb-2.5 border border-line rounded [background:var(--panel-soft-bg)] [padding:10px_12px] down-mobile:[align-items:stretch] down-mobile:flex-col"
          aria-label="Reading order entry pages"
        >
          <span>Page {{ entryPageState.page + 1 }} of {{ entryPageState.pageCount }}</span>
          <div>
            <button
              type="button"
              class="secondary-button"
              :disabled="entryPageState.page === 0"
              @click="goToEntryPage(entryPageState.page - 1)"
            >
              Previous
            </button>
            <button
              type="button"
              class="secondary-button"
              :disabled="entryPageState.page === entryPageState.pageCount - 1"
              @click="goToEntryPage(entryPageState.page + 1)"
            >
              Next
            </button>
          </div>
        </nav>
      </section>
    </div>
  </form>
</template>
