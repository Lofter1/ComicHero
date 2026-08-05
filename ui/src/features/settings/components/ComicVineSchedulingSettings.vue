<script setup>
import { reactive, watch } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseSelect from '@/shared/components/form/BaseSelect.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'

const props = defineProps({
  comicVineScan: { type: Object, default: null },
  saving: { type: Boolean, default: false },
})

const emit = defineEmits(['save', 'trigger', 'stop'])

const draft = reactive({})
const weekdays = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']
const incompleteFieldOptions = [
  { value: 'publisher', label: 'Publisher' },
  { value: 'coverImage', label: 'Cover image' },
  { value: 'coverDate', label: 'Cover date' },
  { value: 'description', label: 'Description' },
]

watch(
  () => props.comicVineScan?.settings,
  (settings) => {
    Object.assign(draft, settings || {})
    if (!Array.isArray(draft.incompleteFields)) {
      draft.incompleteFields = incompleteFieldOptions.map((option) => option.value)
    }
  },
  { immediate: true },
)

function toggleWeekday(day, checked) {
  const selected = new Set(draft.weekdays || [])
  if (checked) selected.add(day)
  else selected.delete(day)
  draft.weekdays = [...selected]
}

function toggleIncompleteField(field, checked) {
  const selected = new Set(draft.incompleteFields || [])
  if (checked) selected.add(field)
  else selected.delete(field)
  draft.incompleteFields = [...selected]
}

function comicVineScanPayload() {
  return {
    enabled: Boolean(draft.enabled),
    schedule: draft.schedule || 'daily',
    weekdays: draft.schedule === 'weekly' ? draft.weekdays || [] : [],
    startTime: draft.startTime || '02:30',
    dailyCallLimit: Number(draft.dailyCallLimit) || 1,
    minIntervalSeconds: Math.max(0, Number(draft.minIntervalSeconds) || 0),
    recheckCooldownDays: Math.max(0, Number(draft.recheckCooldownDays) || 0),
    incompleteFields: draft.incompleteFields || [],
  }
}

function save() {
  emit('save', comicVineScanPayload())
}

function startComicVineScan() {
  emit('trigger', comicVineScanPayload())
}
</script>

