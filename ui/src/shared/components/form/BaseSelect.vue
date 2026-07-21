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
</script>

<template>
  <select
    ref="select"
    v-bind="$attrs"
    v-model="model"
    class="base-select"
    :class="[`base-select--${variant}`, `base-select--${size}`]"
  >
    <slot />
  </select>
</template>

<style scoped>
@reference '../../../styles.css';

.base-select {
  @apply w-full min-w-0 cursor-pointer rounded border border-line-strong bg-surface pr-8 text-ink;
  @apply disabled:cursor-wait disabled:opacity-65;
  @apply focus:outline-3 focus:outline-offset-2 focus:outline-focus;
}

.base-select--nowrap {
  @apply whitespace-nowrap;
}

.base-select--default {
  @apply min-h-10 py-2.5 pl-3;
}

.base-select--compact {
  @apply min-h-9 py-2 pl-2.5;
}

.base-select--large {
  @apply h-11 min-h-11 py-0 pl-3 leading-tight;
}

.base-select--toolbar {
  @apply h-11 min-h-11 py-2 pl-2.5;
}
</style>
