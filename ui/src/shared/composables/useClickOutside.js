import { onBeforeUnmount, onMounted, unref } from 'vue'

export function isClickOutside(elements, event) {
  const eventPath = typeof event.composedPath === 'function' ? event.composedPath() : []
  return !elements.some((element) => eventPath.includes(element) || element.contains(event.target))
}

export function useClickOutside(targets, handler, enabled = true) {
  const targetList = Array.isArray(targets) ? targets : [targets]

  function handlePointerDown(event) {
    const active = typeof enabled === 'function' ? enabled() : unref(enabled)
    if (!active) return

    const elements = targetList.map(unref).filter(Boolean)
    if (isClickOutside(elements, event)) handler(event)
  }

  onMounted(() => document.addEventListener('pointerdown', handlePointerDown))
  onBeforeUnmount(() => document.removeEventListener('pointerdown', handlePointerDown))
}
