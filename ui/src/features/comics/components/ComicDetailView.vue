<script setup>
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import EmptyState from '@/shared/components/feedback/EmptyState.vue'
import StatusPill from '@/shared/components/feedback/StatusPill.vue'
import ComicMergeDialog from '@/features/comics/components/ComicMergeDialog.vue'
import { computed } from 'vue'
import { assetURL } from '@/api/client.js'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import DetailPanel from '@/shared/components/layout/DetailPanel.vue'
import MetadataGrid from '@/shared/components/layout/MetadataGrid.vue'
import PanelHeader from '@/shared/components/layout/PanelHeader.vue'

const props = defineProps({
  selectedComic: {
    type: Object,
    default: null,
  },
  quickSavingComicId: {
    type: Number,
    default: null,
  },
  metronMetadataOpen: {
    type: Boolean,
    default: false,
  },
  metronMetadataSearching: {
    type: Boolean,
    default: false,
  },
  metronMetadataApplyingId: {
    type: Number,
    default: null,
  },
  metronMetadataStatus: {
    type: String,
    default: '',
  },
  metronMetadataResults: {
    type: Array,
    default: () => [],
  },
  readOnly: {
    type: Boolean,
    default: false,
  },
  canDelete: { type: Boolean, default: false },
  deleting: { type: Boolean, default: false },
  mergeOpen: { type: Boolean, default: false },
  mergeCandidates: { type: Array, default: () => [] },
  mergeSearching: { type: Boolean, default: false },
  mergeSaving: { type: Boolean, default: false },
})

const emit = defineEmits([
  'back',
  'search-metron',
  'apply-metron',
  'reset-metron',
  'toggle-read',
  'toggle-skipped',
  'open-character',
  'open-series',
  'delete',
  'open-merge',
  'close-merge',
  'search-merge',
  'merge',
])

const metronActionDisabled = computed(
  () => props.metronMetadataSearching || props.metronMetadataApplyingId !== null,
)
const metronActionLabel = computed(() =>
  props.selectedComic?.metronIssueId ? 'Refresh Metron' : 'Get Metron metadata',
)
const metronIssueURL = computed(() => {
  const id = Number(props.selectedComic?.metronIssueId)
  return Number.isInteger(id) && id > 0 ? `https://metron.cloud/issue/${id}/` : ''
})
const comicVineIssueURL = computed(() => {
  const id = Number(props.selectedComic?.comicVineId)
  return Number.isInteger(id) && id > 0 ? `https://comicvine.gamespot.com/wd/4000-${id}/` : ''
})

function runMetronAction() {
  if (!props.selectedComic) return
  if (props.selectedComic.metronIssueId) {
    emit('apply-metron', props.selectedComic.metronIssueId)
    return
  }
  emit('search-metron')
}

function seriesLabel(comic) {
  if (!comic) return ''
  return `${comic.series || 'Unknown series'}${comic.seriesYear ? ` (${comic.seriesYear})` : ''}`
}
</script>

