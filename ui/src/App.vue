<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppGlobalOverlays from '@/app/components/AppGlobalOverlays.vue'
import AccountView from '@/features/account/components/AccountView.vue'
import AuthenticationView from '@/features/auth/components/AuthenticationView.vue'
import AppSettingsView from '@/features/settings/components/AppSettingsView.vue'
import ArcDetailView from '@/features/arcs/components/ArcDetailView.vue'
import ArcsBrowseView from '@/features/arcs/components/ArcsBrowseView.vue'
import AppSidebar from '@/app/components/AppSidebar.vue'
import AppToolbar from '@/app/components/AppToolbar.vue'
import InfiniteScrollStatus from '@/app/components/InfiniteScrollStatus.vue'
import CharactersBrowseView from '@/features/characters/components/CharactersBrowseView.vue'
import CharacterDetailView from '@/features/characters/components/CharacterDetailView.vue'
import AddToCollectionDialog from '@/features/collections/components/AddToCollectionDialog.vue'
import CollectionDetailView from '@/features/collections/components/CollectionDetailView.vue'
import CollectionsBrowseView from '@/features/collections/components/CollectionsBrowseView.vue'
import ComicDetailView from '@/features/comics/components/ComicDetailView.vue'
import ComicListView from '@/features/comics/components/ComicListView.vue'
import DashboardView from '@/features/dashboard/components/DashboardView.vue'
import EmptyState from '@/shared/components/feedback/EmptyState.vue'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'
import MetronImport from '@/features/metron/components/MetronImport.vue'
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
import { useCharacterCollections } from '@/features/collections/useCharacterCollections.js'
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
import { useAppController } from '@/app/useAppController.js'
import { useAppSearchState } from '@/app/useAppSearchState.js'
import { useInfiniteScroll } from '@/app/useInfiniteScroll.js'

const activeView = ref('dashboard')
const viewMode = ref('browse')
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const systemInfo = ref(null)
const route = useRoute()
const router = useRouter()
const isEditing = computed(() => viewMode.value === 'edit')
const isDetail = computed(() => viewMode.value === 'detail')
let appController
let paginationPageState

function applyCurrentRoute(...args) {
  return appController?.applyCurrentRoute(...args)
}

function refreshActiveListData(...args) {
  return appController?.refreshActiveListData(...args)
}

function refreshActiveLibraryData(...args) {
  return appController?.refreshActiveLibraryData(...args)
}

function handleMetronImported(...args) {
  return appController?.handleMetronImported(...args)
}

function loadMoreActiveViewData(...args) {
  return appController?.loadMoreActiveViewData(...args)
}

function backToPreviousPage(...args) {
  return appController?.backToPreviousPage(...args)
}

function cancelEdit(...args) {
  return appController?.cancelEdit(...args)
}

function showError(...args) {
  return appController?.showError(...args)
}

function clearError(...args) {
  return appController?.clearError(...args)
}

const { search, searchTerm, updateSearch } = useAppSearchState({
  activeView,
  isEditing,
  isDetail,
  getPageState: () => paginationPageState?.value,
  onRefresh: refreshActiveListData,
})
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
  auditPagination,
  auditLoading,
  savingPermissionsUserId: savingUserID,
  savingAdminUserId: savingAdminUserID,
  deletingUserId: deletingUserID,
  loadUsers: loadUserAdminRows,
  loadAuditEvents,
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
  cblRepositorySync,
  cblRepositoryFiles,
  savingComicScan: savingMetronComicScan,
  savingComicDiscovery: savingMetronComicDiscovery,
  savingCBLRepositorySync,
  loadingCBLRepositoryFiles,
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
  saveCBLRepositorySync,
  loadCBLRepositoryFiles,
  runCBLRepositorySync,
  cancelCBLRepositorySync,
  resolveCBLMetronIssue,
  generateInvite: generateUserInvite,
  saveRegistration: saveRegistrationMode,
  savePublic: savePublicAccess,
} = useMetronSettings({ activeView, error, userStatus, registrationMode, publicAccess })
const { listOptions, activeListParams, updateListOption } = useListOptions({
  activeView,
  onChange: refreshActiveListData,
})

