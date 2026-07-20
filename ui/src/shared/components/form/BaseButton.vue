<script setup>
import { computed, ref } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  type: {
    type: String,
    default: 'button',
  },
  variant: {
    type: String,
    default: 'secondary',
    validator: (value) =>
      ['primary', 'secondary', 'neutral', 'danger', 'danger-ghost'].includes(value),
  },
  size: {
    type: String,
    default: 'default',
    validator: (value) => ['default', 'compact', 'icon'].includes(value),
  },
})

const button = ref(null)

defineExpose({
  focus: (options) => button.value?.focus(options),
})

const variantClasses = {
  primary: 'primary-button border-primary bg-primary text-white',
  secondary:
    'secondary-button text-control bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]',
  neutral:
    'secondary-action border-line-strong bg-surface text-control font-extrabold [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:bg-primary-soft focus-visible:border-primary focus-visible:bg-primary-soft',
  danger:
    'danger-button [border-color:color-mix(in_srgb,_var(--danger)_42%,_var(--line-strong))] bg-danger-soft text-danger',
  'danger-ghost':
    'danger-text-button border-danger-border bg-surface text-danger font-black [&:hover:not(:disabled)]:border-danger-border [&:hover:not(:disabled)]:bg-danger-soft focus-visible:border-danger-border focus-visible:bg-danger-soft',
}

const sizeClasses = {
  compact: 'min-h-9 py-2 px-2.5',
  icon: 'inline-flex size-11 min-h-10 items-center justify-center p-0',
}

const defaultSizeClasses = computed(() =>
  ['neutral', 'danger-ghost'].includes(props.variant)
    ? 'min-h-10 py-2 px-3'
    : 'min-h-10 py-2.5 px-3.5',
)

const classes = computed(() => [
  'base-button border rounded',
  variantClasses[props.variant],
  props.size === 'default' ? defaultSizeClasses.value : sizeClasses[props.size],
])
</script>

<template>
  <button ref="button" v-bind="$attrs" :type="type" :class="classes">
    <slot />
  </button>
</template>
