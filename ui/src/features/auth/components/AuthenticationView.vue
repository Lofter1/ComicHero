<script setup>
import ErrorToast from '@/shared/components/feedback/ErrorToast.vue'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'

defineProps({
  loading: Boolean,
  saving: Boolean,
  setupRequired: Boolean,
  verificationRequired: Boolean,
  passwordResetMode: Boolean,
  authRequired: Boolean,
  registrationMode: {
    type: String,
    default: 'invite_only',
  },
  verificationEmail: {
    type: String,
    default: '',
  },
  error: {
    type: String,
    default: '',
  },
})

const setupForm = defineModel('setupForm', { type: Object, required: true })
const authForm = defineModel('authForm', { type: Object, required: true })
const verificationForm = defineModel('verificationForm', { type: Object, required: true })
const passwordResetForm = defineModel('passwordResetForm', { type: Object, required: true })
const authMode = defineModel('authMode', { type: String, required: true })

defineEmits([
  'clear-error',
  'retry',
  'submit-setup',
  'submit-auth',
  'submit-verification',
  'resend-verification',
  'submit-forgot-password',
  'submit-password-reset',
  'show-forgot-password',
  'show-login',
])
</script>

<template>
  <main v-if="loading" class="auth-shell min-h-screen grid place-items-center p-6">
    <LoadingState />
  </main>

  <main v-else-if="setupRequired" class="auth-shell min-h-screen grid place-items-center p-6">
    <form
      class="auth-panel [width:min(100%,_460px)] border border-line rounded bg-panel shadow-dialog p-7 grid gap-5 [&_h1]:text-3xl"
      @submit.prevent="$emit('submit-setup')"
    >
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">First Run</p>
        <h1>Choose user mode</h1>
      </div>

      <fieldset
        class="mode-options border-0 p-0 m-0 grid gap-2.5 [&_legend]:mb-2 [&_legend]:text-label [&_legend]:font-extrabold [&_label]:grid [&_label]:[grid-template-columns:auto_minmax(0,_1fr)] [&_label]:gap-3 [&_label]:items-start [&_label]:border [&_label]:border-line-strong [&_label]:rounded [&_label]:bg-surface [&_label]:p-3.5 [&_label]:min-w-0 [&_input]:mt-1 [&_span]:grid [&_span]:gap-1.5 [&_span]:min-w-0 [&_strong]:break-anywhere [&_small]:break-anywhere [&_small]:text-muted"
      >
        <legend>User environment</legend>
        <label>
          <input v-model="setupForm.mode" type="radio" value="single" />
          <span>
            <strong>Single user</strong>
            <small>No login. Existing read status stays with the default user.</small>
          </span>
        </label>
        <label>
          <input v-model="setupForm.mode" type="radio" value="multi" />
          <span>
            <strong>Multi user</strong>
            <small>Register and log in. Existing read status becomes the first account.</small>
          </span>
        </label>
      </fieldset>

      <div
        v-if="setupForm.mode === 'multi'"
        class="auth-fields grid gap-1.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold [&_input]:min-h-10 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-ink [&_input]:py-2.5 [&_input]:px-3"
      >
        <label>
          <span>Name</span>
          <input v-model.trim="setupForm.name" type="text" autocomplete="name" required />
        </label>
        <label>
          <span>Email</span>
          <input v-model.trim="setupForm.email" type="email" autocomplete="email" required />
        </label>
        <label>
          <span>Password</span>
          <input
            v-model="setupForm.password"
            type="password"
            autocomplete="new-password"
            minlength="6"
            required
          />
        </label>
      </div>

      <ErrorToast :message="error" @dismiss="$emit('clear-error')" />

      <button
        class="primary-action min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 font-extrabold border-primary bg-primary text-white"
        type="submit"
        :disabled="saving"
      >
        {{ saving ? 'Saving...' : 'Continue' }}
      </button>
    </form>
  </main>

  <main
    v-else-if="verificationRequired"
    class="auth-shell min-h-screen grid place-items-center p-6"
  >
    <form
      class="auth-panel [width:min(100%,_460px)] border border-line rounded bg-panel shadow-dialog p-7 grid gap-5 [&_h1]:text-3xl"
      @submit.prevent="$emit('submit-verification')"
    >
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Verify Email</p>
        <h1>Check your email</h1>
        <p>Enter the verification token sent to {{ verificationEmail }}.</p>
      </div>

      <div
        class="auth-fields grid gap-1.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold [&_input]:min-h-10 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-ink [&_input]:py-2.5 [&_input]:px-3"
      >
        <label>
          <span>Verification token</span>
          <input
            v-model.trim="verificationForm.token"
            type="text"
            autocomplete="one-time-code"
            required
          />
        </label>
        <label>
          <span>Password</span>
          <input
            v-model="verificationForm.password"
            type="password"
            autocomplete="current-password"
            minlength="6"
          />
        </label>
      </div>

      <ErrorToast :message="error" @dismiss="$emit('clear-error')" />

      <button
        class="primary-action min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 font-extrabold border-primary bg-primary text-white"
        type="submit"
        :disabled="saving"
      >
        {{ saving ? 'Verifying...' : 'Verify email' }}
      </button>
      <button
        class="secondary-action min-h-10 border border-line-strong rounded bg-surface text-control py-2 px-3 font-extrabold [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:bg-primary-soft focus-visible:border-primary focus-visible:bg-primary-soft"
        type="button"
        :disabled="saving"
        @click="$emit('resend-verification')"
      >
        Resend email
      </button>
    </form>
  </main>

  <main v-else-if="passwordResetMode" class="auth-shell min-h-screen grid place-items-center p-6">
    <form
      class="auth-panel [width:min(100%,_460px)] border border-line rounded bg-panel shadow-dialog p-7 grid gap-5 [&_h1]:text-3xl"
      @submit.prevent="
        passwordResetForm.requested
          ? $emit('submit-password-reset')
          : $emit('submit-forgot-password')
      "
    >
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Account</p>
        <h1>Reset password</h1>
        <p v-if="!passwordResetForm.requested">
          Enter your email and we will send a password reset token if the account exists.
        </p>
        <p v-else>Enter the token from your email and choose a new password.</p>
      </div>

      <div
        class="auth-fields grid gap-1.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold [&_input]:min-h-10 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-ink [&_input]:py-2.5 [&_input]:px-3"
      >
        <label v-if="!passwordResetForm.requested">
          <span>Email</span>
          <input
            v-model.trim="passwordResetForm.email"
            type="email"
            autocomplete="email"
            required
          />
        </label>
        <template v-else>
          <label>
            <span>Reset token</span>
            <input
              v-model.trim="passwordResetForm.token"
              type="text"
              autocomplete="one-time-code"
              required
            />
          </label>
          <label>
            <span>New password</span>
            <input
              v-model="passwordResetForm.password"
              type="password"
              autocomplete="new-password"
              minlength="6"
              required
            />
          </label>
          <label>
            <span>Confirm new password</span>
            <input
              v-model="passwordResetForm.passwordConfirmation"
              type="password"
              autocomplete="new-password"
              minlength="6"
              required
            />
          </label>
        </template>
      </div>

      <ErrorToast :message="error" @dismiss="$emit('clear-error')" />

      <button
        class="primary-action min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 font-extrabold border-primary bg-primary text-white"
        type="submit"
        :disabled="saving"
      >
        {{
          saving
            ? 'Working...'
            : passwordResetForm.requested
              ? 'Reset password'
              : 'Send reset email'
        }}
      </button>
      <button
        class="secondary-action min-h-10 border border-line-strong rounded bg-surface text-control py-2 px-3 font-extrabold [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:bg-primary-soft focus-visible:border-primary focus-visible:bg-primary-soft"
        type="button"
        :disabled="saving"
        @click="$emit('show-login')"
      >
        Back to login
      </button>
    </form>
  </main>

  <main v-else-if="authRequired" class="auth-shell min-h-screen grid place-items-center p-6">
    <form
      class="auth-panel [width:min(100%,_460px)] border border-line rounded bg-panel shadow-dialog p-7 grid gap-5 [&_h1]:text-3xl"
      @submit.prevent="$emit('submit-auth')"
    >
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Multi User</p>
        <h1>{{ authMode === 'register' ? 'Register' : 'Log in' }}</h1>
      </div>

      <div
        class="auth-tabs grid grid-cols-2 gap-2 [&_button]:min-h-10 [&_button]:border [&_button]:border-line-strong [&_button]:rounded [&_button]:bg-surface [&_button]:text-control [&_button]:py-2.5 [&_button]:px-3.5 [&_button]:font-extrabold [&_button.active]:border-primary [&_button.active]:bg-primary [&_button.active]:text-white"
        role="group"
        aria-label="Authentication mode"
      >
        <button type="button" :class="{ active: authMode === 'login' }" @click="authMode = 'login'">
          Log in
        </button>
        <button
          type="button"
          :class="{ active: authMode === 'register' }"
          @click="authMode = 'register'"
        >
          Register
        </button>
      </div>

      <div
        class="auth-fields grid gap-1.5 min-w-0 [&_label]:grid [&_label]:gap-1.5 [&_label]:text-label [&_label]:font-extrabold [&_input]:min-h-10 [&_input]:border [&_input]:border-line-strong [&_input]:rounded [&_input]:bg-surface [&_input]:text-ink [&_input]:py-2.5 [&_input]:px-3"
      >
        <label v-if="authMode === 'register'">
          <span>Name</span>
          <input v-model.trim="authForm.name" type="text" autocomplete="name" required />
        </label>
        <label>
          <span>Email</span>
          <input v-model.trim="authForm.email" type="email" autocomplete="email" required />
        </label>
        <label v-if="authMode === 'register'">
          <span>Confirm email</span>
          <input
            v-model.trim="authForm.emailConfirmation"
            type="email"
            autocomplete="email"
            required
          />
        </label>
        <label>
          <span>Password</span>
          <input
            v-model="authForm.password"
            type="password"
            :autocomplete="authMode === 'register' ? 'new-password' : 'current-password'"
            minlength="6"
            required
          />
        </label>
        <label v-if="authMode === 'register'">
          <span>Confirm password</span>
          <input
            v-model="authForm.passwordConfirmation"
            type="password"
            autocomplete="new-password"
            minlength="6"
            required
          />
        </label>
        <label v-if="authMode === 'register' && registrationMode === 'invite_only'">
          <span>Invite token</span>
          <input
            v-model.trim="authForm.inviteToken"
            type="text"
            autocomplete="one-time-code"
            required
          />
        </label>
      </div>

      <ErrorToast :message="error" @dismiss="$emit('clear-error')" />

      <button
        class="primary-action min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 font-extrabold border-primary bg-primary text-white"
        type="submit"
        :disabled="saving"
      >
        {{ saving ? 'Working...' : authMode === 'register' ? 'Register' : 'Log in' }}
      </button>
      <button
        v-if="authMode === 'login'"
        class="secondary-action min-h-10 border border-line-strong rounded bg-surface text-control py-2 px-3 font-extrabold [&:hover:not(:disabled)]:border-primary [&:hover:not(:disabled)]:bg-primary-soft focus-visible:border-primary focus-visible:bg-primary-soft"
        type="button"
        :disabled="saving"
        @click="$emit('show-forgot-password')"
      >
        Forgot password?
      </button>
    </form>
  </main>

  <main v-else class="auth-shell min-h-screen grid place-items-center p-6">
    <section
      class="auth-panel [width:min(100%,_460px)] border border-line rounded bg-panel shadow-dialog p-7 grid gap-5 [&_h1]:text-3xl"
    >
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Setup</p>
        <h1>Could not load user setup</h1>
      </div>
      <ErrorToast :message="error" @dismiss="$emit('clear-error')" />
      <button
        class="primary-action min-h-10 border border-line-strong rounded bg-surface text-control py-2.5 px-3.5 font-extrabold border-primary bg-primary text-white"
        type="button"
        :disabled="loading"
        @click="$emit('retry')"
      >
        Retry
      </button>
    </section>
  </main>
</template>
