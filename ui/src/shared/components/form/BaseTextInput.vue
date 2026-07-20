<script setup>
import { ref } from 'vue'

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
    class="base-text-input w-full min-w-0 min-h-10 border border-line-strong rounded bg-surface text-ink py-2.5 px-3"
    :value="modelValue"
    @input="!modelModifiers.lazy && updateValue($event)"
    @change="modelModifiers.lazy && updateValue($event)"
  />
</template>
