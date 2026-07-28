import { reactive } from 'vue'

// Keyed by an arbitrary scope string (e.g. "arc:5", "series:12"). Lives only
// for the lifetime of the SPA session (resets on a full page reload) —
// intentionally not persisted to the URL or browser storage. This lets any
// detail view's embedded ComicListView keep its search/status/sort/direction
// filters when navigating away to a comic's detail and back, without making
// the URL itself stateful.
const stateByScope = reactive(new Map())

function defaultState(overrides = {}) {
  return {
    search: '',
    status: 'all',
    sort: 'series',
    direction: 'asc',
    ...overrides,
  }
}

/**
 * @param {string|null} scope - a stable, unique key for the list being
 *   filtered, e.g. `arc:${arc.id}`, `series:${series.id}`,
 *   `readingOrder:${order.id}`. Pass null if there's nothing to key on yet
 *   (e.g. before the entity has loaded) — a throwaway, non-persisted state
 *   object is returned in that case.
 * @param {object} initial - default values for this scope's first visit,
 *   e.g. `{ sort: 'date' }` for character/collection appearances.
 */
export function useComicListFilterState(scope, initial = {}) {
  if (scope == null) return reactive(defaultState(initial))

  if (!stateByScope.has(scope)) {
    stateByScope.set(scope, defaultState(initial))
  }
  return stateByScope.get(scope)
}
