import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

const STORAGE_KEY = 'comichero-theme'
const VALID_PREFERENCES = new Set(['dark', 'light', 'system'])

export function useTheme() {
  const themePreference = ref(initialThemePreference())
  const systemTheme = ref(currentSystemTheme())
  const resolvedTheme = computed(() =>
    themePreference.value === 'system' ? systemTheme.value : themePreference.value,
  )
  let mediaQuery = null

  function setThemePreference(value) {
    if (VALID_PREFERENCES.has(value)) themePreference.value = value
  }

  function handleSystemThemeChange(event) {
    systemTheme.value = event.matches ? 'dark' : 'light'
  }

  watch(
    resolvedTheme,
    (value) => {
      if (typeof document === 'undefined') return
      document.documentElement.dataset.theme = value
      document.documentElement.style.colorScheme = value
    },
    { immediate: true },
  )

  watch(
    themePreference,
    (value) => {
      if (typeof window !== 'undefined') window.localStorage.setItem(STORAGE_KEY, value)
    },
    { immediate: true },
  )

  onMounted(() => {
    mediaQuery = window.matchMedia?.('(prefers-color-scheme: dark)')
    mediaQuery?.addEventListener('change', handleSystemThemeChange)
  })

  onUnmounted(() => {
    mediaQuery?.removeEventListener('change', handleSystemThemeChange)
  })

  return { themePreference, resolvedTheme, setThemePreference }
}

function initialThemePreference() {
  if (typeof window === 'undefined') return 'system'
  const savedTheme = window.localStorage.getItem(STORAGE_KEY)
  return VALID_PREFERENCES.has(savedTheme) ? savedTheme : 'system'
}

function currentSystemTheme() {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}
