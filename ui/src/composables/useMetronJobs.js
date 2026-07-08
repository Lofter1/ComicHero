import { ref } from 'vue'
import {
  cancelMetronImportJob,
  continueMetronImportJob,
  dismissMetronImportJob as removeMetronImportJob,
  getMetronImportJob,
  getMetronQuota,
  importMetronArc,
  importMetronCharacterAppearances,
  importMetronComic,
  importMetronReadingList,
  importMetronSeries,
  listMetronImportJobs,
  metronImportEventsURL,
} from '@/api/client.js'

export function useMetronJobs({ activeView, error, handleImported }) {
  const metronImportJobs = ref([])
  const metronImportMonitorOpen = ref(false)
  const metronQuota = ref(null)
  let metronImportEvents = null
  let metronImportEventsFallbackLoaded = false

  async function loadMetronImportJobs() {
    if (metronImportEvents || typeof EventSource !== 'undefined') {
      connectMetronImportEvents()
      return
    }

    if (!metronImportEventsFallbackLoaded) {
      metronImportEventsFallbackLoaded = true
      await loadMetronImportJobSnapshot()
    }
  }

  async function loadMetronQuota() {
    const { data } = await getMetronQuota()
    metronQuota.value = data
  }

  function updateMetronQuota(quota) {
    if (quota?.known) {
      metronQuota.value = quota
    }
  }

  function trackMetronImportJob(job) {
    if (!job?.id) return
    metronImportMonitorOpen.value = true
    upsertMetronImportJob(job)
    connectMetronImportEvents()
    if (activeView.value === 'metron') {
      loadMetronQuota().catch(() => {})
    }
  }

  function upsertMetronImportJob(job) {
    const normalizedJob = normalizeMetronImportJob(job)
    const index = metronImportJobs.value.findIndex((item) => item.id === job.id)
    if (index === -1) {
      metronImportJobs.value = [normalizedJob, ...metronImportJobs.value]
      return
    }

    metronImportJobs.value = metronImportJobs.value.map((item) => {
      return item.id === job.id
        ? { ...item, ...normalizedJob, displayName: normalizedJob.displayName || item.displayName }
        : item
    })
  }

  function normalizeMetronImportJob(job) {
    if (
      job.status === 'failed' &&
      String(job.message || '')
        .toLowerCase()
        .includes('context canceled')
    ) {
      return { ...job, status: 'canceled', message: 'Import canceled.' }
    }
    return job
  }

  function connectMetronImportEvents() {
    if (metronImportEvents || typeof EventSource === 'undefined') return

    metronImportEvents = new EventSource(metronImportEventsURL())
    metronImportEvents.addEventListener('job', (event) => {
      handleMetronJobEvent(event)
    })
    metronImportEvents.onerror = () => {
      if (metronImportEvents?.readyState === EventSource.CLOSED) {
        closeMetronImportEvents()
        loadMetronImportJobSnapshot().catch((err) => {
          error.value = err.message
        })
      }
    }
  }

  async function handleMetronJobEvent(event) {
    try {
      const payload = JSON.parse(event.data)
      const job = normalizeMetronImportJob(payload.job || payload)
      const previous = metronImportJobs.value.find((item) => item.id === job.id)
      upsertMetronImportJob(job)

      if (
        job.type === 'readingList' &&
        job.status === 'running' &&
        previous &&
        (job.completed !== previous.completed || job.total !== previous.total)
      ) {
        await handleImported(job)
        return
      }
      if (job.status === 'succeeded' && previous && previous.status !== 'succeeded') {
        await handleImported(job)
        if (activeView.value === 'metron') {
          loadMetronQuota().catch(() => {})
        }
        return
      }
      if (job.status === 'failed' && previous && previous.status !== 'failed') {
        error.value = job.message || 'Metron import failed.'
        if (activeView.value === 'metron') {
          loadMetronQuota().catch(() => {})
        }
        return
      }
    } catch (err) {
      error.value = err.message
    }
  }

  async function loadMetronImportJobSnapshot() {
    const jobs = await listMetronImportJobs()
    metronImportJobs.value = []
    jobs.forEach((job) => {
      upsertMetronImportJob(job)
    })
    if (jobs.length) {
      metronImportMonitorOpen.value = jobs.some(isActiveMetronJob)
    }
  }

  function dismissMetronJob(id) {
    removeMetronImportJob(id).catch(() => {})
    metronImportJobs.value = metronImportJobs.value.filter((job) => job.id !== id)
  }

  async function retryMetronJob(job) {
    try {
      const { data: nextJob } = await startMetronRetry(job)
      dismissMetronJob(job.id)
      trackMetronImportJob({ ...nextJob, displayName: job.displayName })
    } catch (err) {
      error.value = err.message
    }
  }

  function startMetronRetry(job) {
    const options = job.options || {}
    if (job.type === 'comic') return importMetronComic(job.metronId, options)
    if (job.type === 'readingList') return importMetronReadingList(job.metronId, options)
    if (job.type === 'arc') return importMetronArc(job.metronId, options)
    if (job.type === 'character') return importMetronCharacterAppearances(job.metronId, options)
    return importMetronSeries(job.metronId, options)
  }

  async function continueMetronJob(job) {
    try {
      const nextJob = await continueMetronImportJob(job.id)
      dismissMetronJob(job.id)
      trackMetronImportJob({ ...nextJob, displayName: job.displayName })
    } catch (err) {
      error.value = err.message
    }
  }

  async function cancelMetronJob(id) {
    try {
      const job = await cancelMetronImportJob(id)
      upsertMetronImportJob(job)
      if (job.status === 'canceling') {
        pollCanceledMetronJob(id)
      }
    } catch (err) {
      error.value = err.message
    }
  }

  async function pollCanceledMetronJob(id, attempts = 10) {
    for (let attempt = 0; attempt < attempts; attempt += 1) {
      await wait(600)
      const current = metronImportJobs.value.find((job) => job.id === id)
      if (!current || current.status !== 'canceling') return
      try {
        const next = await getMetronImportJob(id)
        upsertMetronImportJob(next)
        if (next.status !== 'canceling') return
      } catch {
        return
      }
    }
  }

  function wait(ms) {
    return new Promise((resolve) => window.setTimeout(resolve, ms))
  }

  function closeMetronImportEvents() {
    if (metronImportEvents) {
      metronImportEvents.close()
      metronImportEvents = null
    }
  }

  function isActiveMetronJob(job) {
    return job.status === 'queued' || job.status === 'running' || job.status === 'canceling'
  }

  return {
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
  }
}
