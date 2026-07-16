<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AccountView from '@/features/account/components/AccountView.vue'
import AuthenticationView from '@/features/auth/components/AuthenticationView.vue'
import AppSettingsView from '@/features/settings/components/AppSettingsView.vue'
import ArcDetailView from '@/features/arcs/components/ArcDetailView.vue'
import ArcsBrowseView from '@/features/arcs/components/ArcsBrowseView.vue'
import AppSidebar from '@/app/components/AppSidebar.vue'
import AppToolbar from '@/app/components/AppToolbar.vue'
import CharactersBrowseView from '@/features/characters/components/CharactersBrowseView.vue'
import CharacterDetailView from '@/features/characters/components/CharacterDetailView.vue'
import ComicDetailView from '@/features/comics/components/ComicDetailView.vue'
import ComicListView from '@/features/comics/components/ComicListView.vue'
import DashboardView from '@/features/dashboard/components/DashboardView.vue'
import ErrorToast from '@/shared/components/feedback/ErrorToast.vue'
import MetronImport from '@/features/metron/components/MetronImport.vue'
import MetronImportMonitor from '@/features/metron/components/MetronImportMonitor.vue'
import ProgressView from '@/features/account/components/ProgressView.vue'
import ReadingOrderDetailView from '@/features/reading-orders/components/ReadingOrderDetailView.vue'
import ReadingOrderEditView from '@/features/reading-orders/components/ReadingOrderEditView.vue'
import ReadingOrdersBrowseView from '@/features/reading-orders/components/ReadingOrdersBrowseView.vue'
import SeriesBrowseView from '@/features/series/components/SeriesBrowseView.vue'
import SeriesDetailView from '@/features/series/components/SeriesDetailView.vue'
import UserManagementView from '@/features/users/components/UserManagementView.vue'
import { useAccount } from '@/features/account/useAccount.js'
import { useArcs } from '@/features/arcs/useArcs.js'
import { useAuthSession } from '@/features/auth/useAuthSession.js'
import { useCharacters } from '@/features/characters/useCharacters.js'
import { useComics } from '@/features/comics/useComics.js'
import { useDashboard } from '@/features/dashboard/useDashboard.js'
import { useListOptions } from '@/shared/composables/useListOptions.js'
import { useMetronJobs } from '@/features/metron/useMetronJobs.js'
import { useMetronSettings } from '@/features/settings/useMetronSettings.js'
import { usePagination } from '@/shared/composables/usePagination.js'
import { useReadingOrders } from '@/features/reading-orders/useReadingOrders.js'
import { useSeries } from '@/features/series/useSeries.js'
import { useTheme } from '@/shared/composables/useTheme.js'
import { useUserAdministration } from '@/features/users/useUserAdministration.js'
import {
  browseRouteLocation,
  detailRouteLocation,
  editRouteLocation,
  routeToAppState,
} from '@/router/appRouteState.js'
import { routeAccessRedirect, setRouteAccessContext } from '@/router/index.js'
import { getSystemInfo } from '@/api/client.js'

