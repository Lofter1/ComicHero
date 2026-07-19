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
    class="toolbar sticky-toolbar justify-between down-tablet:items-center down-mobile:gap-1.5 down-mobile:pb-2.5 flex items-center gap-3.5 [&_>_div]:min-w-0 [&_h2]:min-w-0 [&_h2]:break-anywhere sticky [top:var(--sticky-toolbar-top)] z-20 [margin-inline:calc(var(--sticky-toolbar-inline-offset)_*_-1)] [padding:14px_var(--sticky-toolbar-inline-offset)] border-b border-sticky-border bg-sticky-bg shadow-sticky backdrop-blur-ui [&.sticky-toolbar]:[margin-top:calc(var(--content-padding)_*_-1)] max-w-none down-tablet:[&.sticky-toolbar]:mb-0 down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none down-mobile:[&.sticky-toolbar]:mt-0 down-mobile:[&_.eyebrow]:hidden down-mobile:[&_h2]:text-xl down-mobile:[&_input]:w-full"
  >
    <div>
      <p v-if="showEyebrow" class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">
        {{ eyebrow }}
      </p>
      <h2>{{ title }}</h2>
      <p
        v-if="showSummary"
        class="toolbar-summary mt-1.5 mx-0 mb-0 text-muted text-sm down-mobile:mt-1 down-mobile:text-sm"
      >
        Showing {{ resultCount }} of {{ totalCount }}
      </p>
    </div>
  </header>
</template>
