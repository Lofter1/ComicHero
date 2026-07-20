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
    class="toast error-toast fixed right-6 bottom-6 [z-index:50] flex items-start gap-3 [width:min(420px,_calc(100vw_-_32px))] border border-danger-border rounded bg-danger-soft text-danger shadow-monitor py-3 px-3.5 [&_span]:flex-auto [&_span]:min-w-0 [&_span]:break-anywhere"
    role="alert"
    aria-live="assertive"
  >
    <span>{{ message }}</span>
    <!-- Native button: toast dismissal inherits the toast's compact contextual styling. -->
    <button
      class="flex-none min-h-0 border-0 bg-transparent text-inherit p-0 text-sm font-extrabold"
      type="button"
      aria-label="Dismiss error"
      @click="$emit('dismiss')"
    >
      Dismiss
    </button>
  </div>
</template>
