import { computed, ref, watch } from 'vue'

const STORAGE_KEY = 'comichero-list-options-v2'
const DEFAULT_OPTIONS = {
  readingOrders: { filter: 'all', sort: 'name', direction: 'asc' },
  arcs: { filter: 'all', sort: 'name', direction: 'asc' },
  comics: { status: 'all', sort: 'series', direction: 'asc' },
  series: { filter: 'all', sort: 'name', direction: 'asc' },
  characters: { filter: 'all', sort: 'name', direction: 'asc' },
}

export function useListOptions({ activeView, onChange }) {
  const listOptions = ref(initialOptions())
  const activeListParams = computed(() => apiParams(listOptions.value[activeView.value]))

  function updateListOption(view, key, value) {
    listOptions.value = {
      ...listOptions.value,
      [view]: { ...(listOptions.value[view] || {}), [key]: value },
    }
    onChange?.()
  }

  watch(
    listOptions,
    (value) => {
      if (typeof window !== 'undefined') {
        window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
      }
    },
    { deep: true },
  )

  return { listOptions, activeListParams, updateListOption }
}

function initialOptions() {
  if (typeof window === 'undefined') return mergeOptions({})
  try {
    return mergeOptions(JSON.parse(window.localStorage.getItem(STORAGE_KEY) || '{}'))
  } catch {
    return mergeOptions({})
  }
}

function mergeOptions(savedOptions) {
  return Object.fromEntries(
    Object.entries(DEFAULT_OPTIONS).map(([view, defaults]) => {
      const saved = savedOptions?.[view]
      return [view, { ...defaults, ...(saved && typeof saved === 'object' ? saved : {}) }]
    }),
  )
}

function apiParams(options = {}) {
  const params = {}
  if (options.filter === 'favorites') params.favorite = true
  if (options.filter === 'other') params.favorite = false
  if (options.filter === 'started') params.started = true
  if (options.status && options.status !== 'all') params.status = options.status
  if (options.sort) params.sort = options.sort
  if (options.direction) params.direction = options.direction
  return params
}
