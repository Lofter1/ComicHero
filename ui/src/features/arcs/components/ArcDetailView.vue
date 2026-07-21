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
  selectedArc: {
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
  quickSavingArcId: {
    type: Number,
    default: null,
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
  'open-comic',
  'toggle-read',
  'toggle-skipped',
  'delete',
])
</script>

<template>
  <div class="detail-view grid gap-4 w-full">
    <DetailNavigation @back="$emit('back')">
      <BaseButton
        v-if="selectedArc && canDelete"
        variant="danger"
        :disabled="deleting"
        @click="$emit('delete')"
      >
        {{ deleting ? 'Deleting...' : 'Delete arc' }}
      </BaseButton>
      <FavoriteToggle
        v-if="selectedArc && !readOnly"
        class="detail-favorite-toggle self-center"
        :favorite="selectedArc.favorite"
        :disabled="quickSavingArcId === selectedArc.id"
        @toggle="$emit('toggle-favorite', selectedArc)"
      />
      <BaseButton
        v-if="selectedArc && !readOnly"
        :variant="selectedArc.startedAt ? 'secondary' : 'primary'"
        :disabled="startSaving"
        @click="$emit('toggle-started')"
      >
        {{ startSaving ? 'Saving...' : selectedArc.startedAt ? 'Stop reading' : 'Start reading' }}
      </BaseButton>
    </DetailNavigation>

    <DetailPanel>
      <div v-if="selectedArc" class="read-only-detail grid gap-4">
        <PanelHeader eyebrow="Arc" :title="selectedArc.name" />

        <div v-if="selectedArc.image" class="character-portrait">
          <img
            :src="assetURL(selectedArc.image)"
            :alt="`${selectedArc.name} arc artwork`"
            loading="lazy"
          />
        </div>

        <p class="detail-description text-body leading-normal">
          {{ selectedArc.description || 'No description' }}
        </p>
        <ProgressBar :value="formatProgress(selectedArc.progress)" label="Arc progress" />
        <MetadataGrid>
          <span>
            <strong>{{ formatProgress(selectedArc.progress) }}</strong>
            <small>Progress</small>
          </span>
          <span>
            <strong>{{ selectedArc.comics.length }}</strong>
            <small>Comics</small>
          </span>
          <span>
            <strong>{{ selectedArc.favoriteCount }}</strong>
            <small>Favorites</small>
          </span>
          <span>
            <strong>{{ selectedArc.startedCount }}</strong>
            <small>Currently reading</small>
          </span>
          <span v-if="selectedArc.startedAt"
            ><strong>Started</strong
            ><small>{{ new Date(selectedArc.startedAt).toLocaleDateString() }}</small></span
          >
        </MetadataGrid>

        <ComicListView
          embedded
          title="Comics"
          :comics="selectedArc.comics"
          :source-params="{ arcId: selectedArc.id }"
          :selected-comic-id="selectedComicId"
          :quick-saving-comic-id="quickSavingComicId"
          show-comment
          paginate-local
          server-source
          :read-only="readOnly"
          empty-message="No comics in this arc yet."
          filtered-empty-message="No comics match these filters."
          @open-comic="$emit('open-comic', $event)"
          @toggle-read="$emit('toggle-read', $event)"
          @toggle-skipped="$emit('toggle-skipped', $event)"
        />
      </div>
      <EmptyState v-else tag="p"> Select an arc to view it. </EmptyState>
    </DetailPanel>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.character-portrait {
  @apply overflow-hidden border border-line rounded bg-surface-muted max-w-44 [&_img]:block [&_img]:w-full [&_img]:aspect-square [&_img]:object-cover;
}
</style>
