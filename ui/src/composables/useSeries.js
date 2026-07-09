import { computed, ref } from 'vue'
import { getSeries, importMetronSeries, listSeries, updateSeriesFavorite } from '@/api/client.js'

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

  const visibleSeries = computed(() => series.value)
  const favoriteVisibleSeries = computed(() => series.value.filter((item) => item.favorite))
  const remainingVisibleSeries = computed(() => series.value.filter((item) => !item.favorite))
  const seriesBrowseSections = computed(() => {
    if (!favoriteVisibleSeries.value.length) {
      return [{ key: 'all', title: 'All Series', series: series.value }]
    }
    return [
      { key: 'favorites', title: 'Favorites', series: favoriteVisibleSeries.value },
      { key: 'other', title: 'Other Series', series: remainingVisibleSeries.value },
    ].filter((section) => section.series.length)
  })

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

  async function loadSeries(options = {}) {
    await loadPagedList('series', series, listSeries, options)
  }

  return {
    series,
    selectedSeries,
    visibleSeries,
    seriesBrowseSections,
    openSeries,
    toggleSeriesFavorite,
    importSelectedSeriesFromMetron,
    seriesImportRunning,
    refreshSelectedSeriesDetail,
    loadSeries,
  }
}
