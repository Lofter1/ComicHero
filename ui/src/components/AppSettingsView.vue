<script setup>
import { reactive, watch } from 'vue'

const props = defineProps({
  metronComicScan: { type: Object, default: null },
  saving: { type: Boolean, default: false },
  registrationMode: { type: String, default: 'invite_only' },
  savingRegistrationMode: { type: Boolean, default: false },
  publicAccess: { type: Boolean, default: false },
  savingPublicAccess: { type: Boolean, default: false },
  invite: { type: Object, default: null },
  generatingInvite: { type: Boolean, default: false },
})

const emit = defineEmits([
  'save',
  'trigger',
  'stop',
  'update-registration-mode',
  'update-public-access',
  'generate-invite',
])
const draft = reactive({})
const weekdays = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']

watch(
  () => props.metronComicScan?.settings,
  (settings) => Object.assign(draft, settings || {}),
  { immediate: true },
)

function toggleWeekday(day, checked) {
  const selected = new Set(draft.weekdays || [])
  if (checked) selected.add(day)
  else selected.delete(day)
  draft.weekdays = [...selected]
}

function save() {
  emit('save', {
    enabled: Boolean(draft.enabled),
    scanComics: true,
    schedule: draft.schedule || 'daily',
    weekdays: draft.schedule === 'weekly' ? draft.weekdays || [] : [],
    startTime: draft.startTime || '02:00',
    dailyCallLimit: Number(draft.dailyCallLimit) || 1,
    minIntervalSeconds: Math.max(0, Number(draft.minIntervalSeconds) || 0),
  })
}

function registrationModeLabel(mode) {
  return mode === 'open' ? 'Open registration' : 'Invite only'
}
</script>

<template>
  <section class="browse-view app-settings-view">
    <section class="user-access-panel">
      <div class="user-registration-panel">
        <div>
          <p class="eyebrow">Registration</p>
          <h3>{{ registrationModeLabel(registrationMode) }}</h3>
          <p class="muted">
            {{
              registrationMode === 'open'
                ? 'Anyone who can reach this server can register without an invite, then verify their email.'
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
        <p v-if="registrationMode === 'open'" class="access-note">
          Open registration gives verified new accounts full read/write access to the shared
          library.
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
        <p v-if="publicAccess" class="access-note">
          Public visitors cannot edit data, but they can see your shared library.
        </p>
      </div>
    </section>

    <section v-if="metronComicScan" class="account-settings-panel metron-scan-panel">
      <header class="metron-scan-heading">
        <div class="metron-scan-heading-copy">
          <p class="eyebrow">Metron maintenance</p>
          <h3>Incomplete comic data</h3>
          <p class="muted">
            Fill missing publisher, cover, cover date, and description fields. Issue responses also
            create missing arc and character links without extra detail calls.
          </p>
        </div>
        <label class="compact-toggle metron-scan-toggle">
          <input v-model="draft.enabled" type="checkbox" />
          <span>{{ draft.enabled ? 'Enabled' : 'Disabled' }}</span>
        </label>
      </header>

      <div class="metron-scan-fields">
        <label class="metron-scan-field">
          <span>Schedule</span>
          <select v-model="draft.schedule">
            <option value="daily">Daily</option>
            <option value="weekly">Specific weekdays</option>
          </select>
        </label>
        <label class="metron-scan-field">
          <span>Start time (server time)</span>
          <input v-model="draft.startTime" type="time" />
        </label>
        <label class="metron-scan-field">
          <span>Calls per day</span>
          <input v-model.number="draft.dailyCallLimit" min="1" step="1" type="number" />
        </label>
        <label class="metron-scan-field">
          <span>Minimum Metron interval (seconds)</span>
          <input v-model.number="draft.minIntervalSeconds" min="0" step="1" type="number" />
        </label>
      </div>

      <fieldset v-if="draft.schedule === 'weekly'" class="permission-scopes">
        <legend>Run on</legend>
        <label v-for="day in weekdays" :key="day">
          <input
            type="checkbox"
            :checked="(draft.weekdays || []).includes(day)"
            @change="toggleWeekday(day, $event.target.checked)"
          />
          <span>{{ day.charAt(0).toUpperCase() + day.slice(1) }}</span>
        </label>
      </fieldset>

      <div class="metron-scan-status" aria-live="polite">
        <div>
          <strong>{{ metronComicScan.callsUsedToday }} / {{ draft.dailyCallLimit }}</strong>
          <span>calls used today</span>
        </div>
        <div v-if="metronComicScan.running">
          <strong>{{ metronComicScan.updated }}</strong>
          <span>updated from {{ metronComicScan.scanned }} scanned</span>
        </div>
        <p v-else>
          Quota resets daily · {{ metronComicScan.usageDate }}
          <template v-if="metronComicScan.stopReason">
            · Last scan: {{ metronComicScan.stopReason }}
          </template>
        </p>
      </div>

      <div class="metron-scan-actions">
        <button type="button" class="primary-button" :disabled="saving" @click="save">
          {{ saving ? 'Saving...' : 'Save settings' }}
        </button>
        <button
          v-if="!metronComicScan.running"
          type="button"
          class="secondary-action"
          :disabled="!draft.enabled"
          @click="$emit('trigger')"
        >
          Scan now
        </button>
        <button v-else type="button" class="danger-text-button" @click="$emit('stop')">
          Stop scan
        </button>
      </div>
    </section>
    <div v-else class="empty-panel">Loading app settings...</div>
  </section>
</template>