<template>
  <section
    v-if="comicVineScan"
    id="settings-panel-comicvine"
    class="account-settings-panel comicvine-scan-panel"
    role="tabpanel"
    aria-labelledby="settings-tab-comicvine"
  >
    <header class="comicvine-scan-heading">
      <div class="comicvine-scan-heading-copy grid gap-1.5 max-w-prose">
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Comic Vine</p>
        <h3>Incomplete Comic Vine data</h3>
        <p class="muted block text-muted">
          Fill missing publisher, cover, and description metadata for comics that already have a
          Comic Vine ID, either entered manually or linked automatically by Metron maintenance.
        </p>
      </div>
      <label class="compact-toggle comicvine-scan-toggle">
        <input v-model="draft.enabled" type="checkbox" />
        <span>{{ draft.enabled ? 'Enabled' : 'Disabled' }}</span>
      </label>
    </header>

    <fieldset class="permission-scopes comicvine-incomplete-fields">
      <legend>Incomplete when missing</legend>
      <label v-for="option in incompleteFieldOptions" :key="option.value">
        <input
          type="checkbox"
          :checked="(draft.incompleteFields || []).includes(option.value)"
          @change="toggleIncompleteField(option.value, $event.target.checked)"
        />
        <span>{{ option.label }}</span>
      </label>
    </fieldset>
    <p v-if="!(draft.incompleteFields || []).length" class="access-note">
      Select at least one comic field.
    </p>

    <div class="comicvine-scan-fields">
      <label class="comicvine-scan-field grid gap-2 text-label font-extrabold">
        <span>Schedule</span>
        <BaseSelect v-model="draft.schedule" size="large">
          <option value="daily">Daily</option>
          <option value="weekly">Specific weekdays</option>
        </BaseSelect>
      </label>
      <label class="comicvine-scan-field grid gap-2 text-label font-extrabold">
        <span>Start time (server time)</span>
        <BaseTextInput v-model="draft.startTime" size="large" type="time" />
      </label>
      <label class="comicvine-scan-field grid gap-2 text-label font-extrabold">
        <span>Calls per day</span>
        <BaseTextInput
          v-model.number="draft.dailyCallLimit"
          size="large"
          min="1"
          step="1"
          type="number"
        />
      </label>
      <label class="comicvine-scan-field grid gap-2 text-label font-extrabold">
        <span>Minimum Comic Vine interval (seconds)</span>
        <BaseTextInput
          v-model.number="draft.minIntervalSeconds"
          size="large"
          min="0"
          step="1"
          type="number"
        />
      </label>
      <label class="comicvine-scan-field grid gap-2 text-label font-extrabold">
        <span>Re-check cooldown (days)</span>
        <BaseTextInput
          v-model.number="draft.recheckCooldownDays"
          size="large"
          min="0"
          step="1"
          type="number"
        />
      </label>
    </div>
    <p class="muted comicvine-scan-hint block text-muted">
      Publisher lookups use one extra Comic Vine call per comic. Some comics have fields that are
      also blank on Comic Vine, so they can stay "incomplete" after a check; the cooldown prevents
      those from consuming the daily call budget on every run. Set the cooldown to 0 to recheck
      everything every run.
    </p>

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

    <div class="comicvine-scan-status" aria-live="polite">
      <div>
        <strong>{{ comicVineScan.callsUsedToday }} / {{ draft.dailyCallLimit }}</strong>
        <span>calls used today</span>
      </div>
      <div v-if="comicVineScan.running">
        <strong>{{ comicVineScan.updated }}</strong>
        <span>updated ({{ comicVineScan.failed }} failed) from {{ comicVineScan.scanned }} scanned</span>
      </div>
      <p v-else>
        Quota resets daily · {{ comicVineScan.usageDate }}
        <template v-if="comicVineScan.stopReason">
          · Last scan: {{ comicVineScan.stopReason }} ({{ comicVineScan.updated }} updated,
          {{ comicVineScan.failed }} failed)
        </template>
      </p>
    </div>
    <p v-if="comicVineScan.lastError" class="text-danger text-sm m-0">
      Last error: {{ comicVineScan.lastError }}
    </p>

    <div class="comicvine-scan-actions">
      <BaseButton
        variant="primary"
        size="large"
        :disabled="saving || !(draft.incompleteFields || []).length"
        @click="save"
      >
        {{ saving ? 'Saving...' : 'Save settings' }}
      </BaseButton>
      <BaseButton
        v-if="!comicVineScan.running"
        variant="neutral"
        size="large"
        :disabled="saving || !draft.enabled || !(draft.incompleteFields || []).length"
        @click="startComicVineScan"
      >
        {{ saving ? 'Saving and starting...' : 'Scan now' }}
      </BaseButton>
      <BaseButton v-else variant="danger-ghost" size="large" @click="$emit('stop')">
        Stop scan
      </BaseButton>
    </div>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.comicvine-scan-field :is(input, select, textarea) {
  @apply w-full;
}

.permission-scopes label {
  @apply inline-flex min-h-8 items-center gap-2 rounded border border-line bg-surface px-2.5 py-2 font-extrabold leading-ui-tight text-label;
}

.permission-scopes legend {
  @apply mb-0.5 w-full text-sm font-extrabold uppercase text-muted;
}

.comicvine-scan-status > div {
  @apply grid gap-px;
}

.comicvine-scan-status strong {
  @apply text-base text-ink;
}

.comicvine-scan-status :is(span, p) {
  @apply m-0 text-sm font-bold;
}

.comicvine-scan-status p {
  @apply ml-auto;
}

@media (width <= 720px) {
  .comicvine-scan-status p {
    @apply ml-0 mr-0;
  }
}

.comicvine-scan-heading {
  @apply flex items-start justify-between gap-6 down-mobile:items-stretch down-mobile:flex-col;
}

.compact-toggle.comicvine-scan-toggle {
  @apply flex-none min-w-36 justify-center border border-line rounded bg-surface py-3 px-3.5 down-mobile:self-start inline-flex items-center gap-2 min-h-8 text-label font-extrabold leading-ui-tight;
}

.comicvine-scan-fields {
  @apply grid grid-cols-[repeat(2,minmax(220px,360px))] gap-y-4 gap-x-6 down-mobile:grid-cols-1;
}

.permission-scopes {
  @apply border-0 p-0 m-0 grid grid-cols-[repeat(auto-fit,minmax(126px,1fr))] gap-2 min-w-0 disabled:opacity-55 down-mobile:grid-cols-1;
}

.comicvine-scan-status {
  @apply flex items-center gap-7 border border-line rounded bg-surface py-3 px-3.5 text-muted down-mobile:items-stretch down-mobile:flex-col;
}

.comicvine-scan-actions {
  @apply flex items-center flex-wrap gap-2.5 down-mobile:items-stretch down-mobile:flex-col [&_>_button]:w-40 down-mobile:[&_>_button]:w-full;
}

.account-settings-panel.comicvine-scan-panel {
  @apply gap-6 rounded-xl p-6 down-mobile:p-4 grid border border-line bg-surface-soft;
}
</style>