<template>
  <div class="detail-view grid gap-4 w-full">
    <DetailNavigation @back="$emit('back')">
      <BaseButton
        v-if="selectedComic && canDelete"
        :disabled="deleting || mergeSaving"
        @click="$emit('open-merge')"
      >
        Merge duplicate
      </BaseButton>
      <BaseButton
        v-if="selectedComic && canDelete"
        variant="danger"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? 'Deleting...' : 'Delete comic' }}
      </BaseButton>
      <!-- Native buttons: read/skip controls expose persistent state-specific styling. -->
      <button
        v-if="selectedComic && !readOnly"
        class="read-toggle-button large"
        type="button"
        :disabled="quickSavingComicId === selectedComic.id"
        @click="$emit('toggle-read', selectedComic)"
      >
        {{ selectedComic.read ? 'Mark unread' : 'Mark read' }}
      </button>
      <button
        v-if="selectedComic && !readOnly"
        class="read-toggle-button large"
        :class="{ skipped: selectedComic.skipped }"
        type="button"
        :disabled="quickSavingComicId === selectedComic.id"
        @click="$emit('toggle-skipped', selectedComic)"
      >
        {{ selectedComic.skipped ? 'Unskip' : 'Skip' }}
      </button>
      <BaseButton
        v-if="selectedComic && !readOnly"
        :disabled="metronActionDisabled"
        @click="runMetronAction"
      >
        {{ metronActionLabel }}
      </BaseButton>
    </DetailNavigation>

    <DetailPanel>
      <div v-if="selectedComic" class="read-only-detail grid gap-4">
        <PanelHeader eyebrow="Comic" :title="selectedComic.title" />

        <div v-if="selectedComic.coverImage" class="cover-preview">
          <img
            :src="assetURL(selectedComic.coverImage)"
            :alt="`${selectedComic.title} cover`"
            loading="lazy"
          />
        </div>

        <MetadataGrid>
          <span>
            <strong>
              {{ selectedComic.skipped ? 'Skipped' : selectedComic.read ? 'Read' : 'Unread' }}
            </strong>
            <small>Status</small>
          </span>
          <span>
            <!-- Native button: metadata navigation is styled as an inline text link. -->
            <button
              v-if="selectedComic.seriesId"
              class="metadata-link-button"
              type="button"
              @click="$emit('open-series', { id: selectedComic.seriesId })"
            >
              {{ seriesLabel(selectedComic) }}
            </button>
            <strong v-else>{{ seriesLabel(selectedComic) }}</strong>
            <small>Series</small>
          </span>
          <span>
            <strong>{{ selectedComic.publisher || 'Unknown' }}</strong>
            <small>Publisher</small>
          </span>
          <span>
            <strong>{{ selectedComic.coverDate || 'Unknown' }}</strong>
            <small>Cover Date</small>
          </span>
        </MetadataGrid>

        <nav
          v-if="metronIssueURL || comicVineIssueURL"
          class="comic-source-links"
          aria-label="External comic sources"
        >
          <span>Sources</span>
          <a v-if="metronIssueURL" :href="metronIssueURL" target="_blank" rel="noreferrer noopener">
            Metron <span aria-hidden="true">↗</span>
            <span class="sr-only">(opens in a new tab)</span>
          </a>
          <a
            v-if="comicVineIssueURL"
            :href="comicVineIssueURL"
            target="_blank"
            rel="noreferrer noopener"
          >
            Comic Vine <span aria-hidden="true">↗</span>
            <span class="sr-only">(opens in a new tab)</span>
          </a>
        </nav>

        <div
          v-if="!readOnly && (metronMetadataOpen || metronMetadataStatus)"
          class="metron-metadata-panel"
        >
          <header class="section-title">
            <div>
              <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Metron</p>
              <h4>Metadata matches</h4>
            </div>
            <!-- Native button: this is a borderless inline panel command. -->
            <button
              v-if="metronMetadataOpen || metronMetadataStatus"
              class="ghost-button"
              type="button"
              @click="$emit('reset-metron')"
            >
              Close
            </button>
          </header>
          <p v-if="metronMetadataSearching" class="muted block text-muted">Searching Metron...</p>
          <p v-else-if="metronMetadataStatus" class="muted block text-muted">
            {{ metronMetadataStatus }}
          </p>
          <div v-if="metronMetadataResults.length" class="list grid gap-2.5 down-mobile:gap-2">
            <!-- Native buttons: metadata matches are full-card selection targets. -->
            <button
              v-for="issue in metronMetadataResults"
              :key="issue.id"
              class="row"
              type="button"
              :disabled="metronMetadataApplyingId !== null"
              @click="$emit('apply-metron', issue.id)"
            >
              <span>
                <strong
                  >{{ issue.series }} #{{ issue.number || issue.issue }}:
                  {{ issue.title || 'Untitled issue' }}</strong
                >
                <small
                  >{{ issue.publisher || 'Unknown publisher' }} ·
                  {{ issue.coverDate || 'Unknown date' }}</small
                >
              </span>
              <StatusPill>
                {{ metronMetadataApplyingId === issue.id ? 'Applying...' : 'Apply' }}
              </StatusPill>
            </button>
          </div>
        </div>

        <p class="detail-description text-body leading-normal">
          {{ selectedComic.description || 'No description' }}
        </p>

        <div v-if="selectedComic.characters?.length" class="detail-section">
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Characters</p>
          <div class="alias-list">
            <!-- Native buttons: character chips use pill-shaped navigation styling. -->
            <button
              v-for="character in selectedComic.characters"
              :key="character.id"
              class="character-chip"
              type="button"
              @click="$emit('open-character', character)"
            >
              {{ character.name }}
            </button>
          </div>
        </div>
      </div>
      <EmptyState v-else tag="p"> Select a comic to view it. </EmptyState>
    </DetailPanel>

    <ComicMergeDialog
      v-if="mergeOpen && selectedComic"
      :target="selectedComic"
      :candidates="mergeCandidates"
      :searching="mergeSearching"
      :saving="mergeSaving"
      @close="$emit('close-merge')"
      @search="$emit('search-merge', $event)"
      @merge="$emit('merge', $event)"
    />
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.read-toggle-button.large {
  @apply flex-none min-h-8 border border-line-strong rounded bg-surface text-label py-1.5 px-2.5 text-sm font-bold whitespace-nowrap [&.skipped]:border-muted [&.skipped]:text-muted [&.large]:min-h-10 [&.large]:py-2.5 [&.large]:px-3.5 [&.large]:text-base;
}

