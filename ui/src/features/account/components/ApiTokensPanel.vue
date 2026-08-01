<script setup>
import { onMounted, reactive, ref } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import PanelHeader from '@/shared/components/layout/PanelHeader.vue'
import ModalShell from '@/shared/components/overlay/ModalShell.vue'
import StatusPill from '@/shared/components/feedback/StatusPill.vue'
import { useApiTokens } from '../useApiTokens.js'

const SCOPE_OPTIONS = [
  {
    value: 'readingOrders:search',
    label: 'Search reading orders',
    description: 'List and search reading orders.',
  },
  {
    value: 'readingOrders:read',
    label: 'View reading order details',
    description: 'View a single reading order, including its comics and progress.',
  },
  {
    value: 'readingOrders:next',
    label: 'Get the next comic',
    description: 'Fetch the next unread comic in a reading order.',
  },
  {
    value: 'readingOrders:start',
    label: 'Start / stop reading orders',
    description: 'Mark a reading order as started or stopped.',
  },
  {
    value: 'comics:markRead',
    label: 'Mark comics read',
    description: 'Mark, unmark, or skip a comic as read.',
  },
]

const {
  tokens,
  loading,
  error,
  creating,
  revokingId,
  createdToken,
  loadTokens,
  createToken,
  revokeToken,
  dismissCreatedToken,
} = useApiTokens()

const showCreateDialog = ref(false)
const copied = ref(false)
const form = reactive({
  name: '',
  scopes: [],
  expiresAt: '',
})

onMounted(loadTokens)

function openCreateDialog() {
  form.name = ''
  form.scopes = []
  form.expiresAt = ''
  showCreateDialog.value = true
}

function toggleScope(value, checked) {
  form.scopes = checked ? [...form.scopes, value] : form.scopes.filter((scope) => scope !== value)
}

async function submitCreate() {
  const payload = { name: form.name, scopes: form.scopes }
  if (form.expiresAt) {
    payload.expiresAt = new Date(`${form.expiresAt}T00:00:00Z`).toISOString()
  }
  const ok = await createToken(payload)
  if (ok) showCreateDialog.value = false
}

async function copyCreatedToken() {
  if (!createdToken.value) return
  try {
    await navigator.clipboard.writeText(createdToken.value.token)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    // Clipboard access can be denied by the browser; the token stays
    // visible on screen so the user can still select and copy it by hand.
  }
}

function closeCreatedToken() {
  copied.value = false
  dismissCreatedToken()
}

