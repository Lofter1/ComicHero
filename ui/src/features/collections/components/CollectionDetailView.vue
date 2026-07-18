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
  <div class="detail-view">
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
      <button class="danger-button" type="button" :disabled="saving" @click="$emit('delete')">
        {{ saving ? 'Saving...' : 'Delete collection' }}
      </button>
    </DetailNavigation>

    <article class="detail-panel">
      <div v-if="collection" class="read-only-detail">
        <header class="collection-detail-hero">
          <span class="collection-detail-monogram" aria-hidden="true">
            {{ monogram(collection.name) }}
          </span>
          <div class="collection-detail-copy min-w-0">
            <p class="eyebrow">Private character collection</p>
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
        <div class="progress-meter" aria-label="Collection read progress">
          <span :style="{ width: formatProgress(collection.progress) }"></span>
        </div>

        <section
          class="collection-members grid gap-3 mt-2.5 pt-5 [border-top:1px_solid_var(--line)]"
        >
          <header class="collection-members-header">
            <div>
              <p class="eyebrow">Members</p>
              <h3>Characters</h3>
            </div>
            <button
              class="secondary-button icon-text-button"
              type="button"
              :disabled="saving"
              @click="pickerOpen = true"
            >
              <span class="button-icon" aria-hidden="true">+</span>
              Add character
            </button>
          </header>
          <div v-if="collection.characters?.length" class="collection-member-list">
            <div v-for="character in collection.characters" :key="character.id" class="row">
              <button
                type="button"
                class="row-main collection-member-main grid [grid-template-columns:42px_minmax(0,_1fr)] items-center gap-2.5"
                @click="$emit('open-character', character)"
              >
                <span
                  class="collection-member-avatar grid place-items-center [width:42px] [height:42px] overflow-hidden border border-line-strong rounded-lg bg-surface-muted text-muted font-black"
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
                class="danger-text-button collection-member-remove [flex:0_0_auto]"
                type="button"
                :disabled="saving"
                @click="$emit('remove-character', character)"
              >
                Remove
              </button>
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
          class="preview-list"
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
