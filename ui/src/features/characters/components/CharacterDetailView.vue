<script setup>
import { assetURL } from '@/api/client.js'
import ComicListView from '@/features/comics/components/ComicListView.vue'
import { formatProgress } from '@/features/reading-orders/model.js'
import DetailNavigation from '@/shared/components/detail/DetailNavigation.vue'
import EmptyState from '@/shared/components/feedback/EmptyState.vue'
import FavoriteToggle from '@/shared/components/feedback/FavoriteToggle.vue'
import ProgressBar from '@/shared/components/feedback/ProgressBar.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import DetailPanel from '@/shared/components/layout/DetailPanel.vue'
import MetadataGrid from '@/shared/components/layout/MetadataGrid.vue'
import PanelHeader from '@/shared/components/layout/PanelHeader.vue'

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

    <DetailPanel>
      <div v-if="selectedCharacter" class="read-only-detail grid gap-4">
        <PanelHeader eyebrow="Character" :title="selectedCharacter.name" />

        <div v-if="selectedCharacter.image" class="character-portrait">
          <img
            :src="assetURL(selectedCharacter.image)"
            :alt="`${selectedCharacter.name} portrait`"
            loading="lazy"
          />
        </div>

        <MetadataGrid>
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
        </MetadataGrid>
        <ProgressBar
          :value="characterProgress(selectedCharacter)"
          label="Character read progress"
        />

        <div v-if="selectedCharacter.aliases?.length" class="alias-list">
          <span v-for="alias in selectedCharacter.aliases" :key="alias">{{ alias }}</span>
        </div>

        <p class="detail-description text-body leading-normal">
          {{ selectedCharacter.description || 'No description' }}
        </p>

        <ComicListView
          embedded
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
      <EmptyState v-else tag="p"> Select a character to view appearances. </EmptyState>
    </DetailPanel>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.character-portrait {
  @apply overflow-hidden border border-line rounded bg-surface-muted max-w-44 [&_img]:block [&_img]:w-full [&_img]:aspect-square [&_img]:object-cover;
}

.alias-list {
  @apply flex flex-wrap gap-2 [&_span]:min-h-8 [&_span]:[border:1px_solid_color-mix(in_srgb,var(--primary)_32%,var(--line-strong))] [&_span]:rounded-full [&_span]:bg-primary-soft [&_span]:text-primary-strong [&_span]:py-1 [&_span]:px-2.5 [&_span]:text-sm [&_span]:font-extrabold;
}
</style>
