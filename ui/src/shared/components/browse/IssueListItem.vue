<script setup>
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

function entryTags() {
  return String(props.comic.tags || '')
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean)
}
</script>

<template>
  <div
    class="issue-list-item read-accent [border-bottom-width:4px] [border-bottom-style:solid] flex [align-items:flex-start] justify-between gap-3 border border-line rounded bg-surface-soft [padding:10px_12px] down-mobile:flex-wrap"
    :class="{ read: comic.read, unread: !comic.read, skipped: comic.skipped, selected }"
  >
    <button
      class="issue-list-main flex items-center gap-2.5 [flex:1_1_auto] border-0 bg-transparent [color:inherit] [padding:0] text-left"
      type="button"
      @click="$emit('open', comic)"
    >
      <span v-if="showCover && comic.coverImage" class="issue-list-cover" aria-hidden="true">
        <img :src="assetURL(comic.coverImage)" alt="" loading="lazy" />
      </span>
      <span class="issue-list-copy min-w-0">
        <strong>{{ comic.title }}</strong>
        <small v-if="showComment && comic.comment">{{ comic.comment }}</small>
        <span
          v-if="showComment && entryTags().length"
          class="entry-tags flex flex-wrap [gap:5px] [margin-top:5px]"
        >
          <span v-for="tag in entryTags()" :key="tag">{{ tag }}</span>
        </span>
        <small v-if="!showComment || (!comic.comment && !comic.tags)"
          >{{ comic.publisher || 'Unknown publisher' }} ·
          {{ comic.coverDate || 'Unknown date' }}</small
        >
      </span>
    </button>

    <span class="read-state-actions flex items-center justify-end [flex:0_0_auto] flex-wrap gap-2">
      <span
        class="read-state-pill inline-flex items-center justify-center min-h-7 rounded-full [padding:4px_10px] [font-size:0.78rem] font-extrabold leading-none whitespace-nowrap"
        :class="{ read: comic.read, unread: !comic.read }"
      >
        {{ comic.read ? 'Read' : 'Unread' }}
      </span>
      <span
        v-if="comic.skipped"
        class="read-state-pill skipped inline-flex items-center justify-center min-h-7 rounded-full [padding:4px_10px] [font-size:0.78rem] font-extrabold leading-none whitespace-nowrap"
        >Skipped</span
      >
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
