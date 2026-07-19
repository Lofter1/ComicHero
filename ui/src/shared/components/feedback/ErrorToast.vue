<script setup>
defineProps({
  message: {
    type: String,
    default: '',
  },
})

defineEmits(['dismiss'])
</script>

<template>
  <div
    v-if="message"
    class="toast error-toast fixed right-6 bottom-6 [z-index:50] flex items-start gap-3 [width:min(420px,_calc(100vw_-_32px))] border border-danger-border rounded bg-danger-soft text-danger shadow-monitor py-3 px-3.5 [&_span]:flex-auto [&_span]:min-w-0 [&_span]:break-anywhere [&_button]:flex-none [&_button]:min-h-0 [&_button]:border-0 [&_button]:bg-transparent [&_button]:text-inherit [&_button]:p-0 [&_button]:text-ui-sm [&_button]:font-extrabold"
    role="alert"
    aria-live="assertive"
  >
    <span>{{ message }}</span>
    <button type="button" aria-label="Dismiss error" @click="$emit('dismiss')">Dismiss</button>
  </div>
</template>