function formatTimestamp(value) {
  if (!value) return 'Never'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString([], {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function scopeLabel(value) {
  return SCOPE_OPTIONS.find((option) => option.value === value)?.label || value
}

function isExpired(token) {
  if (!token.expiresAt) return false
  const date = new Date(token.expiresAt)
  return !Number.isNaN(date.getTime()) && date.getTime() <= Date.now()
}
</script>

<template>
  <article class="account-settings-panel">
    <div class="section-heading">
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Integrations</p>
        <h3>API tokens</h3>
        <p class="muted block text-muted">
          Let an external service - a reading server, a collection manager - connect to your account
          with a token scoped to just what it needs.
        </p>
      </div>
      <BaseButton variant="primary" :disabled="loading" @click="openCreateDialog">
        Create token
      </BaseButton>
    </div>

    <div v-if="createdToken" class="created-token-banner">
      <p class="created-token-title">
        "{{ createdToken.name }}" created. Copy the token now - it won't be shown again.
      </p>
      <div class="created-token-value">
        <code>{{ createdToken.token }}</code>
        <BaseButton variant="neutral" size="compact" @click="copyCreatedToken">
          {{ copied ? 'Copied!' : 'Copy' }}
        </BaseButton>
      </div>
      <BaseButton
        variant="neutral"
        size="compact"
        class="justify-self-start"
        @click="closeCreatedToken"
      >
        Done
      </BaseButton>
    </div>

    <p v-if="error" class="error-text">{{ error }}</p>

    <p v-if="loading" class="muted block text-muted">Loading tokens...</p>
    <p v-else-if="!tokens.length" class="empty-panel">
      No API tokens yet. Create one to connect an external service.
    </p>

    <ul v-else class="token-list">
      <li v-for="token in tokens" :key="token.id" class="token-row">
        <span class="token-row-main">
          <strong>{{ token.name }}</strong>
          <small>{{ token.tokenPrefix }}...</small>
          <span class="token-scopes">
            <StatusPill v-for="scope in token.scopes" :key="scope" tone="primary">
              {{ scopeLabel(scope) }}
            </StatusPill>
            <StatusPill v-if="isExpired(token)" tone="danger">Expired</StatusPill>
          </span>
          <small
            >Created {{ formatTimestamp(token.createdAt) }} · Last used
            {{ formatTimestamp(token.lastUsedAt) }}
            <template v-if="token.expiresAt"
              >· Expires {{ formatTimestamp(token.expiresAt) }}</template
            ></small
          >
        </span>
        <BaseButton
          variant="danger-ghost"
          size="compact"
          :disabled="revokingId === token.id"
          @click="revokeToken(token.id)"
        >
          {{ revokingId === token.id ? 'Revoking...' : 'Revoke' }}
        </BaseButton>
      </li>
    </ul>
  </article>

  <ModalShell v-if="showCreateDialog" v-slot="{ titleId }" @close="showCreateDialog = false">
    <PanelHeader
      eyebrow="Integrations"
      title="Create API token"
      :title-id="titleId"
      closable
      close-label="Close create token dialog"
      @close="showCreateDialog = false"
    />

    <form class="auth-fields" @submit.prevent="submitCreate">
      <label>
        <span>Name</span>
        <BaseTextInput
          v-model.trim="form.name"
          type="text"
          placeholder="e.g. Tachiyomi bridge"
          required
          autofocus
        />
      </label>

      <fieldset class="permission-scopes">
        <legend>Scopes</legend>
        <label v-for="option in SCOPE_OPTIONS" :key="option.value">
          <input
            type="checkbox"
            :checked="form.scopes.includes(option.value)"
            @change="toggleScope(option.value, $event.target.checked)"
          />
          <span>
            {{ option.label }}
            <small class="block text-muted font-normal">{{ option.description }}</small>
          </span>
        </label>
      </fieldset>

      <label>
        <span>Expires (optional)</span>
        <BaseTextInput v-model="form.expiresAt" type="date" />
      </label>

      <BaseButton
        class="justify-self-start"
        variant="primary"
        type="submit"
        :disabled="creating || !form.name || !form.scopes.length"
      >
        {{ creating ? 'Creating...' : 'Create token' }}
      </BaseButton>
    </form>
  </ModalShell>
</template>

<style scoped>
@reference '../../../styles.css';

.account-settings-panel {
  @apply grid gap-3.5 border border-line rounded bg-surface-soft p-4;
}

.section-heading {
  @apply flex items-start justify-between gap-3 down-mobile:flex-col;
}

.empty-panel {
  @apply border border-dashed border-line-strong rounded bg-surface-soft text-muted p-5 font-extrabold;
}

.error-text {
  @apply text-danger font-extrabold;
}

.created-token-banner {
  @apply grid gap-2.5 border border-primary rounded bg-primary-soft p-3.5;
}

.created-token-title {
  @apply font-extrabold text-control;
}

.created-token-value {
  @apply flex min-w-0 items-center gap-2.5;
}

.created-token-value code {
  @apply min-w-0 flex-1 overflow-x-auto rounded border border-line-strong bg-surface px-2.5 py-2 text-sm whitespace-nowrap;
}

.token-list {
  @apply grid gap-2 list-none m-0 p-0;
}

.token-row {
  @apply flex min-w-0 items-start justify-between gap-3 border border-line rounded bg-surface p-3.5 down-mobile:flex-col;
}

.token-row-main {
  @apply grid min-w-0 gap-1;
}

.token-row-main strong {
  overflow-wrap: anywhere;
}

.token-row-main small {
  @apply text-muted;
}

.token-scopes {
  @apply flex flex-wrap gap-1.5;
}

.auth-fields {
  @apply grid gap-3.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold;
}

.permission-scopes {
  @apply border-0 p-0 m-0 grid gap-2 min-w-0;
}

.permission-scopes legend {
  @apply mb-0.5 w-full text-sm font-extrabold uppercase text-muted;
}

.permission-scopes label {
  @apply flex items-start gap-2 rounded border border-line bg-surface px-2.5 py-2 font-extrabold leading-ui-tight text-label;
}

.permission-scopes input[type='checkbox'] {
  @apply mt-1;
}
</style>
