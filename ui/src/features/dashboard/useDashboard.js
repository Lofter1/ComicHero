import { ref } from 'vue'
import { getDashboard, updateComicReadStatus } from '@/api/client.js'

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

  async function setComicStatus(comic, payload) {
    if (!comic?.id || quickSavingComicId.value) return
    quickSavingComicId.value = comic.id
    error.value = ''
    try {
      await updateComicReadStatus(comic.id, payload)
      await loadDashboard()
    } catch (err) {
      error.value = err.message
    } finally {
      quickSavingComicId.value = null
    }
  }

  return { dashboard, loading, loadDashboard, markComicRead, markComicSkipped }
}
