<script setup>
import { computed, reactive, ref, watch } from 'vue'

const scopeOptions = [
  { value: 'search', label: 'Search' },
  { value: 'detail', label: 'Details' },
  { value: 'import', label: 'Import' },
  { value: 'monitor', label: 'Monitor' },
]

const props = defineProps({
  users: {
    type: Array,
    default: () => [],
  },
  auditEvents: {
    type: Array,
    default: () => [],
  },
  savingUserID: {
    type: Number,
    default: null,
  },
  savingAdminUserID: {
    type: Number,
    default: null,
  },
  deletingUserID: {
    type: Number,
    default: null,
  },
  currentUserID: {
    type: Number,
    default: null,
  },
})

const emit = defineEmits(['save', 'save-admin', 'delete-user'])
const drafts = reactive({})
const userQuery = ref('')
const roleFilter = ref('all')
const creationSort = ref('newest')

const filteredUsers = computed(() => {
  const query = userQuery.value.trim().toLocaleLowerCase()
  return props.users
    .filter(({ user }) => {
      const matchesQuery =
        !query ||
        [user.name, user.email].some((value) => value?.toLocaleLowerCase().includes(query))
      const matchesRole =
        roleFilter.value === 'all' || (roleFilter.value === 'admin' ? user.isAdmin : !user.isAdmin)
      return matchesQuery && matchesRole
    })
    .sort((left, right) => {
      const direction = creationSort.value === 'newest' ? -1 : 1
      return left.user.createdAt.localeCompare(right.user.createdAt) * direction
    })
})

watch(
  () => props.users,
  (users) => {
    users.forEach((entry) => {
      const userID = entry.user.id
      const permissions = entry.metronPermissions || {}
      drafts[userID] = {
        allowed: Boolean(permissions.allowed),
        scopes: normalizeScopes(permissions.scopes),
        hourlyLimit: permissions.hourlyLimit ?? 0,
        isAdmin: Boolean(entry.user.isAdmin),
      }
    })
  },
  { immediate: true },
)

function normalizeScopes(scopes = []) {
  if (!Array.isArray(scopes)) return []
  if (scopes.includes('*')) return ['*']
  return scopes.filter((scope) => scopeOptions.some((option) => option.value === scope))
}

function draftFor(userID) {
  if (!drafts[userID]) {
    drafts[userID] = { allowed: false, scopes: [], hourlyLimit: 0, isAdmin: false }
  }
  return drafts[userID]
}

function scopesFor(userID) {
  return normalizeScopes(draftFor(userID).scopes)
}

function toggleAll(userID, checked) {
  draftFor(userID).scopes = checked ? ['*'] : []
}

function toggleScope(userID, scope, checked) {
  const draft = draftFor(userID)
  const current = new Set(scopesFor(userID).filter((value) => value !== '*'))
  if (checked) {
    current.add(scope)
  } else {
    current.delete(scope)
  }
  draft.scopes = Array.from(current)
}

function save(entry) {
  const draft = draftFor(entry.user.id)
  emit('save', entry.user.id, {
    allowed: draft.allowed,
    scopes: draft.allowed ? scopesFor(entry.user.id) : [],
    hourlyLimit: Number(draft.hourlyLimit) || 0,
  })
}

function saveAdmin(entry) {
  emit('save-admin', entry.user.id, {
    isAdmin: Boolean(draftFor(entry.user.id).isAdmin),
  })
}

function formatTimestamp(value) {
  if (!value) return 'Unknown'
  const normalized = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(value)
    ? `${value.replace(' ', 'T')}Z`
    : value
  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}
</script>

