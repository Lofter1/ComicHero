import { computed, ref } from 'vue'
import {
  deleteSeries as removeSeries,
  getSeries,
  importMetronSeries,
  listSeries,
  setSeriesStarted,
  updateSeriesFavorite,
} from '@/api/client.js'

export function useSeries({
  activeView,
  viewMode,
  error,
  loadPagedList,
  metronImportJobs,
  trackMetronImportJob,
}) {
  const series = ref([])
  const selectedSeries = ref(null)
  const startSaving = ref(false)
  const deleting = ref(false)

  const visibleSeries = computed(() => series.value)

  async function openSeries(item) {
    if (!item?.id) return
    error.value = ''
    activeView.value = 'series'
    selectedSeries.value = null
    viewMode.value = 'detail'
    selectedSeries.value = await getSeries(item.id)
  }

  async function toggleSeriesFavorite(item) {
    if (!item?.id) return

    error.value = ''
    try {
      const detail = await updateSeriesFavorite(item.id, !item.favorite)
      applySeriesFavoriteState(detail)
    } catch (err) {
      error.value = err.message
    }
  }

  function applySeriesFavoriteState(detail) {
    series.value = series.value.map((item) => {
      return item.id === detail.id ? { ...item, favorite: detail.favorite } : item
    })

    if (selectedSeries.value?.id === detail.id) {
      selectedSeries.value = { ...selectedSeries.value, ...detail }
    }
  }

  async function toggleSelectedSeriesStarted() {
    if (!selectedSeries.value?.id || startSaving.value) return
    startSaving.value = true
    error.value = ''
    try {
      const detail = await setSeriesStarted(
        selectedSeries.value.id,
        !selectedSeries.value.startedAt,
      )
      selectedSeries.value = detail
      series.value = series.value.map((item) =>
        item.id === detail.id ? { ...item, startedAt: detail.startedAt || null } : item,
      )
    } catch (err) {
      error.value = err.message
    } finally {
      startSaving.value = false
    }
  }

  async function importSelectedSeriesFromMetron() {
    if (!selectedSeries.value?.metronSeriesId || seriesImportRunning(selectedSeries.value)) return

    error.value = ''
    try {
      const { data: job } = await importMetronSeries(selectedSeries.value.metronSeriesId)
      trackMetronImportJob({ ...job, displayName: selectedSeries.value.name })
    } catch (err) {
      error.value = err.message
    }
  }

  function seriesImportRunning(item) {
    if (!item?.metronSeriesId) return false
    return metronImportJobs.value.some((job) => {
      return (
        job.type === 'series' &&
        job.metronId === item.metronSeriesId &&
        (job.status === 'queued' || job.status === 'running' || job.status === 'canceling')
      )
    })
  }

  async function refreshSelectedSeriesDetail() {
    if (selectedSeries.value?.id) {
      selectedSeries.value = await getSeries(selectedSeries.value.id)
    }
  }

  async function deleteSelectedSeries() {
    if (!selectedSeries.value?.id || deleting.value) return
    const comicCount = selectedSeries.value.entryCount || selectedSeries.value.comics?.length || 0
    if (
      !confirm(
        `Delete ${selectedSeries.value.name} and its ${comicCount} linked comic${comicCount === 1 ? '' : 's'}? This cannot be undone.`,
      )
    )
      return

    deleting.value = true
    error.value = ''
    try {
      await removeSeries(selectedSeries.value.id)
      selectedSeries.value = null
      await loadSeries({ force: true })
      viewMode.value = 'browse'
    } catch (err) {
      error.value = err.message
    } finally {
      deleting.value = false
    }
  }

  async function loadSeries(options = {}) {
    await loadPagedList('series', series, listSeries, options)
  }

  return {
    series,
    selectedSeries,
    startSaving,
    deleting,
    visibleSeries,
    openSeries,
    toggleSeriesFavorite,
    toggleSelectedSeriesStarted,
    importSelectedSeriesFromMetron,
    seriesImportRunning,
    refreshSelectedSeriesDetail,
    deleteSelectedSeries,
    loadSeries,
  }
}
