<script setup>
import { reactive, watch } from 'vue'

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
  registrationMode: {
    type: String,
    default: 'invite_only',
  },
  savingRegistrationMode: {
    type: Boolean,
    default: false,
  },
  publicAccess: {
    type: Boolean,
    default: false,
  },
  savingPublicAccess: {
    type: Boolean,
    default: false,
  },
  invite: {
    type: Object,
    default: null,
  },
  generatingInvite: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits([
  'save',
  'save-admin',
  'delete-user',
  'update-registration-mode',
  'update-public-access',
  'generate-invite',
])
const drafts = reactive({})

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

function registrationModeLabel(mode) {
  return mode === 'open' ? 'Open registration' : 'Invite only'
}
</script>

<template>
  <section class="browse-view user-management-view">
    <section class="user-access-panel">
      <div class="user-registration-panel">
        <div>
          <p class="eyebrow">Registration</p>
          <h3>{{ registrationModeLabel(registrationMode) }}</h3>
          <p class="muted">
            {{
              registrationMode === 'open'
                ? 'Anyone who can reach this server can register without an invite.'
                : 'New accounts need a single-use invite token to register.'
            }}
          </p>
        </div>

        <div class="registration-mode-toggle" role="group" aria-label="Registration mode">
          <button
            type="button"
            :class="{ active: registrationMode === 'invite_only' }"
            :disabled="savingRegistrationMode"
            @click="$emit('update-registration-mode', 'invite_only')"
          >
            Invite only
          </button>
          <button
            type="button"
            :class="{ active: registrationMode === 'open' }"
            :disabled="savingRegistrationMode"
            @click="$emit('update-registration-mode', 'open')"
          >
            Open registration
          </button>
        </div>

        <p v-if="registrationMode === 'open'" class="warning-copy">
          Open registration gives new accounts full read/write access to the shared library.
        </p>
      </div>

      <div class="user-invite-panel">
        <div>
          <p class="eyebrow">Invites</p>
          <h3>Invite a user</h3>
          <p class="muted">
            {{
              registrationMode === 'open'
                ? 'Open registration is enabled, so invite tokens are optional right now.'
                : 'Generate a single-use token for a new account.'
            }}
          </p>
        </div>
        <button
          class="primary-button"
          type="button"
          :disabled="generatingInvite"
          @click="$emit('generate-invite')"
        >
          {{ generatingInvite ? 'Generating...' : 'Generate invite' }}
        </button>
        <div v-if="invite?.token" class="invite-token-box">
          <span>Invite token</span>
          <code>{{ invite.token }}</code>
          <small>Expires at {{ invite.expiresAt }}</small>
        </div>
      </div>

      <div class="user-public-panel">
        <div>
          <p class="eyebrow">Public access</p>
          <h3>{{ publicAccess ? 'Read-only visitors' : 'Private library' }}</h3>
          <p class="muted">
            {{
              publicAccess
                ? 'Anonymous visitors can browse and export reading orders as CBL.'
                : 'Anonymous visitors must log in before seeing the library.'
            }}
          </p>
        </div>
        <div class="registration-mode-toggle" role="group" aria-label="Public access">
          <button
            type="button"
            :class="{ active: !publicAccess }"
            :disabled="savingPublicAccess"
            @click="$emit('update-public-access', false)"
          >
            Private
          </button>
          <button
            type="button"
            :class="{ active: publicAccess }"
            :disabled="savingPublicAccess"
            @click="$emit('update-public-access', true)"
          >
            Public read-only
          </button>
        </div>
        <p v-if="publicAccess" class="warning-copy">
          Public visitors cannot edit data, but they can see your shared library.
        </p>
      </div>
    </section>

    <div v-if="users.length === 0" class="empty-panel">No users yet.</div>

    <div v-else class="user-permission-list">
      <article v-for="entry in users" :key="entry.user.id" class="user-permission-row">
        <div class="user-permission-header">
          <div class="user-summary">
            <h3>{{ entry.user.name }}</h3>
            <p>{{ entry.user.isAdmin ? 'Admin' : 'User' }}</p>
          </div>
        </div>

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
      </article>
    </div>
  </section>
</template>
