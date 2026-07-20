<script setup>
import { computed, ref } from 'vue'

// Appearance belongs here; parents may only control width, margin, positioning, and placement.
// Add a variant or size instead of styling the rendered button through a parent selector.
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
      [
        'primary',
        'primary-start',
        'primary-strong',
        'secondary',
        'secondary-start',
        'secondary-stacked',
        'neutral',
        'danger',
        'danger-ghost',
      ].includes(value),
  },
  size: {
    type: String,
    default: 'default',
    validator: (value) =>
      [
        'default',
        'compact',
        'compact-label',
        'compact-wide',
        'dense',
        'large',
        'single-line',
        'sidebar',
        'toolbar',
        'toolbar-nowrap',
        'icon',
      ].includes(value),
  },
})

const button = ref(null)

defineExpose({
  focus: (options) => button.value?.focus(options),
})

const variantClasses = {
  primary: 'primary-button border-primary bg-primary text-white',
  'primary-start': 'primary-button border-primary bg-primary text-white',
  'primary-strong': 'primary-button border-primary bg-primary text-white font-extrabold',
  secondary:
    'secondary-button text-control bg-primary-soft border-[color-mix(in_srgb,var(--primary)_42%,var(--line-strong))]',
  'secondary-start':
    'secondary-button text-control bg-primary-soft border-[color-mix(in_srgb,var(--primary)_42%,var(--line-strong))]',
  'secondary-stacked':
    'secondary-button text-control bg-primary-soft border-[color-mix(in_srgb,var(--primary)_42%,var(--line-strong))] [&_small]:text-muted [&_small]:text-xs',
  neutral:
    'secondary-action border-line-strong bg-surface text-control font-extrabold [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:bg-primary-soft focus-visible:border-primary focus-visible:bg-primary-soft',
  danger:
    'danger-button border-[color-mix(in_srgb,var(--danger)_42%,var(--line-strong))] bg-danger-soft text-danger',
  'danger-ghost':
    'danger-text-button border-danger-border bg-surface text-danger font-black [&:hover:not(:disabled)]:border-danger-border [&:hover:not(:disabled)]:bg-danger-soft focus-visible:border-danger-border focus-visible:bg-danger-soft',
}

const sizeClasses = {
  compact: 'min-h-9 py-2 px-2.5',
  'compact-label': 'min-h-9 py-2 px-2.5 text-xs whitespace-nowrap',
  'compact-wide': 'min-h-9 py-2 px-3 text-sm',
  dense: 'min-h-10 py-2 px-3.5',
  sidebar: 'min-h-10 py-2 px-3 down-tablet:py-0 down-tablet:px-4',
  toolbar: 'min-h-11 py-0 px-3.5',
  'toolbar-nowrap': 'min-h-11 py-0 px-3.5 whitespace-nowrap',
  icon: 'size-11 min-h-10 p-0',
}

const defaultPaddingClasses = computed(() =>
  ['neutral', 'danger-ghost'].includes(props.variant) ? 'py-2 px-3' : 'py-2.5 px-3.5',
)

const responsiveSizeClasses = computed(() =>
  props.size === 'large'
    ? `min-h-11 ${defaultPaddingClasses.value}`
    : `min-h-10 ${defaultPaddingClasses.value}${props.size === 'single-line' ? ' whitespace-nowrap' : ''}`,
)

const displayClasses = computed(() =>
  props.variant === 'secondary-stacked' ? 'grid gap-0.5' : 'inline-flex items-center gap-2',
)

const alignmentClasses = computed(() =>
  ['primary-start', 'secondary-start'].includes(props.variant) ? 'justify-start' : 'justify-center',
)

const classes = computed(() => [
  'base-button border rounded cursor-pointer transition-interactive duration-140 disabled:cursor-wait disabled:opacity-65 [&:hover:not(:disabled)]:-translate-y-px [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:shadow-interactive focus-visible:outline-3 focus-visible:outline-offset-2 focus-visible:outline-focus',
  displayClasses.value,
  alignmentClasses.value,
  variantClasses[props.variant],
  ['default', 'large', 'single-line'].includes(props.size)
    ? responsiveSizeClasses.value
    : sizeClasses[props.size],
])
</script>

<template>
  <button ref="button" v-bind="$attrs" :type="type" :class="classes">
    <slot />
  </button>
</template>
