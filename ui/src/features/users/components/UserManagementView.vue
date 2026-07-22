<script setup>
import { computed, reactive, ref, watch } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseSelect from '@/shared/components/form/BaseSelect.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import UserAuditLog from './UserAuditLog.vue'

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
  auditPagination: {
    type: Object,
    default: () => ({ limit: 25, offset: 0, total: 0, hasMore: false }),
  },
  auditLoading: {
    type: Boolean,
    default: false,
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

const emit = defineEmits(['save', 'save-admin', 'delete-user', 'load-audit'])
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
    <div v-if="users.length === 0" class="empty-panel">No users yet.</div>

    <template v-else>
      <header class="user-directory-toolbar">
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Accounts</p>
          <h3>{{ users.length }} {{ users.length === 1 ? 'user' : 'users' }}</h3>
        </div>
        <div class="user-directory-filters">
          <label class="user-search-field">
            <span class="sr-only">Search users</span>
            <BaseTextInput
              v-model="userQuery"
              type="search"
              placeholder="Search by name or email"
            />
          </label>
          <label>
            <span class="sr-only">Filter by role</span>
            <BaseSelect v-model="roleFilter" variant="nowrap">
              <option value="all">All roles</option>
              <option value="admin">Admins</option>
              <option value="user">Users</option>
            </BaseSelect>
          </label>
          <label>
            <span class="sr-only">Sort by creation date</span>
            <BaseSelect v-model="creationSort" variant="nowrap">
              <option value="newest">Newest first</option>
              <option value="oldest">Oldest first</option>
            </BaseSelect>
          </label>
        </div>
      </header>

      <div v-if="filteredUsers.length === 0" class="empty-panel">
        No users match the current filters.
      </div>

      <div v-else class="user-permission-list grid gap-2 min-w-0">
        <details v-for="entry in filteredUsers" :key="entry.user.id" class="user-permission-row">
          <summary class="user-permission-header">
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
                <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Account</p>
                <h4>Role and access</h4>
              </div>
              <div
                class="account-control-grid grid grid-cols-1 gap-2 max-w-64 down-mobile:max-w-none"
              >
                <label class="compact-toggle">
                  <input
                    v-model="draftFor(entry.user.id).isAdmin"
                    type="checkbox"
                    :disabled="entry.user.id === currentUserID"
                  />
                  <span>Admin user</span>
                </label>
                <BaseButton
                  variant="neutral"
                  :disabled="savingAdminUserID === entry.user.id || entry.user.id === currentUserID"
                  @click="saveAdmin(entry)"
                >
                  {{ savingAdminUserID === entry.user.id ? 'Saving...' : 'Save role' }}
                </BaseButton>
                <BaseButton
                  variant="danger-ghost"
                  :disabled="deletingUserID === entry.user.id || entry.user.id === currentUserID"
                  @click="$emit('delete-user', entry.user.id)"
                >
                  {{ deletingUserID === entry.user.id ? 'Deleting...' : 'Delete user' }}
                </BaseButton>
              </div>
            </section>

            <section class="user-card-section metron-permission-editor">
              <div class="section-heading">
                <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Metron</p>
                <h4>API permissions</h4>
              </div>

              <div class="metron-settings-grid">
                <div class="metron-control-column grid gap-2.5 min-w-0">
                  <label class="compact-toggle">
                    <input v-model="draftFor(entry.user.id).allowed" type="checkbox" />
                    <span>{{ draftFor(entry.user.id).allowed ? 'Enabled' : 'Disabled' }}</span>
                  </label>
                  <label class="hourly-limit-field grid gap-1 w-full text-label font-extrabold">
                    <span>Hourly endpoint limit</span>
                    <BaseTextInput
                      v-model.number="draftFor(entry.user.id).hourlyLimit"
                      size="dense"
                      min="0"
                      step="1"
                      type="number"
                    />
                    <small>0 means unlimited.</small>
                  </label>
                  <BaseButton
                    variant="neutral"
                    :disabled="savingUserID === entry.user.id"
                    @click="save(entry)"
                  >
                    {{ savingUserID === entry.user.id ? 'Saving...' : 'Save permissions' }}
                  </BaseButton>
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

    <UserAuditLog
      :users="users"
      :audit-events="auditEvents"
      :audit-pagination="auditPagination"
      :audit-loading="auditLoading"
      @load="$emit('load-audit', $event)"
    />
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.user-summary :is(h3, p) {
  overflow-wrap: anywhere;
}

.user-account-dates {
  @apply mt-1 flex flex-wrap gap-x-3.5 gap-y-1.5 text-xs font-ui-semibold;
}

.user-summary-badges span {
  @apply rounded-full border border-line bg-surface px-2 py-1;
}

.user-summary-badges .user-role-badge {
  @apply border-warning-border bg-warning-soft text-warning;
}

.section-heading h4 {
  @apply mt-0.5 mb-0 text-base font-bold text-label;
}

.hourly-limit-field small {
  @apply text-sm font-bold text-muted;
}

.permission-scopes label {
  @apply inline-flex min-h-8 items-center gap-2 rounded border border-line bg-surface px-2.5 py-2 font-extrabold leading-ui-tight text-label;
}

.permission-scopes legend {
  @apply mb-0.5 w-full text-sm font-extrabold uppercase text-muted;
}

.empty-panel {
  @apply border border-dashed border-line-strong rounded bg-surface-soft text-muted p-5 font-extrabold;
}

.user-directory-toolbar {
  @apply flex min-w-0 items-end justify-between gap-4 border-b border-line py-1 px-3 down-tablet:flex-col down-tablet:items-stretch [&_h3]:mt-0.5 [&_h3]:mx-0 [&_h3]:mb-0;
}

.user-directory-filters {
  @apply grid w-full max-w-[680px] min-w-0 grid-cols-[minmax(220px,1fr)_auto_auto] gap-2 down-tablet:max-w-none down-tablet:grid-cols-[1fr_1fr] down-mobile:[&_.user-search-field]:col-span-full;
}

.user-search-field {
  @apply w-full min-w-0 max-w-[360px] down-tablet:col-span-2 down-tablet:max-w-none;
}

.user-permission-row {
  @apply border border-line rounded bg-surface-soft p-0 grid min-w-0 max-w-full overflow-hidden [&[open]_.user-permission-header]:border-b [&[open]_.user-permission-header]:border-line [&[open]_.user-permission-header::after]:transform-[translateY(-50%)_rotate(90deg)];
}

.user-permission-header {
  @apply flex items-center justify-between gap-3.5 min-w-0 flex-wrap pt-3 pr-12 pb-3 pl-4 cursor-pointer [list-style:none] relative down-mobile:items-stretch down-mobile:flex-col [&::-webkit-details-marker]:hidden after:[content:'›'] after:absolute after:right-4 after:top-[50%] after:text-muted after:text-2xl after:leading-none after:transform-[translateY(-50%)] after:[transition:transform_140ms_ease] [&_h3]:mb-0.5 [&_p]:text-muted [&_p]:text-sm [&_p]:font-bold;
}

.user-summary-badges {
  @apply flex items-center justify-end gap-2 text-muted text-xs font-extrabold down-mobile:justify-start down-mobile:flex-wrap;
}

.user-card-sections {
  @apply grid min-w-0 grid-cols-[minmax(220px,0.7fr)_minmax(420px,1.3fr)] items-stretch gap-0 down-tablet:grid-cols-1 down-mobile:grid-cols-1;
}

.user-card-section.account-section {
  @apply grid min-w-0 content-start gap-3 border-r border-line py-4 px-4 down-tablet:border-r-0 down-tablet:border-b down-tablet:border-line;
}

.compact-toggle {
  @apply inline-flex items-center gap-2 min-h-8 border border-line rounded bg-surface text-label py-2 px-2.5 font-extrabold leading-ui-tight;
}

.user-card-section.metron-permission-editor {
  @apply grid content-start gap-3 min-w-0 py-4 px-4;
}

.metron-settings-grid {
  @apply grid min-w-0 grid-cols-[minmax(180px,240px)_minmax(0,1fr)] items-start gap-3 down-tablet:grid-cols-1 down-mobile:grid-cols-1;
}

.permission-scopes {
  @apply border-0 p-0 m-0 grid grid-cols-[repeat(auto-fit,minmax(126px,1fr))] gap-2 min-w-0 disabled:opacity-55 down-mobile:grid-cols-1;
}

.section-heading {
  @apply flex items-end justify-between gap-3 down-mobile:items-start down-mobile:flex-col;
}
</style>
