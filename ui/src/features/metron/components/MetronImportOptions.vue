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
  <div
    class="metron-import-options inline-flex items-center gap-3 justify-self-start w-fit max-w-full flex-wrap border border-line-strong rounded bg-surface-soft py-2.5 px-3 down-mobile:items-stretch down-mobile:flex-col"
  >
    <div
      class="metron-modes compact inline-grid grid-cols-2 gap-1.5 w-[min(260px,100%)] down-mobile:w-full"
      role="tablist"
      aria-label="Metron import depth"
    >
      <!-- Native buttons: import depth is a stateful tablist. -->
      <button
        v-for="mode in ['quick', 'full']"
        :key="mode"
        type="button"
        class="min-h-8 border border-line-strong rounded bg-surface text-control py-2 px-2.5 [&.active]:border-primary [&.active]:bg-primary [&.active]:text-white"
        :class="{ active: importMode === mode }"
        role="tab"
        :aria-selected="importMode === mode"
        @click="$emit('update:import-mode', mode)"
      >
        {{ mode === 'quick' ? 'Quick' : 'Full' }}
      </button>
    </div>
    <fieldset
      v-if="importMode === 'full'"
      class="metron-data-options inline-flex items-center gap-2.5 flex-wrap min-w-0 m-0 p-0 border-0"
    >
      <label
        v-for="option in DATA_OPTIONS"
        :key="option.value"
        class="inline-toggle inline-flex items-center gap-2 text-label text-sm font-bold whitespace-nowrap"
      >
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
