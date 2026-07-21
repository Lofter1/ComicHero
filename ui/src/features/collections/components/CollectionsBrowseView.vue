<script setup>
import { nextTick, ref } from 'vue'
import { formatProgress } from '@/features/reading-orders/model.js'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import ProgressBar from '@/shared/components/feedback/ProgressBar.vue'

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
  <section class="collections-view grid gap-6 pt-4 down-compact:pt-1.5">
    <header class="collection-page-intro">
      <div>
        <p>
          Group related characters and follow all of their appearances in one release-date order.
        </p>
      </div>
      <BaseButton
        v-if="!createOpen"
        class="flex-none down-compact:w-full"
        variant="primary"
        size="single-line"
        @click="openCreate"
      >
        <span class="button-icon" aria-hidden="true">+</span>
        New collection
      </BaseButton>
    </header>

    <form v-if="createOpen" class="collection-create-panel" @submit.prevent="create">
      <div class="collection-create-heading">
        <div>
          <strong>Create a collection</strong>
          <small>Give this character group a memorable name.</small>
        </div>
        <!-- Native button: this compact close control has editor-panel-specific dimensions. -->
        <button
          v-if="collections.length"
          class="icon-button collection-create-close"
          type="button"
          aria-label="Cancel creating collection"
          @click="closeCreate"
        >
          ×
        </button>
      </div>
      <div class="collection-create-controls">
        <label class="sr-only" for="collection-name">Collection name</label>
        <BaseTextInput id="collection-name" ref="nameInput" v-model="name" maxlength="120" />
        <BaseButton variant="primary" type="submit" :disabled="saving || !name.trim()">
          {{ saving ? 'Creating...' : 'Create' }}
        </BaseButton>
      </div>
    </form>

    <section
      v-if="collections.length"
      class="collection-library grid gap-3.5"
      aria-labelledby="collection-list-title"
    >
      <header class="collection-library-heading">
        <div>
          <h3 id="collection-list-title">Your collections</h3>
          <p>{{ countLabel(collections.length, 'collection') }}</p>
        </div>
      </header>
      <div class="collection-grid">
        <!-- Native buttons: collections are full-card navigation targets. -->
        <button
          v-for="collection in collections"
          :key="collection.id"
          class="collection-card"
          type="button"
          @click="emit('open', collection)"
        >
          <span class="collection-card-monogram" aria-hidden="true">
            {{ collectionMonogram(collection) }}
          </span>
          <span class="collection-card-body grid content-start min-w-0">
            <span class="collection-card-title-row">
              <strong>{{ collection.name }}</strong>
              <span class="collection-card-chevron" aria-hidden="true">›</span>
            </span>
            <span class="collection-card-meta">
              <span>{{ countLabel(collection.characterCount, 'character') }}</span>
              <span>{{ countLabel(collection.appearanceCount, 'appearance') }}</span>
            </span>
            <span class="collection-card-progress-copy">
              <span v-if="collection.startedAt" class="collection-card-status">Reading</span>
              <span v-else>Not started</span>
              <strong>{{ formatProgress(collection.progress) }}</strong>
            </span>
            <ProgressBar
              tag="span"
              compact
              :value="formatProgress(collection.progress)"
              :label="`${collection.name} progress`"
            />
          </span>
        </button>
      </div>
    </section>
    <section v-else class="collection-empty-state">
      <span class="collection-empty-icon" aria-hidden="true">◇</span>
      <h3>Build your first character collection</h3>
      <p>Characters you add will become one combined, release-date reading queue.</p>
    </section>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.collection-page-intro {
  @apply flex items-center justify-between gap-6 pb-4 border-b border-line down-narrow:items-start down-compact:items-stretch down-compact:flex-col down-compact:gap-3.5 [&_>_div]:min-w-0 [&_.eyebrow]:mb-1 [&_h3]:m-0 [&_p:last-child]:mt-1.5 [&_p:last-child]:mx-0 [&_p:last-child]:mb-0 [&_p:last-child]:text-muted [&_p:last-child]:leading-ui [&_p:last-child]:max-w-intro;
}

.button-icon {
  @apply inline-flex items-center justify-center size-5 text-xl font-extrabold leading-none;
}

.collection-create-panel {
  @apply grid grid-cols-[minmax(190px,0.55fr)_minmax(300px,1fr)] items-end gap-4 [border:1px_solid_color-mix(in_srgb,var(--primary)_35%,var(--line))] rounded-xl [background:var(--primary-soft),var(--panel-bg)] p-4 [box-shadow:0_10px_26px_var(--shadow-soft)] down-narrow:grid-cols-1;
}

