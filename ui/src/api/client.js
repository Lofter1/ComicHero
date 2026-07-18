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
    credentials: 'include',
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
      message = resetLabel(rateLimit)
        ? `Metron rate limit reached. Try again ${resetLabel(rateLimit)}.`
        : message
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

export function getSystemInfo() {
  return request('/system')
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

export function startReadingOrder(id) {
  return send(`/readingOrders/${id}/start`, 'POST', {})
}

export function stopReadingOrder(id) {
  return request(`/readingOrders/${id}/start`, { method: 'DELETE' })
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

export function setArcStarted(id, started) {
  return request(`/arcs/${id}/start`, { method: started ? 'POST' : 'DELETE' })
}

export function getSeries(id) {
  return request(`/series/${id}`)
}

export function deleteSeries(id) {
  return request(`/series/${id}`, { method: 'DELETE' })
}

export function updateSeriesFavorite(id, favorite) {
  return send(`/series/${id}/favorite`, 'PATCH', { favorite })
}

export function setSeriesStarted(id, started) {
  return request(`/series/${id}/start`, { method: started ? 'POST' : 'DELETE' })
}

export function importLocalSeriesFromMetron(id) {
  return sendWithMeta(`/series/${id}/metron/import`, 'POST', {})
}

export function getCharacter(id) {
  return request(`/characters/${id}`)
}

export function deleteCharacter(id) {
  return request(`/characters/${id}`, { method: 'DELETE' })
}

export function updateCharacterFavorite(id, favorite) {
  return send(`/characters/${id}/favorite`, 'PATCH', { favorite })
}

export function setCharacterStarted(id, started) {
  return request(`/characters/${id}/start`, { method: started ? 'POST' : 'DELETE' })
}

export function listCharacterCollections(params = {}) {
  return request(`/collections${queryString(params)}`)
}

export function getCharacterCollection(id) {
  return request(`/collections/${id}`)
}

export function createCharacterCollection(payload) {
  return send('/collections', 'POST', payload)
}

export function deleteCharacterCollection(id) {
  return request(`/collections/${id}`, { method: 'DELETE' })
}

export function setCharacterCollectionStarted(id, started) {
  return request(`/collections/${id}/start`, { method: started ? 'POST' : 'DELETE' })
}

export function addCharacterToCollection(id, characterId) {
  return send(`/collections/${id}/characters`, 'POST', { characterId })
}

export function removeCharacterFromCollection(id, characterId) {
  return request(`/collections/${id}/characters/${characterId}`, { method: 'DELETE' })
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

export function updateComicReadStatus(id, payload) {
  return send(`/comic/${id}/read`, 'PATCH', payload)
}

export function getUserStatus() {
  return request('/auth/status')
}

export function setupUsers(payload) {
  return send('/auth/setup', 'POST', payload)
}

export function registerUser(payload) {
  return send('/auth/register', 'POST', payload)
}

export function loginUser(payload) {
  return send('/auth/login', 'POST', payload)
}

export function verifyEmail(payload) {
  return send('/auth/verify-email', 'POST', payload)
}

export function resendEmailVerification(payload) {
  return send('/auth/verify-email/resend', 'POST', payload)
}

export function requestPasswordReset(payload) {
  return send('/auth/forgot-password', 'POST', payload)
}

export function resetPassword(payload) {
  return send('/auth/reset-password', 'POST', payload)
}

export function logoutUser() {
  return request('/auth/logout', { method: 'POST' })
}

export function updateAccount(payload) {
  return send('/account', 'PUT', payload)
}

export function getAccountStatistics() {
  return request('/account/statistics')
}

export function getDashboard() {
  return request('/dashboard')
}

export function deleteAccount(payload) {
  return send('/account', 'DELETE', payload)
}

export function listUsers() {
  return request('/users')
}

export function createUserInvite() {
  return send('/users/invites', 'POST', {})
}

export function updateRegistrationMode(payload) {
  return send('/users/registration-mode', 'PUT', payload)
}

export function updatePublicAccess(payload) {
  return send('/users/public-access', 'PUT', payload)
}

export function getMetronComicScan() {
  return request('/metron/scans/comics')
}

export function metronComicScanEventsURL() {
  return `${API_BASE}/metron/scans/comics/events`
}

export function updateMetronComicScan(payload) {
  return send('/metron/scans/comics', 'PUT', payload)
}

export function triggerMetronComicScan() {
  return send('/metron/scans/comics/trigger', 'POST', {})
}

export function stopMetronComicScan() {
  return send('/metron/scans/comics/stop', 'POST', {})
}

export function getMetronComicDiscovery() {
  return request('/metron/discovery/comics')
}

export function metronComicDiscoveryEventsURL() {
  return `${API_BASE}/metron/discovery/comics/events`
}

export function updateMetronComicDiscovery(payload) {
  return send('/metron/discovery/comics', 'PUT', payload)
}

export function triggerMetronComicDiscovery() {
  return send('/metron/discovery/comics/trigger', 'POST', {})
}

export function stopMetronComicDiscovery() {
  return send('/metron/discovery/comics/stop', 'POST', {})
}

export function getCBLRepositorySync() {
  return request('/readingOrders/repository-sync')
}

export function cblRepositorySyncEventsURL() {
  return `${API_BASE}/readingOrders/repository-sync/events`
}

export function updateCBLRepositorySync(payload) {
  return send('/readingOrders/repository-sync', 'PUT', payload)
}

export function listCBLRepositoryFiles() {
  return request('/readingOrders/repository-sync/files')
}

export function triggerCBLRepositorySync(payload = {}) {
  return send('/readingOrders/repository-sync/trigger', 'POST', payload)
}

export function resolveCBLRepositoryMetronIssue(payload) {
  return send('/readingOrders/repository-sync/resolve', 'POST', payload)
}

export function stopCBLRepositorySync() {
  return send('/readingOrders/repository-sync/stop', 'POST', {})
}

export function updateUserMetronPermissions(id, payload) {
  return send(`/users/${id}/metron-permissions`, 'PUT', payload)
}

export function updateUserAdmin(id, payload) {
  return send(`/users/${id}/admin`, 'PUT', payload)
}

export function deleteUser(id) {
  return request(`/users/${id}`, { method: 'DELETE' })
}

export function listAuditEvents(params = {}) {
  return request(`/audit-events${queryString(params)}`)
}

export function deleteComic(id) {
  return request(`/comics/${id}`, { method: 'DELETE' })
}

export function mergeComic(id, sourceId) {
  return send(`/comics/${id}/merge`, 'POST', { sourceId })
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

export function rateReadingOrder(id, rating) {
  return send(`/readingOrders/${id}/rating`, 'PATCH', { rating })
}

export function copyReadingOrder(id) {
  return send(`/readingOrders/${id}/copy`, 'POST', {})
}

export function deleteReadingOrder(id) {
  return request(`/readingOrders/${id}`, { method: 'DELETE' })
}

export function setReadingOrderComics(id, payload) {
  return send(`/readingOrders/${id}/comics`, 'PUT', payload)
}

export function importReadingOrderCBL(payload) {
  return send('/readingOrders/cbl/import', 'POST', payload)
}

export function exportReadingOrderCBL(id) {
  return request(`/readingOrders/${id}/cbl`)
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

export function getMetronImportJob(id) {
  return request(`/metron/imports/${id}`)
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

export function updateComicFromMetron(id, metronIssueId) {
  return sendWithMeta(`/comics/${id}/metron`, 'PATCH', {
    metronIssueId,
  })
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

export function importAllMetronReadingLists(options = {}) {
  return sendWithMeta('/metron/readingLists/importAll', 'POST', options)
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
