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
  <section class="browse-view user-management-view">
    <div v-if="users.length === 0" class="empty-panel">No users yet.</div>

    <template v-else>
      <header class="user-directory-toolbar">
        <div>
          <p class="eyebrow">Accounts</p>
          <h3>{{ users.length }} {{ users.length === 1 ? 'user' : 'users' }}</h3>
        </div>
        <div class="user-directory-filters">
          <label class="user-search-field">
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

      <div v-else class="user-permission-list">
        <details v-for="entry in filteredUsers" :key="entry.user.id" class="user-permission-row">
          <summary class="user-permission-header">
            <div class="user-summary">
              <h3>{{ entry.user.name }}</h3>
              <p>{{ entry.user.email || 'No email address' }}</p>
              <p class="user-account-dates">
                <span>Created {{ formatTimestamp(entry.user.createdAt) }}</span>
                <span v-if="entry.user.emailVerifiedAt">
                  Email verified {{ formatTimestamp(entry.user.emailVerifiedAt) }}
                </span>
                <span v-else>Email not verified</span>
              </p>
            </div>
            <div class="user-summary-badges" aria-hidden="true">
              <span v-if="entry.user.isAdmin" class="user-role-badge">Admin</span>
              <span>{{
                entry.metronPermissions?.allowed ? 'Metron enabled' : 'Metron disabled'
              }}</span>
            </div>
          </summary>

          <div class="user-card-sections">
            <section class="user-card-section account-section">
              <div class="section-heading">
                <p class="eyebrow">Account</p>
                <h4>Role and access</h4>
              </div>
              <div class="account-control-grid">
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

            <section class="user-card-section metron-permission-editor">
              <div class="section-heading">
                <p class="eyebrow">Metron</p>
                <h4>API permissions</h4>
              </div>

              <div class="metron-settings-grid">
                <div class="metron-control-column">
                  <label class="compact-toggle">
                    <input v-model="draftFor(entry.user.id).allowed" type="checkbox" />
                    <span>{{ draftFor(entry.user.id).allowed ? 'Enabled' : 'Disabled' }}</span>
                  </label>
                  <label class="hourly-limit-field">
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
  </section>
</template>
