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
  publicAccess: {
    type: Boolean,
    default: false,
  },
  readOnlyGuest: {
    type: Boolean,
    default: false,
  },
  showMetron: {
    type: Boolean,
    default: true,
  },
  authSaving: {
    type: Boolean,
    default: false,
  },
  version: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['set-theme', 'login', 'logout'])
const menuOpen = ref(false)
const accountMenuOpen = ref(false)

const userInitial = computed(
  () => (props.user?.name || '?').trim().slice(0, 1).toUpperCase() || '?',
)

function closeMenus() {
  menuOpen.value = false
  accountMenuOpen.value = false
}

function setTheme(value) {
  emit('set-theme', value)
}

function logout() {
  emit('logout')
  closeMenus()
}

function login() {
  emit('login')
  closeMenus()
}

function toggleAccountMenu() {
  menuOpen.value = false
  accountMenuOpen.value = !accountMenuOpen.value
}

function toggleMobileMenu() {
  accountMenuOpen.value = false
  menuOpen.value = !menuOpen.value
}
</script>

<template>
  <aside class="sidebar" :class="{ 'menu-open': menuOpen }">
    <div class="sidebar-header">
      <div class="sidebar-branding">
        <h1>ComicHero</h1>
        <span v-if="version" class="version-tag">{{ version }}</span>
      </div>
      <button
        type="button"
        class="mobile-menu-button"
        :aria-expanded="menuOpen"
        aria-controls="primary-navigation"
        aria-label="Toggle navigation"
        @click="toggleMobileMenu"
      >
        <span class="menu-bars" aria-hidden="true">
          <span></span>
          <span></span>
          <span></span>
        </span>
      </button>
    </div>

    <nav id="primary-navigation" class="nav-tabs" aria-label="Primary">
      <router-link
        :to="{ name: 'dashboard' }"
        :class="{ active: activeView === 'dashboard' }"
        @click="closeMenus"
      >
        <span>Dashboard</span>
      </router-link>
      <router-link
        :to="{ name: 'readingOrders' }"
        :class="{ active: activeView === 'readingOrders' }"
        @click="closeMenus"
      >
        <span>Reading Orders</span>
      </router-link>
      <router-link
        :to="{ name: 'arcs' }"
        :class="{ active: activeView === 'arcs' }"
        @click="closeMenus"
      >
        <span>Arcs</span>
      </router-link>
      <router-link
        :to="{ name: 'comics' }"
        :class="{ active: activeView === 'comics' }"
        @click="closeMenus"
      >
        <span>Comics</span>
      </router-link>
      <router-link
        :to="{ name: 'series' }"
        :class="{ active: activeView === 'series' }"
        @click="closeMenus"
      >
        <span>Series</span>
      </router-link>
      <router-link
        :to="{ name: 'characters' }"
        :class="{ active: activeView === 'characters' }"
        @click="closeMenus"
      >
        <span>Characters</span>
      </router-link>
      <router-link
        v-if="showMetron"
        :to="{ name: 'metron' }"
        :class="{ active: activeView === 'metron' }"
        @click="closeMenus"
      >
        <span>Metron</span>
      </router-link>
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

          <router-link :to="{ name: 'account' }" class="account-menu-item" @click="closeMenus">
            <span aria-hidden="true">@</span>
            <span>Account settings</span>
          </router-link>
          <router-link :to="{ name: 'progress' }" class="account-menu-item" @click="closeMenus">
            <span aria-hidden="true">%</span>
            <span>Progress</span>
          </router-link>
          <router-link
            v-if="isAdmin"
            :to="{ name: 'settings' }"
            class="account-menu-item"
            @click="closeMenus"
          >
            <span aria-hidden="true">⚙</span>
            <span>App settings</span>
          </router-link>
          <router-link
            v-if="isAdmin"
            :to="{ name: 'users' }"
            class="account-menu-item"
            @click="closeMenus"
          >
            <span aria-hidden="true">#</span>
            <span>Manage users</span>
          </router-link>
          <a
            class="account-menu-item"
            href="https://discord.gg/GebUwAVP"
            target="_blank"
            rel="noreferrer"
            @click="closeMenus"
          >
            <span aria-hidden="true">♥</span>
            <span>Join the community</span>
          </a>
          <a
            class="account-menu-item"
            href="https://github.com/Lofter1/ComicHero/issues/new"
            target="_blank"
            rel="noreferrer"
            @click="closeMenus"
          >
            <span aria-hidden="true">!</span>
            <span>Report a bug</span>
          </a>
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
      <div v-else-if="readOnlyGuest" class="public-session-card">
        <strong>Public access</strong>
        <span>Read-only access</span>
        <button type="button" class="secondary-action" @click="login">Log in</button>
      </div>
    </div>
  </aside>
</template>
