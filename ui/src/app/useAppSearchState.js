import { computed, onUnmounted, reactive } from 'vue'

export function useAppSearchState({ activeView, isEditing, isDetail, getPageState, onRefresh }) {
  const searchByView = reactive({
    readingOrders: '',
    arcs: '',
    comics: '',
    series: '',
    characters: '',
  })
  const search = computed(() => searchByView[activeView.value] || '')
  const searchTerm = computed(() => search.value.trim().toLowerCase())
  let debounceTimer = null

  function updateSearch(value) {
    const view = activeView.value
    if (!(view in searchByView)) return

    searchByView[view] = value
    const pageState = getPageState()
    if (pageState?.[view]) pageState[view].initialized = false

    if (debounceTimer) window.clearTimeout(debounceTimer)
    debounceTimer = window.setTimeout(() => {
      if (activeView.value === view && !isEditing.value && !isDetail.value) onRefresh()
    }, 250)
  }

  onUnmounted(() => {
    if (debounceTimer) window.clearTimeout(debounceTimer)
  })

  return { search, searchTerm, updateSearch }
}
