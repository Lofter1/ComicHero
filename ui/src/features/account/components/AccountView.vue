<script setup>
import { reactive, ref, watch } from 'vue'
import { exportReadHistory, importReadHistory } from '@/api/client.js'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import ApiTokensPanel from './ApiTokensPanel.vue'

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
const historyFile = ref(null)
const historyBusy = ref(false)
const historyMessage = ref('')
const historyError = ref('')
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

async function downloadReadHistory() {
  historyBusy.value = true
  historyMessage.value = ''
  historyError.value = ''
  try {
    const data = await exportReadHistory()
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `comichero-read-history-${new Date().toISOString().slice(0, 10)}.json`
    link.click()
    URL.revokeObjectURL(url)
    historyMessage.value = `${data.comics.length} read comics exported.`
  } catch (err) {
    historyError.value = err.message
  } finally {
    historyBusy.value = false
  }
}

async function uploadReadHistory() {
  const file = historyFile.value?.files?.[0]
  if (!file) return
  historyBusy.value = true
  historyMessage.value = ''
  historyError.value = ''
  try {
    const result = await importReadHistory(JSON.parse(await file.text()))
    historyMessage.value = `${result.imported} comics marked as read. ${result.unmatched.length} unmatched.`
    historyFile.value.value = ''
  } catch (err) {
    historyError.value = err.message
  } finally {
    historyBusy.value = false
  }
}
</script>

<template>
  <section class="browse-view account-view max-w-240 min-w-0 w-full grid gap-5 pt-5">
    <div v-if="!user" class="empty-panel">No active account.</div>

    <div v-else class="account-settings grid gap-5">
      <form class="grid gap-5" @submit.prevent="save">
        <article class="account-settings-panel">
          <div class="account-settings-heading">
            <span class="account-avatar large" aria-hidden="true">{{
              (user.name || '?').slice(0, 1).toUpperCase()
            }}</span>
            <div>
              <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Profile</p>
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
            <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Account data</p>
            <h3>Manage account</h3>
          </div>

          <div class="auth-fields">
            <label>
              <span>Display name</span>
              <BaseTextInput v-model.trim="form.name" type="text" autocomplete="name" required />
            </label>
          </div>

          <template v-if="userMode === 'multi'">
            <hr class="panel-divider" />
            <div>
              <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Password</p>
              <h3>Change password</h3>
            </div>

            <div class="auth-fields">
              <label>
                <span>Current password</span>
                <BaseTextInput
                  v-model="form.currentPassword"
                  type="password"
                  autocomplete="current-password"
                />
              </label>
              <label>
                <span>New password</span>
                <BaseTextInput
                  v-model="form.newPassword"
                  type="password"
                  autocomplete="new-password"
                  minlength="6"
                />
              </label>
              <label>
                <span>Confirm new password</span>
                <BaseTextInput
                  v-model="form.confirmPassword"
                  type="password"
                  autocomplete="new-password"
                  minlength="6"
                />
              </label>
            </div>
          </template>

          <BaseButton class="justify-self-start" variant="primary" type="submit" :disabled="saving">
            {{ saving ? 'Saving...' : 'Save account' }}
          </BaseButton>
        </article>
      </form>

      <ApiTokensPanel />

      <article class="account-settings-panel">
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Library data</p>
          <h3>Transfer read history</h3>
          <p class="muted block text-muted">
            Export the comics you have read and import them into another ComicHero instance.
            Matching uses Metron or Comic Vine IDs, so local database IDs do not need to match.
          </p>
        </div>
        <div class="flex flex-wrap gap-2 items-center">
          <BaseButton variant="secondary" :disabled="historyBusy" @click="downloadReadHistory">
            {{ historyBusy ? 'Working...' : 'Export read history' }}
          </BaseButton>
          <label class="history-import-button">
            Import JSON
            <input
              ref="historyFile"
              type="file"
              accept="application/json,.json"
              :disabled="historyBusy"
              @change="uploadReadHistory"
            />
          </label>
        </div>
        <p v-if="historyMessage" class="text-muted text-sm">{{ historyMessage }}</p>
        <p v-if="historyError" class="text-danger text-sm">{{ historyError }}</p>
      </article>

      <article v-if="userMode === 'multi'" class="account-settings-panel danger-panel">
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Danger Zone</p>
          <h3>Delete account</h3>
          <p class="muted block text-muted">
            This removes your account, sessions, read status, and Metron permissions. Reading lists
            you authored stay in the library without an author.
          </p>
        </div>

        <div class="auth-fields">
          <label>
            <span>Current password</span>
            <BaseTextInput
              v-model="deleteForm.currentPassword"
              type="password"
              autocomplete="current-password"
              @keydown.enter.prevent="deleteAccount"
            />
          </label>
        </div>

        <BaseButton
          class="justify-self-start"
          variant="danger"
          :disabled="deleting"
          @click="deleteAccount"
        >
          {{ deleting ? 'Deleting...' : 'Delete account' }}
        </BaseButton>
      </article>
    </div>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.history-import-button {
  @apply inline-flex cursor-pointer items-center rounded border border-line bg-surface-soft px-3 py-2 text-sm font-semibold text-control hover:border-primary;
}

.history-import-button input {
  @apply sr-only;
}
</style>

<style scoped>
@reference '../../../styles.css';

.empty-panel {
  @apply border border-dashed border-line-strong rounded-xl bg-surface-soft text-muted p-6 down-mobile:p-4 font-extrabold;
}

.account-settings-panel {
  @apply grid gap-4 rounded-xl border border-line bg-surface-soft p-6 down-mobile:p-4;
}

.panel-divider {
  @apply border-0 border-t border-line my-1;
}

.account-avatar.large {
  @apply w-9 min-w-9 h-9 border border-primary rounded-full inline-flex items-center justify-center bg-primary-soft text-primary font-black leading-none [&.large]:w-12 [&.large]:min-w-12 [&.large]:h-12 [&.large]:text-lg;
}

.metadata-grid.account-metadata {
  @apply grid-cols-[repeat(auto-fit,minmax(150px,1fr))] grid gap-2.5 [&_span]:border [&_span]:border-line [&_span]:rounded [&_span]:bg-surface [&_span]:p-3 [&_strong]:block [&_small]:block [&_small]:text-muted [&_small]:mt-1 down-tablet:grid-cols-1;
}

.auth-fields {
  @apply grid gap-2.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold;
}

.account-settings-panel.danger-panel {
  @apply border-danger-border bg-danger-soft grid gap-4 rounded-xl border p-6 down-mobile:p-4;
}

.metadata-grid.account-metadata strong {
  overflow-wrap: anywhere;
}

.account-settings-heading {
  @apply flex min-w-0 items-center gap-3.5;
}

.account-settings-heading h3 {
  overflow-wrap: anywhere;
}
</style>
