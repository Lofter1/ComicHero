<script setup>
import BaseButton from '@/shared/components/form/BaseButton.vue'
import StatusPill from '@/shared/components/feedback/StatusPill.vue'
import DetailPanel from '@/shared/components/layout/DetailPanel.vue'

defineProps({
  activeSearch: { type: String, required: true },
  searching: { type: Boolean, default: false },
  comicResults: { type: Array, default: () => [] },
  readingListResults: { type: Array, default: () => [] },
  seriesResults: { type: Array, default: () => [] },
  arcResults: { type: Array, default: () => [] },
  characterResults: { type: Array, default: () => [] },
  importingAllReadingLists: { type: Boolean, default: false },
  isImporting: { type: Function, required: true },
})

defineEmits([
  'import-comic',
  'open-reading-list',
  'import-all-reading-lists',
  'import-series',
  'import-arc',
  'import-character',
])

function comicTitle(comic) {
  if (comic.title) return comic.title
  const seriesName = comic.series || 'Unknown series'
  const number = comic.number || comic.issue
  return number ? `${seriesName} #${number}` : seriesName
}

function comicMeta(comic) {
  return [
    comic.series && comic.title ? comic.series : '',
    comic.seriesVolume ? `Vol. ${comic.seriesVolume}` : '',
    comic.seriesYear || '',
    comic.publisher || '',
    comic.storeDate ? `Store ${formatDate(comic.storeDate)}` : '',
    comic.coverDate ? `Cover ${formatDate(comic.coverDate)}` : '',
  ]
    .filter(Boolean)
    .join(' · ')
}

function comicStoryLine(comic) {
  if (!Array.isArray(comic.storyNames) || comic.storyNames.length === 0) return ''
  return comic.storyNames.join(', ')
}

function readingListSummary(list) {
  return [
    list.listType || 'Reading list',
    list.user?.username ? `by ${list.user.username}` : '',
    list.attributionSource ? `via ${list.attributionSource}` : '',
    list.ratingCount ? `${list.averageRating || 0} avg from ${list.ratingCount} ratings` : '',
    list.modified ? `Modified ${formatDate(list.modified)}` : '',
  ]
    .filter(Boolean)
    .join(' · ')
}

function arcSummary(item) {
  return [
    item.modified ? `Modified ${formatDate(item.modified)}` : '',
    item.id ? `Metron ID ${item.id}` : '',
  ]
    .filter(Boolean)
    .join(' · ')
}

function formatDate(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString([], { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<template>
  <section class="metron-results single">
    <DetailPanel>
      <!-- Native buttons below are full-card result selection targets. -->
      <template v-if="activeSearch === 'comics'">
        <h3>Comics</h3>
        <p v-if="searching" class="muted block text-muted">Searching Metron comics...</p>
        <p v-else-if="comicResults.length === 0" class="muted block text-muted">
          No Metron comic results yet.
        </p>
        <button
          v-for="comic in comicResults"
          :key="comic.id"
          class="row"
          :disabled="isImporting('comic', comic.id)"
          @click="$emit('import-comic', comic)"
        >
          <span>
            <strong>{{ comicTitle(comic) }}</strong>
            <small v-if="comicMeta(comic)">{{ comicMeta(comic) }}</small>
            <small v-if="comicStoryLine(comic)">{{ comicStoryLine(comic) }}</small>
          </span>
          <StatusPill>{{ isImporting('comic', comic.id) ? 'Importing...' : 'Import' }}</StatusPill>
        </button>
      </template>

      <template v-else-if="activeSearch === 'readingLists'">
        <div class="section-title">
          <h3>Reading Lists</h3>
          <BaseButton
            variant="neutral"
            :disabled="importingAllReadingLists || isImporting('readingLists', 0)"
            @click="$emit('import-all-reading-lists')"
          >
            {{
              importingAllReadingLists || isImporting('readingLists', 0)
                ? 'Pulling all...'
                : 'Pull all'
            }}
          </BaseButton>
        </div>
        <p v-if="searching" class="muted block text-muted">Searching Metron reading lists...</p>
        <p v-else-if="readingListResults.length === 0" class="muted block text-muted">
          No Metron reading-list results yet.
        </p>
        <button
          v-for="list in readingListResults"
          :key="list.id"
          class="row"
          :disabled="isImporting('readingList', list.id)"
          @click="$emit('open-reading-list', list)"
        >
          <span>
            <strong>{{ list.name || 'Untitled reading list' }}</strong>
            <small>{{ readingListSummary(list) }}</small>
          </span>
          <StatusPill>
            {{ isImporting('readingList', list.id) ? 'Importing...' : 'Details' }}
          </StatusPill>
        </button>
      </template>

      <template v-else-if="activeSearch === 'series'">
        <h3>Series</h3>
        <p v-if="searching" class="muted block text-muted">Searching Metron series...</p>
        <p v-else-if="seriesResults.length === 0" class="muted block text-muted">
          No Metron series results yet.
        </p>
        <button
          v-for="item in seriesResults"
          :key="item.id"
          class="row"
          :disabled="isImporting('series', item.id)"
          @click="$emit('import-series', item)"
        >
          <span>
            <strong>{{ item.name || 'Untitled series' }}</strong>
            <small>
              Vol. {{ item.volume }} · {{ item.yearBegan || 'Unknown year' }} ·
              {{ item.issueCount }} issues
            </small>
          </span>
          <StatusPill>{{ isImporting('series', item.id) ? 'Importing...' : 'Import' }}</StatusPill>
        </button>
      </template>

      <template v-else-if="activeSearch === 'arcs'">
        <h3>Arcs</h3>
        <p v-if="searching" class="muted block text-muted">Searching Metron arcs...</p>
        <p v-else-if="arcResults.length === 0" class="muted block text-muted">
          No Metron arc results yet.
        </p>
        <button
          v-for="item in arcResults"
          :key="item.id"
          class="row"
          :disabled="isImporting('arc', item.id)"
          @click="$emit('import-arc', item)"
        >
          <span>
            <strong>{{ item.name || 'Untitled arc' }}</strong>
            <small>{{ arcSummary(item) }}</small>
          </span>
          <StatusPill>{{ isImporting('arc', item.id) ? 'Importing...' : 'Import' }}</StatusPill>
        </button>
      </template>

      <template v-else>
        <h3>Characters</h3>
        <p v-if="searching" class="muted block text-muted">Searching Metron characters...</p>
        <p v-else-if="characterResults.length === 0" class="muted block text-muted">
          No Metron character results yet.
        </p>
        <button
          v-for="character in characterResults"
          :key="character.id"
          class="row"
          :disabled="isImporting('character', character.id)"
          @click="$emit('import-character', character)"
        >
          <span>
            <strong>{{ character.name }}</strong>
            <small>Metron ID {{ character.id }}</small>
          </span>
          <StatusPill>
            {{ isImporting('character', character.id) ? 'Importing...' : 'Import' }}
          </StatusPill>
        </button>
      </template>
    </DetailPanel>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.metron-results.single {
  @apply grid grid-cols-3 gap-5 items-start [&.single]:grid-cols-[minmax(0,1fr)] [&_.row:disabled]:cursor-wait [&_.row:disabled]:opacity-[0.72] down-mobile:gap-3.5 down-mobile:[&_.detail-panel]:py-3.5 down-mobile:[&_.detail-panel]:px-3 down-tablet:grid-cols-1;
}

.row {
  @apply min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1;
}

.section-title {
  @apply justify-between mb-2.5 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5;
}

.row strong,
.row small {
  overflow-wrap: anywhere;
}
</style>