.collection-create-heading {
  @apply flex items-start justify-between gap-3 [&_strong]:block [&_small]:block [&_strong]:text-control [&_strong]:text-base [&_small]:mt-1 [&_small]:text-muted [&_small]:leading-[1.35];
}

.icon-button.collection-create-close {
  @apply w-8 min-w-8 min-h-8 p-0 border border-line-strong rounded bg-surface text-control self-end py-0 px-3 down-mobile:self-stretch down-mobile:w-full;
}

.collection-create-controls {
  @apply grid grid-cols-[minmax(0,1fr)_max-content] gap-2.5 down-compact:grid-cols-1;
}

.collection-library-heading {
  @apply flex items-end justify-between gap-3.5 [&_h3]:m-0 [&_p]:mt-1.5 [&_p]:mx-0 [&_p]:mb-0 [&_p]:text-muted [&_p]:leading-ui [&_p]:text-sm [&_p]:font-bold;
}

.collection-grid {
  @apply grid grid-cols-[repeat(auto-fill,minmax(min(100%,330px),410px))] gap-3.5 items-stretch down-compact:grid-cols-1;
}

.collection-card {
  @apply relative grid grid-cols-[50px_minmax(0,1fr)] gap-3.5 w-full min-h-36 overflow-hidden border border-line rounded-xl bg-surface text-control p-4 text-left shadow-soft down-compact:grid-cols-[44px_minmax(0,1fr)] down-compact:min-h-36 down-compact:p-3.5 after:absolute after:inset-0 after:rounded-[inherit] after:[box-shadow:inset_0_0_0_1px_transparent] after:[content:''] after:pointer-events-none hover:bg-surface-soft hover:[box-shadow:0_14px_32px_var(--shadow-panel)] hover:transform-[translateY(-2px)] focus-visible:bg-surface-soft focus-visible:[box-shadow:0_14px_32px_var(--shadow-panel)] focus-visible:transform-[translateY(-2px)] [&:hover:not(:disabled)]:border-line focus-visible:outline-3 focus-visible:outline-focus focus-visible:outline-offset-2 [&:hover_.collection-card-chevron]:text-(--surface-strong) [&:hover_.collection-card-chevron]:transform-[translateX(3px)] [&:focus-visible_.collection-card-chevron]:text-(--surface-strong) [&:focus-visible_.collection-card-chevron]:transform-[translateX(3px)] [&_.row-progress]:h-1.5 [&_.row-progress]:mt-1.5;
}

.collection-card-monogram {
  @apply w-12 h-12 text-xl down-compact:w-11 down-compact:h-11 inline-flex items-center justify-center flex-none border border-line-strong rounded-xl bg-primary text-white font-black shadow-control;
}

.collection-card-title-row {
  @apply flex items-center justify-between gap-2.5 [&_>_strong]:overflow-hidden [&_>_strong]:text-base [&_>_strong]:text-ellipsis [&_>_strong]:whitespace-nowrap;
}

.collection-card-chevron {
  @apply text-muted text-2xl font-normal leading-[0.8] [transition:transform_140ms_ease];
}

.collection-card-meta {
  @apply flex flex-wrap gap-y-1 gap-x-3 mt-1.5 text-muted text-xs font-bold [&_span+span::before]:mr-3 [&_span+span::before]:text-(--line-strong) [&_span+span::before]:[content:'•'];
}

.collection-card-progress-copy {
  @apply mt-4 text-muted text-xs font-bold flex items-center justify-between gap-2.5 [&_strong]:text-control;
}

.collection-card-status {
  @apply inline-flex items-center w-fit border border-line rounded-full bg-surface text-primary-strong py-1 px-2 text-xs font-black tracking-[0.04em] leading-none uppercase;
}

.collection-empty-state {
  @apply grid justify-items-center max-w-[560px] border border-dashed border-line-strong rounded-xl bg-surface-soft py-8 px-6 text-center [&_h3]:m-0 [&_p]:mt-1.5 [&_p]:mx-0 [&_p]:mb-0 [&_p]:text-muted [&_p]:leading-ui;
}

.collection-empty-icon {
  @apply grid place-items-center w-12 h-12 mb-3 border border-line-strong rounded-2xl bg-surface text-primary-strong text-3xl;
}
</style>