const activeView = ref('dashboard')
const viewMode = ref('browse')
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const systemInfo = ref(null)
const searchByView = reactive({
  readingOrders: '',
  arcs: '',
  comics: '',
  series: '',
  characters: '',
})
const search = computed(() => searchByView[activeView.value] || '')
const loadMoreSentinel = ref(null)
let loadMoreObserver = null
let searchDebounceTimer = null
let routeSyncPaused = false
const route = useRoute()
const router = useRouter()
const { themePreference, setThemePreference } = useTheme()
const {
  authLoading,
  authSaving,
  userStatus,
  authMode,
  setupForm,
  authForm,
  verificationForm,
  passwordResetForm,
  setupRequired,
  userMode,
  registrationMode,
  publicAccess,
  currentUser,
  isAdmin,
  isReadOnlyGuest,
  emailVerificationRequired,
  passwordResetMode,
  authRequired,
  appReady,
  loadUserStatus,
  submitSetup,
  submitAuth,
  submitEmailVerification,
  resendVerificationEmail,
  verifyEmailFromRouteToken,
  showForgotPassword,
  showLogin,
  requestLogin,
  submitForgotPassword,
  submitPasswordReset,
  preparePasswordResetFromRouteToken,
  signOut,
} = useAuthSession({ route, error, onReady: applyCurrentRoute })
const {
  users: userAdminRows,
  auditEvents,
  savingPermissionsUserId: savingUserID,
  savingAdminUserId: savingAdminUserID,
  deletingUserId: deletingUserID,
  loadUsers: loadUserAdminRows,
  savePermissions: saveUserMetronPermissions,
  saveAdmin: saveUserAdmin,
  removeUser,
} = useUserAdministration({ error })
const {
  saving: accountSaving,
  deleting: accountDeleting,
  statistics: accountStatistics,
  statisticsLoading: accountStatisticsLoading,
  statisticsError: accountStatisticsError,
  saveAccount,
  deleteCurrentAccount,
  loadStatistics: loadAccountStatistics,
} = useAccount({ error, userStatus, currentUser, activeView, viewMode, authMode })
const {
  comicScan: metronComicScan,
  comicDiscovery: metronComicDiscovery,
  savingComicScan: savingMetronComicScan,
  savingComicDiscovery: savingMetronComicDiscovery,
  generatedInvite,
  generatingInvite,
  savingRegistrationMode,
  savingPublicAccess,
  loadSettings: loadMetronSettings,
  saveComicScan: saveMetronComicScan,
  runComicScan: runMetronComicScan,
  cancelComicScan: cancelMetronComicScan,
  saveComicDiscovery: saveMetronComicDiscovery,
  runComicDiscovery: runMetronComicDiscovery,
  cancelComicDiscovery: cancelMetronComicDiscovery,
  generateInvite: generateUserInvite,
  saveRegistration: saveRegistrationMode,
  savePublic: savePublicAccess,
} = useMetronSettings({ activeView, error, userStatus, registrationMode, publicAccess })
const { listOptions, activeListParams, updateListOption } = useListOptions({
  activeView,
  onChange: refreshActiveListData,
})

const searchTerm = computed(() => search.value.trim().toLowerCase())
const isEditing = computed(() => viewMode.value === 'edit')
const isDetail = computed(() => viewMode.value === 'detail')
const currentRouteLocation = computed(() => routeLocationForCurrentState())

const { pageState, showInfiniteScrollSentinel, activeListLoadingMore, listTotal, loadPagedList } =
  usePagination({
    activeView,
    loading,
    isEditing,
    isDetail,
    searchTerm,
    listParams: activeListParams,
  })

const {
  metronImportJobs,
  metronImportMonitorOpen,
  metronQuota,
  loadMetronImportJobs,
  loadMetronQuota,
  updateMetronQuota,
  trackMetronImportJob,
  dismissMetronJob,
  retryMetronJob,
  continueMetronJob,
  cancelMetronJob,
  closeMetronImportEvents,
} = useMetronJobs({ activeView, error, handleImported: handleMetronImported })

const {
  selectedSeries,
  startSaving: seriesStartSaving,
  deleting: seriesDeleting,
  visibleSeries,
  openSeries,
  toggleSeriesFavorite,
  toggleSelectedSeriesStarted,
  importSelectedSeriesFromMetron,
  seriesImportRunning,
  refreshSelectedSeriesDetail,
  deleteSelectedSeries,
  loadSeries,
} = useSeries({
  activeView,
  viewMode,
  error,
  loadPagedList,
  metronImportJobs,
  trackMetronImportJob,
})

const {
  selectedCharacter,
  quickSavingCharacterID,
  startSaving: characterStartSaving,
  deleting: characterDeleting,
  visibleCharacters,
  openCharacter,
  toggleCharacterFavorite,
  toggleSelectedCharacterStarted,
  importSelectedCharacterAppearances,
  characterImportRunning,
  refreshSelectedCharacterDetail,
  deleteSelectedCharacter,
  loadCharacters,
} = useCharacters({
  activeView,
  viewMode,
  error,
  loadPagedList,
  metronImportJobs,
  trackMetronImportJob,
})

const {
  selectedArc,
  quickSavingArcID,
  startSaving: arcStartSaving,
  arcForm,
  visibleArcs,
  arcProgress,
  openArc,
  toggleArcFavorite,
  toggleSelectedArcStarted,
  newArc,
  editArc,
  deleteArc,
  loadArcs,
} = useArcs({
  activeView,
  viewMode,
  error,
  saving,
  loadComics: (...args) => loadComics(...args),
  loadPagedList,
})

