<script setup>
import { computed } from 'vue'
import { assetURL } from '@/api/client.js'

const props = defineProps({
  comic: {
    type: Object,
    required: true,
  },
  selected: {
    type: Boolean,
    default: false,
  },
  quickSaving: {
    type: Boolean,
    default: false,
  },
  showCover: {
    type: Boolean,
    default: false,
  },
  showComment: {
    type: Boolean,
    default: false,
  },
  readOnly: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['open', 'toggle-read', 'toggle-skipped'])

const readingState = computed(() => {
  if (props.comic.skipped) return 'skipped'
  if (props.comic.read) return 'read'
  return 'none'
})

const stateLabel = computed(() => {
  if (readingState.value === 'skipped') return 'Skipped'
  if (readingState.value === 'read') return 'Read'
  return 'Not read'
})

const stateIcon = computed(() => {
  if (readingState.value === 'skipped') return '–'
  if (readingState.value === 'read') return '✓'
  return '○'
})

function entryTags() {
  return String(props.comic.tags || '')
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean)
}
</script>

<template>
  <div class="issue-list-item read-accent" :class="[readingState, { selected }]">
    <!-- Native button: the issue body is a full-row navigation target. -->
    <button class="issue-list-main" type="button" @click="$emit('open', comic)">
      <span v-if="showCover && comic.coverImage" class="issue-cover" aria-hidden="true">
        <img :src="assetURL(comic.coverImage)" alt="" loading="lazy" />
      </span>
      <span class="issue-list-copy min-w-0 max-w-full break-anywhere">
        <strong>{{ comic.title }}</strong>
        <small v-if="showComment && comic.comment">{{ comic.comment }}</small>
        <span v-if="showComment && entryTags().length" class="entry-tags mt-1 flex flex-wrap gap-1">
          <span v-for="tag in entryTags()" :key="tag">{{ tag }}</span>
        </span>
        <small v-if="!showComment || (!comic.comment && !comic.tags)"
          >{{ comic.publisher || 'Unknown publisher' }} ·
          {{ comic.coverDate || 'Unknown date' }}</small
        >
      </span>
    </button>

    <span class="read-state-actions flex items-center justify-end flex-none flex-wrap gap-2">
      <span
        class="read-state-pill"
        :class="readingState"
        :aria-label="`Reading status: ${stateLabel}`"
      >
        <span class="read-state-icon" aria-hidden="true">{{ stateIcon }}</span>
        {{ stateLabel }}
      </span>
      <!-- Native buttons: read/skip controls expose persistent state-specific styling. -->
      <button
        v-if="!readOnly"
        type="button"
        class="read-toggle-button"
        :disabled="quickSaving"
        @click="$emit('toggle-read', comic)"
      >
        {{ comic.read ? 'Mark unread' : 'Mark read' }}
      </button>
      <button
        v-if="!readOnly"
        type="button"
        class="read-toggle-button"
        :class="{ skipped: comic.skipped }"
        :disabled="quickSaving"
        @click="$emit('toggle-skipped', comic)"
      >
        {{ comic.skipped ? 'Unskip' : 'Skip' }}
      </button>
    </span>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.issue-list-item > span,
.issue-list-main > span:last-child {
  @apply min-w-0;
}

.issue-list-main:hover:not(:disabled) {
  @apply border-transparent shadow-none transform-none;
}

.issue-list-main strong,
.issue-list-main small,
.issue-cover img {
  @apply block;
}

.issue-cover img {
  @apply h-full w-full object-cover;
}

.entry-tags span {
  @apply rounded-full bg-primary-soft px-2 py-1 text-xs font-extrabold leading-snug text-primary-strong;
}

.read-state-pill.read {
  @apply bg-success-soft text-success;
  box-shadow: inset 0 0 0 1px var(--success-border);
}

.read-state-pill.none {
  @apply bg-surface-muted text-muted;
  box-shadow: inset 0 0 0 1px var(--line-strong);
}

.read-state-pill.skipped {
  @apply bg-warning-soft text-warning;
  box-shadow: inset 0 0 0 1px var(--warning-border);
}

.read-state-icon {
  @apply grid size-4 place-items-center text-sm leading-none font-black;
}

.read-toggle-button.skipped {
  @apply border-muted text-muted;
}

.read-toggle-button.large {
  @apply min-h-10 px-3.5 py-2.5 text-base;
}

@media (width <= 720px) {
  .issue-list-main {
    @apply w-full;
    flex-basis: 100%;
  }

  .read-state-actions {
    @apply w-full justify-start;
    flex: 1 0 100%;
  }
}

.issue-list-item.read-accent {
  @apply flex w-full min-w-0 max-w-full flex-wrap items-start justify-between gap-3 rounded border border-b-4 border-line bg-surface-soft px-3 py-2.5;
}

.issue-list-item.read {
  border-bottom-color: var(--success);
}

.issue-list-item.none {
  border-bottom-color: var(--muted);
}

.issue-list-item.skipped {
  border-bottom-color: var(--warning-border);
}

.issue-list-main {
  @apply flex max-w-full min-w-0 items-center gap-2.5 border-0 bg-transparent p-0 text-left text-inherit flex-[1_1_280px];
}

.issue-cover {
  @apply h-16 w-11 flex-none overflow-hidden rounded-ui-sm border border-line bg-surface-muted down-phone:h-12 down-phone:w-10;
}

.read-state-pill {
  @apply inline-flex min-h-7 items-center justify-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-extrabold leading-none;
}

.read-toggle-button {
  @apply min-h-8 flex-none whitespace-nowrap rounded border border-line-strong bg-surface px-2.5 py-1.5 text-sm font-bold text-label;
}
</style>
