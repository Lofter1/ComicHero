<script setup>
import { ref } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseSelect from '@/shared/components/form/BaseSelect.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import { useClickOutside } from '@/shared/composables/useClickOutside.js'

defineProps({
  title: { type: String, required: true },
  summaryText: { type: String, required: true },
  hasContent: { type: Boolean, default: false },
  serverSource: { type: Boolean, default: false },
  hasFilters: { type: Boolean, default: false },
  effectiveServerMode: { type: Boolean, default: false },
  tagOptions: { type: Array, default: () => [] },
  showReadingOrderSort: { type: Boolean, default: false },
  showNewButton: { type: Boolean, default: false },
  readOnly: { type: Boolean, default: false },
})

defineEmits(['new-comic'])

const search = defineModel('search', { type: String, required: true })
const status = defineModel('status', { type: String, required: true })
const tag = defineModel('tag', { type: String, required: true })
const sort = defineModel('sort', { type: String, required: true })
const direction = defineModel('direction', { type: String, required: true })
const optionsOpen = ref(false)
const optionsTrigger = ref(null)
const filterControls = ref(null)

useClickOutside([optionsTrigger, filterControls], () => (optionsOpen.value = false), optionsOpen)

function statusValues(value) {
  if (!value || value === 'all') return ['unread', 'read', 'skipped']
  return String(value)
    .split(',')
    .map((item) => item.trim())
    .filter((item) => ['unread', 'read', 'skipped'].includes(item))
}

function statusActive(value) {
  return statusValues(status.value).includes(value)
}

function setAllStatuses() {
  status.value = 'all'
}

function toggleStatus(value) {
  const selected = statusValues(status.value)
  const current = new Set(selected.length === 3 ? [] : selected)
  if (current.has(value)) current.delete(value)
  else current.add(value)

  const next = ['unread', 'read', 'skipped'].filter((item) => current.has(item))
  status.value = next.length === 0 || next.length === 3 ? 'all' : next.join(',')
}
</script>

<template>
  <div class="comic-list-sticky">
    <header class="comic-list-header">
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">{{ title }}</p>
        <small>{{ summaryText }}</small>
      </div>

      <BaseButton v-if="showNewButton && !readOnly" variant="primary" @click="$emit('new-comic')">
        New Comic
      </BaseButton>
    </header>

    <div v-if="hasContent || serverSource || hasFilters" class="comic-list-tools">
      <BaseTextInput
        v-model="search"
        class="flex-[1_1_280px] min-w-[min(280px,100%)] down-mobile:flex-[1_1_280px]"
        type="search"
        placeholder="Search issues"
      />

      <!-- Native button: this DOM ref anchors the bespoke mobile filter popover. -->
      <button
        ref="optionsTrigger"
        class="mobile-comic-options-trigger"
        type="button"
        :aria-expanded="optionsOpen"
        @click="optionsOpen = !optionsOpen"
      >
        Filter &amp; sort
        <span aria-hidden="true">⌄</span>
      </button>

      <div ref="filterControls" class="comic-filter-controls" :class="{ open: optionsOpen }">
        <div
          class="inline-filter-tabs issue-status-tabs"
          role="group"
          aria-label="Issue status filters"
        >
          <!-- Native buttons: status filters are a segmented pressed-state control. -->
          <button
            type="button"
            class="status-filter-button"
            :class="{ active: status === 'all' }"
            :aria-pressed="status === 'all'"
            @click="setAllStatuses"
          >
            All
          </button>
          <button
            v-for="option in ['unread', 'read', 'skipped']"
            :key="option"
            type="button"
            class="status-filter-button"
            :class="{ active: statusActive(option) && status !== 'all' }"
            :aria-pressed="statusActive(option) && status !== 'all'"
            @click="toggleStatus(option)"
          >
            {{ option.charAt(0).toUpperCase() + option.slice(1) }}
          </button>
        </div>

        <BaseSelect
          v-if="!effectiveServerMode && tagOptions.length"
          v-model="tag"
          class="filter-select"
          variant="trailing"
          aria-label="Filter by tag"
        >
          <option value="all">All Tags</option>
          <option v-for="option in tagOptions" :key="option" :value="option.toLowerCase()">
            {{ option }}
          </option>
        </BaseSelect>

        <BaseSelect
          v-model="sort"
          class="filter-select"
          variant="trailing"
          aria-label="Sort issues"
        >
          <option v-if="showReadingOrderSort" value="readingOrder">Reading Order</option>
          <option value="series">Series</option>
          <option value="title">Title</option>
          <option value="date">Date</option>
          <option value="publisher">Publisher</option>
          <option value="read">Read Status</option>
        </BaseSelect>

        <BaseSelect
          v-model="direction"
          class="filter-select"
          variant="trailing"
          aria-label="Sort direction"
        >
          <option value="asc">Ascending</option>
          <option value="desc">Descending</option>
        </BaseSelect>
      </div>
    </div>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.comic-list-sticky {
  @apply grid w-full min-w-0 gap-2.5 pb-3 border-b border-sticky-border bg-sticky-bg mt-8 max-w-none down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:pb-3 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none;
}

