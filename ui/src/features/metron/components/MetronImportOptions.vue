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
    <fieldset v-if="importMode === 'full'" class="metron-data-options">
      <label v-for="option in DATA_OPTIONS" :key="option.value" class="inline-toggle">
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
