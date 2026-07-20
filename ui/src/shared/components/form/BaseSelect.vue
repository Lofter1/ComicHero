<script setup>
import { computed, ref } from 'vue'

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
    class="base-select w-full min-w-0 min-h-10 border border-line-strong rounded bg-surface text-ink py-2.5 px-3"
  >
    <slot />
  </select>
</template>
