<script setup>
import { ref } from 'vue'
import { assetURL } from '@/api/client.js'
import ComicListView from '@/features/comics/components/ComicListView.vue'
import { formatProgress } from '@/features/reading-orders/model.js'
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import CharacterPickerDialog from './CharacterPickerDialog.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import ProgressBar from '@/shared/components/feedback/ProgressBar.vue'
import DetailPanel from '@/shared/components/layout/DetailPanel.vue'

defineProps({
  collection: { type: Object, default: null },
  selectedComicId: { type: Number, default: null },
  quickSavingComicId: { type: Number, default: null },
  saving: { type: Boolean, default: false },
  startSaving: { type: Boolean, default: false },
})
defineEmits([
  'back',
  'toggle-started',
  'delete',
  'add-character',
  'remove-character',
  'open-character',
  'open-comic',
  'toggle-read',
  'toggle-skipped',
])
const pickerOpen = ref(false)

function monogram(name) {
  return (name || '?').trim().slice(0, 1).toUpperCase() || '?'
}
</script>

<template>
  <div class="detail-view grid gap-4 w-full">
    <DetailNavigation @back="$emit('back')">
      <BaseButton
        v-if="collection"
        :variant="collection.startedAt ? 'secondary' : 'primary'"
        :disabled="startSaving"
        @click="$emit('toggle-started')"
      >
        {{ startSaving ? 'Saving...' : collection.startedAt ? 'Stop reading' : 'Start reading' }}
      </BaseButton>
      <BaseButton variant="danger" :disabled="saving" @click="$emit('delete')">
        {{ saving ? 'Saving...' : 'Delete collection' }}
      </BaseButton>
    </DetailNavigation>

    <DetailPanel>
      <div v-if="collection" class="read-only-detail grid gap-4">
        <header class="collection-detail-hero">
          <span class="collection-detail-monogram" aria-hidden="true">
            {{ monogram(collection.name) }}
          </span>
          <div class="collection-detail-copy">
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">
              Private character collection
            </p>
            <h3>{{ collection.name }}</h3>
            <p>One release-date queue across every character in this collection.</p>
          </div>
          <span class="collection-detail-status" :class="{ active: collection.startedAt }">
            {{ collection.startedAt ? 'Currently reading' : 'Not started' }}
          </span>
        </header>
        <div class="collection-detail-summary">
          <span
            ><strong>{{ formatProgress(collection.progress) }}</strong
            ><small>Read</small></span
          >
          <span
            ><strong>{{ collection.characterCount }}</strong
            ><small>Characters</small></span
          >
          <span>
            <strong>{{ collection.appearanceCount }}</strong>
            <small>Distinct appearances</small>
          </span>
        </div>
        <ProgressBar
          :value="formatProgress(collection.progress)"
          label="Collection read progress"
        />

        <section class="collection-members grid gap-3 mt-2.5 pt-5 border-t border-line">
          <header class="collection-members-header">
            <div>
              <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Members</p>
              <h3>Characters</h3>
            </div>
            <BaseButton :disabled="saving" @click="pickerOpen = true">
              <span class="button-icon" aria-hidden="true">+</span>
              Add character
            </BaseButton>
          </header>
          <div v-if="collection.characters?.length" class="collection-member-list">
            <div v-for="character in collection.characters" :key="character.id" class="row">
              <!-- Native button: the member body is a full-row navigation target. -->
              <button
                type="button"
                class="collection-member-main"
                @click="$emit('open-character', character)"
              >
                <span class="collection-member-avatar" aria-hidden="true">
                  <img
                    v-if="character.image"
                    :src="assetURL(character.image)"
                    alt=""
                    loading="lazy"
                  />
                  <span v-else>{{ monogram(character.name) }}</span>
                </span>
                <span>
                  <strong>{{ character.name }}</strong>
                  <small>{{ character.appearanceCount }} appearances</small>
                </span>
              </button>
              <BaseButton
                class="flex-none"
                variant="danger-ghost"
                :disabled="saving"
                @click="$emit('remove-character', character)"
              >
                Remove
              </BaseButton>
            </div>
          </div>
          <div v-else class="collection-members-empty">
            <span>
              <strong>No characters yet</strong>
              <small>Search your library to start building this collection.</small>
            </span>
          </div>
        </section>

        <ComicListView
          embedded
          title="Combined appearances"
          :comics="collection.comics || []"
          :selected-comic-id="selectedComicId"
          :quick-saving-comic-id="quickSavingComicId"
          initial-sort="date"
          paginate-local
          empty-message="Add characters to build this reading queue."
          filtered-empty-message="No appearances match these filters."
          @open-comic="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
          @toggle-skipped="$emit('toggle-skipped', $event)"
        />
      </div>
    </DetailPanel>

    <CharacterPickerDialog
      v-if="pickerOpen"
      :saving="saving"
      @close="pickerOpen = false"
      @add="$emit('add-character', $event)"
    />
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.collection-detail-hero {
  @apply grid grid-cols-[72px_minmax(0,1fr)_auto] items-center gap-4 border border-line rounded-xl [background:var(--primary-soft),var(--surface-soft)] p-4 down-narrow:grid-cols-[58px_minmax(0,1fr)];
}

