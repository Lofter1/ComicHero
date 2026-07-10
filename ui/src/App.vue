<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AccountView from '@/components/AccountView.vue'
import ArcDetailView from '@/components/ArcDetailView.vue'
import ArcsBrowseView from '@/components/ArcsBrowseView.vue'
import AppSidebar from '@/components/AppSidebar.vue'
import AppToolbar from '@/components/AppToolbar.vue'
import CharactersBrowseView from '@/components/CharactersBrowseView.vue'
import CharacterDetailView from '@/components/CharacterDetailView.vue'
import ComicDetailView from '@/components/ComicDetailView.vue'
import ComicListView from '@/components/ComicListView.vue'
import MetronImport from '@/components/MetronImport.vue'
import MetronImportMonitor from '@/components/MetronImportMonitor.vue'
import ProgressView from '@/components/ProgressView.vue'
import ReadingOrderDetailView from '@/components/ReadingOrderDetailView.vue'
import ReadingOrderEditView from '@/components/ReadingOrderEditView.vue'
import ReadingOrdersBrowseView from '@/components/ReadingOrdersBrowseView.vue'
import SeriesBrowseView from '@/components/SeriesBrowseView.vue'
import SeriesDetailView from '@/components/SeriesDetailView.vue'
import UserManagementView from '@/components/UserManagementView.vue'
import { useArcs } from '@/composables/useArcs.js'
import { useCharacters } from '@/composables/useCharacters.js'
import { useComics } from '@/composables/useComics.js'
import { useMetronJobs } from '@/composables/useMetronJobs.js'
import { usePagination } from '@/composables/usePagination.js'
import { useReadingOrders } from '@/composables/useReadingOrders.js'
import { useSeries } from '@/composables/useSeries.js'
import { routeAccessRedirect, setRouteAccessContext } from '@/router/index.js'
import {
  createUserInvite,
  deleteAccount as deleteAccountRequest,
  deleteUser as deleteUserRequest,
  getAccountStatistics,
  getUserStatus,
  listUsers,
  loginUser,
  logoutUser,
  registerUser,
  resendEmailVerification,
  requestPasswordReset,
  resetPassword,
  setupUsers,
  updateAccount,
  updatePublicAccess,
  updateRegistrationMode,
  updateUserAdmin,
  updateUserMetronPermissions,
  verifyEmail,
} from '@/api/client.js'

const activeView = ref('readingOrders')
const viewMode = ref('browse')
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const authLoading = ref(true)
const authSaving = ref(false)
const userStatus = ref(null)
const authMode = ref('login')
const loginRequested = ref(false)
const setupForm = ref({ mode: 'single', name: '', email: '', password: '' })
const authForm = ref({
  name: '',
  email: '',
  emailConfirmation: '',
  password: '',
  passwordConfirmation: '',
  inviteToken: '',
})
const verificationForm = ref({ token: '', email: '', password: '' })
const passwordResetForm = ref({
  email: '',
  token: '',
  password: '',
  passwordConfirmation: '',
  requested: false,
  completed: false,
})
const userAdminRows = ref([])
const generatedInvite = ref(null)
const accountSaving = ref(false)
const accountDeleting = ref(false)
const accountStatistics = ref(null)
const accountStatisticsLoading = ref(false)
const accountStatisticsError = ref('')
const savingUserID = ref(null)
const savingAdminUserID = ref(null)
const deletingUserID = ref(null)
const generatingInvite = ref(false)
const savingRegistrationMode = ref(false)
const savingPublicAccess = ref(false)
const search = ref('')
const defaultListOptions = {
  readingOrders: { filter: 'all', sort: 'name', direction: 'asc' },
  arcs: { filter: 'all', sort: 'name', direction: 'asc' },
  comics: { status: 'all', sort: 'series', direction: 'asc' },
  series: { filter: 'all', sort: 'name', direction: 'asc' },
  characters: { filter: 'all', sort: 'name', direction: 'asc' },
}
const listOptionsStorageKey = 'comichero-list-options-v2'
const listOptions = ref(getInitialListOptions())
const loadMoreSentinel = ref(null)
const themePreference = ref(getInitialThemePreference())
const systemTheme = ref(getSystemTheme())
let loadMoreObserver = null
let searchDebounceTimer = null
let themeMediaQuery = null
let routeSyncPaused = false
const route = useRoute()
const router = useRouter()

