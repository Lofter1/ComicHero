import { onUnmounted, ref, watch } from 'vue'
import {
  createUserInvite,
  getMetronComicDiscovery,
  getMetronComicScan,
  metronComicDiscoveryEventsURL,
  metronComicScanEventsURL,
  stopMetronComicDiscovery,
  stopMetronComicScan,
  triggerMetronComicDiscovery,
  triggerMetronComicScan,
  updateMetronComicDiscovery,
  updateMetronComicScan,
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
  const savingComicScan = ref(false)
  const savingComicDiscovery = ref(false)
  const generatedInvite = ref(null)
  const generatingInvite = ref(false)
  const savingRegistrationMode = ref(false)
  const savingPublicAccess = ref(false)
  let comicScanEvents = null
  let comicDiscoveryEvents = null

  async function loadSettings() {
    if (!connectComicScanEvents()) comicScan.value = await getMetronComicScan()
    if (!connectComicDiscoveryEvents()) comicDiscovery.value = await getMetronComicDiscovery()
  }

  async function saveComicScan(settings) {
    await withSaving(savingComicScan, async () => {
      comicScan.value = await updateMetronComicScan(settings)
    })
  }

  async function runComicScan() {
    await run(async () => {
      comicScan.value = await triggerMetronComicScan()
    })
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

  async function runComicDiscovery() {
    await run(async () => {
      comicDiscovery.value = await triggerMetronComicDiscovery()
    })
  }

  async function cancelComicDiscovery() {
    await run(async () => {
      comicDiscovery.value = await stopMetronComicDiscovery()
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
  })

  onUnmounted(() => {
    closeComicScanEvents()
    closeComicDiscoveryEvents()
  })

  return {
    comicScan,
    comicDiscovery,
    savingComicScan,
    savingComicDiscovery,
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
    generateInvite,
    saveRegistration,
    savePublic,
  }
}
