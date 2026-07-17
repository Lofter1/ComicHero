import assert from 'node:assert/strict'
import test from 'node:test'
import {
  backFallbackRouteLocation,
  browseRouteLocation,
  detailRouteLocation,
  editRouteLocation,
  routeToAppState,
} from './appRouteState.js'

test('maps browse routes to application state', () => {
  assert.deepEqual(routeToAppState({ name: 'series', params: {} }), {
    view: 'series',
    mode: 'browse',
  })
})

test('maps valid entity routes and rejects invalid IDs', () => {
  assert.deepEqual(routeToAppState({ name: 'comicDetail', params: { id: '42' } }), {
    view: 'comics',
    mode: 'detail',
    id: 42,
  })
  assert.deepEqual(routeToAppState({ name: 'comicDetail', params: { id: 'invalid' } }), {
    view: 'comics',
    mode: 'browse',
    replace: true,
  })
})

test('builds browse, detail, and edit route locations', () => {
  assert.deepEqual(browseRouteLocation('readingOrders'), { name: 'readingOrders' })
  assert.equal(browseRouteLocation('metron'), null)
  assert.deepEqual(detailRouteLocation('characters', 7), {
    name: 'characterDetail',
    params: { id: 7 },
  })
  assert.deepEqual(editRouteLocation('arcs'), { name: 'arcsNew' })
  assert.deepEqual(editRouteLocation('arcs', 9), { name: 'arcEdit', params: { id: 9 } })
})

test('builds detail-aware Back fallbacks without creating a second navigation model', () => {
  assert.deepEqual(backFallbackRouteLocation('comics', 42), {
    name: 'comicDetail',
    params: { id: 42 },
  })
  assert.deepEqual(backFallbackRouteLocation('comics'), { name: 'comics' })
  assert.deepEqual(backFallbackRouteLocation('unknown'), { name: 'readingOrders' })
})

test('falls back to the dashboard for unknown routes', () => {
  assert.deepEqual(routeToAppState({ name: 'unknown', params: {} }), {
    view: 'dashboard',
    mode: 'browse',
    replace: true,
  })
})
