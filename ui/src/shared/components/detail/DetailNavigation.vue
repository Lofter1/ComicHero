<script setup>
defineEmits(['back'])
</script>

<template>
  <header class="detail-nav sticky-toolbar">
    <button class="secondary-button" type="button" @click="$emit('back')">Back</button>
    <div class="detail-nav-actions">
      <slot />
    </div>
  </header>
</template>
