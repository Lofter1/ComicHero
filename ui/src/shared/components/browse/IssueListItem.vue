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
    class="issue-list-item read-accent flex w-full min-w-0 max-w-full flex-wrap items-start justify-between gap-3 rounded border border-line border-b-4 bg-surface-soft py-2.5 px-3 [&.read]:[border-bottom-color:var(--primary)] [&.unread]:[border-bottom-color:#e2c46a] [&.skipped]:[border-bottom-color:var(--muted)] [&.read-accent]:[border-bottom-width:4px] [&.read-accent.read]:[border-bottom-color:var(--primary)] [&.read-accent.unread]:[border-bottom-color:#e2c46a] [&.read-accent.skipped]:[border-bottom-color:var(--muted)] [&_>_span]:min-w-0 down-mobile:[&_.issue-list-main]:[flex-basis:100%] down-mobile:[&_.issue-list-main]:w-full down-mobile:[&_.read-state-actions]:[flex:1_0_100%] down-mobile:[&_.read-state-actions]:w-full down-mobile:[&_.read-state-actions]:justify-start"
    :class="{ read: comic.read, unread: !comic.read, skipped: comic.skipped, selected }"
  >
    <button
      class="issue-list-main flex max-w-full min-w-0 items-center gap-2.5 [flex:1_1_280px] border-0 bg-transparent text-inherit p-0 text-left [&_>_span:last-child]:min-w-0 hover:[border-color:transparent] hover:shadow-none hover:transform-none [&_strong]:block [&_small]:block"
      type="button"
      @click="$emit('open', comic)"
    >
      <span
        v-if="showCover && comic.coverImage"
        class="flex-none w-11 h-16 overflow-hidden border border-line rounded-[6px] bg-surface-muted [&_img]:block [&_img]:w-full [&_img]:h-full [&_img]:object-cover down-phone:w-10 down-phone:h-12"
        aria-hidden="true"
      >
        <img :src="assetURL(comic.coverImage)" alt="" loading="lazy" />
      </span>
      <span class="issue-list-copy min-w-0 max-w-full break-anywhere">
        <strong>{{ comic.title }}</strong>
        <small v-if="showComment && comic.comment">{{ comic.comment }}</small>
        <span
          v-if="showComment && entryTags().length"
          class="entry-tags flex flex-wrap gap-1 mt-1 [&_span]:rounded-full [&_span]:bg-primary-soft [&_span]:text-primary-strong [&_span]:py-1 [&_span]:px-2 [&_span]:text-xs [&_span]:font-extrabold [&_span]:leading-snug"
        >
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
        class="read-state-pill inline-flex items-center justify-center min-h-7 rounded-full py-1 px-2.5 text-xs font-extrabold leading-none whitespace-nowrap [&.read]:bg-primary-soft [&.read]:text-primary [&.unread]:bg-warning-soft [&.unread]:text-warning [&.unread]:[box-shadow:inset_0_0_0_1px_var(--warning-border)] [&.skipped]:bg-surface-muted [&.skipped]:text-muted [&.skipped]:[box-shadow:inset_0_0_0_1px_var(--line-strong)]"
        :class="{ read: comic.read, unread: !comic.read }"
      >
        {{ comic.read ? 'Read' : 'Unread' }}
      </span>
      <span
        v-if="comic.skipped"
        class="read-state-pill skipped inline-flex items-center justify-center min-h-7 rounded-full py-1 px-2.5 text-xs font-extrabold leading-none whitespace-nowrap [&.read]:bg-primary-soft [&.read]:text-primary [&.unread]:bg-warning-soft [&.unread]:text-warning [&.unread]:[box-shadow:inset_0_0_0_1px_var(--warning-border)] [&.skipped]:bg-surface-muted [&.skipped]:text-muted [&.skipped]:[box-shadow:inset_0_0_0_1px_var(--line-strong)]"
        >Skipped</span
      >
      <button
        v-if="!readOnly"
        type="button"
        class="read-toggle-button flex-none min-h-8 border border-line-strong rounded bg-surface text-label py-1.5 px-2.5 text-sm font-bold whitespace-nowrap [&.skipped]:border-muted [&.skipped]:text-muted [&.large]:min-h-10 [&.large]:py-2.5 [&.large]:px-3.5 [&.large]:text-base"
        :disabled="quickSaving"
        @click="$emit('toggle-read', comic)"
      >
        {{ comic.read ? 'Mark unread' : 'Mark read' }}
      </button>
      <button
        v-if="!readOnly"
        type="button"
        class="read-toggle-button flex-none min-h-8 border border-line-strong rounded bg-surface text-label py-1.5 px-2.5 text-sm font-bold whitespace-nowrap [&.skipped]:border-muted [&.skipped]:text-muted [&.large]:min-h-10 [&.large]:py-2.5 [&.large]:px-3.5 [&.large]:text-base"
        :class="{ skipped: comic.skipped }"
        :disabled="quickSaving"
        @click="$emit('toggle-skipped', comic)"
      >
        {{ comic.skipped ? 'Unskip' : 'Skip' }}
      </button>
    </span>
  </div>
</template>