.collection-detail-monogram {
  @apply w-20 h-20 text-3xl down-narrow:w-14 down-narrow:h-14 down-narrow:text-2xl inline-flex items-center justify-center flex-none border border-line-strong rounded-xl bg-primary text-white font-black shadow-control;
}

.collection-detail-copy {
  @apply min-w-0 [&_h3]:my-1 [&_h3]:mx-1 [&_>_p:last-child]:m-0 [&_>_p:last-child]:text-muted;
}

.collection-detail-status {
  @apply self-start border border-line-strong rounded-full bg-surface text-muted py-1 px-2.5 text-xs font-extrabold down-narrow:col-span-full down-narrow:justify-self-start [&.active]:border-primary [&.active]:bg-primary-soft [&.active]:text-primary-strong;
}

.collection-detail-summary {
  @apply grid grid-cols-3 gap-2.5 down-compact:grid-cols-1 [&_span]:border [&_span]:border-line [&_span]:rounded-lg [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_small]:block [&_strong]:text-control [&_strong]:text-base [&_small]:mt-1 [&_small]:text-muted;
}

.collection-members-header {
  @apply flex items-center justify-between gap-3.5 [&_h3]:mt-0.5 [&_h3]:mx-0 [&_h3]:mb-0 down-compact:items-stretch down-compact:flex-col down-compact:[&_button]:w-full;
}

.button-icon {
  @apply inline-flex items-center justify-center size-5 text-xl font-extrabold leading-none;
}

.collection-member-list {
  @apply grid grid-cols-[minmax(0,1fr)] gap-2 down-compact:grid-cols-1 [&_.row]:items-center [&_.row]:min-h-[70px] [&_.row]:p-2.5;
}

.row {
  @apply min-h-10 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-12 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1;
}

.collection-member-main {
  @apply grid grid-cols-[42px_minmax(0,1fr)] items-center gap-2.5 flex-auto min-w-0 border-0 bg-transparent text-inherit p-0 text-left [&:hover:not(:disabled)]:border-transparent [&:hover:not(:disabled)]:shadow-none [&:hover:not(:disabled)]:transform-none *:min-w-0 [&_strong]:block [&_strong]:overflow-hidden [&_strong]:text-ellipsis [&_strong]:whitespace-nowrap [&_small]:block [&_small]:overflow-hidden [&_small]:text-ellipsis [&_small]:whitespace-nowrap [&_small]:mt-1 [&_small]:text-muted;
}

.collection-member-avatar {
  @apply grid place-items-center w-10 h-10 overflow-hidden border border-line-strong rounded-lg bg-surface-muted text-muted font-black [&_img]:w-full [&_img]:h-full [&_img]:object-cover;
}

.collection-members-empty {
  @apply flex items-center justify-between gap-4 border border-dashed border-line-strong rounded-lg bg-surface-soft p-4 [&_strong]:block [&_small]:block [&_small]:mt-1 [&_small]:text-muted down-compact:items-stretch down-compact:flex-col down-compact:[&_button]:w-full;
}

.collection-detail-copy h3 {
  overflow-wrap: anywhere;
}

.row strong {
  overflow-wrap: anywhere;
}

.row small {
  overflow-wrap: anywhere;
}

.collection-member-main span {
  overflow-wrap: anywhere;
}
</style>
