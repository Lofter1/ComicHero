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
  <div v-if="message" class="toast error-toast" role="alert" aria-live="assertive">
    <span>{{ message }}</span>
    <button type="button" aria-label="Dismiss error" @click="$emit('dismiss')">Dismiss</button>
  </div>
</template>
