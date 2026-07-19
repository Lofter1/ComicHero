<script setup>
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import ComicMergeDialog from '@/features/comics/components/ComicMergeDialog.vue'
import { computed } from 'vue'
import { assetURL } from '@/api/client.js'

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
      <button
        v-if="selectedComic && canDelete"
        class="secondary-button min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]"
        type="button"
        :disabled="deleting || mergeSaving"
        @click="$emit('open-merge')"
      >
        Merge duplicate
      </button>
      <button
        v-if="selectedComic && canDelete"
        class="danger-button min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 [border-color:color-mix(in_srgb,_var(--danger)_42%,_var(--line-strong))] bg-danger-soft text-danger"
        type="button"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? 'Deleting...' : 'Delete comic' }}
      </button>
      <button
        v-if="selectedComic && !readOnly"
        class="read-toggle-button large flex-none min-h-8 border border-line-strong rounded bg-surface text-label py-1.5 px-2.5 text-sm font-bold whitespace-nowrap [&.skipped]:border-muted [&.skipped]:text-muted [&.large]:min-h-10 [&.large]:py-2.5 [&.large]:px-3.5 [&.large]:text-base"
        type="button"
        :disabled="quickSavingComicId === selectedComic.id"
        @click="$emit('toggle-read', selectedComic)"
      >
        {{ selectedComic.read ? 'Mark unread' : 'Mark read' }}
      </button>
      <button
        v-if="selectedComic && !readOnly"
        class="read-toggle-button large flex-none min-h-8 border border-line-strong rounded bg-surface text-label py-1.5 px-2.5 text-sm font-bold whitespace-nowrap [&.skipped]:border-muted [&.skipped]:text-muted [&.large]:min-h-10 [&.large]:py-2.5 [&.large]:px-3.5 [&.large]:text-base"
        :class="{ skipped: selectedComic.skipped }"
        type="button"
        :disabled="quickSavingComicId === selectedComic.id"
        @click="$emit('toggle-skipped', selectedComic)"
      >
        {{ selectedComic.skipped ? 'Unskip' : 'Skip' }}
      </button>
      <button
        v-if="selectedComic && !readOnly"
        class="secondary-button min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]"
        type="button"
        :disabled="metronActionDisabled"
        @click="runMetronAction"
      >
        {{ metronActionLabel }}
      </button>
    </DetailNavigation>

    <article
      class="detail-panel min-h-panel border border-line rounded bg-panel p-5 shadow-detail down-mobile:min-h-0 down-mobile:p-3.5"
    >
      <div v-if="selectedComic" class="read-only-detail grid gap-4">
        <header
          class="panel-header justify-between mb-4 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5"
        >
          <div>
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Comic</p>
            <h3>{{ selectedComic.title }}</h3>
          </div>
        </header>

        <div
          v-if="selectedComic.coverImage"
          class="cover-preview overflow-hidden border border-line rounded bg-surface-muted max-w-56 [&_img]:block [&_img]:w-full [&_img]:aspect-portrait [&_img]:object-cover"
        >
          <img
            :src="assetURL(selectedComic.coverImage)"
            :alt="`${selectedComic.title} cover`"
            loading="lazy"
          />
        </div>

        <div
          class="metadata-grid grid grid-cols-3 gap-2.5 [&_span]:border [&_span]:border-line [&_span]:rounded [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_strong]:break-anywhere [&_small]:block [&_small]:text-muted [&_small]:mt-1 down-tablet:grid-cols-1"
        >
          <span>
            <strong>
              {{ selectedComic.skipped ? 'Skipped' : selectedComic.read ? 'Read' : 'Unread' }}
            </strong>
            <small>Status</small>
          </span>
          <span>
            <button
              v-if="selectedComic.seriesId"
              class="metadata-link-button block w-full border-0 bg-transparent text-accent p-0 [font:inherit] font-extrabold text-left break-anywhere cursor-pointer hover:[text-decoration:underline]"
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
        </div>

        <nav
          v-if="metronIssueURL || comicVineIssueURL"
          class="comic-source-links flex items-center flex-wrap gap-y-1.5 gap-x-3 mt-2.5 text-muted text-xs [&_>_span]:font-bold [&_a]:text-muted [&_a]:no-underline [&_a:hover]:text-accent [&_a:hover]:[text-decoration:underline] [&_a:focus-visible]:rounded-xs [&_a:focus-visible]:outline-3 [&_a:focus-visible]:outline-focus [&_a:focus-visible]:outline-offset-2"
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
          class="metron-metadata-panel grid gap-3 border-t border-line pt-3.5 [&_.section-title]:mb-0"
        >
          <header
            class="section-title justify-between mb-2.5 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5"
          >
            <div>
              <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Metron</p>
              <h4>Metadata matches</h4>
            </div>
            <button
              v-if="metronMetadataOpen || metronMetadataStatus"
              class="ghost-button min-h-8 border-0 rounded-[7px] bg-transparent text-accent py-1.5 px-2 font-bold"
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
            <button
              v-for="issue in metronMetadataResults"
              :key="issue.id"
              class="row min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
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
              <span
                class="status-pill border-0 rounded-full bg-primary-soft text-primary py-1 px-2 text-xs flex-none font-bold down-mobile:ml-auto down-phone:justify-self-start down-phone:ml-0"
              >
                {{ metronMetadataApplyingId === issue.id ? 'Applying...' : 'Apply' }}
              </span>
            </button>
          </div>
        </div>

        <p class="detail-description text-body leading-normal">
          {{ selectedComic.description || 'No description' }}
        </p>

        <div
          v-if="selectedComic.characters?.length"
          class="[&_small]:block [&_small]:text-muted border-t border-line pt-3.5 [&_ol]:mb-0 [&_ol]:pl-6 [&_ul]:mb-0 [&_ul]:pl-6 [&_li]:mb-2.5"
        >
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Characters</p>
          <div
            class="alias-list flex flex-wrap gap-2 [&_span]:min-h-8 [&_span]:[border:1px_solid_color-mix(in_srgb,_var(--primary)_32%,_var(--line-strong))] [&_span]:rounded-full [&_span]:bg-primary-soft [&_span]:text-primary-strong [&_span]:py-1 [&_span]:px-2.5 [&_span]:text-sm [&_span]:font-extrabold [&_button]:min-h-8 [&_button]:[border:1px_solid_color-mix(in_srgb,_var(--primary)_32%,_var(--line-strong))] [&_button]:rounded-full [&_button]:bg-primary-soft [&_button]:text-primary-strong [&_button]:py-1 [&_button]:px-2.5 [&_button]:text-sm [&_button]:font-extrabold [&_button]:cursor-pointer"
          >
            <button
              v-for="character in selectedComic.characters"
              :key="character.id"
              type="button"
              @click="$emit('open-character', character)"
            >
              {{ character.name }}
            </button>
          </div>
        </div>
      </div>
      <p
        v-else
        class="empty-state grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4"
      >
        Select a comic to view it.
      </p>
    </article>

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