const searchTerm = computed(() => search.value.trim().toLowerCase())
const activeListParams = computed(() => {
  const options = listOptions.value[activeView.value] || {}
  const params = {}
  if (options.filter === 'favorites') params.favorite = true
  if (options.filter === 'other') params.favorite = false
  if (options.filter === 'started') params.started = true
  if (options.status && options.status !== 'all') params.status = options.status
  if (options.sort) params.sort = options.sort
  if (options.direction) params.direction = options.direction
  return params
})
const isEditing = computed(() => viewMode.value === 'edit')
const isDetail = computed(() => viewMode.value === 'detail')
const resolvedTheme = computed(() =>
  themePreference.value === 'system' ? systemTheme.value : themePreference.value,
)
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
  visibleSeries,
  seriesBrowseSections,
  openSeries,
  toggleSeriesFavorite,
  toggleSelectedSeriesStarted,
  importSelectedSeriesFromMetron,
  seriesImportRunning,
  refreshSelectedSeriesDetail,
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
  visibleCharacters,
  characterBrowseSections,
  openCharacter,
  toggleCharacterFavorite,
  toggleSelectedCharacterStarted,
  importSelectedCharacterAppearances,
  characterImportRunning,
  refreshSelectedCharacterDetail,
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
  arcBrowseSections,
  arcProgress,
  openArc,
  toggleArcFavorite,
  toggleSelectedArcStarted,
  newArc,
  editArc,
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
  readingOrderBrowseSections,
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

