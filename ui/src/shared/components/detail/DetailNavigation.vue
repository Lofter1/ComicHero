<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useClickOutside } from '@/shared/composables/useClickOutside.js'

defineEmits(['back'])

const navigation = ref(null)
const backButton = ref(null)
const actionRoot = ref(null)
const actionList = ref(null)
const actionsCollapsed = ref(false)
const actionsOpen = ref(false)

let resizeObserver
let mutationObserver
let measureFrame
let naturalActionsWidth = 0

useClickOutside(actionRoot, () => (actionsOpen.value = false), actionsOpen)

function navigationContentWidth() {
  if (!navigation.value) return 0
  const style = window.getComputedStyle(navigation.value)
  return (
    navigation.value.clientWidth -
    (Number.parseFloat(style.paddingLeft) || 0) -
    (Number.parseFloat(style.paddingRight) || 0)
  )
}

function navigationGap() {
  if (!navigation.value) return 0
  const style = window.getComputedStyle(navigation.value)
  return Number.parseFloat(style.columnGap || style.gap) || 0
}

function measureActions() {
  measureFrame = undefined
  if (!navigation.value || !backButton.value || !actionList.value) return

  if (!actionList.value.children.length) {
    actionsCollapsed.value = false
    actionsOpen.value = false
    naturalActionsWidth = 0
    return
  }

  const availableWidth = navigationContentWidth()
  if (actionsCollapsed.value) {
    if (backButton.value.offsetWidth + navigationGap() + naturalActionsWidth <= availableWidth) {
      actionsCollapsed.value = false
      nextTick(scheduleMeasurement)
    }
    return
  }

  naturalActionsWidth = actionList.value.scrollWidth
  const shouldCollapse =
    backButton.value.offsetWidth + navigationGap() + naturalActionsWidth > availableWidth
  actionsCollapsed.value = shouldCollapse
  if (!shouldCollapse) actionsOpen.value = false
}

function scheduleMeasurement() {
  if (measureFrame !== undefined) window.cancelAnimationFrame(measureFrame)
  measureFrame = window.requestAnimationFrame(measureActions)
}

function remeasureNaturalActions() {
  if (actionsCollapsed.value) {
    actionsCollapsed.value = false
    actionsOpen.value = false
    nextTick(scheduleMeasurement)
    return
  }
  scheduleMeasurement()
}

function closeActionsAfterClick(event) {
  if (actionsCollapsed.value && event.target.closest('button')) actionsOpen.value = false
}

function handleEscape(event) {
  if (event.key === 'Escape') actionsOpen.value = false
}

onMounted(() => {
  resizeObserver = new ResizeObserver(scheduleMeasurement)
  resizeObserver.observe(navigation.value)

  mutationObserver = new MutationObserver(remeasureNaturalActions)
  mutationObserver.observe(actionList.value, {
    childList: true,
    characterData: true,
    subtree: true,
  })

  document.addEventListener('keydown', handleEscape)
  scheduleMeasurement()
  document.fonts?.ready.then(scheduleMeasurement)
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  mutationObserver?.disconnect()
  document.removeEventListener('keydown', handleEscape)
  if (measureFrame !== undefined) window.cancelAnimationFrame(measureFrame)
})
</script>

<template>
  <header ref="navigation" class="detail-nav sticky-toolbar">
    <!-- Native button: measurement logic requires direct access to this DOM element's width. -->
    <button ref="backButton" class="back-button" type="button" @click="$emit('back')">Back</button>
    <div ref="actionRoot" class="detail-nav-action-root relative ml-auto flex-none">
      <!-- Native button: this trigger is part of the measured overflow-navigation structure. -->
      <button
        v-if="actionsCollapsed"
        class="more-button"
        type="button"
        :aria-expanded="actionsOpen"
        aria-label="More actions"
        title="More actions"
        @click="actionsOpen = !actionsOpen"
      >
        <span aria-hidden="true" class="text-2xl font-extrabold leading-none">⋮</span>
      </button>
      <div
        v-show="!actionsCollapsed || actionsOpen"
        ref="actionList"
        class="detail-nav-actions"
        :class="{
          'detail-nav-actions--collapsed': actionsCollapsed,
          'detail-nav-actions--expanded': !actionsCollapsed,
        }"
        @click="closeActionsAfterClick"
      >
        <slot />
      </div>
    </div>
  </header>
</template>

<style scoped>
@reference '../../../styles.css';

.detail-nav.sticky-toolbar {
  @apply sticky top-(--sticky-toolbar-top) z-20 mx-[calc(var(--sticky-toolbar-inline-offset)*-1)] p-[14px_var(--sticky-toolbar-inline-offset)] border-b border-sticky-border bg-sticky-bg shadow-sticky backdrop-blur-ui [&.sticky-toolbar]:mt-[calc(var(--content-padding)*-1)] max-w-none flex flex-nowrap items-center justify-between gap-2.5 *:min-w-0 down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:pb-3 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none down-mobile:[&.sticky-toolbar]:mt-[calc(var(--content-padding)*-1)];
}

.back-button,
.more-button {
  @apply flex-none rounded border bg-primary-soft text-control;
  border-color: color-mix(in srgb, var(--primary) 42%, var(--line-strong));
}

.back-button {
  @apply min-h-10 px-3.5 py-2.5;
}

.more-button {
  @apply inline-flex size-11 min-h-10 items-center justify-center p-0;
}

.detail-nav-actions {
  @apply gap-2.5;
}

.detail-nav-actions :deep(> *) {
  @apply min-w-0;
}

.detail-nav-actions--expanded {
  @apply flex flex-nowrap items-center justify-end;
}

.detail-nav-actions--collapsed {
  @apply absolute top-[calc(100%+8px)] right-0 z-26 grid w-max min-w-[210px] max-w-[calc(100vw-24px)] items-stretch rounded-lg border border-line-strong bg-surface p-2.5 shadow-monitor;
}

.detail-nav-actions--collapsed :deep(> button) {
  @apply w-full;
}
</style>
