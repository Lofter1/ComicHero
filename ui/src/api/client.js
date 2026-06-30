const API_BASE = normalizeApiBase(import.meta.env.VITE_API_BASE)

class ApiError extends Error {
  constructor(message, { status, rateLimit } = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.rateLimit = rateLimit
  }
}

function normalizeApiBase(value) {
  const base = String(value || '').trim()
  if (!base) return '/api'
  return base.endsWith('/') ? base.slice(0, -1) : base
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
  const pagination = paginationFromHeaders(response.headers)

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

  if (response.status === 204) return { data: null, rateLimit, pagination }
  return { data: await response.json(), rateLimit, pagination }
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

function paginationFromHeaders(headers) {
  const limit = headerNumber(headers, 'X-Page-Limit')
  const offset = headerNumber(headers, 'X-Page-Offset')
  const total = headerNumber(headers, 'X-Total-Count')
  const hasMoreRaw = headers.get('X-Has-More')
  if (limit === null && offset === null && total === null && hasMoreRaw === null) return null
  return {
    limit: limit ?? 0,
    offset: offset ?? 0,
    total: total ?? 0,
    hasMore: hasMoreRaw === 'true',
  }
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

function pagedListResult(data, pagination) {
  const items = Array.isArray(data) ? data : []
  return {
    items,
    limit: pagination?.limit ?? items.length,
    offset: pagination?.offset ?? 0,
    total: pagination?.total ?? items.length,
    hasMore: Boolean(pagination?.hasMore),
  }
}

export async function listComics(params = {}) {
  const { data, pagination } = await requestWithMeta(`/comics${queryString(params)}`)
  return pagedListResult(data, pagination)
}

export async function listCharacters(params = {}) {
  const { data, pagination } = await requestWithMeta(`/characters${queryString(params)}`)
  return pagedListResult(data, pagination)
}

export async function listSeries(params = {}) {
  const { data, pagination } = await requestWithMeta(`/series${queryString(params)}`)
  return pagedListResult(data, pagination)
}

export async function listArcs(params = {}) {
  const { data, pagination } = await requestWithMeta(`/arcs${queryString(params)}`)
  return pagedListResult(data, pagination)
}

export function getArc(id) {
  return request(`/arcs/${id}`)
}

export function createArc(payload) {
  return send('/arcs', 'POST', payload)
}

export function updateArc(id, payload) {
  return send(`/arcs/${id}`, 'PUT', payload)
}

export function deleteArc(id) {
  return request(`/arcs/${id}`, { method: 'DELETE' })
}

export function setArcComics(id, payload) {
  return send(`/arcs/${id}/comics`, 'PUT', payload)
}

export function getSeries(id) {
  return request(`/series/${id}`)
}

export function updateSeriesFavorite(id, favorite) {
  return send(`/series/${id}/favorite`, 'PATCH', { favorite })
}

export function importLocalSeriesFromMetron(id) {
  return sendWithMeta(`/series/${id}/metron/import`, 'POST', {})
}

export function getCharacter(id) {
  return request(`/characters/${id}`)
}

export function updateCharacterFavorite(id, favorite) {
  return send(`/characters/${id}/favorite`, 'PATCH', { favorite })
}

export function searchMetronCharacters(params) {
  return requestWithMeta(`/metron/characters${queryString(params)}`)
}

export function importMetronCharacterAppearances(id, options = {}) {
  return sendWithMeta(`/metron/characters/${id}/import`, 'POST', options)
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

export async function listReadingOrders(params = {}) {
  const { data, pagination } = await requestWithMeta(`/readingOrders${queryString(params)}`)
  return pagedListResult(data, pagination)
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

export function importMetronComic(id, options = {}) {
  return sendWithMeta(`/metron/comics/${id}/import`, 'POST', options)
}

export function metronImportEventsURL() {
  return `${API_BASE}/metron/imports/events`
}

export async function listMetronImportJobs() {
  const jobs = await request('/metron/imports')
  return Array.isArray(jobs) ? jobs : []
}

export function dismissMetronImportJob(id) {
  return request(`/metron/imports/${id}`, { method: 'DELETE' })
}

export function getMetronQuota() {
  return requestWithMeta('/metron/quota')
}

export function cancelMetronImportJob(id) {
  return send(`/metron/imports/${id}/cancel`, 'POST', {})
}

export function continueMetronImportJob(id) {
  return send(`/metron/imports/${id}/continue`, 'POST', {})
}

export function updateComicFromMetron(id, metronIssueId, options = {}) {
  return sendWithMeta(`/comics/${id}/metron`, 'PATCH', { metronIssueId, force: Boolean(options.force) })
}

export function searchMetronReadingLists(params) {
  return requestWithMeta(`/metron/readingLists${queryString(params)}`)
}

export function getMetronReadingList(id) {
  return requestWithMeta(`/metron/readingLists/${id}`)
}

export function importMetronReadingList(id, options = {}) {
  return sendWithMeta(`/metron/readingLists/${id}/import`, 'POST', options)
}

export function searchMetronSeries(params) {
  return requestWithMeta(`/metron/series${queryString(params)}`)
}

export function importMetronSeries(id, options = {}) {
  return sendWithMeta(`/metron/series/${id}/import`, 'POST', options)
}

export function searchMetronArcs(params) {
  return requestWithMeta(`/metron/arcs${queryString(params)}`)
}

export function importMetronArc(id, options = {}) {
  return sendWithMeta(`/metron/arcs/${id}/import`, 'POST', options)
}
