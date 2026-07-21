<script setup>
import { computed, ref } from 'vue'
import { useClickOutside } from '@/shared/composables/useClickOutside.js'
import BaseButton from '@/shared/components/form/BaseButton.vue'

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
  <aside class="sidebar" :class="{ 'menu-open': menuOpen }">
    <div class="sidebar-header flex items-center justify-between gap-3">
      <div class="sidebar-branding grid justify-items-start gap-1.5">
        <h1>ComicHero</h1>
        <span v-if="version" class="version-tag">{{ version }}</span>
      </div>
      <!-- Native button: this measured DOM trigger controls the responsive navigation panel. -->
      <button
        ref="mobileMenuButton"
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

    <nav id="primary-navigation" ref="primaryNavigation" class="nav-tabs" aria-label="Primary">
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

    <div class="sidebar-actions">
      <div v-if="user" ref="accountMenu" class="account-menu" :class="{ open: accountMenuOpen }">
        <!-- Native button: the account trigger is a composite avatar/menu disclosure. -->
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
          <span class="account-menu-caret text-muted text-sm down-tablet:hidden" aria-hidden="true"
            >▾</span
          >
        </button>

        <div v-if="accountMenuOpen" id="account-menu-panel" class="account-menu-panel">
          <div class="account-menu-profile">
            <span class="account-avatar large" aria-hidden="true">{{ userInitial }}</span>
            <span>
              <strong>{{ user.name }}</strong>
            </span>
          </div>

          <div class="account-menu-section grid gap-2 border-t border-line border-b py-2.5 px-0">
            <div class="account-menu-label">
              <span aria-hidden="true">◐</span>
              <strong>Display Mode</strong>
            </div>
            <div class="theme-selector account-theme-selector" role="group" aria-label="Theme">
              <!-- Native buttons: theme choices are a stateful segmented control. -->
              <button
                type="button"
                class="theme-option"
                :class="{ active: themePreference === 'light' }"
                :aria-pressed="themePreference === 'light'"
                @click="setTheme('light')"
              >
                Light
              </button>
              <button
                type="button"
                class="theme-option"
                :class="{ active: themePreference === 'dark' }"
                :aria-pressed="themePreference === 'dark'"
                @click="setTheme('dark')"
              >
                Dark
              </button>
              <button
                type="button"
                class="theme-option"
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
          <router-link :to="{ name: 'collections' }" class="account-menu-item" @click="closeMenus">
            <span aria-hidden="true">◆</span>
            <span>My collections</span>
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
          <!-- Native button: logout is styled as an inline account-menu item. -->
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
        <BaseButton class="mt-1.5 down-tablet:mt-0" variant="neutral" size="sidebar" @click="login">
          Log in
        </BaseButton>
      </div>
    </div>
  </aside>
</template>

<style scoped>
@reference '../../styles.css';

.sidebar {
  @apply sticky top-0 z-30 h-screen border-r border-line [background:var(--sidebar-bg)] p-7 flex flex-col gap-7 [backdrop-filter:blur(12px)] down-tablet:sticky down-tablet:top-0 down-tablet:z-30 down-tablet:h-auto down-tablet:border-r-0 down-tablet:border-b down-tablet:border-line down-tablet:py-3 down-tablet:px-4 down-tablet:gap-3 down-tablet:[box-shadow:0_6px_18px_var(--shadow-soft)] down-tablet:overflow-visible down-mobile:p-3 down-phone:p-2.5 down-tablet:[&:not(.menu-open)_.nav-tabs]:hidden down-tablet:[&.menu-open_.nav-tabs]:grid down-tablet:[&.menu-open_.nav-tabs]:absolute down-tablet:[&.menu-open_.nav-tabs]:top-[calc(100%+8px)] down-tablet:[&.menu-open_.nav-tabs]:right-4 down-tablet:[&.menu-open_.nav-tabs]:w-[min(360px,calc(100vw-36px))] down-tablet:[&.menu-open_.nav-tabs]:grid-cols-1 down-tablet:[&.menu-open_.nav-tabs]:gap-2 down-tablet:[&.menu-open_.nav-tabs]:border down-tablet:[&.menu-open_.nav-tabs]:border-line-strong down-tablet:[&.menu-open_.nav-tabs]:rounded-lg down-tablet:[&.menu-open_.nav-tabs]:z-40 down-tablet:[&.menu-open_.nav-tabs]:bg-surface down-tablet:[&.menu-open_.nav-tabs]:p-2.5 down-tablet:[&.menu-open_.nav-tabs]:[box-shadow:0_18px_40px_var(--shadow-panel)] down-tablet:[&.menu-open_.nav-tabs]:backdrop-filter-none down-tablet:[&.menu-open_.nav-tabs]:isolate down-mobile:[&_.eyebrow]:m-0 down-mobile:[&.menu-open_.nav-tabs]:right-3 down-mobile:[&.menu-open_.nav-tabs]:w-[min(360px,calc(100vw-24px))] down-mobile:[&.menu-open_.sidebar-actions]:grid-cols-1;
}

.version-tag {
  @apply inline-flex border border-line-strong rounded-full bg-surface text-muted py-0.5 px-2 text-xs font-bold leading-tight;
}

.mobile-menu-button {
  @apply hidden place-items-center w-10 min-w-10 min-h-10 border border-line-strong rounded bg-surface text-control p-0 down-tablet:grid;
}

