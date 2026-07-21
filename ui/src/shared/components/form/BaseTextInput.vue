<script setup>
import { ref } from 'vue'

// Appearance belongs here; parents may only control width, margin, positioning, and placement.
// Add a variant or size instead of styling the rendered input through a parent selector.
defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
  modelModifiers: {
    type: Object,
    default: () => ({}),
  },
  size: {
    type: String,
    default: 'default',
    validator: (value) =>
      ['default', 'compact', 'dense', 'fill', 'large', 'toolbar'].includes(value),
  },
  variant: {
    type: String,
    default: 'default',
    validator: (value) => ['default', 'embedded'].includes(value),
  },
})

const emit = defineEmits(['update:modelValue'])
const input = ref(null)

defineExpose({
  focus: (options) => input.value?.focus(options),
  select: () => input.value?.select(),
})

function normalizedValue(value) {
  if (props.modelModifiers.trim) value = value.trim()
  if (props.modelModifiers.number) {
    const number = Number.parseFloat(value)
    if (!Number.isNaN(number)) value = number
  }
  return value
}

function updateValue(event) {
  emit('update:modelValue', normalizedValue(event.target.value))
}
</script>

<template>
  <input
    ref="input"
    v-bind="$attrs"
    class="base-text-input"
    :class="[`base-text-input--variant-${variant}`, `base-text-input--size-${size}`]"
    :value="modelValue"
    @input="!modelModifiers.lazy && updateValue($event)"
    @change="modelModifiers.lazy && updateValue($event)"
  />
</template>

<style scoped>
@reference '../../../styles.css';

.base-text-input {
  @apply w-full min-w-0 rounded text-ink disabled:cursor-wait disabled:opacity-65;
  @apply focus:outline-3 focus:outline-offset-2 focus:outline-focus;
}

.base-text-input--variant-default {
  @apply border border-line-strong bg-surface;
}

.base-text-input--variant-embedded {
  @apply border-0 bg-transparent;
}

.base-text-input--size-default {
  @apply min-h-10 px-3 py-2.5;
}

.base-text-input--size-compact {
  @apply min-h-9 px-2.5 py-2;
}

.base-text-input--size-dense {
  @apply min-h-10 px-2.5 py-2;
}

.base-text-input--size-fill {
  @apply h-full min-h-0 px-3 py-2.5;
}

.base-text-input--size-large {
  @apply h-11 min-h-11 px-3 py-0 leading-tight;
}

.base-text-input--size-toolbar {
  @apply h-11 min-h-11 px-2.5 py-2;
}
</style>
