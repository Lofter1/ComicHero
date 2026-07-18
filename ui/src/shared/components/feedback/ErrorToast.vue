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
    class="toast error-toast fixed [right:24px] [bottom:24px] [z-index:50] flex [align-items:flex-start] gap-3 [width:min(420px,_calc(100vw_-_32px))] border border-danger-border rounded bg-danger-soft text-danger [box-shadow:0_18px_42px_var(--shadow-panel)] [padding:12px_14px]"
    role="alert"
    aria-live="assertive"
  >
    <span>{{ message }}</span>
    <button type="button" aria-label="Dismiss error" @click="$emit('dismiss')">Dismiss</button>
  </div>
</template>
