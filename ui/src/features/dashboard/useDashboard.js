import { ref } from 'vue'
import { ApiError, getDashboard, updateComicReadStatus } from '@/api/client.js'

export function useDashboard({ error, quickSavingComicId }) {
  const dashboard = ref(null)
  const loading = ref(false)

  async function loadDashboard() {
    loading.value = true
    try {
      dashboard.value = await getDashboard()
    } finally {
      loading.value = false
    }
  }

  async function markComicRead(comic) {
    await setComicStatus(comic, { read: true })
  }

  async function markComicSkipped(comic) {
    await setComicStatus(comic, { skipped: true })
  }

  function applyOptimisticStatus(comicId, payload) {
    if (!dashboard.value) return
    for (const section of Object.values(dashboard.value)) {
      if (!Array.isArray(section)) continue
      for (const comic of section) {
        if (comic.id === comicId) Object.assign(comic, payload)
      }
    }
  }

  async function setComicStatus(comic, payload) {
    if (!comic?.id || quickSavingComicId.value) return
    quickSavingComicId.value = comic.id
    error.value = ''
    applyOptimisticStatus(comic.id, payload)
    try {
      await updateComicReadStatus(comic.id, payload)
      await loadDashboard()
    } catch (err) {
      if (err instanceof ApiError) {
        error.value = err.message
        await loadDashboard()
      }
      // A non-ApiError failure here means the request is queued for
      // background sync (likely offline) rather than lost — keep the
      // optimistic state and skip reloading, since there's nothing newer
      // to fetch yet.
    } finally {
      quickSavingComicId.value = null
    }
  }

  return { dashboard, loading, loadDashboard, markComicRead, markComicSkipped }
}
