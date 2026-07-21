<script setup>
defineProps({
  importMode: {
    type: String,
    required: true,
  },
  selectedData: {
    type: Array,
    default: () => [],
  },
})

defineEmits(['update:import-mode', 'toggle-data'])

const DATA_OPTIONS = [
  { value: 'comics', label: 'Comics' },
  { value: 'series', label: 'Series' },
  { value: 'arcs', label: 'Arcs' },
  { value: 'characters', label: 'Characters' },
]
</script>

<template>
  <div class="metron-import-options">
    <div class="metron-modes compact" role="tablist" aria-label="Metron import depth">
      <!-- Native buttons: import depth is a stateful tablist. -->
      <button
        v-for="mode in ['quick', 'full']"
        :key="mode"
        type="button"
        class="mode-button"
        :class="{ active: importMode === mode }"
        role="tab"
        :aria-selected="importMode === mode"
        @click="$emit('update:import-mode', mode)"
      >
        {{ mode === 'quick' ? 'Quick' : 'Full' }}
      </button>
    </div>
    <fieldset v-if="importMode === 'full'" class="metron-data-options">
      <label v-for="option in DATA_OPTIONS" :key="option.value" class="inline-toggle">
        <input
          class="w-4 h-4 m-0"
          type="checkbox"
          :checked="selectedData.includes(option.value)"
          @change="$emit('toggle-data', option.value, $event.target.checked)"
        />
        {{ option.label }}
      </label>
    </fieldset>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.metron-import-options {
  @apply inline-flex items-center gap-3 justify-self-start w-fit max-w-full flex-wrap border border-line-strong rounded bg-surface-soft py-2.5 px-3 down-mobile:items-stretch down-mobile:flex-col;
}

.metron-modes.compact {
  @apply inline-grid grid-cols-2 gap-1.5 w-[min(260px,100%)] down-mobile:w-full;
}

.metron-data-options {
  @apply inline-flex items-center gap-2.5 flex-wrap min-w-0 m-0 p-0 border-0;
}

.inline-toggle {
  @apply inline-flex items-center gap-2 text-label text-sm font-bold whitespace-nowrap;
}

.mode-button {
  @apply min-h-8 rounded border border-line-strong bg-surface px-2.5 py-2 text-control;
}

.mode-button.active {
  @apply border-primary bg-primary text-white;
}
</style>
