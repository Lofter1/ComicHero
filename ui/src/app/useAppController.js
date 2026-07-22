import { computed, nextTick, onMounted, onUnmounted, watch } from 'vue'
import {
  backFallbackRouteLocation,
  browseRouteLocation,
  detailRouteLocation,
  editRouteLocation,
  routeToAppState,
} from '@/router/appRouteState.js'
import { routeAccessRedirect, setRouteAccessContext } from '@/router/index.js'
import { getSystemInfo } from '@/api/client.js'

export function useAppController({
  state,
  routing,
  auth,
  access,
  entities,
  forms,
  actions,
  loaders,
  metron,
}) {
  const { activeView, viewMode, loading, error, systemInfo, isEditing, isDetail } = state
  const { route, router } = routing
  let routeSyncPaused = false

  const currentRouteLocation = computed(() => routeLocationForCurrentState())

  function syncRouteAccessContext() {
    if (!auth.userStatus.value) return
    setRouteAccessContext({
      canAccessMetron: access.canAccessMetronArea.value,
      isAdmin: auth.isAdmin.value,
      hasUser: Boolean(auth.currentUser.value),
      readOnlyGuest: auth.isReadOnlyGuest.value,
    })
  }

  async function enforceCurrentRouteAccess() {
    syncRouteAccessContext()
    const redirect = routeAccessRedirect(route)
    if (!redirect) return false
    await router.replace(redirect)
    return true
  }

  function routeLocationForCurrentState() {
    if (activeView.value === 'notFound') return route.fullPath
    if (viewMode.value === 'browse') {
      const query =
        activeView.value === 'settings' && route.name === 'settings' ? route.query : null
      return browseRouteLocation(activeView.value, query)
    }
    if (viewMode.value === 'detail') {
      if (activeView.value === 'readingOrders')
        return detailRouteLocation(activeView.value, entities.selectedOrder.value?.id)
      if (activeView.value === 'arcs')
        return detailRouteLocation(activeView.value, entities.selectedArc.value?.id)
      if (activeView.value === 'comics')
        return detailRouteLocation(activeView.value, entities.selectedComic.value?.id)
      if (activeView.value === 'series')
        return detailRouteLocation(activeView.value, entities.selectedSeries.value?.id)
      if (activeView.value === 'characters')
        return detailRouteLocation(activeView.value, entities.selectedCharacter.value?.id)
      if (activeView.value === 'collections')
        return detailRouteLocation(activeView.value, entities.selectedCollection.value?.id)
    }
    if (viewMode.value === 'edit') {
      if (activeView.value === 'readingOrders')
        return editRouteLocation(
          activeView.value,
          forms.orderForm.value?.id || entities.selectedOrder.value?.id,
        )
      if (activeView.value === 'arcs')
        return editRouteLocation(
          activeView.value,
          forms.arcForm.value?.id || entities.selectedArc.value?.id,
        )
      if (activeView.value === 'comics')
        return editRouteLocation(
          activeView.value,
          forms.comicForm.value?.id || entities.selectedComic.value?.id,
        )
    }
    return null
  }

  async function syncRouterRoute(location, { replace = false } = {}) {
    if (routeSyncPaused || !location) return
    if (router.resolve(location).fullPath === route.fullPath) return
    routeSyncPaused = true
    try {
      await router[replace ? 'replace' : 'push'](location)
    } finally {
      await nextTick()
      routeSyncPaused = false
    }
  }

  async function applyCurrentRoute(options = {}) {
    await applyRoute(routeToAppState(route), options)
  }

  async function applyRoute(routeState, { replace = false, force = false } = {}) {
    routeSyncPaused = true
    error.value = ''

    try {
      await activateRoute({ ...routeState, force })
    } catch (err) {
      error.value = err.message
      activeView.value = routeState.view || 'readingOrders'
      viewMode.value = 'browse'
      await loadActiveViewData({ force: true })
    } finally {
      await nextTick()
      routeSyncPaused = false
    }

    const nextLocation = currentRouteLocation.value ||
      browseRouteLocation(activeView.value) || { name: 'readingOrders' }
    await syncRouterRoute(nextLocation, { replace: replace || routeState.replace })
  }

  async function activateRoute(routeState) {
    if (routeState.mode === 'browse') {
      activeView.value = routeState.view
      viewMode.value = 'browse'
      await loadData(Boolean(routeState.force))
      return
    }

    if (routeState.view === 'readingOrders') {
      if (auth.isReadOnlyGuest.value && (routeState.isNew || routeState.mode === 'edit')) {
        viewMode.value = 'browse'
        await loadData(Boolean(routeState.force))
        return
      }
      if (routeState.isNew) return actions.newReadingOrder()
      await actions.openReadingOrder({ id: routeState.id })
      if (routeState.mode === 'edit') actions.editReadingOrder()
      return
    }

    if (routeState.view === 'arcs') {
      if (auth.isReadOnlyGuest.value && (routeState.isNew || routeState.mode === 'edit')) {
        viewMode.value = 'browse'
        await loadData(Boolean(routeState.force))
        return
      }
      if (routeState.isNew) return actions.newArc()
      await actions.openArc({ id: routeState.id })
      if (routeState.mode === 'edit') actions.editArc()
      return
    }

    if (routeState.view === 'comics') {
      if (auth.isReadOnlyGuest.value && (routeState.isNew || routeState.mode === 'edit')) {
        viewMode.value = 'browse'
        await loadData(Boolean(routeState.force))
        return
      }
      if (routeState.isNew) return actions.newComic()
      await actions.openComic({ id: routeState.id })
      if (routeState.mode === 'edit') actions.editComic()
      return
    }

    if (routeState.view === 'series') return actions.openSeries({ id: routeState.id })
    if (routeState.view === 'characters') return actions.openCharacter({ id: routeState.id })
    if (routeState.view === 'collections') await actions.openCollection({ id: routeState.id })
  }

  async function backToPreviousPage() {
    error.value = ''
    await popHistoryOrReplace(backFallbackRouteLocation(activeView.value))
  }

  async function popHistoryOrReplace(fallback) {
    if (window.history.state?.back) {
      router.back()
      return
    }
    await router.replace(fallback)
  }

  async function loadData(force = false) {
    loading.value = true
    error.value = ''
    try {
      await loadActiveViewData({ force })
      await ensureMetronImportMonitor()
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function ensureMetronImportMonitor() {
    if (!auth.appReady.value || !access.canMetronMonitor.value || auth.isReadOnlyGuest.value) return
    await metron.loadMetronImportJobs()
  }

  async function loadActiveViewData(options = {}) {
    if (activeView.value === 'settings') {
      await Promise.all([
        loaders.settings(),
        access.canMetronMonitor.value ? metron.loadMetronQuota() : Promise.resolve(),
      ])
      return
    }
    const viewLoader = loaders[activeView.value]
    if (viewLoader) return viewLoader(options)
  }

  async function refreshActiveLibraryData() {
    await loadActiveViewData({ force: true })
  }

  async function refreshActiveListData() {
    error.value = ''
    try {
      await loadActiveViewData({ force: true })
    } catch (err) {
      error.value = err.message
    }
  }

  async function loadMoreActiveViewData() {
    if (loading.value || isEditing.value || isDetail.value) return
    try {
      await loadActiveViewData({ append: true })
    } catch (err) {
      error.value = err.message
    }
  }

  async function cancelEdit() {
    error.value = ''
    let selectedID = null
    if (activeView.value === 'readingOrders') selectedID = entities.selectedOrder.value?.id || null
    if (activeView.value === 'arcs') selectedID = entities.selectedArc.value?.id || null
    if (activeView.value === 'comics') selectedID = entities.selectedComic.value?.id || null
    await popHistoryOrReplace(backFallbackRouteLocation(activeView.value, selectedID))
  }

  function showError(message) {
    error.value = message
  }

  function clearError() {
    error.value = ''
    actions.clearMetronMergeConflict()
  }

  async function handleMetronImported() {
    error.value = ''
    await refreshActiveLibraryData()
    if (activeView.value === 'readingOrders' && viewMode.value === 'detail') {
      await actions.refreshSelectedReadingOrderDetail()
    }
    if (activeView.value === 'characters' && viewMode.value === 'detail') {
      await actions.refreshSelectedCharacterDetail()
    }
    if (activeView.value === 'series' && viewMode.value === 'detail') {
      await actions.refreshSelectedSeriesDetail()
    }
  }

  onMounted(async () => {
    const [, info] = await Promise.all([auth.loadUserStatus(), getSystemInfo().catch(() => null)])
    systemInfo.value = info
    if (route.name === 'verifyEmail') await auth.verifyEmailFromRouteToken()
    else if (route.name === 'resetPassword') auth.preparePasswordResetFromRouteToken()

    if (auth.appReady.value) {
      if (!(await enforceCurrentRouteAccess())) {
        await applyCurrentRoute({ replace: true, force: true })
      }
      await ensureMetronImportMonitor()
    }
  })

  watch(currentRouteLocation, (location) => {
    if (auth.appReady.value) syncRouterRoute(location)
  })

  watch(
    () => route.fullPath,
    () => {
      if (auth.appReady.value && !routeSyncPaused) applyCurrentRoute()
    },
  )

  watch([access.canAccessMetronArea, auth.isAdmin, auth.currentUser, auth.isReadOnlyGuest], () => {
    syncRouteAccessContext()
    if (auth.appReady.value) {
      enforceCurrentRouteAccess()
      ensureMetronImportMonitor()
    }
  })

  onUnmounted(metron.closeMetronImportEvents)

  return {
    applyCurrentRoute,
    backToPreviousPage,
    cancelEdit,
    clearError,
    handleMetronImported,
    loadMoreActiveViewData,
    refreshActiveLibraryData,
    refreshActiveListData,
    showError,
  }
}
