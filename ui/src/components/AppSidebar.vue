<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  activeView: {
    type: String,
    required: true,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  themePreference: {
    type: String,
    default: 'system',
  },
})

const emit = defineEmits(['change-view', 'refresh', 'set-theme'])
const menuOpen = ref(false)

const themeLabel = computed(() => {
  if (props.themePreference === 'light') return 'Light'
  if (props.themePreference === 'dark') return 'Dark'
  return 'System'
})

function changeView(view) {
  emit('change-view', view)
  menuOpen.value = false
}

function refresh() {
  emit('refresh')
  menuOpen.value = false
}

function setTheme(value) {
  emit('set-theme', value)
}
</script>

<template>
  <aside class="sidebar" :class="{ 'menu-open': menuOpen }">
    <div class="sidebar-header">
      <h1>ComicHero</h1>
      <button
        type="button"
        class="mobile-menu-button"
        :aria-expanded="menuOpen"
        aria-controls="primary-navigation"
        aria-label="Toggle navigation"
        @click="menuOpen = !menuOpen"
      >
        <span class="menu-bars" aria-hidden="true">
          <span></span>
          <span></span>
          <span></span>
        </span>
      </button>
    </div>

    <nav id="primary-navigation" class="nav-tabs" aria-label="Primary">
      <button :class="{ active: activeView === 'readingOrders' }" @click="changeView('readingOrders')">
        <span>Orders</span>
      </button>
      <button :class="{ active: activeView === 'arcs' }" @click="changeView('arcs')">
        <span>Arcs</span>
      </button>
      <button :class="{ active: activeView === 'comics' }" @click="changeView('comics')">
        <span>Comics</span>
      </button>
      <button :class="{ active: activeView === 'series' }" @click="changeView('series')">
        <span>Series</span>
      </button>
      <button :class="{ active: activeView === 'characters' }" @click="changeView('characters')">
        <span>Characters</span>
      </button>
      <button :class="{ active: activeView === 'metron' }" @click="changeView('metron')">
        <span>Metron</span>
      </button>
    </nav>

    <div class="sidebar-actions">
      <details class="theme-menu">
        <summary>Theme: {{ themeLabel }}</summary>
        <div class="theme-selector" role="group" aria-label="Theme">
          <button
            type="button"
            :class="{ active: themePreference === 'light' }"
            :aria-pressed="themePreference === 'light'"
            @click="setTheme('light')"
          >
            Light
          </button>
          <button
            type="button"
            :class="{ active: themePreference === 'dark' }"
            :aria-pressed="themePreference === 'dark'"
            @click="setTheme('dark')"
          >
            Dark
          </button>
          <button
            type="button"
            :class="{ active: themePreference === 'system' }"
            :aria-pressed="themePreference === 'system'"
            @click="setTheme('system')"
          >
            System
          </button>
        </div>
      </details>

      <button class="refresh-button" :disabled="loading" @click="refresh">
        Refresh
      </button>
    </div>
  </aside>
</template>
