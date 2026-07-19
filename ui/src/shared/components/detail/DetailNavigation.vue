<script setup>
defineEmits(['back'])
</script>

<template>
  <header
    class="detail-nav sticky-toolbar justify-between flex-wrap down-tablet:items-center down-tablet:gap-2.5 down-mobile:grid down-mobile:grid-cols-1 down-mobile:items-stretch sticky [top:var(--sticky-toolbar-top)] z-20 [margin-inline:calc(var(--sticky-toolbar-inline-offset)_*_-1)] [padding:14px_var(--sticky-toolbar-inline-offset)] border-b border-sticky-border bg-sticky-bg shadow-sticky backdrop-blur-ui [&.sticky-toolbar]:[margin-top:calc(var(--content-padding)_*_-1)] max-w-none flex items-center gap-2.5 [&_>_button]:flex-none [&_>_*]:min-w-0 down-tablet:[&_>_button]:min-w-19.5 down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:pb-3 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none down-mobile:[&.sticky-toolbar]:[margin-top:calc(var(--content-padding)_*_-1)] down-mobile:[&_>_button]:justify-self-start"
  >
    <button
      class="secondary-button min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 bg-primary-soft [border-color:color-mix(in_srgb,_var(--primary)_42%,_var(--line-strong))]"
      type="button"
      @click="$emit('back')"
    >
      Back
    </button>
    <div
      class="detail-nav-actions down-tablet:[flex:1_1_220px] down-tablet:grid down-tablet:[grid-template-columns:repeat(auto-fit,_minmax(128px,_1fr))] down-tablet:gap-2.5 down-mobile:w-full down-mobile:grid-cols-2 flex items-center gap-2.5 flex-wrap justify-end ml-auto [&_>_button]:min-w-0 [&_>_button]:whitespace-normal [&_>_*]:min-w-0"
    >
      <slot />
    </div>
  </header>
</template>
