<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  activeView: {
    type: String,
    required: true,
  },
  themePreference: {
    type: String,
    default: 'system',
  },
  user: {
    type: Object,
    default: null,
  },
  userMode: {
    type: String,
    default: '',
  },
  isAdmin: {
    type: Boolean,
    default: false,
  },
  authSaving: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['change-view', 'set-theme', 'logout'])
const menuOpen = ref(false)
const accountMenuOpen = ref(false)

const userInitial = computed(() => (props.user?.name || '?').trim().slice(0, 1).toUpperCase() || '?')

function changeView(view) {
  emit('change-view', view)
  menuOpen.value = false
  accountMenuOpen.value = false
}

function setTheme(value) {
  emit('set-theme', value)
}

function logout() {
  emit('logout')
  menuOpen.value = false
  accountMenuOpen.value = false
}

function toggleAccountMenu() {
  accountMenuOpen.value = !accountMenuOpen.value
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
      <div v-if="user" class="account-menu" :class="{ open: accountMenuOpen }">
        <button
          type="button"
          class="account-menu-trigger"
          :aria-expanded="accountMenuOpen"
          aria-controls="account-menu-panel"
          @click="toggleAccountMenu"
        >
          <span class="account-avatar" aria-hidden="true">{{ userInitial }}</span>
          <span class="account-trigger-copy">
            <strong>{{ user.name }}</strong>
          </span>
          <span class="account-menu-caret" aria-hidden="true">▾</span>
        </button>

        <div v-if="accountMenuOpen" id="account-menu-panel" class="account-menu-panel">
          <div class="account-menu-profile">
            <span class="account-avatar large" aria-hidden="true">{{ userInitial }}</span>
            <span>
              <strong>{{ user.name }}</strong>
            </span>
          </div>

          <div class="account-menu-section">
            <div class="account-menu-label">
              <span aria-hidden="true">◐</span>
              <strong>Display Mode</strong>
            </div>
            <div class="theme-selector account-theme-selector" role="group" aria-label="Theme">
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
          </div>

          <button v-if="isAdmin" type="button" class="account-menu-item" @click="changeView('users')">
            <span aria-hidden="true">⚙</span>
            <span>Manage users</span>
          </button>
          <button
            v-if="userMode === 'multi'"
            type="button"
            class="account-menu-item danger"
            :disabled="authSaving"
            @click="logout"
          >
            <span aria-hidden="true">⇥</span>
            <span>{{ authSaving ? 'Logging out...' : 'Log out' }}</span>
          </button>
        </div>
      </div>

    </div>
  </aside>
</template>