const { pageState, showInfiniteScrollSentinel, activeListLoadingMore, listTotal, loadPagedList } =
  usePagination({
    activeView,
    loading,
    isEditing,
    isDetail,
    searchTerm,
    listParams: activeListParams,
  })
paginationPageState = pageState

const { sentinel: loadMoreSentinel } = useInfiniteScroll({
  activeView,
  enabled: showInfiniteScrollSentinel,
  onLoadMore: loadMoreActiveViewData,
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
  loading,
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
  loading,
  error,
  loadPagedList,
  metronImportJobs,
  trackMetronImportJob,
})

const {
  collections: characterCollections,
  selectedCollection,
  saving: collectionSaving,
  startSaving: collectionStartSaving,
  addDialogCharacter,
  addDialogCollections,
  addDialogLoading,
  loadCollections: loadCharacterCollections,
  openCollection,
  createCollection,
  toggleSelectedCollectionStarted,
  deleteSelectedCollection,
  addCharacter: addCharacterToSelectedCollection,
  removeCharacter: removeCharacterFromSelectedCollection,
  openAddDialog: openAddToCollection,
  closeAddDialog: closeAddToCollection,
  addDialogCharacterTo,
  createAndAddDialogCollection,
} = useCharacterCollections({ activeView, viewMode, loading, error })

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
  loading,
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
  loading,
  error,
  saving,
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
  metronMergeConflict,
  metronMergeSaving,
  mergeOpen: comicMergeOpen,
  mergeCandidates: comicMergeCandidates,
  mergeSearching: comicMergeSearching,
  mergeSaving: comicMergeSaving,
  loadComics,
  openComic,
  openOrderComic,
  newComic,
  editComic,
  deleteComic,
  openComicMerge,
  closeComicMerge,
  searchComicMerge,
  mergeSelectedComic,
  toggleComicRead,
  toggleComicSkipped,
  resetMetronMetadata,
  searchSelectedComicMetron,
  applyMetronMetadata,
  mergeMetronConflict,
  clearMetronMergeConflict,
} = useComics({
  activeView,
  viewMode,
  loading,
  error,
  saving,
  loadPagedList,
  refreshActiveLibraryData,
  readingOrderProgress,
  selectedOrder,
  selectedArc,
  selectedCharacter,
  selectedSeries,
  selectedCollection,
  collectionProgress: arcProgress,
})
const {
  dashboard,
  loading: dashboardLoading,
  loadDashboard,
  markComicRead: markDashboardComicRead,
  markComicSkipped: markDashboardComicSkipped,
} = useDashboard({ error, quickSavingComicId: quickSavingComicID })

function openDashboardItem(item) {
  if (item?.type === 'readingOrder') return openReadingOrder(item)
  if (item?.type === 'arc') return openArc(item)
  if (item?.type === 'character') return openCharacter(item)
  if (item?.type === 'characterCollection') return openCollection(item)
  if (item?.type === 'series') return openSeries(item)
}

const toolbarResultCount = computed(() => {
  if (activeView.value === 'dashboard') return dashboard.value?.items?.length || 0
  if (activeView.value === 'readingOrders') return visibleOrders.value.length
  if (activeView.value === 'arcs') return visibleArcs.value.length
  if (activeView.value === 'comics') return comics.value.length
  if (activeView.value === 'series') return visibleSeries.value.length
  if (activeView.value === 'characters') return visibleCharacters.value.length
  if (activeView.value === 'collections') return characterCollections.value.length
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
  if (activeView.value === 'collections') return characterCollections.value.length
  if (activeView.value === 'users') return userAdminRows.value.length
  if (activeView.value === 'account' || activeView.value === 'progress') return 1
  return 0
})
const showBlockingLoading = computed(() => loading.value)
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