.cover-preview {
  @apply overflow-hidden border border-line rounded bg-surface-muted max-w-56 [&_img]:block [&_img]:w-full [&_img]:aspect-portrait [&_img]:object-cover;
}

.metadata-link-button {
  @apply block w-full cursor-pointer border-0 bg-transparent p-0 text-left font-extrabold text-accent [font:inherit] hover:[text-decoration:underline];
  overflow-wrap: anywhere;
}

.comic-source-links {
  @apply flex items-center flex-wrap gap-y-1.5 gap-x-3 mt-2.5 text-muted text-xs [&_>_span]:font-bold [&_a]:text-muted [&_a]:no-underline [&_a:hover]:text-accent [&_a:hover]:[text-decoration:underline] [&_a:focus-visible]:rounded-xs [&_a:focus-visible]:outline-3 [&_a:focus-visible]:outline-focus [&_a:focus-visible]:outline-offset-2;
}

.metron-metadata-panel {
  @apply grid gap-3 border-t border-line pt-3.5 [&_.section-title]:mb-0;
}

.section-title {
  @apply justify-between mb-2.5 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5;
}

.ghost-button {
  @apply min-h-8 border-0 rounded-[7px] bg-transparent text-accent py-1.5 px-2 font-bold;
}

.row {
  @apply min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1;
}

.alias-list {
  @apply flex flex-wrap gap-2 [&_span]:min-h-8 [&_span]:[border:1px_solid_color-mix(in_srgb,var(--primary)_32%,var(--line-strong))] [&_span]:rounded-full [&_span]:bg-primary-soft [&_span]:text-primary-strong [&_span]:py-1 [&_span]:px-2.5 [&_span]:text-sm [&_span]:font-extrabold;
}

.row strong {
  overflow-wrap: anywhere;
}

.row small {
  overflow-wrap: anywhere;
}

.detail-section {
  @apply border-t border-line pt-3.5;
}

.detail-section small {
  @apply block text-muted;
}

.detail-section :is(ol, ul) {
  @apply mb-0 pl-6;
}

.detail-section li {
  @apply mb-2.5;
}

.character-chip {
  @apply min-h-8 cursor-pointer rounded-full bg-primary-soft px-2.5 py-1 text-sm font-extrabold text-primary-strong;
  border: 1px solid color-mix(in srgb, var(--primary) 32%, var(--line-strong));
}
</style>
