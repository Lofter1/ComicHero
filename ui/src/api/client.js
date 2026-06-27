const API_BASE = import.meta.env.VITE_API_BASE ?? '/api'

class ApiError extends Error {
  constructor(message, { status, rateLimit } = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.rateLimit = rateLimit
  }
}

async function request(path, options = {}) {
  const { data } = await requestWithMeta(path, options)
  return data
}

async function requestWithMeta(path, options = {}) {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: options.body ? { 'Content-Type': 'application/json' } : {},
    ...options,
  })
  const rateLimit = metronRateLimitFromHeaders(response.headers)

  if (!response.ok) {
    let message = `Request failed: ${response.status}`
    try {
      const body = await response.json()
      message = body.detail || body.title || message
    } catch {
      // Keep the status-based message for empty error responses.
    }
    if (response.status === 429) {
      message = resetLabel(rateLimit) ? `Metron rate limit reached. Try again ${resetLabel(rateLimit)}.` : message
    }
    throw new ApiError(message, { status: response.status, rateLimit })
  }

  if (response.status === 204) return { data: null, rateLimit }
  return { data: await response.json(), rateLimit }
}

function send(path, method, body) {
  return request(path, {
    method,
    body: JSON.stringify(body),
  })
}

function sendWithMeta(path, method, body) {
  return requestWithMeta(path, {
    method,
    body: JSON.stringify(body),
  })
}

function queryString(params) {
  const values = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      values.set(key, value)
    }
  })
  const query = values.toString()
  return query ? `?${query}` : ''
}

function metronRateLimitFromHeaders(headers) {
  const rateLimit = {
    burstLimit: headerNumber(headers, 'X-RateLimit-Burst-Limit'),
    burstRemaining: headerNumber(headers, 'X-RateLimit-Burst-Remaining'),
    burstReset: headerNumber(headers, 'X-RateLimit-Burst-Reset'),
    sustainedLimit: headerNumber(headers, 'X-RateLimit-Sustained-Limit'),
    sustainedRemaining: headerNumber(headers, 'X-RateLimit-Sustained-Remaining'),
    sustainedReset: headerNumber(headers, 'X-RateLimit-Sustained-Reset'),
  }

  return Object.values(rateLimit).some((value) => value !== null) ? rateLimit : null
}

function headerNumber(headers, key) {
  const raw = headers.get(key)
  if (raw === null || raw === '') return null
  const value = Number(raw)
  return Number.isFinite(value) && value >= 0 ? value : null
}

function resetLabel(rateLimit) {
  if (!rateLimit) return ''
  const reset = Math.max(rateLimit.burstReset ?? 0, rateLimit.sustainedReset ?? 0)
  if (!reset) return ''
  return `after ${new Date(reset * 1000).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}`
}

export function assetURL(path) {
  if (!path || /^[a-z][a-z\d+.-]*:/i.test(path)) return path
  if (!path.startsWith('/')) return path
  if (!/^https?:\/\//i.test(API_BASE)) return path

  const base = new URL(API_BASE)
  return `${base.origin}${path}`
}

export async function listComics() {
  const comics = await request('/comics')
  return Array.isArray(comics) ? comics : []
}

export function getComic(id) {
  return request(`/comics/${id}`)
}

export function createComic(payload) {
  return send('/comics', 'POST', payload)
}

export function updateComic(id, payload) {
  return send(`/comics/${id}`, 'PUT', payload)
}

export function updateComicReadStatus(id, read) {
  return send(`/comic/${id}/read`, 'PATCH', { read })
}

export function deleteComic(id) {
  return request(`/comics/${id}`, { method: 'DELETE' })
}

export async function listReadingOrders() {
  const readingOrders = await request('/readingOrders')
  return Array.isArray(readingOrders) ? readingOrders : []
}

export function getReadingOrder(id) {
  return request(`/readingOrders/${id}`)
}

export function createReadingOrder(payload) {
  return send('/readingOrders', 'POST', payload)
}

export function updateReadingOrder(id, payload) {
  return send(`/readingOrders/${id}`, 'PUT', payload)
}

export function deleteReadingOrder(id) {
  return request(`/readingOrders/${id}`, { method: 'DELETE' })
}

export function setReadingOrderComics(id, payload) {
  return send(`/readingOrders/${id}/comics`, 'PUT', payload)
}

export function searchMetronComics(params) {
  return requestWithMeta(`/metron/comics${queryString(params)}`)
}

export function importMetronComic(id) {
  return sendWithMeta(`/metron/comics/${id}/import`, 'POST', {})
}

export function getMetronImportJob(id) {
  return request(`/metron/imports/${id}`)
}

export function cancelMetronImportJob(id) {
  return send(`/metron/imports/${id}/cancel`, 'POST', {})
}

export function continueMetronImportJob(id) {
  return send(`/metron/imports/${id}/continue`, 'POST', {})
}

export function updateComicFromMetron(id, metronIssueId) {
  return sendWithMeta(`/comics/${id}/metron`, 'PATCH', { metronIssueId })
}

export function searchMetronReadingLists(params) {
  return requestWithMeta(`/metron/readingLists${queryString(params)}`)
}

export function importMetronReadingList(id) {
  return sendWithMeta(`/metron/readingLists/${id}/import`, 'POST', {})
}

export function searchMetronSeries(params) {
  return requestWithMeta(`/metron/series${queryString(params)}`)
}

export function importMetronSeries(id) {
  return sendWithMeta(`/metron/series/${id}/import`, 'POST', {})
}
