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
    class="sidebar sticky top-0 z-30 h-screen border-r border-line [background:var(--sidebar-bg)] p-7 flex flex-col gap-7 [backdrop-filter:blur(12px)] down-tablet:sticky down-tablet:top-0 down-tablet:z-30 down-tablet:h-auto down-tablet:border-r-0 down-tablet:border-b down-tablet:border-line down-tablet:py-3 down-tablet:px-4.5 down-tablet:gap-3 down-tablet:[box-shadow:0_6px_18px_var(--shadow-soft)] down-tablet:[overflow:visible] down-mobile:p-3 down-phone:p-2.5 down-tablet:[&:not(.menu-open)_.nav-tabs]:hidden down-tablet:[&.menu-open_.nav-tabs]:grid down-tablet:[&.menu-open_.nav-tabs]:absolute down-tablet:[&.menu-open_.nav-tabs]:[top:calc(100%_+_8px)] down-tablet:[&.menu-open_.nav-tabs]:right-4.5 down-tablet:[&.menu-open_.nav-tabs]:[width:min(360px,_calc(100vw_-_36px))] down-tablet:[&.menu-open_.nav-tabs]:grid-cols-1 down-tablet:[&.menu-open_.nav-tabs]:gap-2 down-tablet:[&.menu-open_.nav-tabs]:border down-tablet:[&.menu-open_.nav-tabs]:border-line-strong down-tablet:[&.menu-open_.nav-tabs]:rounded-lg down-tablet:[&.menu-open_.nav-tabs]:z-40 down-tablet:[&.menu-open_.nav-tabs]:bg-surface down-tablet:[&.menu-open_.nav-tabs]:p-2.5 down-tablet:[&.menu-open_.nav-tabs]:[box-shadow:0_18px_40px_var(--shadow-panel)] down-tablet:[&.menu-open_.nav-tabs]:backdrop-filter-none down-tablet:[&.menu-open_.nav-tabs]:[isolation:isolate] down-mobile:[&_.eyebrow]:m-0 down-mobile:[&.menu-open_.nav-tabs]:right-3 down-mobile:[&.menu-open_.nav-tabs]:[width:min(360px,_calc(100vw_-_24px))] down-mobile:[&.menu-open_.sidebar-actions]:grid-cols-1"
    :class="{ 'menu-open': menuOpen }"
  >
    <div class="sidebar-header flex items-center justify-between gap-3">
      <div class="sidebar-branding grid justify-items-start gap-1.5">
        <h1>ComicHero</h1>
        <span
          v-if="version"
          class="version-tag inline-flex border border-line-strong rounded-full bg-surface text-muted py-0.5 px-2 text-ui-2xs font-bold leading-tight"
          >{{ version }}</span
        >
      </div>
      <button
        ref="mobileMenuButton"
        type="button"
        class="mobile-menu-button hidden place-items-center w-10.5 min-w-10.5 min-h-10.5 border border-line-strong rounded bg-surface text-control p-0 down-tablet:grid"
        :aria-expanded="menuOpen"
        aria-controls="primary-navigation"
        aria-label="Toggle navigation"
        @click="toggleMobileMenu"
      >
        <span
          class="menu-bars grid gap-1 w-4.5 [&_span]:block [&_span]:h-0.5 [&_span]:rounded-full [&_span]:[background:currentColor]"
          aria-hidden="true"
        >
          <span></span>
          <span></span>
          <span></span>
        </span>
      </button>
    </div>

    <nav
      id="primary-navigation"
      ref="primaryNavigation"
      class="nav-tabs grid gap-2 [&_:where(a,_button)]:min-h-10.5 [&_:where(a,_button)]:border [&_:where(a,_button)]:border-line-strong [&_:where(a,_button)]:rounded [&_:where(a,_button)]:bg-surface [&_:where(a,_button)]:text-control [&_:where(a,_button)]:py-2.5 [&_:where(a,_button)]:px-3.5 [&_:where(a,_button)]:flex [&_:where(a,_button)]:items-center [&_:where(a,_button)]:justify-between [&_:where(a,_button)]:gap-2.5 [&_:where(a,_button)]:text-left [&_:where(a,_button)]:no-underline [&_:where(a,_button).active]:border-primary [&_:where(a,_button).active]:bg-primary [&_:where(a,_button).active]:text-white down-tablet:[&_:where(a,_button)]:justify-start down-tablet:[&_:where(a,_button)]:text-left down-mobile:[&_:where(a,_button)]:min-h-10 down-mobile:[&_:where(a,_button)]:py-2 down-mobile:[&_:where(a,_button)]:px-1.5 down-mobile:[&_:where(a,_button)]:text-sm down-phone:[&_:where(a,_button)]:text-ui-sm"
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
      class="sidebar-actions grid gap-2.5 mt-auto down-tablet:absolute down-tablet:[top:50%] down-tablet:right-18 down-tablet:block down-tablet:mt-0 down-tablet:w-auto down-tablet:[transform:translateY(-50%)] down-mobile:[right:66px] down-phone:right-16"
    >
      <div
        v-if="user"
        ref="accountMenu"
        class="account-menu relative min-w-0 [&.open_.account-menu-trigger]:border-primary [&.open_.account-menu-trigger]:bg-surface [&.open_.account-menu-trigger]:[box-shadow:0_8px_18px_var(--shadow-soft)]"
        :class="{ open: accountMenuOpen }"
      >
        <button
          type="button"
          class="account-menu-trigger w-full min-h-14.5 grid [grid-template-columns:auto_minmax(0,_1fr)_auto] items-center gap-2.5 border border-line rounded bg-surface-soft text-control py-2.25 px-2.5 text-left down-tablet:w-10.5 down-tablet:h-10.5 down-tablet:min-h-10.5 down-tablet:inline-flex down-tablet:justify-center down-tablet:p-0 hover:border-primary hover:bg-surface hover:[box-shadow:0_8px_18px_var(--shadow-soft)] focus-visible:border-primary focus-visible:bg-surface focus-visible:[box-shadow:0_8px_18px_var(--shadow-soft)] down-tablet:[&_.account-avatar]:w-7.5 down-tablet:[&_.account-avatar]:min-w-7.5 down-tablet:[&_.account-avatar]:h-7.5"
          :aria-expanded="accountMenuOpen"
          aria-controls="account-menu-panel"
          @click="toggleAccountMenu"
        >
          <span
            class="account-avatar w-9 min-w-9 h-9 border border-primary rounded-full inline-flex items-center justify-center bg-primary-soft text-primary font-black leading-none [&.large]:w-11.5 [&.large]:min-w-11.5 [&.large]:h-11.5 [&.large]:text-ui-title-sm"
            aria-hidden="true"
            >{{ userInitial }}</span
          >
          <span
            class="account-trigger-copy min-w-0 grid gap-0.5 [&_strong]:break-anywhere [&_small]:text-muted [&_small]:font-bold down-tablet:hidden"
          >
            <strong>{{ user.name }}</strong>
          </span>
          <span class="account-menu-caret text-muted text-sm down-tablet:hidden" aria-hidden="true"
            >▾</span
          >
        </button>

        <div
          v-if="accountMenuOpen"
          id="account-menu-panel"
          class="account-menu-panel absolute left-0 [bottom:calc(100%_+_10px)] z-40 [width:min(320px,_calc(100vw_-_36px))] border border-line rounded bg-surface shadow-panel p-2.5 grid gap-2 down-tablet:absolute down-tablet:[top:calc(100%_+_8px)] down-tablet:right-0 down-tablet:[bottom:auto] down-tablet:[left:auto] down-tablet:[width:min(320px,_calc(100vw_-_36px))]"
        >
          <div
            class="account-menu-profile grid [grid-template-columns:auto_minmax(0,_1fr)] gap-3 items-center p-2 [&_span]:min-w-0 [&_span]:grid [&_span]:gap-0.5 [&_strong]:break-anywhere [&_small]:text-muted [&_small]:font-bold"
          >
            <span
              class="account-avatar large w-9 min-w-9 h-9 border border-primary rounded-full inline-flex items-center justify-center bg-primary-soft text-primary font-black leading-none [&.large]:w-11.5 [&.large]:min-w-11.5 [&.large]:h-11.5 [&.large]:text-ui-title-sm"
              aria-hidden="true"
              >{{ userInitial }}</span
            >
            <span>
              <strong>{{ user.name }}</strong>
            </span>
          </div>

          <div class="account-menu-section grid gap-2 border-t border-line border-b py-2.5 px-0">
            <div
              class="account-menu-label flex items-center gap-2.5 text-label py-0 px-2 [&_>_span]:w-5.5 [&_>_span]:min-w-5.5 [&_>_span]:text-center [&_>_span]:text-muted"
            >
              <span aria-hidden="true">◐</span>
              <strong>Display Mode</strong>
            </div>
            <div
              class="theme-selector account-theme-selector my-0 mx-2 grid grid-cols-3 gap-1 border border-line-strong rounded bg-panel-soft p-1 [&_button]:min-h-8.5 [&_button]:border-0 [&_button]:rounded-[6px] [&_button]:bg-transparent [&_button]:text-label [&_button]:p-1.5 [&_button]:text-ui-sm [&_button]:font-extrabold [&_button.active]:bg-primary [&_button.active]:text-white down-phone:[&_button]:text-ui-compact-xs down-phone:[&_button]:px-0.75"
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
            class="account-menu-item min-h-10.5 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-5.5 [&_>_span:first-child]:min-w-5.5 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger"
            @click="closeMenus"
          >
            <span aria-hidden="true">@</span>
            <span>Account settings</span>
          </router-link>
          <router-link
            :to="{ name: 'progress' }"
            class="account-menu-item min-h-10.5 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-5.5 [&_>_span:first-child]:min-w-5.5 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger"
            @click="closeMenus"
          >
            <span aria-hidden="true">%</span>
            <span>Progress</span>
          </router-link>
          <router-link
            :to="{ name: 'collections' }"
            class="account-menu-item min-h-10.5 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-5.5 [&_>_span:first-child]:min-w-5.5 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger"
            @click="closeMenus"
          >
            <span aria-hidden="true">◆</span>
            <span>My collections</span>
          </router-link>
          <router-link
            v-if="isAdmin"
            :to="{ name: 'settings' }"
            class="account-menu-item min-h-10.5 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-5.5 [&_>_span:first-child]:min-w-5.5 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger"
            @click="closeMenus"
          >
            <span aria-hidden="true">⚙</span>
            <span>App settings</span>
          </router-link>
          <router-link
            v-if="isAdmin"
            :to="{ name: 'users' }"
            class="account-menu-item min-h-10.5 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-5.5 [&_>_span:first-child]:min-w-5.5 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger"
            @click="closeMenus"
          >
            <span aria-hidden="true">#</span>
            <span>Manage users</span>
          </router-link>
          <a
            class="account-menu-item min-h-10.5 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-5.5 [&_>_span:first-child]:min-w-5.5 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger"
            href="https://discord.gg/GebUwAVP"
            target="_blank"
            rel="noreferrer"
            @click="closeMenus"
          >
            <span aria-hidden="true">♥</span>
            <span>Join the community</span>
          </a>
          <a
            class="account-menu-item min-h-10.5 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-5.5 [&_>_span:first-child]:min-w-5.5 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger"
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
            class="account-menu-item danger min-h-10.5 w-full flex items-center gap-2.5 border-0 rounded bg-transparent text-control py-2.5 px-2 font-extrabold text-left [&_>_span:first-child]:w-5.5 [&_>_span:first-child]:min-w-5.5 [&_>_span:first-child]:text-center [&_>_span:first-child]:text-muted [&:hover:not(:disabled)]:bg-surface-soft focus-visible:bg-surface-soft [&.danger]:text-danger"
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
        class="public-session-card grid gap-1 border border-line rounded bg-surface-soft p-3 down-tablet:block down-tablet:border-0 down-tablet:bg-transparent down-tablet:p-0 [&_span]:text-muted [&_span]:text-ui-md [&_span]:font-bold [&_.secondary-action]:mt-1.5 down-tablet:[&_strong]:hidden down-tablet:[&_span]:hidden down-tablet:[&_.secondary-action]:min-h-10.5 down-tablet:[&_.secondary-action]:mt-0 down-tablet:[&_.secondary-action]:py-0 down-tablet:[&_.secondary-action]:px-4.5"
      >
        <strong>Public access</strong>
        <span>Read-only access</span>
        <button
          type="button"
          class="secondary-action min-h-9.5 border border-line-strong rounded bg-surface text-control py-2 px-3 font-extrabold [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:bg-primary-soft focus-visible:border-primary focus-visible:bg-primary-soft"
          @click="login"
        >
          Log in
        </button>
      </div>
    </div>
  </aside>
</template>
