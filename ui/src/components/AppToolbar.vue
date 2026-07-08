<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

defineProps({
  resultCount: {
    type: Number,
    default: 0,
  },
  totalCount: {
    type: Number,
    default: 0,
  },
})

const route = useRoute()
const eyebrow = computed(() => route.meta.eyebrow || 'ComicHero')
const title = computed(() => route.meta.title || 'ComicHero')
const showSummary = computed(() => Boolean(route.meta.showCount))
</script>

<template>
  <header class="toolbar sticky-toolbar">
    <div>
      <p class="eyebrow">{{ eyebrow }}</p>
      <h2>{{ title }}</h2>
      <p v-if="showSummary" class="toolbar-summary">
        Showing {{ resultCount }} of {{ totalCount }}
      </p>
    </div>
  </header>
</template>
