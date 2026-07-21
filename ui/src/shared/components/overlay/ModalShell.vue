<script setup>
defineOptions({ inheritAttrs: false })

defineProps({
  size: {
    type: String,
    default: 'medium',
    validator: (value) => ['medium', 'large', 'wide'].includes(value),
  },
  structured: { type: Boolean, default: false },
})

defineEmits(['close'])
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <section
      v-bind="$attrs"
      class="modal-shell"
      :class="[`modal-shell--${size}`, { 'modal-shell--structured': structured }]"
      role="dialog"
      aria-modal="true"
    >
      <slot />
    </section>
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.modal-backdrop {
  @apply fixed inset-0 z-60 grid place-items-center bg-backdrop p-4;
}

.modal-shell {
  @apply max-h-[min(720px,calc(100dvh-28px))] w-[calc(100%-28px)] overflow-auto rounded-xl border border-line-strong bg-panel p-5 shadow-elevated;
}

.modal-shell--medium {
  @apply max-w-[620px];
}

.modal-shell--large {
  @apply max-w-[680px];
}

.modal-shell--wide {
  @apply max-w-[760px];
}

.modal-shell--structured {
  @apply grid max-h-[min(760px,calc(100vh-36px))] grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden rounded p-0 shadow-overlay;
}
</style>
