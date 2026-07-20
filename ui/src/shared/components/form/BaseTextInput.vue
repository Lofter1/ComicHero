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

const sizeClasses = {
  default: 'min-h-10 py-2.5 px-3',
  compact: 'min-h-9 py-2 px-2.5',
  dense: 'min-h-10 py-2 px-2.5',
  fill: 'h-full min-h-0 py-2.5 px-3',
  large: 'h-11 min-h-11 py-0 px-3 leading-tight',
  toolbar: 'h-11 min-h-11 py-2 px-2.5',
}

const variantClasses = {
  default: 'border border-line-strong bg-surface text-ink',
  embedded: 'border-0 bg-transparent text-ink',
}
</script>

<template>
  <input
    ref="input"
    v-bind="$attrs"
    class="base-text-input w-full min-w-0 rounded disabled:cursor-wait disabled:opacity-65 focus:outline-3 focus:outline-offset-2 focus:outline-focus"
    :class="[variantClasses[variant], sizeClasses[size]]"
    :value="modelValue"
    @input="!modelModifiers.lazy && updateValue($event)"
    @change="modelModifiers.lazy && updateValue($event)"
  />
</template>
