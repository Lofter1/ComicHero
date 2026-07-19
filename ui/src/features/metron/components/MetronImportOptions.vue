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
      class="metron-modes compact inline-grid grid-cols-5 gap-1.5 [width:min(680px,_100%)] [&_button]:min-h-10.5 [&_button]:border [&_button]:border-line-strong [&_button]:rounded [&_button]:bg-surface [&_button]:text-control [&_button]:py-2.5 [&_button]:px-3 [&_button.active]:border-primary [&_button.active]:bg-primary [&_button.active]:text-white [&.compact]:grid-cols-2 [&.compact]:[width:min(260px,_100%)] [&.compact_button]:min-h-8.5 [&.compact_button]:py-1.75 [&.compact_button]:px-2.5 down-mobile:grid-cols-2 down-mobile:w-full"
      role="tablist"
      aria-label="Metron import depth"
    >
      <button
        v-for="mode in ['quick', 'full']"
        :key="mode"
        type="button"
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
        class="inline-toggle inline-flex items-center gap-1.75 text-label text-ui-md font-bold whitespace-nowrap [&_input]:w-4 [&_input]:h-4 [&_input]:m-0"
      >
        <input
          type="checkbox"
          :checked="selectedData.includes(option.value)"
          @change="$emit('toggle-data', option.value, $event.target.checked)"
        />
        {{ option.label }}
      </label>
    </fieldset>
  </div>
</template>
