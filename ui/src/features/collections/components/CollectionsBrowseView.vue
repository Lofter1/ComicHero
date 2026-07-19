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
  <section class="collections-view grid gap-5.5 pt-4.5 down-compact:pt-1.5">
    <header
      class="collection-page-intro flex items-center justify-between gap-6 pb-4.5 border-b border-line down-narrow:items-start down-compact:items-stretch down-compact:flex-col down-compact:gap-3.5 [&_>_div]:min-w-0 [&_.eyebrow]:mb-1.25 [&_h3]:m-0 [&_p:last-child]:mt-1.5 [&_p:last-child]:mx-0 [&_p:last-child]:mb-0 [&_p:last-child]:text-muted [&_p:last-child]:leading-ui [&_p:last-child]:max-w-170 [&_>_.primary-button]:flex-none [&_>_.primary-button]:whitespace-nowrap down-compact:[&_>_.primary-button]:w-full"
    >
      <div>
        <p>
          Group related characters and follow all of their appearances in one release-date order.
        </p>
      </div>
      <button
        v-if="!createOpen"
        class="primary-button icon-text-button min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 border-primary bg-primary text-white inline-flex items-center justify-center gap-2"
        type="button"
        @click="openCreate"
      >
        <span
          class="button-icon inline-flex items-center justify-center w-em h-em text-xl font-extrabold leading-none"
          aria-hidden="true"
          >+</span
        >
        New collection
      </button>
    </header>

    <form
      v-if="createOpen"
      class="collection-create-panel grid [grid-template-columns:minmax(190px,_0.55fr)_minmax(300px,_1fr)] items-end gap-4.5 [border:1px_solid_color-mix(in_srgb,_var(--primary)_35%,_var(--line))] rounded-xl [background:var(--primary-soft),_var(--panel-bg)] p-4 [box-shadow:0_10px_26px_var(--shadow-soft)] down-narrow:grid-cols-1"
      @submit.prevent="create"
    >
      <div
        class="collection-create-heading flex items-start justify-between gap-3 [&_strong]:block [&_small]:block [&_strong]:text-control [&_strong]:text-base [&_small]:mt-1 [&_small]:text-muted [&_small]:[line-height:1.35]"
      >
        <div>
          <strong>Create a collection</strong>
          <small>Give this character group a memorable name.</small>
        </div>
        <button
          v-if="collections.length"
          class="icon-button collection-create-close w-8.5 min-w-8.5 min-h-8.5 p-0 min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 self-end py-0 px-3 down-mobile:self-stretch down-mobile:w-full"
          type="button"
          aria-label="Cancel creating collection"
          @click="closeCreate"
        >
          ×
        </button>
      </div>
      <div
        class="collection-create-controls grid [grid-template-columns:minmax(0,_1fr)_max-content] gap-2.5 [&_input]:w-full [&_input]:min-w-0 [&_input]:min-h-10.5 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-control [&_input]:py-2.25 [&_input]:px-3 [&_input:focus]:outline-3 [&_input:focus]:outline-focus down-compact:grid-cols-1"
      >
        <label class="sr-only" for="collection-name">Collection name</label>
        <input id="collection-name" ref="nameInput" v-model="name" maxlength="120" />
        <button
          class="primary-button min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 border-primary bg-primary text-white"
          type="submit"
          :disabled="saving || !name.trim()"
        >
          {{ saving ? 'Creating...' : 'Create' }}
        </button>
      </div>
    </form>

    <section
      v-if="collections.length"
      class="collection-library grid gap-3.5"
      aria-labelledby="collection-list-title"
    >
      <header
        class="collection-library-heading flex items-end justify-between gap-3.5 [&_h3]:m-0 [&_p]:mt-1.5 [&_p]:mx-0 [&_p]:mb-0 [&_p]:text-muted [&_p]:leading-ui [&_p]:text-ui-sm-plus [&_p]:font-bold"
      >
        <div>
          <h3 id="collection-list-title">Your collections</h3>
          <p>{{ countLabel(collections.length, 'collection') }}</p>
        </div>
      </header>
      <div
        class="collection-grid grid [grid-template-columns:repeat(auto-fill,_minmax(min(100%,_330px),_410px))] gap-3.5 items-stretch down-compact:grid-cols-1"
      >
        <button
          v-for="collection in collections"
          :key="collection.id"
          class="collection-card relative grid [grid-template-columns:50px_minmax(0,_1fr)] gap-3.5 w-full min-h-36.5 overflow-hidden border border-line rounded-xl bg-surface text-control p-4 text-left shadow-soft down-compact:[grid-template-columns:44px_minmax(0,_1fr)] down-compact:min-h-34.5 down-compact:p-3.5 after:absolute after:inset-0 after:[border-radius:inherit] after:[box-shadow:inset_0_0_0_1px_transparent] after:[content:''] after:pointer-events-none hover:bg-surface-soft hover:[box-shadow:0_14px_32px_var(--shadow-panel)] hover:[transform:translateY(-2px)] focus-visible:bg-surface-soft focus-visible:[box-shadow:0_14px_32px_var(--shadow-panel)] focus-visible:[transform:translateY(-2px)] [&:hover:not(:disabled)]:border-line focus-visible:outline-3 focus-visible:outline-focus focus-visible:outline-offset-2 [&:hover_.collection-card-chevron]:[color:var(--surface-strong)] [&:hover_.collection-card-chevron]:[transform:translateX(3px)] [&:focus-visible_.collection-card-chevron]:[color:var(--surface-strong)] [&:focus-visible_.collection-card-chevron]:[transform:translateX(3px)] [&_.row-progress]:h-1.5 [&_.row-progress]:mt-1.5"
          type="button"
          @click="emit('open', collection)"
        >
          <span
            class="collection-card-monogram w-12.5 h-12.5 text-xl down-compact:w-11 down-compact:h-11 inline-flex items-center justify-center flex-none border border-line-strong rounded-xl bg-primary text-white font-black shadow-control"
            aria-hidden="true"
          >
            {{ collectionMonogram(collection) }}
          </span>
          <span class="collection-card-body grid content-start min-w-0">
            <span
              class="collection-card-title-row flex items-center justify-between gap-2.5 [&_>_strong]:overflow-hidden [&_>_strong]:text-base [&_>_strong]:text-ellipsis [&_>_strong]:whitespace-nowrap"
            >
              <strong>{{ collection.name }}</strong>
              <span
                class="collection-card-chevron text-muted [font-size:1.6rem] [font-weight:400] [line-height:0.8] [transition:transform_140ms_ease]"
                aria-hidden="true"
                >›</span
              >
            </span>
            <span
              class="collection-card-meta flex flex-wrap gapy-1 gapx-3 mt-1.5 text-muted text-compact font-bold [&_span_+_span::before]:mr-3 [&_span_+_span::before]:[color:var(--line-strong)] [&_span_+_span::before]:[content:'•']"
            >
              <span>{{ countLabel(collection.characterCount, 'character') }}</span>
              <span>{{ countLabel(collection.appearanceCount, 'appearance') }}</span>
            </span>
            <span
              class="collection-card-progress-copy mt-4.5 text-muted text-xs font-bold flex items-center justify-between gap-2.5 [&_strong]:text-control"
            >
              <span
                v-if="collection.startedAt"
                class="collection-card-status inline-flex items-center w-fit border border-line rounded-full bg-surface text-primary-strong py-0.75 px-1.75 text-xxs font-black [letter-spacing:0.04em] leading-none uppercase"
                >Reading</span
              >
              <span v-else>Not started</span>
              <strong>{{ formatProgress(collection.progress) }}</strong>
            </span>
            <span
              class="row-progress block flex-none w-full h-2 overflow-hidden rounded-full bg-read-progress [&_span]:block [&_span]:h-full [&_span]:min-w-0.5 [&_span]:[border-radius:inherit] [&_span]:bg-progress"
              :aria-label="`${collection.name} progress`"
            >
              <span :style="{ width: formatProgress(collection.progress) }"></span>
            </span>
          </span>
        </button>
      </div>
    </section>
    <section
      v-else
      class="collection-empty-state grid [justify-items:center] [max-width:560px] border border-dashed border-line-strong rounded-xl bg-surface-soft py-8.5 px-6 text-center [&_h3]:m-0 [&_p]:mt-1.5 [&_p]:mx-0 [&_p]:mb-0 [&_p]:text-muted [&_p]:leading-ui"
    >
      <span
        class="collection-empty-icon grid place-items-center w-13 h-13 mb-3 border border-line-strong rounded-2xl bg-surface text-primary-strong text-ui-display-sm"
        aria-hidden="true"
        >◇</span
      >
      <h3>Build your first character collection</h3>
      <p>Characters you add will become one combined, release-date reading queue.</p>
    </section>
  </section>
</template>
