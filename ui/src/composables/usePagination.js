import { computed, ref } from 'vue'

function emptyPageState() {
  return {
    initialized: false,
    hasMore: true,
    nextOffset: 0,
    total: 0,
    refreshing: false,
    loadingMore: false,
  }
}

export function usePagination({ activeView, loading, isEditing, isDetail, searchTerm, listParams, pageSize = 50 } = {}) {
  const pageState = ref({
    readingOrders: emptyPageState(),
    arcs: emptyPageState(),
    comics: emptyPageState(),
    series: emptyPageState(),
    characters: emptyPageState(),
  })

  const activePageState = computed(() => pageState.value[activeView.value] || null)
  const showInfiniteScrollSentinel = computed(() => {
    return Boolean(activePageState.value)
      && !loading.value
      && !isEditing.value
      && !isDetail.value
      && !activePageState.value.refreshing
      && (activePageState.value.hasMore || activePageState.value.loadingMore)
  })
  const activeListLoadingMore = computed(() => Boolean(activePageState.value?.loadingMore))

  function listTotal(key) {
    return pageState.value[key]?.total ?? 0
  }

  async function loadPagedList(key, target, listFn, { append = false, force = false, all = false } = {}) {
    const state = pageState.value[key]
    if (!state) return
    if (append && (!state.initialized || !state.hasMore || state.loadingMore)) return
    if (!append && state.initialized && !force) return

    const offset = append ? state.nextOffset : 0
    const isActiveList = key === activeView.value
    const params = { ...(isActiveList ? (listParams?.value || {}) : {}), limit: pageSize, offset }
    if (isActiveList && searchTerm.value) {
      params.q = searchTerm.value
    }
    state.refreshing = !append
    state.loadingMore = append
    try {
      const page = await listFn(params)
      target.value = append ? [...target.value, ...page.items] : page.items
      state.initialized = true
      state.hasMore = page.hasMore
      state.total = page.total
      state.nextOffset = offset + page.items.length

      if (all && !append) {
        let nextOffset = state.nextOffset
        let hasMore = state.hasMore
        while (hasMore) {
          const nextPage = await listFn({ ...params, offset: nextOffset })
          target.value = [...target.value, ...nextPage.items]
          nextOffset += nextPage.items.length
          hasMore = nextPage.hasMore && nextPage.items.length > 0
          state.hasMore = hasMore
          state.total = nextPage.total
          state.nextOffset = nextOffset
        }
      }
    } finally {
      state.refreshing = false
      state.loadingMore = false
    }
  }

  return {
    pageState,
    activePageState,
    showInfiniteScrollSentinel,
    activeListLoadingMore,
    listTotal,
    loadPagedList,
  }
}
