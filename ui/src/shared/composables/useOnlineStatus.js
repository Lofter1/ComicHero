import { onBeforeUnmount, onMounted, ref } from 'vue'

export function useOnlineStatus() {
  const isOnline = ref(navigator.onLine)

  function handleOnline() {
    isOnline.value = true
  }
  function handleOffline() {
    isOnline.value = false
  }

  onMounted(() => {
    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)
  })
  onBeforeUnmount(() => {
    window.removeEventListener('online', handleOnline)
    window.removeEventListener('offline', handleOffline)
  })

  return { isOnline }
}