<template>
  <section class="browse-view user-management-view grid w-full min-w-0 max-w-[1160px] gap-5">
    <div
      v-if="users.length === 0"
      class="empty-panel border border-dashed border-line-strong rounded bg-surface-soft text-muted p-5 font-extrabold"
    >
      No users yet.
    </div>

    <template v-else>
      <header
        class="user-directory-toolbar flex min-w-0 items-end justify-between gap-4 border-b border-line py-1 px-3 down-tablet:flex-col down-tablet:items-stretch [&_h3]:mt-0.5 [&_h3]:mx-0 [&_h3]:mb-0"
      >
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Accounts</p>
          <h3>{{ users.length }} {{ users.length === 1 ? 'user' : 'users' }}</h3>
        </div>
        <div
          class="user-directory-filters grid w-full max-w-[680px] min-w-0 [grid-template-columns:minmax(220px,_1fr)_auto_auto] gap-2 down-tablet:max-w-none down-tablet:[grid-template-columns:1fr_1fr] [&_select]:min-h-10 [&_select]:whitespace-nowrap down-mobile:[&_.user-search-field]:col-span-full"
        >
          <label
            class="user-search-field w-full min-w-0 max-w-[360px] down-tablet:col-span-2 down-tablet:max-w-none [&_input]:min-h-10"
          >
            <span class="sr-only">Search users</span>
            <input v-model="userQuery" type="search" placeholder="Search by name or email" />
          </label>
          <label>
            <span class="sr-only">Filter by role</span>
            <select v-model="roleFilter">
              <option value="all">All roles</option>
              <option value="admin">Admins</option>
              <option value="user">Users</option>
            </select>
          </label>
          <label>
            <span class="sr-only">Sort by creation date</span>
            <select v-model="creationSort">
              <option value="newest">Newest first</option>
              <option value="oldest">Oldest first</option>
            </select>
          </label>
        </div>
      </header>

      <div
        v-if="filteredUsers.length === 0"
        class="empty-panel border border-dashed border-line-strong rounded bg-surface-soft text-muted p-5 font-extrabold"
      >
        No users match the current filters.
      </div>

      <div v-else class="user-permission-list grid gap-2 min-w-0">
        <details
          v-for="entry in filteredUsers"
          :key="entry.user.id"
          class="user-permission-row border border-line rounded bg-surface-soft p-0 grid min-w-0 max-w-full overflow-hidden [&[open]_.user-permission-header]:border-b [&[open]_.user-permission-header]:border-line [&[open]_.user-permission-header::after]:[transform:translateY(-50%)_rotate(90deg)]"
        >
          <summary
            class="user-permission-header flex items-center justify-between gap-3.5 min-w-0 flex-wrap pt-3.25 pr-12 pb-3.25 pl-4.5 cursor-pointer [list-style:none] relative down-mobile:items-stretch down-mobile:flex-col [&::-webkit-details-marker]:hidden after:[content:'›'] after:absolute after:right-4.5 after:[top:50%] after:text-muted after:[font-size:1.5rem] after:leading-none after:[transform:translateY(-50%)] after:[transition:transform_140ms_ease] [&_h3]:mb-0.5 [&_p]:text-muted [&_p]:text-ui-md [&_p]:font-bold"
          >
            <div
              class="user-summary min-w-0 [&_h3]:break-anywhere [&_p]:break-anywhere [&_.user-account-dates]:flex [&_.user-account-dates]:gap-y-1.5 [&_.user-account-dates]:gap-x-3.5 [&_.user-account-dates]:flex-wrap [&_.user-account-dates]:mt-1.25 [&_.user-account-dates]:text-ui-compact-xs [&_.user-account-dates]:font-ui-semibold"
            >
              <h3>{{ entry.user.name }}</h3>
              <p>{{ entry.user.email || 'No email address' }}</p>
              <p class="user-account-dates">
                <span>Created {{ formatTimestamp(entry.user.createdAt) }}</span>
                <span v-if="entry.user.emailVerifiedAt">
                  Email verified {{ formatTimestamp(entry.user.emailVerifiedAt) }}
                </span>
                <span v-else>Email not verified</span>
                <span>Last login {{ formatTimestamp(entry.user.lastLoginAt) }}</span>
              </p>
            </div>
            <div
              class="user-summary-badges flex items-center justify-end gap-2 text-muted text-ui-compact font-extrabold down-mobile:justify-start down-mobile:flex-wrap [&_span]:border [&_span]:border-line [&_span]:rounded-full [&_span]:bg-surface [&_span]:py-1 [&_span]:px-2 [&_.user-role-badge]:border-warning-border [&_.user-role-badge]:bg-warning-soft [&_.user-role-badge]:text-warning"
              aria-hidden="true"
            >
              <span v-if="entry.user.isAdmin" class="user-role-badge">Admin</span>
              <span>{{
                entry.metronPermissions?.allowed ? 'Metron enabled' : 'Metron disabled'
              }}</span>
            </div>
          </summary>

          <div
            class="user-card-sections grid min-w-0 [grid-template-columns:minmax(220px,_0.7fr)_minmax(420px,_1.3fr)] items-stretch [gap:0] down-tablet:grid-cols-1 down-mobile:grid-cols-1"
          >
            <section
              class="user-card-section account-section grid min-w-0 content-start gap-3 border-r border-line py-4 px-4.5 down-tablet:border-r-0 down-tablet:border-b down-tablet:border-line"
            >
              <div
                class="section-heading [&_h4]:mt-0.5 [&_h4]:mx-0 [&_h4]:mb-0 [&_h4]:text-label [&_h4]:text-base [&_h4]:font-bold"
              >
                <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Account</p>
                <h4>Role and access</h4>
              </div>
              <div
                class="account-control-grid grid grid-cols-1 gap-2 max-w-65 down-mobile:max-w-none"
              >
                <label
                  class="compact-toggle inline-flex items-center gap-2 min-h-8.5 border border-line rounded bg-surface text-label py-1.75 px-2.5 font-extrabold leading-ui-tight"
                >
                  <input
                    v-model="draftFor(entry.user.id).isAdmin"
                    type="checkbox"
                    :disabled="entry.user.id === currentUserID"
                  />
                  <span>Admin user</span>
                </label>
                <button
                  type="button"
                  class="secondary-action min-h-9.5 border border-line-strong rounded bg-surface text-control py-2 px-3 font-extrabold [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:bg-primary-soft focus-visible:border-primary focus-visible:bg-primary-soft"
                  :disabled="savingAdminUserID === entry.user.id || entry.user.id === currentUserID"
                  @click="saveAdmin(entry)"
                >
                  {{ savingAdminUserID === entry.user.id ? 'Saving...' : 'Save role' }}
                </button>
                <button
                  type="button"
                  class="danger-text-button min-h-9.5 border border-danger-border rounded bg-surface text-danger py-2 px-3 font-black [&:hover:not(:disabled)]:border-danger-border [&:hover:not(:disabled)]:bg-danger-soft focus-visible:border-danger-border focus-visible:bg-danger-soft"
                  :disabled="deletingUserID === entry.user.id || entry.user.id === currentUserID"
                  @click="$emit('delete-user', entry.user.id)"
                >
                  {{ deletingUserID === entry.user.id ? 'Deleting...' : 'Delete user' }}
                </button>
              </div>
            </section>

            <section
              class="user-card-section metron-permission-editor grid content-start gap-3 min-w-0 py-4 px-4.5"
            >
              <div
                class="section-heading [&_h4]:mt-0.5 [&_h4]:mx-0 [&_h4]:mb-0 [&_h4]:text-label [&_h4]:text-base [&_h4]:font-bold"
              >
                <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Metron</p>
                <h4>API permissions</h4>
              </div>

              <div
                class="metron-settings-grid grid min-w-0 grid-cols-[minmax(180px,240px)_minmax(0,1fr)] items-start gap-3 down-tablet:grid-cols-1 down-mobile:grid-cols-1"
              >
                <div class="metron-control-column grid gap-2.5 min-w-0">
                  <label
                    class="compact-toggle inline-flex items-center gap-2 min-h-8.5 border border-line rounded bg-surface text-label py-1.75 px-2.5 font-extrabold leading-ui-tight"
                  >
                    <input v-model="draftFor(entry.user.id).allowed" type="checkbox" />
                    <span>{{ draftFor(entry.user.id).allowed ? 'Enabled' : 'Disabled' }}</span>
                  </label>
                  <label
                    class="hourly-limit-field grid gap-1.25 w-full text-label font-extrabold [&_small]:text-muted [&_small]:text-ui-md [&_small]:font-bold [&_input]:w-full [&_input]:min-w-0 [&_input]:min-h-9.5 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-ink [&_input]:py-2 [&_input]:px-2.5"
                  >
                    <span>Hourly endpoint limit</span>
                    <input
                      v-model.number="draftFor(entry.user.id).hourlyLimit"
                      min="0"
                      step="1"
                      type="number"
                    />
                    <small>0 means unlimited.</small>
                  </label>
                  <button
                    type="button"
                    class="secondary-action min-h-9.5 border border-line-strong rounded bg-surface text-control py-2 px-3 font-extrabold [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:bg-primary-soft focus-visible:border-primary focus-visible:bg-primary-soft"
                    :disabled="savingUserID === entry.user.id"
                    @click="save(entry)"
                  >
                    {{ savingUserID === entry.user.id ? 'Saving...' : 'Save permissions' }}
                  </button>
                </div>

                <fieldset
                  class="permission-scopes [&_label]:inline-flex [&_label]:items-center [&_label]:gap-2 [&_label]:min-h-8.5 [&_label]:border [&_label]:border-line [&_label]:rounded [&_label]:bg-surface [&_label]:text-label [&_label]:py-1.75 [&_label]:px-2.5 [&_label]:font-extrabold [&_label]:leading-ui-tight border-0 p-0 m-0 grid [grid-template-columns:repeat(auto-fit,_minmax(126px,_1fr))] gap-2 min-w-0 [&_legend]:w-full [&_legend]:mb-0.5 [&_legend]:text-muted [&_legend]:text-ui-sm [&_legend]:font-extrabold [&_legend]:uppercase disabled:opacity-55 down-mobile:grid-cols-1"
                  :disabled="!draftFor(entry.user.id).allowed"
                >
                  <legend>Allowed endpoint scopes</legend>
                  <label>
                    <input
                      type="checkbox"
                      :checked="scopesFor(entry.user.id).includes('*')"
                      @change="toggleAll(entry.user.id, $event.target.checked)"
                    />
                    <span>All endpoints</span>
                  </label>
                  <label v-for="option in scopeOptions" :key="option.value">
                    <input
                      type="checkbox"
                      :checked="
                        scopesFor(entry.user.id).includes('*') ||
                        scopesFor(entry.user.id).includes(option.value)
                      "
                      :disabled="scopesFor(entry.user.id).includes('*')"
                      @change="toggleScope(entry.user.id, option.value, $event.target.checked)"
                    />
                    <span>{{ option.label }}</span>
                  </label>
                </fieldset>
              </div>
            </section>
          </div>
        </details>
      </div>
    </template>

    <section
      class="detail-panel min-w-0 max-w-full min-h-90 border border-line rounded bg-panel p-5 shadow-detail down-mobile:min-h-0 down-mobile:p-3.5"
    >
      <header
        class="section-heading [&_h4]:mt-0.5 [&_h4]:mx-0 [&_h4]:mb-0 [&_h4]:text-label [&_h4]:text-base [&_h4]:font-bold"
      >
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Audit log</p>
        <h3>Recent changes</h3>
      </header>
      <p
        v-if="auditEvents.length === 0"
        class="empty-state grid gap-3 justify-items-start border border-dashed border-line-strong rounded bg-panel-soft text-muted p-4"
      >
        No state-changing actions recorded yet.
      </p>
      <div
        v-else
        class="table-scroll mt-4 w-full max-w-full overflow-x-auto overscroll-x-contain rounded border border-line"
      >
        <table class="w-full min-w-[640px] border-collapse text-left">
          <thead>
            <tr>
              <th class="border-b border-line bg-surface-muted px-3 py-2.5">When</th>
              <th class="border-b border-line bg-surface-muted px-3 py-2.5">User</th>
              <th class="border-b border-line bg-surface-muted px-3 py-2.5">Action</th>
              <th class="border-b border-line bg-surface-muted px-3 py-2.5">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="event in auditEvents" :key="event.id">
              <td class="border-b border-line px-3 py-2.5 align-top">
                {{ formatTimestamp(event.occurredAt) }}
              </td>
              <td class="border-b border-line px-3 py-2.5 align-top">
                {{ event.userName || 'System / unauthenticated' }}
              </td>
              <td class="border-b border-line px-3 py-2.5 align-top">
                <code class="whitespace-nowrap">{{ event.method }} {{ event.path }}</code>
              </td>
              <td class="border-b border-line px-3 py-2.5 align-top">{{ event.statusCode }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>
