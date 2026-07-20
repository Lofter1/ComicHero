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
  <header
    ref="navigation"
    class="detail-nav sticky-toolbar sticky top-(--sticky-toolbar-top) z-20 mx-[calc(var(--sticky-toolbar-inline-offset)*-1)] p-[14px_var(--sticky-toolbar-inline-offset)] border-b border-sticky-border bg-sticky-bg shadow-sticky backdrop-blur-ui [&.sticky-toolbar]:mt-[calc(var(--content-padding)*-1)] max-w-none flex flex-nowrap items-center justify-between gap-2.5 *:min-w-0 down-mobile:static down-mobile:mx-0 down-mobile:pt-0 down-mobile:px-0 down-mobile:pb-3 down-mobile:border-b down-mobile:border-line down-mobile:bg-transparent down-mobile:shadow-none down-mobile:backdrop-filter-none down-mobile:[&.sticky-toolbar]:mt-[calc(var(--content-padding)*-1)]"
  >
    <!-- Native button: measurement logic requires direct access to this DOM element's width. -->
    <button
      ref="backButton"
      class="secondary-button flex-none min-h-10 border rounded text-control py-2.5 px-3.5 bg-primary-soft border-[color-mix(in_srgb,var(--primary)_42%,var(--line-strong))]"
      type="button"
      @click="$emit('back')"
    >
      Back
    </button>
    <div ref="actionRoot" class="detail-nav-action-root relative ml-auto flex-none">
      <!-- Native button: this trigger is part of the measured overflow-navigation structure. -->
      <button
        v-if="actionsCollapsed"
        class="secondary-button inline-flex size-11 min-h-10 items-center justify-center border rounded bg-primary-soft text-control p-0 border-[color-mix(in_srgb,var(--primary)_42%,var(--line-strong))]"
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
        class="detail-nav-actions gap-2.5 [&_>_button]:min-w-0 *:min-w-0"
        :class="
          actionsCollapsed
            ? 'absolute z-26 top-[calc(100%+8px)] right-0 grid items-stretch w-max min-w-[210px] max-w-[calc(100vw-24px)] border border-line-strong rounded-lg bg-surface p-2.5 [box-shadow:0_18px_40px_var(--shadow-panel)] [&_>_button]:w-full'
            : 'flex flex-nowrap items-center justify-end'
        "
        @click="closeActionsAfterClick"
      >
        <slot />
      </div>
    </div>
  </header>
</template>
