export function routeToAppState(route) {
  const staticRoute = STATIC_ROUTE_STATES[route.name]
  if (staticRoute) return { ...staticRoute }
  if (route.name === 'readingOrdersNew') return editState('readingOrders')
  if (route.name === 'arcsNew') return editState('arcs')
  if (route.name === 'comicsNew') return editState('comics')
  if (route.name === 'readingOrderDetail') return entityState(route, 'readingOrders')
  if (route.name === 'readingOrderEdit') return entityState(route, 'readingOrders', 'edit')
  if (route.name === 'arcDetail') return entityState(route, 'arcs')
  if (route.name === 'arcEdit') return entityState(route, 'arcs', 'edit')
  if (route.name === 'comicDetail') return entityState(route, 'comics')
  if (route.name === 'comicEdit') return entityState(route, 'comics', 'edit')
  if (route.name === 'seriesDetail') return entityState(route, 'series')
  if (route.name === 'characterDetail') return entityState(route, 'characters')
  return { view: 'dashboard', mode: 'browse', replace: true }
}

export function browseRouteLocation(view) {
  const name = BROWSE_ROUTE_NAMES[view]
  return name ? { name } : null
}

export function detailRouteLocation(view, id) {
  const name = DETAIL_ROUTE_NAMES[view]
  return name && id ? { name, params: { id } } : null
}

export function backFallbackRouteLocation(view, id) {
  return detailRouteLocation(view, id) || browseRouteLocation(view) || { name: 'readingOrders' }
}

export function editRouteLocation(view, id) {
  const routes = EDIT_ROUTE_NAMES[view]
  if (!routes) return browseRouteLocation(view)
  return id ? { name: routes.edit, params: { id } } : { name: routes.create }
}

const BROWSE_ROUTE_NAMES = {
  dashboard: 'dashboard',
  readingOrders: 'readingOrders',
  arcs: 'arcs',
  comics: 'comics',
  series: 'series',
  characters: 'characters',
  users: 'users',
  settings: 'settings',
  account: 'account',
  progress: 'progress',
}

const DETAIL_ROUTE_NAMES = {
  readingOrders: 'readingOrderDetail',
  arcs: 'arcDetail',
  comics: 'comicDetail',
  series: 'seriesDetail',
  characters: 'characterDetail',
}

const EDIT_ROUTE_NAMES = {
  readingOrders: { create: 'readingOrdersNew', edit: 'readingOrderEdit' },
  arcs: { create: 'arcsNew', edit: 'arcEdit' },
  comics: { create: 'comicsNew', edit: 'comicEdit' },
}

const STATIC_ROUTE_STATES = Object.fromEntries(
  Object.entries(BROWSE_ROUTE_NAMES).map(([view, name]) => [name, { view, mode: 'browse' }]),
)
STATIC_ROUTE_STATES.notFound = { view: 'notFound', mode: 'browse' }

function entityState(route, view, mode = 'detail') {
  const id = Number(route.params.id)
  if (!Number.isInteger(id) || id <= 0) return { view, mode: 'browse', replace: true }
  return { view, mode, id }
}

function editState(view) {
  return { view, mode: 'edit', isNew: true }
}