const {
  readingOrders,
  selectedOrder,
  quickSavingOrderID,
  ratingSaving,
  startSaving,
  cblImporting,
  orderForm,
  visibleOrders,
  readingOrderProgress,
  openReadingOrder,
  toggleReadingOrderFavorite,
  rateSelectedReadingOrder,
  startSelectedReadingOrder,
  stopSelectedReadingOrder,
  refreshSelectedReadingOrderDetail,
  newReadingOrder,
  saveReadingOrder,
  deleteReadingOrder,
  copySelectedReadingOrder,
  editReadingOrder,
  importReadingOrderCBLFile,
  exportSelectedReadingOrderCBL,
  loadReadingOrders,
} = useReadingOrders({
  activeView,
  viewMode,
  error,
  saving,
  loadComics: (...args) => loadComics(...args),
  loadPagedList,
})

const {
  comics,
  selectedComic,
  quickSavingComicID,
  comicForm,
  metronMetadataOpen,
  metronMetadataSearching,
  metronMetadataApplyingID,
  metronMetadataStatus,
  metronMetadataResults,
  loadComics,
  openComic,
  openOrderComic,
  newComic,
  editComic,
  deleteComic,
  toggleComicRead,
  toggleComicSkipped,
  resetMetronMetadata,
  searchSelectedComicMetron,
  applyMetronMetadata,
} = useComics({
  activeView,
  viewMode,
  error,
  saving,
  loadPagedList,
  refreshActiveLibraryData,
  readingOrderProgress,
  selectedOrder,
  selectedArc,
  selectedCharacter,
  selectedSeries,
  collectionProgress: arcProgress,
})
const {
  dashboard,
  loading: dashboardLoading,
  loadDashboard,
  markComicRead: markDashboardComicRead,
  markComicSkipped: markDashboardComicSkipped,
} = useDashboard({ error, quickSavingComicId: quickSavingComicID })

const toolbarResultCount = computed(() => {
  if (activeView.value === 'dashboard') return dashboard.value?.items?.length || 0
  if (activeView.value === 'readingOrders') return visibleOrders.value.length
  if (activeView.value === 'arcs') return visibleArcs.value.length
  if (activeView.value === 'comics') return comics.value.length
  if (activeView.value === 'series') return visibleSeries.value.length
  if (activeView.value === 'characters') return visibleCharacters.value.length
  if (activeView.value === 'users') return userAdminRows.value.length
  if (activeView.value === 'account' || activeView.value === 'progress') return 1
  return 0
})
const toolbarTotalCount = computed(() => {
  if (activeView.value === 'dashboard') return dashboard.value?.items?.length || 0
  if (activeView.value === 'readingOrders') return listTotal('readingOrders')
  if (activeView.value === 'arcs') return listTotal('arcs')
  if (activeView.value === 'comics') return listTotal('comics')
  if (activeView.value === 'series') return listTotal('series')
  if (activeView.value === 'characters') return listTotal('characters')
  if (activeView.value === 'users') return userAdminRows.value.length
  if (activeView.value === 'account' || activeView.value === 'progress') return 1
  return 0
})
const loadingLabel = computed(() => {
  if (activeView.value === 'dashboard') return 'Loading dashboard...'
  if (activeView.value === 'readingOrders') return 'Loading orders...'
  if (activeView.value === 'arcs') return 'Loading arcs...'
  if (activeView.value === 'comics') return 'Loading comics...'
  if (activeView.value === 'series') return 'Loading series...'
  if (activeView.value === 'characters') return 'Loading characters...'
  if (activeView.value === 'metron') return 'Loading Metron...'
  if (activeView.value === 'users') return 'Loading users...'
  if (activeView.value === 'account') return 'Loading account...'
  if (activeView.value === 'progress') return 'Loading progress...'
  return 'Loading...'
})
const showBlockingLoading = computed(() => loading.value && activeView.value !== 'series')
const seriesListLoading = computed(() => Boolean(pageState.value.series?.refreshing))
const metronPermissions = computed(() => userStatus.value?.metronPermissions || {})
const canMetronSearch = computed(() => hasMetronScope('search'))
const canMetronMonitor = computed(() => hasMetronScope('monitor'))
const canAccessMetronArea = computed(() => canMetronSearch.value)

function hasMetronScope(scope) {
  const permissions = metronPermissions.value
  if (!permissions.allowed) return false
  const scopes = Array.isArray(permissions.scopes) ? permissions.scopes : []
  return scopes.includes('*') || scopes.includes(scope)
}

