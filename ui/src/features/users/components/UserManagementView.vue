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
  <section class="browse-view user-management-view grid gap-5 [max-width:1160px]">
    <div v-if="users.length === 0" class="empty-panel">No users yet.</div>

    <template v-else>
      <header
        class="user-directory-toolbar flex items-end justify-between gap-4 [border-bottom:1px_solid_var(--line)] [padding:4px_0_12px] down-mobile:[align-items:stretch] down-mobile:flex-col"
      >
        <div>
          <p class="eyebrow">Accounts</p>
          <h3>{{ users.length }} {{ users.length === 1 ? 'user' : 'users' }}</h3>
        </div>
        <div
          class="user-directory-filters grid [grid-template-columns:minmax(220px,_1fr)_auto_auto] gap-2 [width:min(680px,_100%)] down-mobile:[grid-template-columns:1fr_1fr] down-mobile:w-full"
        >
          <label class="user-search-field [width:min(360px,_100%)] down-mobile:w-full">
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

      <div v-if="filteredUsers.length === 0" class="empty-panel">
        No users match the current filters.
      </div>

      <div v-else class="user-permission-list grid gap-2 min-w-0">
        <details
          v-for="entry in filteredUsers"
          :key="entry.user.id"
          class="user-permission-row border border-line rounded bg-surface-soft [padding:0] grid min-w-0 [max-width:100%] overflow-hidden"
        >
          <summary
            class="user-permission-header flex items-center justify-between gap-3.5 min-w-0 flex-wrap [padding:13px_48px_13px_18px] cursor-pointer [list-style:none] relative down-mobile:[align-items:stretch] down-mobile:flex-col"
          >
            <div class="user-summary min-w-0">
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
              class="user-summary-badges flex items-center justify-end gap-2 text-muted [font-size:0.78rem] font-extrabold down-mobile:[justify-content:flex-start] down-mobile:flex-wrap"
              aria-hidden="true"
            >
              <span v-if="entry.user.isAdmin" class="user-role-badge">Admin</span>
              <span>{{
                entry.metronPermissions?.allowed ? 'Metron enabled' : 'Metron disabled'
              }}</span>
            </div>
          </summary>

          <div
            class="user-card-sections grid [grid-template-columns:minmax(220px,_0.7fr)_minmax(420px,_1.3fr)] [gap:0] [align-items:stretch] min-w-0"
          >
            <section
              class="user-card-section account-section grid content-start gap-3 min-w-0 [padding:16px_18px] [border-right:1px_solid_var(--line)] down-mobile:[border-right:0] down-mobile:[border-bottom:1px_solid_var(--line)]"
            >
              <div class="section-heading">
                <p class="eyebrow">Account</p>
                <h4>Role and access</h4>
              </div>
              <div
                class="account-control-grid grid [grid-template-columns:1fr] gap-2 [max-width:260px] down-mobile:[max-width:none]"
              >
                <label class="compact-toggle">
                  <input
                    v-model="draftFor(entry.user.id).isAdmin"
                    type="checkbox"
                    :disabled="entry.user.id === currentUserID"
                  />
                  <span>Admin user</span>
                </label>
                <button
                  type="button"
                  class="secondary-action"
                  :disabled="savingAdminUserID === entry.user.id || entry.user.id === currentUserID"
                  @click="saveAdmin(entry)"
                >
                  {{ savingAdminUserID === entry.user.id ? 'Saving...' : 'Save role' }}
                </button>
                <button
                  type="button"
                  class="danger-text-button"
                  :disabled="deletingUserID === entry.user.id || entry.user.id === currentUserID"
                  @click="$emit('delete-user', entry.user.id)"
                >
                  {{ deletingUserID === entry.user.id ? 'Deleting...' : 'Delete user' }}
                </button>
              </div>
            </section>

            <section
              class="user-card-section metron-permission-editor grid content-start gap-3 min-w-0 [padding:16px_18px] min-w-0"
            >
              <div class="section-heading">
                <p class="eyebrow">Metron</p>
                <h4>API permissions</h4>
              </div>

              <div
                class="metron-settings-grid grid [grid-template-columns:minmax(180px,_240px)_1fr] gap-3 items-start min-w-0"
              >
                <div class="metron-control-column grid gap-2.5 min-w-0">
                  <label class="compact-toggle">
                    <input v-model="draftFor(entry.user.id).allowed" type="checkbox" />
                    <span>{{ draftFor(entry.user.id).allowed ? 'Enabled' : 'Disabled' }}</span>
                  </label>
                  <label class="hourly-limit-field grid [gap:5px] w-full text-label font-extrabold">
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
                    class="secondary-action"
                    :disabled="savingUserID === entry.user.id"
                    @click="save(entry)"
                  >
                    {{ savingUserID === entry.user.id ? 'Saving...' : 'Save permissions' }}
                  </button>
                </div>

                <fieldset class="permission-scopes" :disabled="!draftFor(entry.user.id).allowed">
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

    <section class="detail-panel">
      <header class="section-heading">
        <p class="eyebrow">Audit log</p>
        <h3>Recent changes</h3>
      </header>
      <p v-if="auditEvents.length === 0" class="empty-state">
        No state-changing actions recorded yet.
      </p>
      <div v-else class="table-scroll">
        <table>
          <thead>
            <tr>
              <th>When</th>
              <th>User</th>
              <th>Action</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="event in auditEvents" :key="event.id">
              <td>{{ formatTimestamp(event.occurredAt) }}</td>
              <td>{{ event.userName || 'System / unauthenticated' }}</td>
              <td>
                <code>{{ event.method }} {{ event.path }}</code>
              </td>
              <td>{{ event.statusCode }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>
