<script setup>
import BaseButton from '@/shared/components/form/BaseButton.vue'

defineProps({
  registrationMode: { type: String, default: 'invite_only' },
  savingRegistrationMode: { type: Boolean, default: false },
  publicAccess: { type: Boolean, default: false },
  savingPublicAccess: { type: Boolean, default: false },
  invite: { type: Object, default: null },
  generatingInvite: { type: Boolean, default: false },
})

defineEmits(['update-registration-mode', 'update-public-access', 'generate-invite'])
</script>

<template>
  <section class="user-access-panel" role="tabpanel" aria-labelledby="settings-tab-general">
    <div class="access-panel">
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

    <div class="access-panel">
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

    <div class="access-panel">
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
