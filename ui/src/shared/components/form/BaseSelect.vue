<script setup>
import { computed, ref } from 'vue'

// Appearance belongs here; parents may only control width, margin, positioning, and placement.
// Add a variant or size instead of styling the rendered select through a parent selector.
defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: {
    type: [String, Number, Boolean],
    default: '',
  },
  modelModifiers: {
    type: Object,
    default: () => ({}),
  },
  size: {
    type: String,
    default: 'default',
    validator: (value) => ['default', 'compact', 'large', 'toolbar'].includes(value),
  },
  variant: {
    type: String,
    default: 'default',
    validator: (value) => ['default', 'nowrap', 'trailing'].includes(value),
  },
})

const emit = defineEmits(['update:modelValue'])
const select = ref(null)
const model = computed({
  get: () => props.modelValue,
  set: (selectedValue) => {
    let value = selectedValue
    if (props.modelModifiers.number) {
      const number = Number.parseFloat(value)
      if (!Number.isNaN(number)) value = number
    }
    emit('update:modelValue', value)
  },
})

defineExpose({
  focus: (options) => select.value?.focus(options),
})

const sizeClasses = {
  default: 'min-h-10 py-2.5 pl-3',
  compact: 'min-h-9 py-2 pl-2.5',
  large: 'h-11 min-h-11 py-0 pl-3 leading-tight',
  toolbar: 'h-11 min-h-11 py-2 pl-2.5',
}

const variantClasses = {
  default: 'pr-8',
  nowrap: 'pr-8 whitespace-nowrap',
  trailing: 'pr-8',
}
</script>

<template>
  <select
    ref="select"
    v-bind="$attrs"
    v-model="model"
    class="base-select w-full min-w-0 border border-line-strong rounded bg-surface text-ink cursor-pointer disabled:cursor-wait disabled:opacity-65 focus:outline-3 focus:outline-offset-2 focus:outline-focus"
    :class="[variantClasses[variant], sizeClasses[size]]"
  >
    <slot />
  </select>
</template>
