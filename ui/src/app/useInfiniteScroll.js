import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

export function useInfiniteScroll({ activeView, enabled, onLoadMore }) {
  const sentinel = ref(null)
  let observer = null

  function observeSentinel() {
    if (!observer) return
    observer.disconnect()
    if (sentinel.value && enabled.value) observer.observe(sentinel.value.element)
  }

  onMounted(() => {
    if (typeof IntersectionObserver === 'undefined') return
    observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) onLoadMore()
      },
      { rootMargin: '360px 0px' },
    )
    observeSentinel()
  })

  watch(enabled, () => nextTick(observeSentinel))
  watch(activeView, () => nextTick(observeSentinel))

  onUnmounted(() => observer?.disconnect())

  return { sentinel }
}
