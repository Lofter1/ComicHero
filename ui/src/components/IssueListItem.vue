<script setup>
import { assetURL } from '@/api/client.js'

defineProps({
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
        <small v-else>{{ comic.publisher || 'Unknown publisher' }} · {{ comic.coverDate || 'Unknown date' }}</small>
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