function syncRouteAccessContext() {
  if (!userStatus.value) return
  setRouteAccessContext({
    canAccessMetron: canAccessMetronArea.value,
    isAdmin: isAdmin.value,
    hasUser: Boolean(currentUser.value),
    readOnlyGuest: isReadOnlyGuest.value,
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
  if (viewMode.value === 'browse') return browseRouteLocation(activeView.value)
  if (viewMode.value === 'detail') {
    if (activeView.value === 'readingOrders')
      return detailRouteLocation(activeView.value, selectedOrder.value?.id)
    if (activeView.value === 'arcs')
      return detailRouteLocation(activeView.value, selectedArc.value?.id)
    if (activeView.value === 'comics')
      return detailRouteLocation(activeView.value, selectedComic.value?.id)
    if (activeView.value === 'series')
      return detailRouteLocation(activeView.value, selectedSeries.value?.id)
    if (activeView.value === 'characters')
      return detailRouteLocation(activeView.value, selectedCharacter.value?.id)
  }
  if (viewMode.value === 'edit') {
    if (activeView.value === 'readingOrders')
      return editRouteLocation(activeView.value, orderForm.value?.id || selectedOrder.value?.id)
    if (activeView.value === 'arcs')
      return editRouteLocation(activeView.value, arcForm.value?.id || selectedArc.value?.id)
    if (activeView.value === 'comics')
      return editRouteLocation(activeView.value, comicForm.value?.id || selectedComic.value?.id)
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

async function applyRoute(route, { replace = false, force = false } = {}) {
  routeSyncPaused = true
  error.value = ''

  try {
    await activateRoute({ ...route, force })
  } catch (err) {
    error.value = err.message
    activeView.value = route.view || 'readingOrders'
    viewMode.value = 'browse'
    await loadActiveViewData({ force: true })
  } finally {
    await nextTick()
    routeSyncPaused = false
  }

  const nextLocation = currentRouteLocation.value ||
    browseRouteLocation(activeView.value) || { name: 'readingOrders' }
  await syncRouterRoute(nextLocation, { replace: replace || route.replace })
}

async function activateRoute(route) {
  if (route.mode === 'browse') {
    activeView.value = route.view
    viewMode.value = 'browse'
    await loadData(Boolean(route.force))
    return
  }

  if (route.view === 'readingOrders') {
    if (isReadOnlyGuest.value && (route.isNew || route.mode === 'edit')) {
      viewMode.value = 'browse'
      await loadData(Boolean(route.force))
      return
    }
    if (route.isNew) {
      newReadingOrder()
      return
    }
    await openReadingOrder({ id: route.id })
    if (route.mode === 'edit') editReadingOrder()
    return
  }

  if (route.view === 'arcs') {
    if (isReadOnlyGuest.value && (route.isNew || route.mode === 'edit')) {
      viewMode.value = 'browse'
      await loadData(Boolean(route.force))
      return
    }
    if (route.isNew) {
      newArc()
      return
    }
    await openArc({ id: route.id })
    if (route.mode === 'edit') editArc()
    return
  }

  if (route.view === 'comics') {
    if (isReadOnlyGuest.value && (route.isNew || route.mode === 'edit')) {
      viewMode.value = 'browse'
      await loadData(Boolean(route.force))
      return
    }
    if (route.isNew) {
      newComic()
      return
    }
    await openComic({ id: route.id })
    if (route.mode === 'edit') editComic()
    return
  }

  if (route.view === 'series') {
    await openSeries({ id: route.id })
    return
  }

  if (route.view === 'characters') {
    await openCharacter({ id: route.id })
  }
}

async function backToPreviousPage() {
  error.value = ''
  if (window.history.state?.back) {
    router.back()
    return
  }
  await router.push(browseRouteLocation(activeView.value) || { name: 'readingOrders' })
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
  if (!appReady.value || !canMetronMonitor.value || isReadOnlyGuest.value) return
  await loadMetronImportJobs()
}

async function loadActiveViewData(options = {}) {
  if (activeView.value === 'dashboard') {
    await loadDashboard()
    return
  }
  if (activeView.value === 'readingOrders') {
    await loadReadingOrders(options)
    return
  }
  if (activeView.value === 'arcs') {
    await loadArcs(options)
    return
  }
  if (activeView.value === 'comics') {
    await loadComics(options)
    return
  }
  if (activeView.value === 'series') {
    await loadSeries(options)
    return
  }
  if (activeView.value === 'characters') {
    await loadCharacters(options)
    return
  }
  if (activeView.value === 'metron') {
    if (canMetronMonitor.value) {
      await loadMetronQuota()
    }
    return
  }
  if (activeView.value === 'users') {
    await loadUserAdminRows()
    return
  }
  if (activeView.value === 'settings') {
    await loadMetronSettings()
    return
  }
  if (activeView.value === 'account') {
    return
  }
  if (activeView.value === 'progress') {
    await loadAccountStatistics()
    return
  }
}

async function refreshActiveLibraryData() {
  if (activeView.value === 'metron') return
  await loadActiveViewData({ force: true })
}

async function refreshActiveListData() {
  if (activeView.value === 'metron') return
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

function cancelEdit() {
  error.value = ''
  if (activeView.value === 'readingOrders' && selectedOrder.value) {
    viewMode.value = 'detail'
    return
  }
  if (activeView.value === 'arcs' && selectedArc.value) {
    viewMode.value = 'detail'
    return
  }
  if (activeView.value === 'comics' && selectedComic.value) {
    viewMode.value = 'detail'
    return
  }
  viewMode.value = 'browse'
}

function showError(message) {
  error.value = message
}

function clearError() {
  error.value = ''
}

async function handleMetronImported() {
  error.value = ''
  await refreshActiveLibraryData()
  if (activeView.value === 'readingOrders' && viewMode.value === 'detail') {
    await refreshSelectedReadingOrderDetail()
  }
  if (activeView.value === 'characters' && viewMode.value === 'detail') {
    await refreshSelectedCharacterDetail()
  }
  if (activeView.value === 'series' && viewMode.value === 'detail') {
    await refreshSelectedSeriesDetail()
  }
}

function updateSearch(value) {
  const view = activeView.value
  if (!(view in searchByView)) return

  searchByView[view] = value
  if (pageState.value[view]) {
    pageState.value[view].initialized = false
  }

  if (searchDebounceTimer) {
    window.clearTimeout(searchDebounceTimer)
  }
  searchDebounceTimer = window.setTimeout(() => {
    if (activeView.value === view && !isEditing.value && !isDetail.value) {
      refreshActiveListData()
    }
  }, 250)
}

function setupLoadMoreObserver() {
  if (typeof IntersectionObserver === 'undefined') return
  loadMoreObserver = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        loadMoreActiveViewData()
      }
    },
    { rootMargin: '360px 0px' },
  )
  observeLoadMoreSentinel()
}

function observeLoadMoreSentinel() {
  if (!loadMoreObserver) return
  loadMoreObserver.disconnect()
  if (loadMoreSentinel.value && showInfiniteScrollSentinel.value) {
    loadMoreObserver.observe(loadMoreSentinel.value)
  }
}

onMounted(async () => {
  setupLoadMoreObserver()
  const [, info] = await Promise.all([loadUserStatus(), getSystemInfo().catch(() => null)])
  systemInfo.value = info
  if (route.name === 'verifyEmail') {
    await verifyEmailFromRouteToken()
  } else if (route.name === 'resetPassword') {
    preparePasswordResetFromRouteToken()
  }
  if (appReady.value) {
    if (!(await enforceCurrentRouteAccess())) {
      await applyCurrentRoute({ replace: true, force: true })
    }
    await ensureMetronImportMonitor()
  }
})

watch(showInfiniteScrollSentinel, () => {
  nextTick(observeLoadMoreSentinel)
})

watch(activeView, () => {
  nextTick(observeLoadMoreSentinel)
})

watch(currentRouteLocation, (location) => {
  if (!appReady.value) return
  syncRouterRoute(location)
})

watch(
  () => route.fullPath,
  () => {
    if (!appReady.value || routeSyncPaused) return
    applyCurrentRoute()
  },
)

watch([canAccessMetronArea, isAdmin, currentUser, isReadOnlyGuest], () => {
  syncRouteAccessContext()
  if (appReady.value) {
    enforceCurrentRouteAccess()
    ensureMetronImportMonitor()
  }
})

onUnmounted(() => {
  closeMetronImportEvents()
  if (searchDebounceTimer) {
    window.clearTimeout(searchDebounceTimer)
  }
  if (loadMoreObserver) {
    loadMoreObserver.disconnect()
  }
})
</script>

<template>
  <AuthenticationView
    v-if="!appReady"
    v-model:setup-form="setupForm"
    v-model:auth-form="authForm"
    v-model:verification-form="verificationForm"
    v-model:password-reset-form="passwordResetForm"
    v-model:auth-mode="authMode"
    :loading="authLoading"
    :saving="authSaving"
    :setup-required="setupRequired"
    :verification-required="emailVerificationRequired"
    :password-reset-mode="passwordResetMode"
    :auth-required="authRequired"
    :registration-mode="registrationMode"
    :verification-email="userStatus?.emailVerificationEmail || verificationForm.email"
    :error="error"
    @clear-error="clearError"
    @retry="loadUserStatus"
    @submit-setup="submitSetup"
    @submit-auth="submitAuth"
    @submit-verification="submitEmailVerification"
    @resend-verification="resendVerificationEmail"
    @submit-forgot-password="submitForgotPassword"
    @submit-password-reset="submitPasswordReset"
    @show-forgot-password="showForgotPassword"
    @show-login="showLogin"
  />

  <main v-else class="app-shell">
    <AppSidebar
      :active-view="activeView"
      :theme-preference="themePreference"
      :user="currentUser"
      :user-mode="userMode"
      :is-admin="isAdmin"
      :public-access="publicAccess"
      :read-only-guest="isReadOnlyGuest"
      :show-metron="canAccessMetronArea && !isReadOnlyGuest"
      :auth-saving="authSaving"
      :version="systemInfo?.version || ''"
      @set-theme="setThemePreference"
      @login="requestLogin"
      @logout="signOut"
    />

    <section class="content">
      <AppToolbar
        v-if="!isEditing && !isDetail"
        :result-count="toolbarResultCount"
        :total-count="toolbarTotalCount"
      />

      <ErrorToast :message="error" @dismiss="clearError" />

      <MetronImportMonitor
        v-model:open="metronImportMonitorOpen"
        :jobs="metronImportJobs"
        @retry="retryMetronJob"
        @continue="continueMetronJob"
        @cancel="cancelMetronJob"
        @dismiss="dismissMetronJob"
      />

      <div v-if="showBlockingLoading" class="loading-panel" role="status" aria-live="polite">
        <span class="loading-spinner" aria-hidden="true"></span>
        <strong>{{ loadingLabel }}</strong>
      </div>

      <MetronImport
        v-else-if="activeView === 'metron'"
        :import-jobs="metronImportJobs"
        :metron-quota="metronQuota"
        @imported="handleMetronImported"
        @error="showError"
        @job-started="trackMetronImportJob"
        @quota-updated="updateMetronQuota"
      />

      <DashboardView
        v-else-if="activeView === 'dashboard'"
        :dashboard="dashboard"
        :loading="dashboardLoading"
        :quick-saving-comic-id="quickSavingComicID"
        :read-only="isReadOnlyGuest"
        @refresh="loadDashboard"
        @open-comic="openComic"
        @mark-read="markDashboardComicRead"
        @mark-skipped="markDashboardComicSkipped"
      />

      <UserManagementView
        v-else-if="activeView === 'users'"
        :users="userAdminRows"
        :audit-events="auditEvents"
        :saving-user-id="savingUserID"
        :saving-admin-user-id="savingAdminUserID"
        :deleting-user-id="deletingUserID"
        :current-user-id="currentUser?.id"
        @save="saveUserMetronPermissions"
        @save-admin="saveUserAdmin"
        @delete-user="removeUser"
      />

      <AppSettingsView
        v-else-if="activeView === 'settings'"
        :metron-comic-scan="metronComicScan"
        :metron-comic-discovery="metronComicDiscovery"
        :saving="savingMetronComicScan"
        :saving-discovery="savingMetronComicDiscovery"
        :registration-mode="registrationMode"
        :saving-registration-mode="savingRegistrationMode"
        :public-access="publicAccess"
        :saving-public-access="savingPublicAccess"
        :invite="generatedInvite"
        :generating-invite="generatingInvite"
        @save="saveMetronComicScan"
        @trigger="runMetronComicScan"
        @stop="cancelMetronComicScan"
        @save-discovery="saveMetronComicDiscovery"
        @trigger-discovery="runMetronComicDiscovery"
        @stop-discovery="cancelMetronComicDiscovery"
        @update-registration-mode="saveRegistrationMode"
        @update-public-access="savePublicAccess"
        @generate-invite="generateUserInvite"
      />

      <AccountView
        v-else-if="activeView === 'account'"
        :user="currentUser"
        :user-mode="userMode"
        :saving="accountSaving"
        :deleting="accountDeleting"
        @save="saveAccount"
        @delete-account="deleteCurrentAccount"
      />

      <ProgressView
        v-else-if="activeView === 'progress'"
        :statistics-view="accountStatistics"
        :loading="accountStatisticsLoading"
        :error="accountStatisticsError"
        @refresh="loadAccountStatistics"
      />

      <section v-else-if="activeView === 'notFound'" class="empty-panel">
        <h2>Page not found</h2>
        <p>This route does not match a ComicHero view.</p>
        <router-link class="primary-button" :to="{ name: 'readingOrders' }">
          Go to reading orders
        </router-link>
      </section>

      <ReadingOrderEditView
        v-else-if="activeView === 'readingOrders' && isEditing"
        v-model:form="orderForm"
        :selected-order="selectedOrder"
        :comics="comics"
        :reading-orders="readingOrders"
        :saving="saving"
        @back="cancelEdit"
        @delete="deleteReadingOrder"
        @save="saveReadingOrder"
      />

      <ReadingOrderDetailView
        v-else-if="activeView === 'readingOrders' && isDetail"
        :selected-order="selectedOrder"
        :current-user-id="currentUser?.id"
        :selected-comic-id="selectedComic?.id"
        :quick-saving-comic-id="quickSavingComicID"
        :read-only="isReadOnlyGuest"
        :saving="saving"
        :rating-saving="ratingSaving"
        :start-saving="startSaving"
        @back="backToPreviousPage"
        @copy="copySelectedReadingOrder"
        @edit="editReadingOrder"
        @export-cbl="exportSelectedReadingOrderCBL"
        @rate="rateSelectedReadingOrder"
        @start="startSelectedReadingOrder"
        @stop="stopSelectedReadingOrder"
        @open-comic="openOrderComic"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
      />

      <ReadingOrdersBrowseView
        v-else-if="activeView === 'readingOrders'"
        :orders="visibleOrders"
        :selected-order-id="selectedOrder?.id"
        :quick-saving-order-id="quickSavingOrderID"
        :cbl-importing="cblImporting"
        :search="search"
        :search-term="searchTerm"
        :filter="listOptions.readingOrders.filter"
        :sort="listOptions.readingOrders.sort"
        :direction="listOptions.readingOrders.direction"
        :read-only="isReadOnlyGuest"
        @update:search="updateSearch"
        @update:filter="updateListOption('readingOrders', 'filter', $event)"
        @update:sort="updateListOption('readingOrders', 'sort', $event)"
        @update:direction="updateListOption('readingOrders', 'direction', $event)"
        @new-order="newReadingOrder"
        @open-order="openReadingOrder"
        @toggle-favorite="toggleReadingOrderFavorite"
        @import-cbl="importReadingOrderCBLFile"
      />

      <ArcDetailView
        v-else-if="activeView === 'arcs' && isDetail"
        :selected-arc="selectedArc"
        :selected-comic-id="selectedComic?.id"
        :quick-saving-comic-id="quickSavingComicID"
        :quick-saving-arc-id="quickSavingArcID"
        :start-saving="arcStartSaving"
        :read-only="isReadOnlyGuest"
        :can-delete="isAdmin"
        :deleting="saving"
        @back="backToPreviousPage"
        @edit="editArc"
        @delete="deleteArc"
        @toggle-favorite="toggleArcFavorite"
        @toggle-started="toggleSelectedArcStarted"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
      />

      <ArcsBrowseView
        v-else-if="activeView === 'arcs'"
        :arcs="visibleArcs"
        :selected-arc-id="selectedArc?.id"
        :quick-saving-arc-id="quickSavingArcID"
        :search="search"
        :search-term="searchTerm"
        :filter="listOptions.arcs.filter"
        :sort="listOptions.arcs.sort"
        :direction="listOptions.arcs.direction"
        :read-only="isReadOnlyGuest"
        @update:search="updateSearch"
        @update:filter="updateListOption('arcs', 'filter', $event)"
        @update:sort="updateListOption('arcs', 'sort', $event)"
        @update:direction="updateListOption('arcs', 'direction', $event)"
        @new-arc="newArc"
        @open-arc="openArc"
        @toggle-favorite="toggleArcFavorite"
      />

      <SeriesDetailView
        v-else-if="activeView === 'series' && isDetail"
        :selected-series="selectedSeries"
        :selected-comic-id="selectedComic?.id"
        :quick-saving-comic-id="quickSavingComicID"
        :import-running="seriesImportRunning(selectedSeries)"
        :start-saving="seriesStartSaving"
        :read-only="isReadOnlyGuest"
        :can-delete="isAdmin"
        :deleting="seriesDeleting"
        @back="backToPreviousPage"
        @toggle-favorite="toggleSeriesFavorite"
        @toggle-started="toggleSelectedSeriesStarted"
        @import-series="importSelectedSeriesFromMetron"
        @delete="deleteSelectedSeries"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
      />

      <SeriesBrowseView
        v-else-if="activeView === 'series'"
        :loading="seriesListLoading"
        :series="visibleSeries"
        :selected-series-id="selectedSeries?.id"
        :search="search"
        :search-term="searchTerm"
        :filter="listOptions.series.filter"
        :sort="listOptions.series.sort"
        :direction="listOptions.series.direction"
        :read-only="isReadOnlyGuest"
        @update:search="updateSearch"
        @update:filter="updateListOption('series', 'filter', $event)"
        @update:sort="updateListOption('series', 'sort', $event)"
        @update:direction="updateListOption('series', 'direction', $event)"
        @open-series="openSeries"
        @toggle-favorite="toggleSeriesFavorite"
        @new-comic="newComic"
      />

      <CharacterDetailView
        v-else-if="activeView === 'characters' && isDetail"
        :selected-character="selectedCharacter"
        :selected-comic-id="selectedComic?.id"
        :quick-saving-comic-id="quickSavingComicID"
        :quick-saving-character-id="quickSavingCharacterID"
        :import-running="characterImportRunning(selectedCharacter)"
        :start-saving="characterStartSaving"
        :read-only="isReadOnlyGuest"
        :can-delete="isAdmin"
        :deleting="characterDeleting"
        @back="backToPreviousPage"
        @toggle-favorite="toggleCharacterFavorite"
        @toggle-started="toggleSelectedCharacterStarted"
        @import-appearances="importSelectedCharacterAppearances"
        @delete="deleteSelectedCharacter"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
      />

      <CharactersBrowseView
        v-else-if="activeView === 'characters'"
        :characters="visibleCharacters"
        :selected-character-id="selectedCharacter?.id"
        :quick-saving-character-id="quickSavingCharacterID"
        :search="search"
        :search-term="searchTerm"
        :filter="listOptions.characters.filter"
        :sort="listOptions.characters.sort"
        :direction="listOptions.characters.direction"
        :read-only="isReadOnlyGuest"
        @update:search="updateSearch"
        @update:filter="updateListOption('characters', 'filter', $event)"
        @update:sort="updateListOption('characters', 'sort', $event)"
        @update:direction="updateListOption('characters', 'direction', $event)"
        @open-character="openCharacter"
        @toggle-favorite="toggleCharacterFavorite"
      />

      <ComicDetailView
        v-else-if="activeView === 'comics' && isDetail"
        :selected-comic="selectedComic"
        :quick-saving-comic-id="quickSavingComicID"
        :metron-metadata-open="metronMetadataOpen"
        :metron-metadata-searching="metronMetadataSearching"
        :metron-metadata-applying-id="metronMetadataApplyingID"
        :metron-metadata-status="metronMetadataStatus"
        :metron-metadata-results="metronMetadataResults"
        :read-only="isReadOnlyGuest"
        :can-delete="isAdmin"
        :deleting="saving"
        @back="backToPreviousPage"
        @search-metron="searchSelectedComicMetron"
        @apply-metron="applyMetronMetadata"
        @reset-metron="resetMetronMetadata"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
        @edit="editComic"
        @delete="deleteComic"
        @open-character="openCharacter"
        @open-series="openSeries"
      />

      <div v-else class="browse-view comic-browse-view">
        <ComicListView
          title="Comics"
          :comics="comics"
          :total-count="listTotal('comics')"
          :search="search"
          :status="listOptions.comics.status"
          :sort="listOptions.comics.sort"
          :direction="listOptions.comics.direction"
          server-search
          :selected-comic-id="selectedComic?.id"
          :quick-saving-comic-id="quickSavingComicID"
          :read-only="isReadOnlyGuest"
          show-cover
          empty-message="No comics yet."
          filtered-empty-message="No comics match these filters."
          @update:search="updateSearch"
          @update:status="updateListOption('comics', 'status', $event)"
          @update:sort="updateListOption('comics', 'sort', $event)"
          @update:direction="updateListOption('comics', 'direction', $event)"
          @open-comic="openComic"
          @toggle-read="toggleComicRead"
          @toggle-skipped="toggleComicSkipped"
        />
      </div>

      <div
        v-if="showInfiniteScrollSentinel"
        ref="loadMoreSentinel"
        class="load-more-sentinel"
        aria-live="polite"
      >
        <span v-if="activeListLoadingMore" class="loading-spinner small" aria-hidden="true"></span>
        <span>{{ activeListLoadingMore ? 'Loading more...' : 'Scroll for more' }}</span>
      </div>
    </section>
  </main>
</template>
