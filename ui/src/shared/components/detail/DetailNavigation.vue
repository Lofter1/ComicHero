<script setup>
defineEmits(['back'])
</script>

<template>
  <header
    class="detail-nav sticky-toolbar justify-between flex-wrap down-tablet:items-center down-tablet:gap-2.5 down-mobile:grid down-mobile:[grid-template-columns:1fr] down-mobile:[align-items:stretch]"
  >
    <button class="secondary-button" type="button" @click="$emit('back')">Back</button>
    <div
      class="detail-nav-actions down-tablet:[flex:1_1_220px] down-tablet:grid down-tablet:[grid-template-columns:repeat(auto-fit,_minmax(128px,_1fr))] down-tablet:gap-2.5 down-mobile:w-full down-mobile:[grid-template-columns:repeat(2,_minmax(0,_1fr))]"
    >
      <slot />
    </div>
  </header>
</template>