const toolbarResultCount = computed(() => {
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
const setupRequired = computed(() => Boolean(userStatus.value?.setupRequired))
const userMode = computed(() => userStatus.value?.mode || '')
const registrationMode = computed(() => userStatus.value?.registrationMode || 'invite_only')
const publicAccess = computed(() => Boolean(userStatus.value?.publicAccess))
const currentUser = computed(() => userStatus.value?.user || null)
const isAdmin = computed(() => Boolean(currentUser.value?.isAdmin))
const isReadOnlyGuest = computed(
  () => userMode.value === 'multi' && publicAccess.value && !currentUser.value,
)
const metronPermissions = computed(() => userStatus.value?.metronPermissions || {})
const canMetronSearch = computed(() => hasMetronScope('search'))
const canMetronMonitor = computed(() => hasMetronScope('monitor'))
const canAccessMetronArea = computed(() => canMetronSearch.value)
const authRequired = computed(
  () =>
    userMode.value === 'multi' &&
    !currentUser.value &&
    !emailVerificationRequired.value &&
    (!publicAccess.value || loginRequested.value),
)
const emailVerificationRequired = computed(() =>
  Boolean(userStatus.value?.emailVerificationRequired),
)
const passwordResetMode = computed(() => authMode.value === 'forgot')
const appReady = computed(
  () =>
    Boolean(userStatus.value) &&
    !authLoading.value &&
    !setupRequired.value &&
    !authRequired.value &&
    !emailVerificationRequired.value &&
    !passwordResetMode.value,
)

function hasMetronScope(scope) {
  const permissions = metronPermissions.value
  if (!permissions.allowed) return false
  const scopes = Array.isArray(permissions.scopes) ? permissions.scopes : []
  return scopes.includes('*') || scopes.includes(scope)
}

function getInitialThemePreference() {
  if (typeof window === 'undefined') return 'system'
  const savedTheme = window.localStorage.getItem('comichero-theme')
  if (savedTheme === 'dark' || savedTheme === 'light' || savedTheme === 'system') return savedTheme
  return 'system'
}

function getInitialListOptions() {
  if (typeof window === 'undefined') return cloneDefaultListOptions()
  try {
    return mergeListOptions(JSON.parse(window.localStorage.getItem(listOptionsStorageKey) || '{}'))
  } catch {
    return cloneDefaultListOptions()
  }
}

function cloneDefaultListOptions() {
  return mergeListOptions({})
}

function mergeListOptions(savedOptions) {
  return Object.fromEntries(
    Object.entries(defaultListOptions).map(([view, defaults]) => {
      const saved = savedOptions?.[view]
      return [view, { ...defaults, ...(saved && typeof saved === 'object' ? saved : {}) }]
    }),
  )
}

function getSystemTheme() {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function applyTheme(value) {
  if (typeof document === 'undefined') return
  document.documentElement.dataset.theme = value
  document.documentElement.style.colorScheme = value
}

function setThemePreference(value) {
  themePreference.value = value
}

async function loadUserStatus() {
  authLoading.value = true
  error.value = ''
  try {
    userStatus.value = await getUserStatus()
  } catch (err) {
    error.value = err.message
  } finally {
    authLoading.value = false
  }
}

async function submitSetup() {
  authSaving.value = true
  error.value = ''
  try {
    const payload = { mode: setupForm.value.mode }
    if (setupForm.value.mode === 'multi') {
      payload.name = setupForm.value.name
      payload.email = setupForm.value.email
      payload.password = setupForm.value.password
    }
    userStatus.value = await setupUsers(payload)
    if (!authRequired.value) {
      await applyCurrentRoute({ replace: true, force: true })
    }
  } catch (err) {
    error.value = err.message
  } finally {
    authSaving.value = false
  }
}

async function submitAuth() {
  authSaving.value = true
  error.value = ''
  try {
    if (authMode.value === 'register') {
      if (authForm.value.email !== authForm.value.emailConfirmation) {
        throw new Error('Email confirmation must match email.')
      }
      if (authForm.value.password !== authForm.value.passwordConfirmation) {
        throw new Error('Password confirmation must match password.')
      }
    }
    const payload = { email: authForm.value.email, password: authForm.value.password }
    if (authMode.value === 'register') {
      payload.name = authForm.value.name
      payload.emailConfirmation = authForm.value.emailConfirmation
      payload.passwordConfirmation = authForm.value.passwordConfirmation
    }
    if (authMode.value === 'register' && registrationMode.value === 'invite_only') {
      payload.inviteToken = authForm.value.inviteToken
    }
    userStatus.value =
      authMode.value === 'register' ? await registerUser(payload) : await loginUser(payload)
    if (userStatus.value?.emailVerificationRequired) {
      verificationForm.value.email = userStatus.value.emailVerificationEmail || authForm.value.email
      verificationForm.value.password = authForm.value.password
      return
    }
    loginRequested.value = false
    await applyCurrentRoute({ replace: true, force: true })
  } catch (err) {
    error.value = err.message
  } finally {
    authSaving.value = false
  }
}

async function submitEmailVerification() {
  authSaving.value = true
  error.value = ''
  try {
    userStatus.value = await verifyEmail({ token: verificationForm.value.token })
    verificationForm.value.token = ''
    loginRequested.value = false
    await applyCurrentRoute({ replace: true, force: true })
  } catch (err) {
    error.value = err.message
  } finally {
    authSaving.value = false
  }
}

async function resendVerificationEmail() {
  authSaving.value = true
  error.value = ''
  try {
    userStatus.value = await resendEmailVerification({
      email: verificationForm.value.email || userStatus.value?.emailVerificationEmail,
      password: verificationForm.value.password,
    })
  } catch (err) {
    error.value = err.message
  } finally {
    authSaving.value = false
  }
}

async function verifyEmailFromRouteToken() {
  const token = typeof route.query.token === 'string' ? route.query.token.trim() : ''
  if (!token) return
  verificationForm.value.token = token
  await submitEmailVerification()
}

function showForgotPassword() {
  authMode.value = 'forgot'
  passwordResetForm.value.email = authForm.value.email
  passwordResetForm.value.token = ''
  passwordResetForm.value.password = ''
  passwordResetForm.value.passwordConfirmation = ''
  passwordResetForm.value.requested = false
  passwordResetForm.value.completed = false
  error.value = ''
}

function showLogin() {
  authMode.value = 'login'
  loginRequested.value = true
  error.value = ''
}

async function submitForgotPassword() {
  authSaving.value = true
  error.value = ''
  try {
    userStatus.value = await requestPasswordReset({ email: passwordResetForm.value.email })
    passwordResetForm.value.requested = true
  } catch (err) {
    error.value = err.message
  } finally {
    authSaving.value = false
  }
}

async function submitPasswordReset() {
  authSaving.value = true
  error.value = ''
  try {
    if (passwordResetForm.value.password !== passwordResetForm.value.passwordConfirmation) {
      throw new Error('Password confirmation must match password.')
    }
    userStatus.value = await resetPassword({
      token: passwordResetForm.value.token,
      password: passwordResetForm.value.password,
      passwordConfirmation: passwordResetForm.value.passwordConfirmation,
    })
    passwordResetForm.value.completed = true
    authMode.value = 'login'
    authForm.value.email = passwordResetForm.value.email
    authForm.value.password = ''
  } catch (err) {
    error.value = err.message
  } finally {
    authSaving.value = false
  }
}

function preparePasswordResetFromRouteToken() {
  const token = typeof route.query.token === 'string' ? route.query.token.trim() : ''
  if (!token) return
  authMode.value = 'forgot'
  loginRequested.value = true
  passwordResetForm.value.token = token
  passwordResetForm.value.requested = true
  passwordResetForm.value.completed = false
}

async function loadUserAdminRows() {
  userAdminRows.value = await listUsers()
}

async function generateUserInvite() {
  generatingInvite.value = true
  error.value = ''
  try {
    generatedInvite.value = await createUserInvite()
  } catch (err) {
    error.value = err.message
  } finally {
    generatingInvite.value = false
  }
}

async function saveRegistrationMode(mode) {
  if (mode === registrationMode.value) return
  if (
    mode === 'open' &&
    registrationMode.value !== 'open' &&
    typeof window !== 'undefined' &&
    !window.confirm(
      'Anyone who can reach this server will be able to create an account with full read/write access to your shared library. Only enable open registration if you understand the risk.',
    )
  ) {
    return
  }

  savingRegistrationMode.value = true
  error.value = ''
  try {
    userStatus.value = await updateRegistrationMode({ mode })
  } catch (err) {
    error.value = err.message
  } finally {
    savingRegistrationMode.value = false
  }
}

async function savePublicAccess(enabled) {
  if (enabled === publicAccess.value) return
  if (
    enabled &&
    typeof window !== 'undefined' &&
    !window.confirm(
      'Anonymous visitors will be able to browse your shared library and export reading lists as CBL. They will not be able to edit data.',
    )
  ) {
    return
  }

  savingPublicAccess.value = true
  error.value = ''
  try {
    userStatus.value = await updatePublicAccess({ enabled })
  } catch (err) {
    error.value = err.message
  } finally {
    savingPublicAccess.value = false
  }
}

async function saveUserMetronPermissions(userID, payload) {
  savingUserID.value = userID
  error.value = ''
  try {
    const updated = await updateUserMetronPermissions(userID, payload)
    userAdminRows.value = userAdminRows.value.map((entry) =>
      entry.user.id === userID ? updated : entry,
    )
  } catch (err) {
    error.value = err.message
  } finally {
    savingUserID.value = null
  }
}

async function saveUserAdmin(userID, payload) {
  savingAdminUserID.value = userID
  error.value = ''
  try {
    const updated = await updateUserAdmin(userID, payload)
    userAdminRows.value = userAdminRows.value.map((entry) =>
      entry.user.id === userID ? updated : entry,
    )
  } catch (err) {
    error.value = err.message
  } finally {
    savingAdminUserID.value = null
  }
}

async function removeUser(userID) {
  if (
    typeof window !== 'undefined' &&
    !window.confirm(
      'Delete this account? Their sessions, read status, Metron permissions, and account data will be removed.',
    )
  ) {
    return
  }

  deletingUserID.value = userID
  error.value = ''
  try {
    await deleteUserRequest(userID)
    userAdminRows.value = userAdminRows.value.filter((entry) => entry.user.id !== userID)
  } catch (err) {
    error.value = err.message
  } finally {
    deletingUserID.value = null
  }
}

async function saveAccount(payload, validationMessage = '') {
  if (validationMessage) {
    error.value = validationMessage
    return
  }
  if (!payload) return

  accountSaving.value = true
  error.value = ''
  try {
    userStatus.value = await updateAccount(payload)
  } catch (err) {
    error.value = err.message
  } finally {
    accountSaving.value = false
  }
}

async function deleteCurrentAccount(payload) {
  accountDeleting.value = true
  error.value = ''
  try {
    activeView.value = 'readingOrders'
    viewMode.value = 'browse'
    userStatus.value = await deleteAccountRequest(payload)
    authMode.value = 'login'
  } catch (err) {
    error.value = err.message
  } finally {
    accountDeleting.value = false
  }
}

async function loadAccountStatistics() {
  if (!currentUser.value) {
    accountStatistics.value = null
    accountStatisticsError.value = ''
    return
  }

  accountStatisticsLoading.value = true
  accountStatisticsError.value = ''
  try {
    accountStatistics.value = await getAccountStatistics()
  } catch (err) {
    accountStatisticsError.value = err.message
  } finally {
    accountStatisticsLoading.value = false
  }
}

async function signOut() {
  authSaving.value = true
  error.value = ''
  try {
    await logoutUser()
    userStatus.value = await getUserStatus()
    loginRequested.value = false
  } catch (err) {
    error.value = err.message
  } finally {
    authSaving.value = false
  }
}

function requestLogin() {
  authMode.value = 'login'
  loginRequested.value = true
}

function handleSystemThemeChange(event) {
  systemTheme.value = event.matches ? 'dark' : 'light'
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

function parseRouteID(value) {
  const id = Number(value)
  return Number.isInteger(id) && id > 0 ? id : null
}

function entityRouteState(routeView, detailMode = 'detail') {
  const id = parseRouteID(route.params.id)
  if (!id) return { view: routeView, mode: 'browse', replace: true }
  return { view: routeView, mode: detailMode, id }
}

function routeToAppState() {
  if (route.name === 'readingOrders') return { view: 'readingOrders', mode: 'browse' }
  if (route.name === 'readingOrdersNew') return { view: 'readingOrders', mode: 'edit', isNew: true }
  if (route.name === 'readingOrderDetail') return entityRouteState('readingOrders')
  if (route.name === 'readingOrderEdit') return entityRouteState('readingOrders', 'edit')
  if (route.name === 'arcs') return { view: 'arcs', mode: 'browse' }
  if (route.name === 'arcsNew') return { view: 'arcs', mode: 'edit', isNew: true }
  if (route.name === 'arcDetail') return entityRouteState('arcs')
  if (route.name === 'arcEdit') return entityRouteState('arcs', 'edit')
  if (route.name === 'comics') return { view: 'comics', mode: 'browse' }
  if (route.name === 'comicsNew') return { view: 'comics', mode: 'edit', isNew: true }
  if (route.name === 'comicDetail') return entityRouteState('comics')
  if (route.name === 'comicEdit') return entityRouteState('comics', 'edit')
  if (route.name === 'series') return { view: 'series', mode: 'browse' }
  if (route.name === 'seriesDetail') return entityRouteState('series')
  if (route.name === 'characters') return { view: 'characters', mode: 'browse' }
  if (route.name === 'characterDetail') return entityRouteState('characters')
  if (route.name === 'metron') return { view: 'metron', mode: 'browse' }
  if (route.name === 'users') return { view: 'users', mode: 'browse' }
  if (route.name === 'account') return { view: 'account', mode: 'browse' }
  if (route.name === 'progress') return { view: 'progress', mode: 'browse' }
  if (route.name === 'notFound') return { view: 'notFound', mode: 'browse' }
  return { view: 'readingOrders', mode: 'browse', replace: true }
}

function browseRouteLocation(view) {
  if (view === 'readingOrders') return { name: 'readingOrders' }
  if (view === 'arcs') return { name: 'arcs' }
  if (view === 'comics') return { name: 'comics' }
  if (view === 'series') return { name: 'series' }
  if (view === 'characters') return { name: 'characters' }
  if (view === 'metron') return { name: 'metron' }
  if (view === 'users') return { name: 'users' }
  if (view === 'account') return { name: 'account' }
  if (view === 'progress') return { name: 'progress' }
  return null
}

function detailRouteLocation(view, id) {
  if (!id) return null
  if (view === 'readingOrders') return { name: 'readingOrderDetail', params: { id } }
  if (view === 'arcs') return { name: 'arcDetail', params: { id } }
  if (view === 'comics') return { name: 'comicDetail', params: { id } }
  if (view === 'series') return { name: 'seriesDetail', params: { id } }
  if (view === 'characters') return { name: 'characterDetail', params: { id } }
  return null
}

function editRouteLocation(view, id) {
  if (view === 'readingOrders') {
    return id ? { name: 'readingOrderEdit', params: { id } } : { name: 'readingOrdersNew' }
  }
  if (view === 'arcs') return id ? { name: 'arcEdit', params: { id } } : { name: 'arcsNew' }
  if (view === 'comics') return id ? { name: 'comicEdit', params: { id } } : { name: 'comicsNew' }
  return browseRouteLocation(view)
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
  await applyRoute(routeToAppState(), options)
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
  search.value = value
}

function updateListOption(view, key, value) {
  listOptions.value = {
    ...listOptions.value,
    [view]: {
      ...(listOptions.value[view] || {}),
      [key]: value,
    },
  }
  refreshActiveListData()
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
  if (typeof window !== 'undefined') {
    themeMediaQuery = window.matchMedia?.('(prefers-color-scheme: dark)')
    themeMediaQuery?.addEventListener('change', handleSystemThemeChange)
  }
  await loadUserStatus()
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

watch(
  resolvedTheme,
  (value) => {
    applyTheme(value)
  },
  { immediate: true },
)

watch(
  themePreference,
  (value) => {
    if (typeof window === 'undefined') return
    window.localStorage.setItem('comichero-theme', value)
  },
  { immediate: true },
)

watch(
  listOptions,
  (value) => {
    if (typeof window === 'undefined') return
    window.localStorage.setItem(listOptionsStorageKey, JSON.stringify(value))
  },
  { deep: true },
)

watch(search, () => {
  if (searchDebounceTimer) {
    window.clearTimeout(searchDebounceTimer)
  }
  searchDebounceTimer = window.setTimeout(() => {
    if (!isEditing.value && !isDetail.value && activeView.value !== 'metron') {
      refreshActiveListData()
    }
  }, 250)
})

onUnmounted(() => {
  closeMetronImportEvents()
  if (searchDebounceTimer) {
    window.clearTimeout(searchDebounceTimer)
  }
  if (loadMoreObserver) {
    loadMoreObserver.disconnect()
  }
  themeMediaQuery?.removeEventListener('change', handleSystemThemeChange)
})
</script>

<template>
  <main v-if="authLoading" class="auth-shell">
    <section class="auth-panel" role="status" aria-live="polite">
      <span class="loading-spinner" aria-hidden="true"></span>
      <h1>ComicHero</h1>
      <p>Loading user setup...</p>
    </section>
  </main>

  <main v-else-if="setupRequired" class="auth-shell">
    <form class="auth-panel" @submit.prevent="submitSetup">
      <div>
        <p class="eyebrow">First Run</p>
        <h1>Choose user mode</h1>
      </div>

      <fieldset class="mode-options">
        <legend>User environment</legend>
        <label>
          <input v-model="setupForm.mode" type="radio" value="single" />
          <span>
            <strong>Single user</strong>
            <small>No login. Existing read status stays with the default user.</small>
          </span>
        </label>
        <label>
          <input v-model="setupForm.mode" type="radio" value="multi" />
          <span>
            <strong>Multi user</strong>
            <small>Register and log in. Existing read status becomes the first account.</small>
          </span>
        </label>
      </fieldset>

      <div v-if="setupForm.mode === 'multi'" class="auth-fields">
        <label>
          <span>Name</span>
          <input v-model.trim="setupForm.name" type="text" autocomplete="name" required />
        </label>
        <label>
          <span>Email</span>
          <input v-model.trim="setupForm.email" type="email" autocomplete="email" required />
        </label>
        <label>
          <span>Password</span>
          <input
            v-model="setupForm.password"
            type="password"
            autocomplete="new-password"
            minlength="6"
            required
          />
        </label>
      </div>

      <div v-if="error" class="toast error-toast" role="alert">
        <span>{{ error }}</span>
        <button type="button" aria-label="Dismiss error" @click="clearError">Dismiss</button>
      </div>

      <button class="primary-action" type="submit" :disabled="authSaving">
        {{ authSaving ? 'Saving...' : 'Continue' }}
      </button>
    </form>
  </main>

  <main v-else-if="emailVerificationRequired" class="auth-shell">
    <form class="auth-panel" @submit.prevent="submitEmailVerification">
      <div>
        <p class="eyebrow">Verify Email</p>
        <h1>Check your email</h1>
        <p>
          Enter the verification token sent to
          {{ userStatus.emailVerificationEmail || verificationForm.email }}.
        </p>
      </div>

      <div class="auth-fields">
        <label>
          <span>Verification token</span>
          <input
            v-model.trim="verificationForm.token"
            type="text"
            autocomplete="one-time-code"
            required
          />
        </label>
        <label>
          <span>Password</span>
          <input
            v-model="verificationForm.password"
            type="password"
            autocomplete="current-password"
            minlength="6"
          />
        </label>
      </div>

      <div v-if="error" class="toast error-toast" role="alert">
        <span>{{ error }}</span>
        <button type="button" aria-label="Dismiss error" @click="clearError">Dismiss</button>
      </div>

      <button class="primary-action" type="submit" :disabled="authSaving">
        {{ authSaving ? 'Verifying...' : 'Verify email' }}
      </button>
      <button
        class="secondary-action"
        type="button"
        :disabled="authSaving"
        @click="resendVerificationEmail"
      >
        Resend email
      </button>
    </form>
  </main>

  <main v-else-if="passwordResetMode" class="auth-shell">
    <form
      class="auth-panel"
      @submit.prevent="passwordResetForm.requested ? submitPasswordReset() : submitForgotPassword()"
    >
      <div>
        <p class="eyebrow">Account</p>
        <h1>Reset password</h1>
        <p v-if="!passwordResetForm.requested">
          Enter your email and we will send a password reset token if the account exists.
        </p>
        <p v-else>Enter the token from your email and choose a new password.</p>
      </div>

      <div class="auth-fields">
        <label v-if="!passwordResetForm.requested">
          <span>Email</span>
          <input
            v-model.trim="passwordResetForm.email"
            type="email"
            autocomplete="email"
            required
          />
        </label>
        <template v-else>
          <label>
            <span>Reset token</span>
            <input
              v-model.trim="passwordResetForm.token"
              type="text"
              autocomplete="one-time-code"
              required
            />
          </label>
          <label>
            <span>New password</span>
            <input
              v-model="passwordResetForm.password"
              type="password"
              autocomplete="new-password"
              minlength="6"
              required
            />
          </label>
          <label>
            <span>Confirm new password</span>
            <input
              v-model="passwordResetForm.passwordConfirmation"
              type="password"
              autocomplete="new-password"
              minlength="6"
              required
            />
          </label>
        </template>
      </div>

      <div v-if="error" class="toast error-toast" role="alert">
        <span>{{ error }}</span>
        <button type="button" aria-label="Dismiss error" @click="clearError">Dismiss</button>
      </div>

      <button class="primary-action" type="submit" :disabled="authSaving">
        {{
          authSaving
            ? 'Working...'
            : passwordResetForm.requested
              ? 'Reset password'
              : 'Send reset email'
        }}
      </button>
      <button class="secondary-action" type="button" :disabled="authSaving" @click="showLogin">
        Back to login
      </button>
    </form>
  </main>

  <main v-else-if="authRequired" class="auth-shell">
    <form class="auth-panel" @submit.prevent="submitAuth">
      <div>
        <p class="eyebrow">Multi User</p>
        <h1>{{ authMode === 'register' ? 'Register' : 'Log in' }}</h1>
      </div>

      <div class="auth-tabs" role="group" aria-label="Authentication mode">
        <button type="button" :class="{ active: authMode === 'login' }" @click="authMode = 'login'">
          Log in
        </button>
        <button
          type="button"
          :class="{ active: authMode === 'register' }"
          @click="authMode = 'register'"
        >
          Register
        </button>
      </div>

      <div class="auth-fields">
        <label v-if="authMode === 'register'">
          <span>Name</span>
          <input v-model.trim="authForm.name" type="text" autocomplete="name" required />
        </label>
        <label>
          <span>Email</span>
          <input v-model.trim="authForm.email" type="email" autocomplete="email" required />
        </label>
        <label v-if="authMode === 'register'">
          <span>Confirm email</span>
          <input
            v-model.trim="authForm.emailConfirmation"
            type="email"
            autocomplete="email"
            required
          />
        </label>
        <label>
          <span>Password</span>
          <input
            v-model="authForm.password"
            type="password"
            :autocomplete="authMode === 'register' ? 'new-password' : 'current-password'"
            minlength="6"
            required
          />
        </label>
        <label v-if="authMode === 'register'">
          <span>Confirm password</span>
          <input
            v-model="authForm.passwordConfirmation"
            type="password"
            autocomplete="new-password"
            minlength="6"
            required
          />
        </label>
        <label v-if="authMode === 'register' && registrationMode === 'invite_only'">
          <span>Invite token</span>
          <input
            v-model.trim="authForm.inviteToken"
            type="text"
            autocomplete="one-time-code"
            required
          />
        </label>
      </div>

      <div v-if="error" class="toast error-toast" role="alert">
        <span>{{ error }}</span>
        <button type="button" aria-label="Dismiss error" @click="clearError">Dismiss</button>
      </div>

      <button class="primary-action" type="submit" :disabled="authSaving">
        {{ authSaving ? 'Working...' : authMode === 'register' ? 'Register' : 'Log in' }}
      </button>
      <button
        v-if="authMode === 'login'"
        class="secondary-action"
        type="button"
        :disabled="authSaving"
        @click="showForgotPassword"
      >
        Forgot password?
      </button>
    </form>
  </main>

  <main v-else-if="appReady" class="app-shell">
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

      <div v-if="error" class="toast error-toast" role="alert" aria-live="assertive">
        <span>{{ error }}</span>
        <button type="button" aria-label="Dismiss error" @click="clearError">Dismiss</button>
      </div>

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

      <UserManagementView
        v-else-if="activeView === 'users'"
        :users="userAdminRows"
        :saving-user-id="savingUserID"
        :saving-admin-user-id="savingAdminUserID"
        :deleting-user-id="deletingUserID"
        :current-user-id="currentUser?.id"
        :registration-mode="registrationMode"
        :saving-registration-mode="savingRegistrationMode"
        :public-access="publicAccess"
        :saving-public-access="savingPublicAccess"
        :invite="generatedInvite"
        :generating-invite="generatingInvite"
        @save="saveUserMetronPermissions"
        @save-admin="saveUserAdmin"
        @delete-user="removeUser"
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
        :sections="readingOrderBrowseSections"
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
        @back="backToPreviousPage"
        @edit="editArc"
        @toggle-favorite="toggleArcFavorite"
        @toggle-started="toggleSelectedArcStarted"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
      />

      <ArcsBrowseView
        v-else-if="activeView === 'arcs'"
        :arcs="visibleArcs"
        :sections="arcBrowseSections"
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
        @back="backToPreviousPage"
        @toggle-favorite="toggleSeriesFavorite"
        @toggle-started="toggleSelectedSeriesStarted"
        @import-series="importSelectedSeriesFromMetron"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
      />

      <SeriesBrowseView
        v-else-if="activeView === 'series'"
        :loading="seriesListLoading"
        :series="visibleSeries"
        :sections="seriesBrowseSections"
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
        @back="backToPreviousPage"
        @toggle-favorite="toggleCharacterFavorite"
        @toggle-started="toggleSelectedCharacterStarted"
        @import-appearances="importSelectedCharacterAppearances"
        @open-comic="openComic"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
      />

      <CharactersBrowseView
        v-else-if="activeView === 'characters'"
        :characters="visibleCharacters"
        :sections="characterBrowseSections"
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
        @back="backToPreviousPage"
        @search-metron="searchSelectedComicMetron"
        @apply-metron="applyMetronMetadata"
        @reset-metron="resetMetronMetadata"
        @toggle-read="toggleComicRead"
        @toggle-skipped="toggleComicSkipped"
        @edit="editComic"
        @open-character="openCharacter"
        @open-series="openSeries"
      />

      <div v-else class="browse-view">
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

  <main v-else class="auth-shell">
    <section class="auth-panel">
      <div>
        <p class="eyebrow">Setup</p>
        <h1>Could not load user setup</h1>
      </div>
      <div v-if="error" class="toast error-toast" role="alert">
        <span>{{ error }}</span>
        <button type="button" aria-label="Dismiss error" @click="clearError">Dismiss</button>
      </div>
      <button class="primary-action" type="button" :disabled="authLoading" @click="loadUserStatus">
        Retry
      </button>
    </section>
  </main>
</template>
