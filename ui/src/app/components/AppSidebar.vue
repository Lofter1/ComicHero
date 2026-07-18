<script setup>
import { computed, ref } from 'vue'
import { useClickOutside } from '@/shared/composables/useClickOutside.js'

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
const mobileMenuButton = ref(null)
const primaryNavigation = ref(null)
const accountMenu = ref(null)

useClickOutside([mobileMenuButton, primaryNavigation], () => (menuOpen.value = false), menuOpen)
useClickOutside(accountMenu, () => (accountMenuOpen.value = false), accountMenuOpen)

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
  <aside
    class="sidebar sticky top-0 z-30 h-screen [border-right:1px_solid_var(--line)] [background:var(--sidebar-bg)] p-7 flex flex-col gap-7 [backdrop-filter:blur(12px)] down-tablet:sticky down-tablet:top-0 down-tablet:z-30 down-tablet:[height:auto] down-tablet:[border-right:0] down-tablet:[border-bottom:1px_solid_var(--line)] down-tablet:[padding:12px_18px] down-tablet:gap-3 down-tablet:[box-shadow:0_6px_18px_var(--shadow-soft)] down-tablet:[overflow:visible] down-mobile:p-3 down-phone:p-2.5"
    :class="{ 'menu-open': menuOpen }"
  >
    <div class="sidebar-header flex items-center justify-between gap-3">
      <div class="sidebar-branding grid justify-items-start gap-1.5">
        <h1>ComicHero</h1>
        <span
          v-if="version"
          class="version-tag inline-flex border border-line-strong rounded-full bg-surface text-muted [padding:2px_8px] [font-size:0.7rem] font-bold leading-tight"
          >{{ version }}</span
        >
      </div>
      <button
        ref="mobileMenuButton"
        type="button"
        class="mobile-menu-button hidden place-items-center [width:42px] [min-width:42px] [min-height:42px] border border-line-strong rounded bg-surface text-control [padding:0] down-tablet:grid"
        :aria-expanded="menuOpen"
        aria-controls="primary-navigation"
        aria-label="Toggle navigation"
        @click="toggleMobileMenu"
      >
        <span class="menu-bars grid gap-1 w-4.5" aria-hidden="true">
          <span></span>
          <span></span>
          <span></span>
        </span>
      </button>
    </div>

    <nav
      id="primary-navigation"
      ref="primaryNavigation"
      class="nav-tabs grid gap-2"
      aria-label="Primary"
    >
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
    </nav>

    <div
      class="sidebar-actions grid gap-2.5 [margin-top:auto] down-tablet:absolute down-tablet:[top:50%] down-tablet:[right:72px] down-tablet:block down-tablet:[margin-top:0] down-tablet:[width:auto] down-tablet:[transform:translateY(-50%)] down-mobile:[right:66px] down-phone:[right:64px]"
    >
      <div
        v-if="user"
        ref="accountMenu"
        class="account-menu relative min-w-0"
        :class="{ open: accountMenuOpen }"
      >
        <button
          type="button"
          class="account-menu-trigger w-full [min-height:58px] grid [grid-template-columns:auto_minmax(0,_1fr)_auto] items-center gap-2.5 border border-line rounded bg-surface-soft text-control [padding:9px_10px] text-left down-tablet:[width:42px] down-tablet:[height:42px] down-tablet:[min-height:42px] down-tablet:inline-flex down-tablet:justify-center down-tablet:[padding:0]"
          :aria-expanded="accountMenuOpen"
          aria-controls="account-menu-panel"
          @click="toggleAccountMenu"
        >
          <span class="account-avatar" aria-hidden="true">{{ userInitial }}</span>
          <span class="account-trigger-copy">
            <strong>{{ user.name }}</strong>
          </span>
          <span class="account-menu-caret text-muted [font-size:0.9rem]" aria-hidden="true">▾</span>
        </button>

        <div
          v-if="accountMenuOpen"
          id="account-menu-panel"
          class="account-menu-panel absolute left-0 [bottom:calc(100%_+_10px)] z-40 [width:min(320px,_calc(100vw_-_36px))] border border-line rounded bg-surface shadow-panel p-2.5 grid gap-2 down-tablet:absolute down-tablet:[top:calc(100%_+_8px)] down-tablet:right-0 down-tablet:[bottom:auto] down-tablet:[left:auto] down-tablet:[width:min(320px,_calc(100vw_-_36px))]"
        >
          <div
            class="account-menu-profile grid [grid-template-columns:auto_minmax(0,_1fr)] gap-3 items-center p-2"
          >
            <span class="account-avatar large" aria-hidden="true">{{ userInitial }}</span>
            <span>
              <strong>{{ user.name }}</strong>
            </span>
          </div>

          <div
            class="account-menu-section grid gap-2 [border-top:1px_solid_var(--line)] [border-bottom:1px_solid_var(--line)] [padding:10px_0]"
          >
            <div class="account-menu-label flex items-center gap-2.5 text-label [padding:0_8px]">
              <span aria-hidden="true">◐</span>
              <strong>Display Mode</strong>
            </div>
            <div
              class="theme-selector account-theme-selector [margin:0_8px] grid [grid-template-columns:repeat(3,_minmax(0,_1fr))] gap-1 border border-line-strong rounded [background:var(--panel-soft-bg)] p-1"
              role="group"
              aria-label="Theme"
            >
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

          <router-link
            :to="{ name: 'account' }"
            class="account-menu-item [min-height:42px] w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control [padding:10px_8px] font-extrabold text-left"
            @click="closeMenus"
          >
            <span aria-hidden="true">@</span>
            <span>Account settings</span>
          </router-link>
          <router-link
            :to="{ name: 'progress' }"
            class="account-menu-item [min-height:42px] w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control [padding:10px_8px] font-extrabold text-left"
            @click="closeMenus"
          >
            <span aria-hidden="true">%</span>
            <span>Progress</span>
          </router-link>
          <router-link
            :to="{ name: 'collections' }"
            class="account-menu-item [min-height:42px] w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control [padding:10px_8px] font-extrabold text-left"
            @click="closeMenus"
          >
            <span aria-hidden="true">◆</span>
            <span>My collections</span>
          </router-link>
          <router-link
            v-if="isAdmin"
            :to="{ name: 'settings' }"
            class="account-menu-item [min-height:42px] w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control [padding:10px_8px] font-extrabold text-left"
            @click="closeMenus"
          >
            <span aria-hidden="true">⚙</span>
            <span>App settings</span>
          </router-link>
          <router-link
            v-if="isAdmin"
            :to="{ name: 'users' }"
            class="account-menu-item [min-height:42px] w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control [padding:10px_8px] font-extrabold text-left"
            @click="closeMenus"
          >
            <span aria-hidden="true">#</span>
            <span>Manage users</span>
          </router-link>
          <a
            class="account-menu-item [min-height:42px] w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control [padding:10px_8px] font-extrabold text-left"
            href="https://discord.gg/GebUwAVP"
            target="_blank"
            rel="noreferrer"
            @click="closeMenus"
          >
            <span aria-hidden="true">♥</span>
            <span>Join the community</span>
          </a>
          <a
            class="account-menu-item [min-height:42px] w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control [padding:10px_8px] font-extrabold text-left"
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
            class="account-menu-item danger [min-height:42px] w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control [padding:10px_8px] font-extrabold text-left"
            :disabled="authSaving"
            @click="logout"
          >
            <span aria-hidden="true">⇥</span>
            <span>{{ authSaving ? 'Logging out...' : 'Log out' }}</span>
          </button>
        </div>
      </div>
      <div
        v-else-if="readOnlyGuest"
        class="public-session-card grid gap-1 border border-line rounded bg-surface-soft p-3 down-tablet:block down-tablet:border-0 down-tablet:bg-transparent down-tablet:[padding:0]"
      >
        <strong>Public access</strong>
        <span>Read-only access</span>
        <button type="button" class="secondary-action" @click="login">Log in</button>
      </div>
    </div>
  </aside>
</template>
