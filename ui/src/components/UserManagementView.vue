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
})

const emit = defineEmits(['save'])
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
    drafts[userID] = { allowed: false, scopes: [], hourlyLimit: 0 }
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
</script>

<template>
  <section class="browse-view user-management-view">
    <div v-if="users.length === 0" class="empty-panel">
      No users yet.
    </div>

    <div v-else class="user-permission-list">
      <article v-for="entry in users" :key="entry.user.id" class="user-permission-row">
        <div class="user-permission-header">
          <div>
            <h3>{{ entry.user.name }}</h3>
            <p>{{ entry.user.isAdmin ? 'Admin' : 'User' }}</p>
          </div>
          <label class="compact-toggle">
            <input v-model="draftFor(entry.user.id).allowed" type="checkbox">
            <span>Metron access</span>
          </label>
        </div>

        <fieldset class="permission-scopes" :disabled="!draftFor(entry.user.id).allowed">
          <legend>Allowed endpoint scopes</legend>
          <label>
            <input
              type="checkbox"
              :checked="scopesFor(entry.user.id).includes('*')"
              @change="toggleAll(entry.user.id, $event.target.checked)"
            >
            <span>All Metron endpoints</span>
          </label>
          <label v-for="option in scopeOptions" :key="option.value">
            <input
              type="checkbox"
              :checked="scopesFor(entry.user.id).includes('*') || scopesFor(entry.user.id).includes(option.value)"
              :disabled="scopesFor(entry.user.id).includes('*')"
              @change="toggleScope(entry.user.id, option.value, $event.target.checked)"
            >
            <span>{{ option.label }}</span>
          </label>
        </fieldset>

        <label class="hourly-limit-field">
          <span>Hourly endpoint limit</span>
          <input v-model.number="draftFor(entry.user.id).hourlyLimit" min="0" step="1" type="number">
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
      </article>
    </div>
  </section>
</template>
