import { onUnmounted, ref, watch } from 'vue'
import {
  cblRepositorySyncEventsURL,
  createUserInvite,
  getCBLRepositorySync,
  listCBLRepositoryFiles,
  getMetronComicDiscovery,
  getMetronComicScan,
  metronComicDiscoveryEventsURL,
  metronComicScanEventsURL,
  resolveCBLRepositoryMetronIssue,
  stopMetronComicDiscovery,
  stopMetronComicScan,
  stopCBLRepositorySync,
  triggerCBLRepositorySync,
  triggerMetronComicDiscovery,
  triggerMetronComicScan,
  updateMetronComicDiscovery,
  updateMetronComicScan,
  updateCBLRepositorySync,
  updatePublicAccess,
  updateRegistrationMode,
} from '@/api/client.js'

export function useMetronSettings({
  activeView,
  error,
  userStatus,
  registrationMode,
  publicAccess,
}) {
  const comicScan = ref(null)
  const comicDiscovery = ref(null)
  const cblRepositorySync = ref(null)
  const cblRepositoryFiles = ref([])
  const savingComicScan = ref(false)
  const savingComicDiscovery = ref(false)
  const savingCBLRepositorySync = ref(false)
  const loadingCBLRepositoryFiles = ref(false)
  const generatedInvite = ref(null)
  const generatingInvite = ref(false)
  const savingRegistrationMode = ref(false)
  const savingPublicAccess = ref(false)
  let comicScanEvents = null
  let comicDiscoveryEvents = null
  let cblRepositorySyncEvents = null

  async function loadSettings() {
    if (!connectComicScanEvents()) comicScan.value = await getMetronComicScan()
    if (!connectComicDiscoveryEvents()) comicDiscovery.value = await getMetronComicDiscovery()
    if (!connectCBLRepositorySyncEvents()) {
      cblRepositorySync.value = await getCBLRepositorySync()
    }
  }

  async function saveComicScan(settings) {
    await withSaving(savingComicScan, async () => {
      comicScan.value = await updateMetronComicScan(settings)
    })
  }

  async function runComicScan(settings) {
    savingComicScan.value = true
    await run(async () => {
      comicScan.value = await updateMetronComicScan(settings)
      comicScan.value = await triggerMetronComicScan()
    })
    savingComicScan.value = false
  }

  async function cancelComicScan() {
    await run(async () => {
      comicScan.value = await stopMetronComicScan()
    })
  }

  async function saveComicDiscovery(settings) {
    await withSaving(savingComicDiscovery, async () => {
      comicDiscovery.value = await updateMetronComicDiscovery(settings)
    })
  }

  async function runComicDiscovery(settings) {
    savingComicDiscovery.value = true
    await run(async () => {
      comicDiscovery.value = await updateMetronComicDiscovery(settings)
      comicDiscovery.value = await triggerMetronComicDiscovery()
    })
    savingComicDiscovery.value = false
  }

  async function cancelComicDiscovery() {
    await run(async () => {
      comicDiscovery.value = await stopMetronComicDiscovery()
    })
  }

  async function saveCBLRepositorySync(settings) {
    await withSaving(savingCBLRepositorySync, async () => {
      cblRepositorySync.value = await updateCBLRepositorySync(settings)
    })
  }

  async function loadCBLRepositoryFiles(settings) {
    loadingCBLRepositoryFiles.value = true
    cblRepositoryFiles.value = []
    await run(async () => {
      cblRepositorySync.value = await updateCBLRepositorySync(settings)
      const files = await listCBLRepositoryFiles()
      cblRepositoryFiles.value = Array.isArray(files) ? files : []
    })
    loadingCBLRepositoryFiles.value = false
  }

  async function runCBLRepositorySync(request) {
    const settings = request?.settings || request
    const files = Array.isArray(request?.files) ? request.files : []
    const resolveMissingOnMetron = Boolean(request?.resolveMissingOnMetron)
    savingCBLRepositorySync.value = true
    await run(async () => {
      cblRepositorySync.value = await updateCBLRepositorySync(settings)
      cblRepositorySync.value = await triggerCBLRepositorySync({
        files,
        resolveMissingOnMetron,
      })
    })
    savingCBLRepositorySync.value = false
  }

  async function cancelCBLRepositorySync() {
    await run(async () => {
      cblRepositorySync.value = await stopCBLRepositorySync()
    })
  }

  async function resolveCBLMetronIssue(selection) {
    await run(async () => {
      cblRepositorySync.value = await resolveCBLRepositoryMetronIssue(selection)
    })
  }

  async function generateInvite() {
    await withSaving(generatingInvite, async () => {
      generatedInvite.value = await createUserInvite()
    })
  }

  async function saveRegistration(mode) {
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

    await withSaving(savingRegistrationMode, async () => {
      userStatus.value = await updateRegistrationMode({ mode })
    })
  }

  async function savePublic(enabled) {
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

    await withSaving(savingPublicAccess, async () => {
      userStatus.value = await updatePublicAccess({ enabled })
    })
  }

  function connectComicScanEvents() {
    if (comicScanEvents || typeof EventSource === 'undefined') return false
    comicScanEvents = connectEvents({
      url: metronComicScanEventsURL(),
      eventName: 'scan',
      payloadKey: 'scan',
      target: comicScan,
      close: closeComicScanEvents,
      fallback: getMetronComicScan,
    })
    return true
  }

  function connectComicDiscoveryEvents() {
    if (comicDiscoveryEvents || typeof EventSource === 'undefined') return false
    comicDiscoveryEvents = connectEvents({
      url: metronComicDiscoveryEventsURL(),
      eventName: 'discovery',
      payloadKey: 'discovery',
      target: comicDiscovery,
      close: closeComicDiscoveryEvents,
      fallback: getMetronComicDiscovery,
    })
    return true
  }

  function connectCBLRepositorySyncEvents() {
    if (cblRepositorySyncEvents || typeof EventSource === 'undefined') return false
    cblRepositorySyncEvents = connectEvents({
      url: cblRepositorySyncEventsURL(),
      eventName: 'sync',
      payloadKey: 'sync',
      target: cblRepositorySync,
      close: closeCBLRepositorySyncEvents,
      fallback: getCBLRepositorySync,
    })
    return true
  }

  function connectEvents({ url, eventName, payloadKey, target, close, fallback }) {
    const source = new EventSource(url, { withCredentials: true })
    source.addEventListener(eventName, (event) => {
      try {
        const payload = JSON.parse(event.data)
        target.value = payload[payloadKey] || payload
      } catch (err) {
        error.value = err.message
      }
    })
    source.onerror = () => {
      if (source.readyState !== EventSource.CLOSED) return
      close()
      fallback()
        .then((status) => {
          target.value = status
        })
        .catch((err) => {
          error.value = err.message
        })
    }
    return source
  }

  function closeComicScanEvents() {
    comicScanEvents?.close()
    comicScanEvents = null
  }

  function closeComicDiscoveryEvents() {
    comicDiscoveryEvents?.close()
    comicDiscoveryEvents = null
  }

  function closeCBLRepositorySyncEvents() {
    cblRepositorySyncEvents?.close()
    cblRepositorySyncEvents = null
  }

  async function withSaving(savingRef, action) {
    savingRef.value = true
    await run(action)
    savingRef.value = false
  }

  async function run(action) {
    error.value = ''
    try {
      await action()
    } catch (err) {
      error.value = err.message
    }
  }

  watch(activeView, (view) => {
    if (view === 'settings') return
    closeComicScanEvents()
    closeComicDiscoveryEvents()
    closeCBLRepositorySyncEvents()
  })

  onUnmounted(() => {
    closeComicScanEvents()
    closeComicDiscoveryEvents()
    closeCBLRepositorySyncEvents()
  })

  return {
    comicScan,
    comicDiscovery,
    cblRepositorySync,
    cblRepositoryFiles,
    savingComicScan,
    savingComicDiscovery,
    savingCBLRepositorySync,
    loadingCBLRepositoryFiles,
    generatedInvite,
    generatingInvite,
    savingRegistrationMode,
    savingPublicAccess,
    loadSettings,
    saveComicScan,
    runComicScan,
    cancelComicScan,
    saveComicDiscovery,
    runComicDiscovery,
    cancelComicDiscovery,
    saveCBLRepositorySync,
    loadCBLRepositoryFiles,
    runCBLRepositorySync,
    cancelCBLRepositorySync,
    resolveCBLMetronIssue,
    generateInvite,
    saveRegistration,
    savePublic,
  }
}
