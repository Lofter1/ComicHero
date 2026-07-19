<script setup>
import { ref } from 'vue'
import { assetURL } from '@/api/client.js'
import ComicListView from '@/features/comics/components/ComicListView.vue'
import { formatProgress } from '@/features/reading-orders/model.js'
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import CharacterPickerDialog from './CharacterPickerDialog.vue'

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
      <button
        v-if="collection"
        :class="collection.startedAt ? 'secondary-button' : 'primary-button'"
        type="button"
        :disabled="startSaving"
        @click="$emit('toggle-started')"
      >
        {{ startSaving ? 'Saving...' : collection.startedAt ? 'Stop reading' : 'Start reading' }}
      </button>
      <button
        class="danger-button min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 [border-color:color-mix(in_srgb,_var(--danger)_42%,_var(--line-strong))] bg-danger-soft text-danger"
        type="button"
        :disabled="saving"
        @click="$emit('delete')"
      >
        {{ saving ? 'Saving...' : 'Delete collection' }}
      </button>
    </DetailNavigation>

    <article
      class="detail-panel min-h-90 border border-line rounded bg-panel p-5 shadow-detail down-mobile:min-h-0 down-mobile:p-3.5"
    >
      <div v-if="collection" class="read-only-detail grid gap-4.5">
        <header
          class="collection-detail-hero grid [grid-template-columns:72px_minmax(0,_1fr)_auto] items-center gap-4 border border-line rounded-xl [background:var(--primary-soft),_var(--surface-soft)] p-4.5 down-narrow:[grid-template-columns:58px_minmax(0,_1fr)]"
        >
          <span
            class="collection-detail-monogram w-18 h-18 text-ui-display-sm down-narrow:w-14.5 down-narrow:h-14.5 down-narrow:[font-size:1.4rem] inline-flex items-center justify-center flex-none border border-line-strong rounded-xl bg-primary text-white font-black shadow-control"
            aria-hidden="true"
          >
            {{ monogram(collection.name) }}
          </span>
          <div
            class="collection-detail-copy min-w-0 [&_h3]:my-0.75 [&_h3]:mx-1.25 [&_h3]:break-anywhere [&_>_p:last-child]:m-0 [&_>_p:last-child]:text-muted"
          >
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">
              Private character collection
            </p>
            <h3>{{ collection.name }}</h3>
            <p>One release-date queue across every character in this collection.</p>
          </div>
          <span
            class="collection-detail-status self-start border border-line-strong rounded-full bg-surface text-muted py-1.25 px-2.5 text-ui-compact-xs font-extrabold down-narrow:col-span-full down-narrow:justify-self-start [&.active]:border-primary [&.active]:bg-primary-soft [&.active]:text-primary-strong"
            :class="{ active: collection.startedAt }"
          >
            {{ collection.startedAt ? 'Currently reading' : 'Not started' }}
          </span>
        </header>
        <div
          class="collection-detail-summary grid grid-cols-3 gap-2.5 down-compact:grid-cols-1 [&_span]:border [&_span]:border-line [&_span]:rounded-lg [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_small]:block [&_strong]:text-control [&_strong]:text-lead [&_small]:mt-0.75 [&_small]:text-muted"
        >
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
        <div
          class="progress-meter h-2.5 overflow-hidden rounded-full bg-read-progress [&_span]:block [&_span]:h-full [&_span]:min-w-0.5 [&_span]:[border-radius:inherit] [&_span]:bg-progress"
          aria-label="Collection read progress"
        >
          <span :style="{ width: formatProgress(collection.progress) }"></span>
        </div>

        <section class="collection-members grid gap-3 mt-2.5 pt-5 border-t border-line">
          <header
            class="collection-members-header flex items-center justify-between gap-3.5 [&_h3]:mt-0.5 [&_h3]:mx-0 [&_h3]:mb-0 down-compact:items-stretch down-compact:flex-col down-compact:[&_button]:w-full"
          >
            <div>
              <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Members</p>
              <h3>Characters</h3>
            </div>
            <button
              class="secondary-button icon-text-button min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))] inline-flex items-center justify-center gap-2"
              type="button"
              :disabled="saving"
              @click="pickerOpen = true"
            >
              <span
                class="button-icon inline-flex items-center justify-center w-em h-em text-xl font-extrabold leading-none"
                aria-hidden="true"
                >+</span
              >
              Add character
            </button>
          </header>
          <div
            v-if="collection.characters?.length"
            class="collection-member-list grid [grid-template-columns:minmax(0,_1fr)] gap-2 down-compact:grid-cols-1 [&_.row]:items-center [&_.row]:[min-height:70px] [&_.row]:p-2.5"
          >
            <div
              v-for="character in collection.characters"
              :key="character.id"
              class="row min-h-10.5 border border-line-strong rounded bg-surface text-control w-full p-3.5 flex justify-between items-start gap-3 text-left hover:bg-surface-soft [&_>_span:first-child]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&.selected]:border-primary [&.selected]:shadow-selected [&_small]:block [&_small]:text-muted down-mobile:min-h-13 down-mobile:p-3 down-mobile:flex-wrap down-phone:grid down-phone:grid-cols-1"
            >
              <button
                type="button"
                class="row-main collection-member-main grid [grid-template-columns:42px_minmax(0,_1fr)] items-center gap-2.5 flex-auto min-w-0 border-0 bg-transparent text-inherit p-0 text-left [&:hover:not(:disabled)]:[border-color:transparent] [&:hover:not(:disabled)]:shadow-none [&:hover:not(:disabled)]:transform-none [&_span]:break-anywhere [&_>_*]:min-w-0 [&_strong]:block [&_strong]:overflow-hidden [&_strong]:text-ellipsis [&_strong]:whitespace-nowrap [&_small]:block [&_small]:overflow-hidden [&_small]:text-ellipsis [&_small]:whitespace-nowrap [&_small]:mt-0.75 [&_small]:text-muted"
                @click="$emit('open-character', character)"
              >
                <span
                  class="collection-member-avatar grid place-items-center w-10.5 h-10.5 overflow-hidden border border-line-strong rounded-lg bg-surface-muted text-muted font-black [&_img]:w-full [&_img]:h-full [&_img]:object-cover"
                  aria-hidden="true"
                >
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
              <button
                class="danger-text-button collection-member-remove flex-none min-h-9.5 border border-danger-border rounded bg-surface text-danger py-2 px-3 font-black [&:hover:not(:disabled)]:border-danger-border [&:hover:not(:disabled)]:bg-danger-soft focus-visible:border-danger-border focus-visible:bg-danger-soft"
                type="button"
                :disabled="saving"
                @click="$emit('remove-character', character)"
              >
                Remove
              </button>
            </div>
          </div>
          <div
            v-else
            class="collection-members-empty flex items-center justify-between gap-4 border border-dashed border-line-strong rounded-lg bg-surface-soft p-4 [&_strong]:block [&_small]:block [&_small]:mt-1 [&_small]:text-muted down-compact:items-stretch down-compact:flex-col down-compact:[&_button]:w-full"
          >
            <span>
              <strong>No characters yet</strong>
              <small>Search your library to start building this collection.</small>
            </span>
          </div>
        </section>

        <ComicListView
          class="preview-list [&_small]:block [&_small]:text-muted border-t border-line pt-3.5 [&_ol]:mb-0 [&_ol]:pl-5.5 [&_ul]:mb-0 [&_ul]:pl-5.5 [&_li]:mb-2.5"
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
    </article>

    <CharacterPickerDialog
      v-if="pickerOpen"
      :saving="saving"
      @close="pickerOpen = false"
      @add="$emit('add-character', $event)"
    />
  </div>
</template>
