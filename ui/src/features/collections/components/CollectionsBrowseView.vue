<script setup>
import { nextTick, ref } from 'vue'
import { formatProgress } from '@/features/reading-orders/model.js'

const props = defineProps({
  collections: { type: Array, default: () => [] },
  saving: { type: Boolean, default: false },
})
const emit = defineEmits(['create', 'open'])
const name = ref('')
const nameInput = ref(null)
const createOpen = ref(!props.collections.length)

function create() {
  const value = name.value.trim()
  if (!value) return
  emit('create', value)
}

async function openCreate() {
  createOpen.value = true
  await nextTick()
  nameInput.value?.focus()
}

function closeCreate() {
  if (props.saving) return
  createOpen.value = false
  name.value = ''
}

function collectionMonogram(collection) {
  return (collection?.name || '?').trim().slice(0, 1).toUpperCase() || '?'
}

function countLabel(count, singular) {
  return `${count} ${count === 1 ? singular : `${singular}s`}`
}
</script>

<template>
  <section class="collections-view grid [gap:22px] pt-4.5 down-compact:pt-1.5">
    <header
      class="collection-page-intro flex items-center justify-between gap-6 pb-4.5 [border-bottom:1px_solid_var(--line)] down-narrow:[align-items:flex-start] down-compact:[align-items:stretch] down-compact:flex-col down-compact:gap-3.5"
    >
      <div>
        <p>
          Group related characters and follow all of their appearances in one release-date order.
        </p>
      </div>
      <button
        v-if="!createOpen"
        class="primary-button icon-text-button"
        type="button"
        @click="openCreate"
      >
        <span class="button-icon" aria-hidden="true">+</span>
        New collection
      </button>
    </header>

    <form
      v-if="createOpen"
      class="collection-create-panel grid [grid-template-columns:minmax(190px,_0.55fr)_minmax(300px,_1fr)] items-end gap-4.5 [border:1px_solid_color-mix(in_srgb,_var(--primary)_35%,_var(--line))] rounded-xl [background:var(--primary-soft),_var(--panel-bg)] p-4 [box-shadow:0_10px_26px_var(--shadow-soft)] down-narrow:[grid-template-columns:1fr]"
      @submit.prevent="create"
    >
      <div class="collection-create-heading flex [align-items:flex-start] justify-between gap-3">
        <div>
          <strong>Create a collection</strong>
          <small>Give this character group a memorable name.</small>
        </div>
        <button
          v-if="collections.length"
          class="icon-button collection-create-close [width:34px] [min-width:34px] [min-height:34px] [padding:0]"
          type="button"
          aria-label="Cancel creating collection"
          @click="closeCreate"
        >
          ×
        </button>
      </div>
      <div class="collection-create-controls">
        <label class="sr-only" for="collection-name">Collection name</label>
        <input id="collection-name" ref="nameInput" v-model="name" maxlength="120" />
        <button class="primary-button" type="submit" :disabled="saving || !name.trim()">
          {{ saving ? 'Creating...' : 'Create' }}
        </button>
      </div>
    </form>

    <section
      v-if="collections.length"
      class="collection-library grid gap-3.5"
      aria-labelledby="collection-list-title"
    >
      <header class="collection-library-heading flex items-end justify-between gap-3.5">
        <div>
          <h3 id="collection-list-title">Your collections</h3>
          <p>{{ countLabel(collections.length, 'collection') }}</p>
        </div>
      </header>
      <div
        class="collection-grid grid [grid-template-columns:repeat(auto-fill,_minmax(min(100%,_330px),_410px))] gap-3.5 [align-items:stretch] down-compact:[grid-template-columns:1fr]"
      >
        <button
          v-for="collection in collections"
          :key="collection.id"
          class="collection-card relative grid [grid-template-columns:50px_minmax(0,_1fr)] gap-3.5 w-full [min-height:146px] overflow-hidden border border-line rounded-xl bg-surface text-control p-4 text-left shadow-soft down-compact:[grid-template-columns:44px_minmax(0,_1fr)] down-compact:[min-height:138px] down-compact:p-3.5"
          type="button"
          @click="emit('open', collection)"
        >
          <span
            class="collection-card-monogram [width:50px] [height:50px] [font-size:1.25rem] down-compact:[width:44px] down-compact:[height:44px]"
            aria-hidden="true"
          >
            {{ collectionMonogram(collection) }}
          </span>
          <span class="collection-card-body grid content-start min-w-0">
            <span class="collection-card-title-row">
              <strong>{{ collection.name }}</strong>
              <span
                class="collection-card-chevron text-muted [font-size:1.6rem] [font-weight:400] [line-height:0.8] [transition:transform_140ms_ease]"
                aria-hidden="true"
                >›</span
              >
            </span>
            <span
              class="collection-card-meta flex flex-wrap [gap:4px_12px] mt-1.5 text-muted [font-size:0.8rem] font-bold"
            >
              <span>{{ countLabel(collection.characterCount, 'character') }}</span>
              <span>{{ countLabel(collection.appearanceCount, 'appearance') }}</span>
            </span>
            <span
              class="collection-card-progress-copy mt-4.5 text-muted [font-size:0.75rem] font-bold"
            >
              <span
                v-if="collection.startedAt"
                class="collection-card-status inline-flex items-center [width:fit-content] border border-line rounded-full bg-surface text-primary-strong [padding:3px_7px] [font-size:0.68rem] font-black [letter-spacing:0.04em] leading-none uppercase"
                >Reading</span
              >
              <span v-else>Not started</span>
              <strong>{{ formatProgress(collection.progress) }}</strong>
            </span>
            <span class="row-progress" :aria-label="`${collection.name} progress`">
              <span :style="{ width: formatProgress(collection.progress) }"></span>
            </span>
          </span>
        </button>
      </div>
    </section>
    <section
      v-else
      class="collection-empty-state grid [justify-items:center] [max-width:560px] [border:1px_dashed_var(--line-strong)] rounded-xl bg-surface-soft [padding:34px_24px] text-center"
    >
      <span
        class="collection-empty-icon grid place-items-center w-13 h-13 mb-3 border border-line-strong rounded-2xl bg-surface text-primary-strong [font-size:1.8rem]"
        aria-hidden="true"
        >◇</span
      >
      <h3>Build your first character collection</h3>
      <p>Characters you add will become one combined, release-date reading queue.</p>
    </section>
  </section>
</template>
