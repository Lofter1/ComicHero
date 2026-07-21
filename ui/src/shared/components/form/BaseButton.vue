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

const classes = computed(() => [
  'base-button',
  `base-button--${props.variant}`,
  `base-button--${props.size}`,
])
</script>

<template>
  <button ref="button" v-bind="$attrs" :type="type" :class="classes">
    <slot />
  </button>
</template>

<style scoped>
@reference '../../../styles.css';

.base-button {
  @apply inline-flex cursor-pointer items-center justify-center gap-2 rounded border px-3.5 py-2.5 transition-interactive duration-140;
  @apply disabled:cursor-wait disabled:opacity-65;
  @apply hover:not-disabled:-translate-y-px hover:not-disabled:border-primary hover:not-disabled:shadow-interactive;
  @apply focus-visible:outline-3 focus-visible:outline-offset-2 focus-visible:outline-focus;
}

.base-button--primary,
.base-button--primary-start,
.base-button--primary-strong {
  @apply border-primary bg-primary text-white;
}

.base-button--primary-strong {
  @apply font-extrabold;
}

.base-button--secondary,
.base-button--secondary-start,
.base-button--secondary-stacked {
  @apply bg-primary-soft text-control;
  border-color: color-mix(in srgb, var(--primary) 42%, var(--line-strong));
}

.base-button--secondary-stacked {
  @apply grid gap-0.5;
}

.base-button--secondary-stacked :deep(small) {
  @apply text-xs text-muted;
}

.base-button--primary-start,
.base-button--secondary-start {
  @apply justify-start;
}

.base-button--neutral {
  @apply border-line-strong bg-surface px-3 py-2 font-extrabold text-control;
  @apply hover:not-disabled:bg-primary-soft focus-visible:border-primary focus-visible:bg-primary-soft;
}

.base-button--danger {
  @apply bg-danger-soft text-danger;
  border-color: color-mix(in srgb, var(--danger) 42%, var(--line-strong));
}

.base-button--danger-ghost {
  @apply border-danger-border bg-surface px-3 py-2 font-black text-danger;
  @apply hover:not-disabled:border-danger-border hover:not-disabled:bg-danger-soft;
  @apply focus-visible:border-danger-border focus-visible:bg-danger-soft;
}

.base-button--default,
.base-button--single-line {
  @apply min-h-10;
}

.base-button--single-line,
.base-button--toolbar-nowrap,
.base-button--compact-label {
  @apply whitespace-nowrap;
}

.base-button--large {
  @apply min-h-11;
}

.base-button--compact,
.base-button--compact-label {
  @apply min-h-9 px-2.5 py-2;
}

.base-button--compact-label {
  @apply text-xs;
}

.base-button--compact-wide {
  @apply min-h-9 px-3 py-2 text-sm;
}

.base-button--dense {
  @apply min-h-10 px-3.5 py-2;
}

.base-button--sidebar {
  @apply min-h-10 px-3 py-2;
}

.base-button--toolbar,
.base-button--toolbar-nowrap {
  @apply min-h-11 px-3.5 py-0;
}

.base-button--icon {
  @apply size-11 min-h-10 p-0;
}

@media (width <= 960px) {
  .base-button--sidebar {
    @apply px-4 py-0;
  }
}
</style>