.comic-list-header {
  @apply flex items-center justify-between gap-3 *:min-w-0 [&_.eyebrow]:mb-0.5 [&_small]:text-muted desktop-compact:items-stretch desktop-compact:flex-wrap;
}

.comic-list-tools {
  @apply flex w-full min-w-0 max-w-full flex-wrap items-center gap-2 [&_.list-sort-select]:min-w-44 [&_.inline-filter-tabs]:flex-[1_1_230px] [&_.inline-filter-tabs]:min-w-[min(230px,100%)] [&_.issue-status-tabs]:basis-[320px] [&_.issue-status-tabs]:min-w-[min(320px,100%)] [&_.four-filter-tabs]:min-w-96 down-mobile:[&:has(>_.comic-filter-controls)]:relative down-mobile:[&:has(>_.comic-filter-controls)]:flex down-mobile:[&:has(>_.comic-filter-controls)]:flex-wrap down-mobile:[&:has(>_.comic-filter-controls)]:items-center down-mobile:[&:has(>_.comic-filter-controls)]:gap-2 down-mobile:[&_.issue-status-tabs]:min-w-0 down-mobile:w-full;
}

.mobile-comic-options-trigger {
  @apply down-mobile:inline-flex down-mobile:items-center down-mobile:justify-between down-mobile:flex-none down-mobile:min-w-48 down-mobile:pr-3 hidden down-mobile:min-h-11 down-mobile:border down-mobile:border-line-strong down-mobile:rounded down-mobile:bg-surface down-mobile:text-control down-mobile:pt-3 down-mobile:pb-3 down-mobile:pl-3 down-mobile:font-bold down-mobile:[&_span]:ml-5 down-mobile:[&_span]:text-muted down-mobile:[&[aria-expanded='true']_span]:transform-[rotate(180deg)];
}

.comic-filter-controls {
  @apply contents down-mobile:hidden down-mobile:w-[min(360px,calc(100vw-28px))] down-mobile:absolute down-mobile:z-25 down-mobile:top-[calc(100%+8px)] down-mobile:right-0 down-mobile:left-auto down-mobile:gap-2.5 down-mobile:border down-mobile:border-line-strong down-mobile:rounded-lg down-mobile:bg-surface down-mobile:p-3 down-mobile:[box-shadow:0_18px_40px_var(--shadow-panel)] down-mobile:[&_.inline-filter-tabs]:grid down-mobile:[&_.inline-filter-tabs]:w-full down-mobile:[&_.inline-filter-tabs]:min-w-0 down-mobile:[&.open]:grid;
}

.inline-filter-tabs.issue-status-tabs {
  @apply inline-grid gap-1 border border-line rounded bg-panel-soft p-1 grid-cols-4 down-mobile:w-full;
}

.status-filter-button {
  @apply min-h-8 rounded-[6px] border-0 bg-transparent px-2 py-1.5 text-sm font-bold text-label;
}

.status-filter-button.active {
  @apply bg-primary text-white;
}

.filter-select {
  @apply min-w-32 max-w-44 flex-[1_1_140px];
}

@media (width <= 720px) {
  .filter-select {
    @apply block w-full max-w-none;
  }
}

@media (width <= 420px) {
  .status-filter-button {
    @apply px-1.5 text-xs;
  }
}
</style>
