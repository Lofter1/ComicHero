<script setup>
import { reactive, watch } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'

const props = defineProps({
  settings: { type: Object, default: null },
  saving: { type: Boolean, default: false },
})
const emit = defineEmits(['save'])
const draft = reactive({ enabled: false, message: '', level: 'info', startsAt: '', endsAt: '' })

function toLocalTime(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const offset = date.getTimezoneOffset() * 60000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

function toRFC3339(value) {
  return value ? new Date(value).toISOString() : ''
}

watch(
  () => props.settings,
  (settings) => {
    Object.assign(draft, {
      enabled: Boolean(settings?.enabled),
      message: settings?.message || '',
      level: settings?.level || 'info',
      startsAt: toLocalTime(settings?.startsAt),
      endsAt: toLocalTime(settings?.endsAt),
    })
  },
  { immediate: true },
)

function save() {
  emit('save', {
    enabled: draft.enabled,
    message: draft.message,
    level: draft.level,
    startsAt: toRFC3339(draft.startsAt),
    endsAt: toRFC3339(draft.endsAt),
  })
}
</script>

<template>
  <section
    class="announcement-settings settings-tab-panel"
    role="tabpanel"
    aria-labelledby="settings-tab-general"
  >
    <div class="account-settings-panel">
      <div>
        <p class="eyebrow">User message</p>
        <h3>Announcements</h3>
        <p class="muted">
          Show an optional message to users, such as a planned maintenance window.
        </p>
      </div>
      <label class="announcement-toggle">
        <input v-model="draft.enabled" type="checkbox" />
        <span>Publish announcement</span>
      </label>
      <label class="auth-fields">
        <span>Message</span>
        <textarea
          v-model="draft.message"
          maxlength="4000"
          rows="4"
          placeholder="Planned maintenance starts at 02:00 UTC."
        />
      </label>
      <div class="announcement-fields">
        <label>
          <span>Severity</span>
          <select v-model="draft.level">
            <option value="info">Information</option>
            <option value="warning">Warning</option>
            <option value="critical">Critical</option>
          </select>
        </label>
        <label>
          <span>Starts at</span>
          <BaseTextInput v-model="draft.startsAt" type="datetime-local" />
        </label>
        <label>
          <span>Ends at</span>
          <BaseTextInput v-model="draft.endsAt" type="datetime-local" />
        </label>
      </div>
      <BaseButton class="justify-self-start" variant="primary" :disabled="saving" @click="save">
        {{ saving ? 'Saving...' : 'Save announcement' }}
      </BaseButton>
    </div>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.announcement-settings {
  @apply grid gap-5;
}

.account-settings-panel {
  @apply grid gap-4 rounded-xl border border-line bg-surface-soft p-6 down-mobile:p-4;
}

.announcement-toggle,
.announcement-fields label,
.auth-fields {
  @apply grid gap-1.5 text-label font-extrabold;
}

.announcement-toggle {
  @apply flex flex-row items-center gap-2;
}

textarea,
select {
  @apply w-full rounded border border-line-strong bg-surface p-2.5 text-ink;
}

.announcement-fields {
  @apply grid gap-3 down-tablet:grid-cols-1;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}
</style>
