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
})

defineEmits(['open', 'toggle-read'])

function entryTags() {
  return String(props.comic.tags || '')
    .split(',')
    .map(tag => tag.trim())
    .filter(Boolean)
}
</script>

<template>
  <div
    class="issue-list-item read-accent"
    :class="{ read: comic.read, unread: !comic.read, selected }"
  >
    <button class="issue-list-main" type="button" @click="$emit('open', comic)">
      <span v-if="showCover && comic.coverImage" class="issue-list-cover" aria-hidden="true">
        <img :src="assetURL(comic.coverImage)" alt="" loading="lazy" />
      </span>
      <span class="issue-list-copy">
        <strong>{{ comic.title }}</strong>
        <small v-if="showComment && comic.comment">{{ comic.comment }}</small>
        <span v-if="showComment && entryTags().length" class="entry-tags">
          <span v-for="tag in entryTags()" :key="tag">{{ tag }}</span>
        </span>
        <small v-if="!showComment || (!comic.comment && !comic.tags)">{{ comic.publisher || 'Unknown publisher' }} · {{ comic.coverDate || 'Unknown date' }}</small>
      </span>
    </button>

    <span class="read-state-actions">
      <span class="read-state-pill" :class="{ read: comic.read, unread: !comic.read }">
        {{ comic.read ? 'Read' : 'Unread' }}
      </span>
      <button
        type="button"
        class="read-toggle-button"
        :disabled="quickSaving"
        @click="$emit('toggle-read', comic)"
      >
        {{ comic.read ? 'Mark unread' : 'Mark read' }}
      </button>
    </span>
  </div>
</template>
