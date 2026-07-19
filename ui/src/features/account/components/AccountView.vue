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
  <section class="browse-view account-view [max-width:960px] min-w-0 w-full">
    <div
      v-if="!user"
      class="empty-panel border border-dashed border-line-strong rounded bg-surface-soft text-muted p-5 font-extrabold"
    >
      No active account.
    </div>

    <form v-else class="account-settings grid gap-3.5" @submit.prevent="save">
      <article
        class="account-settings-panel grid gap-3.5 border border-line rounded bg-surface-soft p-4"
      >
        <div
          class="account-settings-heading flex items-center gap-3.5 min-w-0 [&_h3]:break-anywhere"
        >
          <span
            class="account-avatar large w-9 min-w-9 h-9 border border-primary rounded-full inline-flex items-center justify-center bg-primary-soft text-primary font-black leading-none [&.large]:w-11.5 [&.large]:min-w-11.5 [&.large]:h-11.5 [&.large]:text-ui-title-sm"
            aria-hidden="true"
            >{{ (user.name || '?').slice(0, 1).toUpperCase() }}</span
          >
          <div>
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Profile</p>
            <h3>{{ user.name }}</h3>
          </div>
        </div>

        <div
          class="metadata-grid account-metadata [grid-template-columns:repeat(auto-fit,_minmax(150px,_1fr))] grid grid-cols-3 gap-2.5 [&_span]:border [&_span]:border-line [&_span]:rounded [&_span]:bg-surface-soft [&_span]:p-3 [&_strong]:block [&_strong]:break-anywhere [&_small]:block [&_small]:text-muted [&_small]:mt-1 down-tablet:grid-cols-1"
        >
          <span>
            <strong>{{ user.isAdmin ? 'Admin' : 'User' }}</strong>
            <small>Role</small>
          </span>
        </div>
      </article>

      <article
        class="account-settings-panel grid gap-3.5 border border-line rounded bg-surface-soft p-4"
      >
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Account Data</p>
          <h3>Manage account</h3>
        </div>

        <div
          class="auth-fields grid gap-1.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold [&_input]:min-h-10.5 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-ink [&_input]:py-2.5 [&_input]:px-3"
        >
          <label>
            <span>Display name</span>
            <input v-model.trim="form.name" type="text" autocomplete="name" required />
          </label>
        </div>
      </article>

      <article
        v-if="userMode === 'multi'"
        class="account-settings-panel grid gap-3.5 border border-line rounded bg-surface-soft p-4"
      >
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Password</p>
          <h3>Change password</h3>
        </div>

        <div
          class="auth-fields grid gap-1.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold [&_input]:min-h-10.5 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-ink [&_input]:py-2.5 [&_input]:px-3"
        >
          <label>
            <span>Current password</span>
            <input v-model="form.currentPassword" type="password" autocomplete="current-password" />
          </label>
          <label>
            <span>New password</span>
            <input
              v-model="form.newPassword"
              type="password"
              autocomplete="new-password"
              minlength="6"
            />
          </label>
          <label>
            <span>Confirm new password</span>
            <input
              v-model="form.confirmPassword"
              type="password"
              autocomplete="new-password"
              minlength="6"
            />
          </label>
        </div>
      </article>

      <button
        class="primary-button account-save-button justify-self-start min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 border-primary bg-primary text-white"
        type="submit"
        :disabled="saving"
      >
        {{ saving ? 'Saving...' : 'Save account' }}
      </button>

      <article
        v-if="userMode === 'multi'"
        class="account-settings-panel danger-panel border-danger-border bg-danger-soft grid gap-3.5 border border-line rounded bg-surface-soft p-4"
      >
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Danger Zone</p>
          <h3>Delete account</h3>
          <p class="muted block text-muted">
            This removes your account, sessions, read status, and Metron permissions. Reading lists
            you authored stay in the library without an author.
          </p>
        </div>

        <div
          class="auth-fields grid gap-1.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold [&_input]:min-h-10.5 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-ink [&_input]:py-2.5 [&_input]:px-3"
        >
          <label>
            <span>Current password</span>
            <input
              v-model="deleteForm.currentPassword"
              type="password"
              autocomplete="current-password"
              @keydown.enter.prevent="deleteAccount"
            />
          </label>
        </div>

        <button
          class="danger-button account-save-button justify-self-start min-h-10.5 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 [border-color:color-mix(in_srgb,_var(--danger)_42%,_var(--line-strong))] bg-danger-soft text-danger"
          type="button"
          :disabled="deleting"
          @click="deleteAccount"
        >
          {{ deleting ? 'Deleting...' : 'Delete account' }}
        </button>
      </article>
    </form>
  </section>
</template>