appController = useAppController({
  state: { activeView, viewMode, loading, error, systemInfo, isEditing, isDetail },
  routing: { route, router },
  auth: {
    userStatus,
    isAdmin,
    currentUser,
    isReadOnlyGuest,
    appReady,
    loadUserStatus,
    verifyEmailFromRouteToken,
    preparePasswordResetFromRouteToken,
  },
  access: { canAccessMetronArea, canMetronMonitor },
  entities: {
    selectedOrder,
    selectedArc,
    selectedComic,
    selectedSeries,
    selectedCharacter,
    selectedCollection,
  },
  forms: { orderForm, arcForm, comicForm },
  actions: {
    newReadingOrder,
    openReadingOrder,
    editReadingOrder,
    newArc,
    openArc,
    editArc,
    newComic,
    openComic,
    editComic,
    openSeries,
    openCharacter,
    openCollection,
    clearMetronMergeConflict,
    refreshSelectedReadingOrderDetail,
    refreshSelectedCharacterDetail,
    refreshSelectedSeriesDetail,
  },
  loaders: {
    dashboard: loadDashboard,
    readingOrders: loadReadingOrders,
    arcs: loadArcs,
    comics: loadComics,
    series: loadSeries,
    characters: loadCharacters,
    collections: loadCharacterCollections,
    users: loadUserAdminRows,
    settings: loadMetronSettings,
    progress: loadAccountStatistics,
  },
  metron: { loadMetronImportJobs, loadMetronQuota, closeMetronImportEvents },
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

      <AppGlobalOverlays
        v-model:open="metronImportMonitorOpen"
        :error="error"
        :is-admin="isAdmin"
        :conflict-message="metronMergeConflict?.message || ''"
        :merge-saving="metronMergeSaving"
        :jobs="metronImportJobs"
        @merge="mergeMetronConflict"
        @dismiss-error="clearError"
        @retry="retryMetronJob"
        @continue="continueMetronJob"
        @cancel="cancelMetronJob"
        @dismiss-job="dismissMetronJob"
      />

      <LoadingState v-if="showBlockingLoading" />

      <DashboardView
        v-else-if="activeView === 'dashboard'"
        :dashboard="dashboard"
        :loading="dashboardLoading"
        :quick-saving-comic-id="quickSavingComicID"
        :read-only="isReadOnlyGuest"
        @refresh="loadDashboard"
        @open-comic="openComic"
        @open-item="openDashboardItem"
        @mark-read="markDashboardComicRead"
        @mark-skipped="markDashboardComicSkipped"
      />

      <UserManagementView
        v-else-if="activeView === 'users'"
        :users="userAdminRows"
        :audit-events="auditEvents"
        :audit-pagination="auditPagination"
        :audit-loading="auditLoading"
        :saving-user-id="savingUserID"
        :saving-admin-user-id="savingAdminUserID"
        :deleting-user-id="deletingUserID"
        :current-user-id="currentUser?.id"
        @save="saveUserMetronPermissions"
        @save-admin="saveUserAdmin"
        @delete-user="removeUser"
        @load-audit="loadAuditEvents"
      />

      <AppSettingsView
        v-else-if="activeView === 'settings'"
        :metron-comic-scan="metronComicScan"
        :metron-comic-discovery="metronComicDiscovery"
        :cbl-repository-sync="cblRepositorySync"
        :cbl-repository-files="cblRepositoryFiles"
        :saving="savingMetronComicScan"
        :saving-discovery="savingMetronComicDiscovery"
        :saving-cbl-repository-sync="savingCBLRepositorySync"
        :loading-cbl-repository-files="loadingCBLRepositoryFiles"
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
        @save-cbl-repository-sync="saveCBLRepositorySync"
        @load-cbl-repository-files="loadCBLRepositoryFiles"
        @trigger-cbl-repository-sync="runCBLRepositorySync"
        @stop-cbl-repository-sync="cancelCBLRepositorySync"
        @resolve-cbl-metron-issue="resolveCBLMetronIssue"
        @update-registration-mode="saveRegistrationMode"
        @update-public-access="savePublicAccess"
        @generate-invite="generateUserInvite"
      >
        <template #metron-import>
          <MetronImport
            :import-jobs="metronImportJobs"
            :metron-quota="metronQuota"
            @imported="handleMetronImported"
            @error="showError"
            @job-started="trackMetronImportJob"
            @quota-updated="updateMetronQuota"
          />
        </template>
      </AppSettingsView>

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

      <EmptyState v-else-if="activeView === 'notFound'" tag="section" roomy>
        <h2>Page not found</h2>
        <p>This route does not match a ComicHero view.</p>
        <router-link class="primary-button" :to="{ name: 'readingOrders' }">
          Go to reading orders
        </router-link>
      </EmptyState>

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
        @add-to-collection="openAddToCollection"
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
        @add-to-collection="openAddToCollection"
      />

      <CollectionDetailView
        v-else-if="activeView === 'collections' && isDetail"
        :collection="selectedCollection"
        :selected-comic-id="selectedComic?.id"
        :quick-saving-comic-id="quickSavingComicID"
        :saving="collectionSaving"
        :start-saving="collectionStartSaving"
        @back="backToPreviousPage"
        @toggle-started="toggleSelectedCollectionStarted"
        @delete="deleteSelectedCollection"
        @add-character="addCharacterToSelectedCollection"
        @remove-character="removeCharacterFromSelectedCollection"
        @open-character="openCharacter"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
      />

      <CollectionsBrowseView
        v-else-if="activeView === 'collections'"
        :collections="characterCollections"
        :saving="collectionSaving"
        @create="createCollection"
        @open="openCollection"
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
        :merge-open="comicMergeOpen"
        :merge-candidates="comicMergeCandidates"
        :merge-searching="comicMergeSearching"
        :merge-saving="comicMergeSaving"
        @back="backToPreviousPage"
        @search-metron="searchSelectedComicMetron"
        @apply-metron="applyMetronMetadata"
        @reset-metron="resetMetronMetadata"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
        @edit="editComic"
        @delete="deleteComic"
        @open-merge="openComicMerge"
        @close-merge="closeComicMerge"
        @search-merge="searchComicMerge"
        @merge="mergeSelectedComic"
        @open-character="openCharacter"
        @open-series="openSeries"
      />

      <div
        v-else
        class="browse-view comic-browse-view mt-[-24px] down-mobile:mt-[-12px] min-w-0 w-full"
      >
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

      <InfiniteScrollStatus
        v-if="showInfiniteScrollSentinel"
        ref="loadMoreSentinel"
        :loading="activeListLoadingMore"
      />

      <AddToCollectionDialog
        v-if="addDialogCharacter"
        :character="addDialogCharacter"
        :collections="addDialogCollections"
        :loading="addDialogLoading"
        :saving="collectionSaving"
        @close="closeAddToCollection"
        @add="addDialogCharacterTo"
        @create="createAndAddDialogCollection"
      />
    </section>
  </main>
</template>

<style scoped>
@reference 'styles.css';

.app-shell {
  @apply min-h-screen grid grid-cols-[260px_minmax(0,1fr)] down-tablet:block down-tablet:grid-cols-1;
}

.content {
  @apply [--content-padding:28px] [--sticky-toolbar-top:0px] [--sticky-toolbar-inline-offset:28px] [--comic-list-sticky-top:82px] p-(--content-padding) min-w-0 w-full down-tablet:[--content-padding:22px] down-tablet:[--sticky-toolbar-top:65px] down-tablet:[--sticky-toolbar-inline-offset:22px] down-tablet:[--comic-list-sticky-top:146px] down-tablet:max-w-none down-tablet:p-(--content-padding) down-mobile:[--content-padding:14px] down-mobile:[--sticky-toolbar-top:0px] down-mobile:[--sticky-toolbar-inline-offset:14px] down-mobile:[--comic-list-sticky-top:0px] down-mobile:p-(--content-padding) down-phone:p-2.5 *:min-w-0 *:max-w-full [&_>_.sticky-toolbar]:max-w-none;
}

.primary-button {
  @apply min-h-10 border rounded py-2.5 px-3.5 border-primary bg-primary text-white;
}
</style>
