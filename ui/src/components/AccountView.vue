<script setup>
import { reactive, watch } from 'vue'

const props = defineProps({
  user: {
    type: Object,
    default: null,
  },
  userMode: {
    type: String,
    default: '',
  },
  saving: {
    type: Boolean,
    default: false,
  },
  deleting: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['save', 'delete-account'])
const form = reactive({
  name: '',
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
})
const deleteForm = reactive({
  currentPassword: '',
})

watch(
  () => props.user,
  (user) => {
    form.name = user?.name || ''
    form.currentPassword = ''
    form.newPassword = ''
    form.confirmPassword = ''
    deleteForm.currentPassword = ''
  },
  { immediate: true },
)

function save() {
  const payload = { name: form.name }
  if (props.userMode === 'multi' && form.newPassword) {
    if (form.newPassword !== form.confirmPassword) {
      emit('save', null, 'New passwords do not match.')
      return
    }
    payload.currentPassword = form.currentPassword
    payload.newPassword = form.newPassword
  }
  emit('save', payload)
}

function deleteAccount() {
  if (props.userMode !== 'multi') return
  if (!deleteForm.currentPassword) {
    emit('save', null, 'Current password is required to delete your account.')
    return
  }
  if (!window.confirm('Delete this account permanently? This cannot be undone.')) return
  emit('delete-account', { currentPassword: deleteForm.currentPassword })
}
</script>

<template>
  <section class="browse-view account-view">
    <div v-if="!user" class="empty-panel">
      No active account.
    </div>

    <form v-else class="account-settings" @submit.prevent="save">
      <article class="account-settings-panel">
        <div class="account-settings-heading">
          <span class="account-avatar large" aria-hidden="true">{{ (user.name || '?').slice(0, 1).toUpperCase() }}</span>
          <div>
            <p class="eyebrow">Profile</p>
            <h3>{{ user.name }}</h3>
          </div>
        </div>

        <div class="metadata-grid account-metadata">
          <span>
            <strong>{{ user.isAdmin ? 'Admin' : 'User' }}</strong>
            <small>Role</small>
          </span>
        </div>
      </article>

      <article class="account-settings-panel">
        <div>
          <p class="eyebrow">Account Data</p>
          <h3>Manage account</h3>
        </div>

        <div class="auth-fields">
          <label>
            <span>Display name</span>
            <input v-model.trim="form.name" type="text" autocomplete="name" required>
          </label>
        </div>
      </article>

      <article v-if="userMode === 'multi'" class="account-settings-panel">
        <div>
          <p class="eyebrow">Password</p>
          <h3>Change password</h3>
        </div>

        <div class="auth-fields">
          <label>
            <span>Current password</span>
            <input v-model="form.currentPassword" type="password" autocomplete="current-password">
          </label>
          <label>
            <span>New password</span>
            <input v-model="form.newPassword" type="password" autocomplete="new-password" minlength="6">
          </label>
          <label>
            <span>Confirm new password</span>
            <input v-model="form.confirmPassword" type="password" autocomplete="new-password" minlength="6">
          </label>
        </div>
      </article>

      <button class="primary-button account-save-button" type="submit" :disabled="saving">
        {{ saving ? 'Saving...' : 'Save account' }}
      </button>

      <article v-if="userMode === 'multi'" class="account-settings-panel danger-panel">
        <div>
          <p class="eyebrow">Danger Zone</p>
          <h3>Delete account</h3>
          <p class="muted">This removes your account, sessions, read status, and Metron permissions. Reading lists you authored stay in the library without an author.</p>
        </div>

        <div class="auth-fields">
          <label>
            <span>Current password</span>
            <input v-model="deleteForm.currentPassword" type="password" autocomplete="current-password" @keydown.enter.prevent="deleteAccount">
          </label>
        </div>

        <button class="danger-button account-save-button" type="button" :disabled="deleting" @click="deleteAccount">
          {{ deleting ? 'Deleting...' : 'Delete account' }}
        </button>
      </article>
    </form>
  </section>
</template>