.menu-bars {
  @apply grid gap-1 w-4 [&_span]:block [&_span]:h-0.5 [&_span]:rounded-full [&_span]:[background:currentColor];
}

.nav-tabs {
  @apply grid gap-2 [&_:where(a,button)]:min-h-10 [&_:where(a,button)]:border [&_:where(a,button)]:border-line-strong [&_:where(a,button)]:rounded [&_:where(a,button)]:bg-surface [&_:where(a,button)]:text-control [&_:where(a,button)]:py-2.5 [&_:where(a,button)]:px-3.5 [&_:where(a,button)]:flex [&_:where(a,button)]:items-center [&_:where(a,button)]:justify-between [&_:where(a,button)]:gap-2.5 [&_:where(a,button)]:text-left [&_:where(a,button)]:no-underline [&_:where(a,button).active]:border-primary [&_:where(a,button).active]:bg-primary [&_:where(a,button).active]:text-white down-tablet:[&_:where(a,button)]:justify-start down-tablet:[&_:where(a,button)]:text-left down-mobile:[&_:where(a,button)]:min-h-10 down-mobile:[&_:where(a,button)]:py-2 down-mobile:[&_:where(a,button)]:px-1.5 down-mobile:[&_:where(a,button)]:text-sm down-phone:[&_:where(a,button)]:text-sm;
}

.sidebar-actions {
  @apply grid gap-2.5 mt-auto down-tablet:absolute down-tablet:top-[50%] down-tablet:right-20 down-tablet:block down-tablet:mt-0 down-tablet:w-auto down-tablet:transform-[translateY(-50%)] down-mobile:right-[66px] down-phone:right-16;
}

.account-menu {
  @apply relative min-w-0 [&.open_.account-menu-trigger]:border-primary [&.open_.account-menu-trigger]:bg-surface [&.open_.account-menu-trigger]:[box-shadow:0_8px_18px_var(--shadow-soft)];
}

.account-menu-trigger {
  @apply w-full min-h-14 grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2.5 border border-line rounded bg-surface-soft text-control py-2 px-2.5 text-left down-tablet:w-10 down-tablet:h-10 down-tablet:min-h-10 down-tablet:inline-flex down-tablet:justify-center down-tablet:p-0 hover:border-primary hover:bg-surface hover:[box-shadow:0_8px_18px_var(--shadow-soft)] focus-visible:border-primary focus-visible:bg-surface focus-visible:[box-shadow:0_8px_18px_var(--shadow-soft)] down-tablet:[&_.account-avatar]:w-8 down-tablet:[&_.account-avatar]:min-w-8 down-tablet:[&_.account-avatar]:h-8;
}

.account-avatar {
  @apply w-9 min-w-9 h-9 border border-primary rounded-full inline-flex items-center justify-center bg-primary-soft text-primary font-black leading-none [&.large]:w-12 [&.large]:min-w-12 [&.large]:h-12 [&.large]:text-lg;
}

.account-trigger-copy {
  @apply min-w-0 grid gap-0.5 [&_small]:text-muted [&_small]:font-bold down-tablet:hidden;
}

.account-menu-panel {
  @apply absolute left-0 bottom-[calc(100%+10px)] z-40 w-[min(320px,calc(100vw-36px))] border border-line rounded bg-surface shadow-panel p-2.5 grid gap-2 down-tablet:absolute down-tablet:top-[calc(100%+8px)] down-tablet:right-0 down-tablet:bottom-auto down-tablet:left-auto down-tablet:w-[min(320px,calc(100vw-36px))];
}

.account-menu-profile {
  @apply grid grid-cols-[auto_minmax(0,1fr)] gap-3 items-center p-2 [&_span]:min-w-0 [&_span]:grid [&_span]:gap-0.5 [&_small]:text-muted [&_small]:font-bold;
}

.account-avatar.large {
  @apply w-9 min-w-9 h-9 border border-primary rounded-full inline-flex items-center justify-center bg-primary-soft text-primary font-black leading-none [&.large]:w-12 [&.large]:min-w-12 [&.large]:h-12 [&.large]:text-lg;
}

.account-menu-label {
  @apply flex items-center gap-2.5 text-label py-0 px-2 [&_>_span]:w-6 [&_>_span]:min-w-6 [&_>_span]:text-center [&_>_span]:text-muted;
}

.theme-selector.account-theme-selector {
  @apply my-0 mx-2 grid grid-cols-3 gap-1 border border-line-strong rounded bg-panel-soft p-1;
}

.account-menu-item {
  @apply min-h-10 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-6 [&_>_span:first-child]:min-w-6 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger;
}

.account-menu-item.danger {
  @apply min-h-10 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-6 [&_>_span:first-child]:min-w-6 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger;
}

.public-session-card {
  @apply grid gap-1 border border-line rounded bg-surface-soft p-3 down-tablet:block down-tablet:border-0 down-tablet:bg-transparent down-tablet:p-0 [&_span]:text-muted [&_span]:text-sm [&_span]:font-bold down-tablet:[&_strong]:hidden down-tablet:[&_span]:hidden;
}

.account-trigger-copy strong {
  overflow-wrap: anywhere;
}

.account-menu-profile strong {
  overflow-wrap: anywhere;
}

.theme-option {
  @apply min-h-8 rounded-[6px] border-0 bg-transparent p-1.5 text-sm font-extrabold text-label;
}

.theme-option.active {
  @apply bg-primary text-white;
}

@media (width <= 420px) {
  .theme-option {
    @apply px-1 text-xs;
  }
}
</style>
