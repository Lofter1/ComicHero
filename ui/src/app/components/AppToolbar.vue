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
const showEyebrow = computed(
  () => eyebrow.value.trim().toLocaleLowerCase() !== title.value.trim().toLocaleLowerCase(),
)
const showSummary = computed(() => Boolean(route.meta.showCount))
</script>

<template>
  <header
    class="toolbar sticky-toolbar justify-between down-tablet:items-center down-mobile:gap-1.5 down-mobile:pb-2.5"
  >
    <div>
      <p v-if="showEyebrow" class="eyebrow">{{ eyebrow }}</p>
      <h2>{{ title }}</h2>
      <p
        v-if="showSummary"
        class="toolbar-summary [margin:6px_0_0] text-muted [font-size:0.9rem] down-mobile:[margin-top:3px] down-mobile:[font-size:0.82rem]"
      >
        Showing {{ resultCount }} of {{ totalCount }}
      </p>
    </div>
  </header>
</template>
