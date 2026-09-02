<script setup>
import { reactive } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'

const props = defineProps({
  userMode: { type: String, default: 'single' },
  registrationMode: { type: String, default: 'invite_only' },
  savingRegistrationMode: { type: Boolean, default: false },
  publicAccess: { type: Boolean, default: false },
  savingPublicAccess: { type: Boolean, default: false },
  invite: { type: Object, default: null },
  generatingInvite: { type: Boolean, default: false },
  switchingToMultiUser: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update-registration-mode',
  'update-public-access',
  'generate-invite',
  'switch-to-multi-user',
])

const multiUserForm = reactive({
  name: '',
  email: '',
  password: '',
  confirmPassword: '',
})
const multiUserFormError = reactive({ message: '' })

function switchToMultiUser() {
  multiUserFormError.message = ''
  if (!multiUserForm.name.trim()) {
    multiUserFormError.message = 'Name is required.'
    return
  }
  if (!multiUserForm.email.trim()) {
    multiUserFormError.message = 'Email is required.'
    return
  }
  if (multiUserForm.password.length < 6) {
    multiUserFormError.message = 'Password must be at least 6 characters.'
    return
  }
  if (multiUserForm.password !== multiUserForm.confirmPassword) {
    multiUserFormError.message = 'Passwords do not match.'
    return
  }
  if (
    typeof window !== 'undefined' &&
    !window.confirm(
      "This switches ComicHero to multi-user mode. You'll need this email and password to log back in, and this cannot be undone through the app.",
    )
  ) {
    return
  }
  emit('switch-to-multi-user', {
    name: multiUserForm.name.trim(),
    email: multiUserForm.email.trim(),
    password: multiUserForm.password,
  })
}
</script>

<template>
  <section class="user-access-panel" role="tabpanel" aria-labelledby="settings-tab-general">
    <div v-if="props.userMode === 'single'" class="access-panel multi-user-panel">
      <div>
        <p class="eyebrow">User mode</p>
        <h3>Switch to multi-user mode</h3>
        <p class="muted">
          ComicHero is currently running in single-user mode with no login. Switching to multi-user
          mode turns this account into a login-protected admin account and lets you invite other
          people. Your existing reading history stays attached to this account. This cannot be
          undone from within the app.
        </p>
      </div>

      <form class="auth-fields" @submit.prevent="switchToMultiUser">
        <label>
          <span>Display name</span>
          <BaseTextInput
            v-model.trim="multiUserForm.name"
            type="text"
            autocomplete="name"
            required
          />
        </label>
        <label>
          <span>Email</span>
          <BaseTextInput
            v-model.trim="multiUserForm.email"
            type="email"
            autocomplete="email"
            required
          />
        </label>
        <label>
          <span>Password</span>
          <BaseTextInput
            v-model="multiUserForm.password"
            type="password"
            autocomplete="new-password"
            minlength="6"
            required
          />
        </label>
        <label>
          <span>Confirm password</span>
          <BaseTextInput
            v-model="multiUserForm.confirmPassword"
            type="password"
            autocomplete="new-password"
            minlength="6"
            required
          />
        </label>
        <p v-if="multiUserFormError.message" class="access-note">
          {{ multiUserFormError.message }}
        </p>
        <BaseButton
          class="justify-self-start"
          variant="primary"
          type="submit"
          :disabled="switchingToMultiUser"
        >
          {{ switchingToMultiUser ? 'Switching...' : 'Switch to multi-user mode' }}
        </BaseButton>
      </form>
    </div>

    <div v-if="props.userMode === 'multi'" class="access-panel">
      <div>
        <p class="eyebrow">Registration</p>
        <h3>{{ registrationMode === 'open' ? 'Open registration' : 'Invite only' }}</h3>
        <p class="muted">
          {{
            registrationMode === 'open'
              ? 'Anyone who can reach this server can register without an invite, then verify their email.'
              : 'New accounts need a single-use invite token to register.'
          }}
        </p>
      </div>
      <div class="access-toggle" role="group" aria-label="Registration mode">
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
      <p v-if="registrationMode === 'open'" class="access-note">
        Open registration gives verified new accounts full read/write access to the shared library.
      </p>
    </div>

    <div v-if="props.userMode === 'multi'" class="access-panel">
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
      <BaseButton
        class="invite-button"
        variant="primary"
        size="dense"
        :disabled="generatingInvite"
        @click="$emit('generate-invite')"
      >
        {{ generatingInvite ? 'Generating...' : 'Generate invite' }}
      </BaseButton>
      <div v-if="invite?.token" class="invite-token-box">
        <span>Invite token</span>
        <code>{{ invite.token }}</code>
        <small>Expires at {{ invite.expiresAt }}</small>
      </div>
    </div>

    <div v-if="props.userMode === 'multi'" class="access-panel">
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
      <div class="access-toggle" role="group" aria-label="Public access">
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
      <p v-if="publicAccess" class="access-note">
        Public visitors cannot edit data, but they can see your shared library.
      </p>
    </div>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.user-access-panel {
  @apply min-w-0 grid grid-cols-[repeat(auto-fit,minmax(min(100%,280px),1fr))] gap-4 items-stretch;
  container: user-access / inline-size;
}

@container user-access (width < 872px) {
  .access-panel:last-child {
    @apply col-span-full;
  }
}

.access-panel {
  @apply grid gap-4 content-start border border-line rounded-xl bg-surface-soft p-5 shadow-float [&_>_.access-toggle]:mt-auto;
}

.multi-user-panel {
  @apply col-span-full;
}

.auth-fields {
  @apply grid gap-2.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold;
}

.eyebrow {
  @apply mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase;
}

.muted {
  @apply block text-muted;
}

.access-toggle {
  @apply grid grid-cols-2 gap-1 border border-line rounded bg-surface p-1;
}

.access-toggle button {
  @apply min-h-10 border-0 rounded bg-transparent text-muted py-2 px-2.5 text-sm font-extrabold;
}

.access-toggle button.active {
  @apply bg-primary text-white shadow-selected;
}

.access-note {
  @apply m-0 border border-warning-border rounded bg-warning-soft text-warning py-2.5 px-3 text-sm font-bold leading-ui;
}

.invite-button {
  @apply mt-auto justify-self-start;
}

.invite-token-box {
  @apply grid gap-1 border border-line rounded bg-surface p-3 [&_span]:text-muted [&_span]:text-sm [&_span]:font-bold [&_small]:text-muted [&_small]:text-sm [&_small]:font-bold [&_code]:text-(--heading) [&_code]:font-extrabold;
}

.invite-token-box code {
  overflow-wrap: anywhere;
}
</style>
