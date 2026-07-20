<script setup>
import { assetURL } from '@/api/client.js'
import ComicListView from '@/features/comics/components/ComicListView.vue'
import { formatProgress } from '@/features/reading-orders/model.js'
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import FavoriteToggle from '@/shared/components/feedback/FavoriteToggle.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'

defineProps({
  selectedCharacter: {
    type: Object,
    default: null,
  },
  selectedComicId: {
    type: Number,
    default: null,
  },
  quickSavingComicId: {
    type: Number,
    default: null,
  },
  quickSavingCharacterId: {
    type: Number,
    default: null,
  },
  importRunning: {
    type: Boolean,
    default: false,
  },
  readOnly: {
    type: Boolean,
    default: false,
  },
  canDelete: { type: Boolean, default: false },
  deleting: { type: Boolean, default: false },
  startSaving: { type: Boolean, default: false },
})

defineEmits([
  'back',
  'toggle-favorite',
  'toggle-started',
  'import-appearances',
  'open-comic',
  'toggle-read',
  'toggle-skipped',
  'delete',
  'add-to-collection',
])

function characterProgress(character) {
  return formatProgress(character?.progress ?? 0)
}
</script>

<template>
  <div class="detail-view grid gap-4 w-full">
    <DetailNavigation @back="$emit('back')">
      <BaseButton
        v-if="selectedCharacter && canDelete"
        variant="danger"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? 'Deleting...' : 'Delete character' }}
      </BaseButton>
      <FavoriteToggle
        v-if="selectedCharacter && !readOnly"
        class="detail-favorite-toggle self-center"
        :favorite="selectedCharacter.favorite"
        :disabled="quickSavingCharacterId === selectedCharacter.id"
        @toggle="$emit('toggle-favorite', selectedCharacter)"
      />
      <BaseButton
        v-if="selectedCharacter && !readOnly"
        @click="$emit('add-to-collection', selectedCharacter)"
      >
        Add to collection
      </BaseButton>
      <BaseButton
        v-if="selectedCharacter && !readOnly"
        :variant="selectedCharacter.startedAt ? 'secondary' : 'primary'"
        :disabled="startSaving"
        @click="$emit('toggle-started')"
      >
        {{
          startSaving ? 'Saving...' : selectedCharacter.startedAt ? 'Stop reading' : 'Start reading'
        }}
      </BaseButton>
      <BaseButton
        v-if="selectedCharacter?.metronCharacterId && !readOnly"
        variant="primary"
        :disabled="importRunning"
        @click="$emit('import-appearances')"
      >
        {{ importRunning ? 'Importing...' : 'Import from Metron' }}
      </BaseButton>
    </DetailNavigation>

    <article
      class="detail-panel min-h-panel border border-line rounded bg-panel p-5 shadow-detail down-mobile:min-h-0 down-mobile:p-3.5"
    >
      <div v-if="selectedCharacter" class="read-only-detail grid gap-4">
        <header
          class="panel-header justify-between mb-4 down-mobile:items-stretch down-mobile:flex-col down-mobile:gap-2.5 down-mobile:[&_button]:w-full flex items-center gap-3.5"
        >
          <div>
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Character</p>
            <h3>{{ selectedCharacter.name }}</h3>
          </div>
        </header>

        <div
          v-if="selectedCharacter.image"
          class="character-portrait overflow-hidden border border-line rounded bg-surface-muted max-w-44 [&_img]:block [&_img]:w-full [&_img]:aspect-square [&_img]:object-cover"
        >
          <img
            :src="assetURL(selectedCharacter.image)"
            :alt="`${selectedCharacter.name} portrait`"
            loading="lazy"
          />
        </div>

        <div
          class="metadata-grid grid grid-cols-3 gap-2.5 [&_span]:border [&_span]:border-line [&_span]:rounded [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_strong]:break-anywhere [&_small]:block [&_small]:text-muted [&_small]:mt-1 down-tablet:grid-cols-1"
        >
          <span>
            <strong>{{ characterProgress(selectedCharacter) }}</strong>
            <small>Progress</small>
          </span>
          <span>
            <strong>{{ selectedCharacter.appearanceCount }}</strong>
            <small>Appearances</small>
          </span>
          <span>
            <strong>{{ selectedCharacter.aliases?.length || 0 }}</strong>
            <small>Aliases</small>
          </span>
          <span>
            <strong>{{ selectedCharacter.favoriteCount }}</strong>
            <small>Favorites</small>
          </span>
          <span>
            <strong>{{ selectedCharacter.startedCount }}</strong>
            <small>Currently reading</small>
          </span>
          <span v-if="selectedCharacter.startedAt"
            ><strong>Started</strong
            ><small>{{ new Date(selectedCharacter.startedAt).toLocaleDateString() }}</small></span
          >
        </div>
        <div
          class="progress-meter h-2.5 overflow-hidden rounded-full bg-read-progress [&_span]:block [&_span]:h-full [&_span]:min-w-0.5 [&_span]:[border-radius:inherit] [&_span]:bg-progress"
          aria-label="Character read progress"
        >
          <span :style="{ width: characterProgress(selectedCharacter) }"></span>
        </div>

        <div
          v-if="selectedCharacter.aliases?.length"
          class="alias-list flex flex-wrap gap-2 [&_span]:min-h-8 [&_span]:[border:1px_solid_color-mix(in_srgb,_var(--primary)_32%,_var(--line-strong))] [&_span]:rounded-full [&_span]:bg-primary-soft [&_span]:text-primary-strong [&_span]:py-1 [&_span]:px-2.5 [&_span]:text-sm [&_span]:font-extrabold"
        >
          <span v-for="alias in selectedCharacter.aliases" :key="alias">{{ alias }}</span>
        </div>

        <p class="detail-description text-body leading-normal">
          {{ selectedCharacter.description || 'No description' }}
        </p>

        <ComicListView
          class="[&_small]:block [&_small]:text-muted border-t border-line pt-3.5 [&_ol]:mb-0 [&_ol]:pl-6 [&_ul]:mb-0 [&_ul]:pl-6 [&_li]:mb-2.5"
          title="Appearances"
          :comics="selectedCharacter.comics || []"
          :source-params="{ characterId: selectedCharacter.id }"
          :selected-comic-id="selectedComicId"
          :quick-saving-comic-id="quickSavingComicId"
          initial-sort="date"
          paginate-local
          server-source
          :read-only="readOnly"
          empty-message="No appearances saved yet."
          filtered-empty-message="No appearances match these filters."
          @open-comic="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
          @toggle-skipped="$emit('toggle-skipped', $event)"
        />
      </div>
      <p
        v-else
        class="empty-state grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4"
      >
        Select a character to view appearances.
      </p>
    </article>
  </div>
</template>
