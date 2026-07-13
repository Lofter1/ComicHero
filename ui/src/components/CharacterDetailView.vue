<script setup>
import { assetURL } from '@/api/client.js'
import ComicListView from '@/components/ComicListView.vue'
import { formatProgress } from '@/domain/readingOrders.js'

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
])

function characterProgress(character) {
  return formatProgress(character?.progress ?? 0)
}
</script>

<template>
  <div class="detail-view">
    <header class="detail-nav sticky-toolbar">
      <button class="secondary-button" type="button" @click="$emit('back')">Back</button>
      <div class="detail-nav-actions">
        <button
          v-if="selectedCharacter && canDelete"
          class="danger-button"
          type="button"
          :disabled="deleting"
          @click="$emit('delete')"
        >
          {{ deleting ? 'Deleting...' : 'Delete character' }}
        </button>
        <button
          v-if="selectedCharacter && !readOnly"
          type="button"
          class="favorite-toggle detail-favorite-toggle"
          :class="{ active: selectedCharacter.favorite }"
          :disabled="quickSavingCharacterId === selectedCharacter.id"
          :aria-label="selectedCharacter.favorite ? 'Remove from favorites' : 'Add to favorites'"
          :title="selectedCharacter.favorite ? 'Remove from favorites' : 'Add to favorites'"
          @click="$emit('toggle-favorite', selectedCharacter)"
        >
          <span aria-hidden="true">{{ selectedCharacter.favorite ? '★' : '☆' }}</span>
        </button>
        <button
          v-if="selectedCharacter && !readOnly"
          :class="selectedCharacter.startedAt ? 'secondary-button' : 'primary-button'"
          type="button"
          :disabled="startSaving"
          @click="$emit('toggle-started')"
        >
          {{
            startSaving
              ? 'Saving...'
              : selectedCharacter.startedAt
                ? 'Stop reading'
                : 'Start reading'
          }}
        </button>
        <button
          v-if="selectedCharacter?.metronCharacterId && !readOnly"
          class="primary-button"
          type="button"
          :disabled="importRunning"
          @click="$emit('import-appearances')"
        >
          {{ importRunning ? 'Importing...' : 'Import from Metron' }}
        </button>
      </div>
    </header>

    <article class="detail-panel">
      <div v-if="selectedCharacter" class="read-only-detail">
        <header class="panel-header">
          <div>
            <p class="eyebrow">Character</p>
            <h3>{{ selectedCharacter.name }}</h3>
          </div>
        </header>

        <div v-if="selectedCharacter.image" class="character-portrait">
          <img
            :src="assetURL(selectedCharacter.image)"
            :alt="`${selectedCharacter.name} portrait`"
            loading="lazy"
          />
        </div>

        <div class="metadata-grid">
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
          <span v-if="selectedCharacter.startedAt"
            ><strong>Started</strong
            ><small>{{ new Date(selectedCharacter.startedAt).toLocaleDateString() }}</small></span
          >
        </div>
        <div class="progress-meter" aria-label="Character read progress">
          <span :style="{ width: characterProgress(selectedCharacter) }"></span>
        </div>

        <div v-if="selectedCharacter.aliases?.length" class="alias-list">
          <span v-for="alias in selectedCharacter.aliases" :key="alias">{{ alias }}</span>
        </div>

        <p class="detail-description">{{ selectedCharacter.description || 'No description' }}</p>

        <ComicListView
          class="preview-list"
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
      <p v-else class="empty-state">Select a character to view appearances.</p>
    </article>
  </div>
</template>
