<script setup>
import { reactive, watch } from 'vue'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseSelect from '@/shared/components/form/BaseSelect.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'

const props = defineProps({
  metronComicScan: { type: Object, default: null },
  metronComicDiscovery: { type: Object, default: null },
  saving: { type: Boolean, default: false },
  savingDiscovery: { type: Boolean, default: false },
})

const emit = defineEmits([
  'save',
  'trigger',
  'stop',
  'save-discovery',
  'trigger-discovery',
  'stop-discovery',
])

const draft = reactive({})
const discoveryDraft = reactive({})
const weekdays = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']
const incompleteFieldOptions = [
  { value: 'comicVineId', label: 'Comic Vine ID' },
  { value: 'publisher', label: 'Publisher' },
  { value: 'coverImage', label: 'Cover image' },
  { value: 'coverDate', label: 'Cover date' },
  { value: 'description', label: 'Description' },
]

watch(
  () => props.metronComicScan?.settings,
  (settings) => {
    Object.assign(draft, settings || {})
    if (!Array.isArray(draft.incompleteFields)) {
      draft.incompleteFields = incompleteFieldOptions.map((option) => option.value)
    }
  },
  { immediate: true },
)

watch(
  () => props.metronComicDiscovery?.settings,
  (settings) => Object.assign(discoveryDraft, settings || {}),
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

function comicScanPayload() {
  return {
    enabled: Boolean(draft.enabled),
    scanComics: true,
    schedule: draft.schedule || 'daily',
    weekdays: draft.schedule === 'weekly' ? draft.weekdays || [] : [],
    startTime: draft.startTime || '02:00',
    dailyCallLimit: Number(draft.dailyCallLimit) || 1,
    minIntervalSeconds: Math.max(0, Number(draft.minIntervalSeconds) || 0),
    recheckCooldownDays: Math.max(0, Number(draft.recheckCooldownDays) || 0),
    incompleteFields: draft.incompleteFields || [],
  }
}

function save() {
  emit('save', comicScanPayload())
}

function startComicScan() {
  emit('trigger', comicScanPayload())
}

function toggleDiscoveryWeekday(day, checked) {
  const selected = new Set(discoveryDraft.weekdays || [])
  if (checked) selected.add(day)
  else selected.delete(day)
  discoveryDraft.weekdays = [...selected]
}

function comicDiscoveryPayload() {
  return {
    enabled: Boolean(discoveryDraft.enabled),
    pullComics: Boolean(discoveryDraft.pullComics),
    pullReadingLists: Boolean(discoveryDraft.pullReadingLists),
    schedule: discoveryDraft.schedule || 'daily',
    weekdays: discoveryDraft.schedule === 'weekly' ? discoveryDraft.weekdays || [] : [],
    monthDay: Math.min(31, Math.max(1, Number(discoveryDraft.monthDay) || 1)),
    startTime: discoveryDraft.startTime || '03:00',
    publisherName: String(discoveryDraft.publisherName || '').trim(),
    seriesName: String(discoveryDraft.seriesName || '').trim(),
  }
}

function saveDiscovery() {
  emit('save-discovery', comicDiscoveryPayload())
}

function startComicDiscovery() {
  emit('trigger-discovery', comicDiscoveryPayload())
}
</script>

<template>
  <div
    id="settings-panel-metron"
    class="settings-tab-panel settings-metron-panel min-w-0 grid gap-5"
    role="tabpanel"
    aria-labelledby="settings-tab-metron"
  >
    <slot name="metron-import"></slot>

    <section v-if="metronComicDiscovery" class="account-settings-panel metron-scan-panel">
      <header class="metron-scan-heading">
        <div class="metron-scan-heading-copy grid gap-1.5 max-w-prose">
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">
            Metron discovery
          </p>
          <h3>Automatic new Metron data</h3>
          <p class="muted block text-muted">
            Import recently modified comics, reading lists, or both from Metron on one schedule.
          </p>
        </div>
        <label class="compact-toggle metron-scan-toggle">
          <input v-model="discoveryDraft.enabled" type="checkbox" />
          <span>{{ discoveryDraft.enabled ? 'Enabled' : 'Disabled' }}</span>
        </label>
      </header>

      <fieldset class="permission-scopes metron-discovery-types">
        <legend>Pull content</legend>
        <label>
          <input v-model="discoveryDraft.pullComics" type="checkbox" />
          <span>Comics</span>
        </label>
        <label>
          <input v-model="discoveryDraft.pullReadingLists" type="checkbox" />
          <span>Reading lists</span>
        </label>
      </fieldset>

      <div class="metron-scan-fields">
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Schedule</span>
          <BaseSelect v-model="discoveryDraft.schedule" size="large">
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
          </BaseSelect>
        </label>
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Start time (server time)</span>
          <BaseTextInput v-model="discoveryDraft.startTime" size="large" type="time" />
        </label>
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Publisher name filter</span>
          <BaseTextInput
            v-model="discoveryDraft.publisherName"
            size="large"
            type="text"
            placeholder="All publishers"
            :disabled="!discoveryDraft.pullComics"
          />
        </label>
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Series name filter</span>
          <BaseTextInput
            v-model="discoveryDraft.seriesName"
            size="large"
            type="text"
            placeholder="All series"
            :disabled="!discoveryDraft.pullComics"
          />
        </label>
        <label
          v-if="discoveryDraft.schedule === 'monthly'"
          class="metron-scan-field grid gap-2 text-label font-extrabold"
        >
          <span>Day of month</span>
          <BaseTextInput
            v-model.number="discoveryDraft.monthDay"
            size="large"
            type="number"
            min="1"
            max="31"
          />
        </label>
      </div>

      <fieldset v-if="discoveryDraft.schedule === 'weekly'" class="permission-scopes">
        <legend>Run on</legend>
        <label v-for="day in weekdays" :key="`discovery-${day}`">
          <input
            type="checkbox"
            :checked="(discoveryDraft.weekdays || []).includes(day)"
            @change="toggleDiscoveryWeekday(day, $event.target.checked)"
          />
          <span>{{ day.charAt(0).toUpperCase() + day.slice(1) }}</span>
        </label>
      </fieldset>

      <div class="metron-scan-status" aria-live="polite">
        <div>
          <strong>{{ metronComicDiscovery.found }}</strong>
          <span>list results found</span>
        </div>
        <div>
          <strong>{{ metronComicDiscovery.imported }}</strong>
          <span>imported</span>
        </div>
        <div v-if="metronComicDiscovery.alreadyPresent">
          <strong>{{ metronComicDiscovery.alreadyPresent }}</strong>
          <span>already present</span>
        </div>
        <p>
          <template v-if="metronComicDiscovery.running">Import running</template>
          <template v-else-if="metronComicDiscovery.stopReason">
            Last pull: {{ metronComicDiscovery.stopReason }}
          </template>
          <template v-else>Not run yet</template>
        </p>
      </div>
      <div class="metron-scan-actions">
        <BaseButton
          variant="primary"
          size="large"
          :disabled="
            savingDiscovery || (!discoveryDraft.pullComics && !discoveryDraft.pullReadingLists)
          "
          @click="saveDiscovery"
        >
          {{ savingDiscovery ? 'Saving...' : 'Save settings' }}
        </BaseButton>
        <BaseButton
          v-if="!metronComicDiscovery.running"
          variant="neutral"
          size="large"
          :disabled="
            savingDiscovery ||
            !discoveryDraft.enabled ||
            (!discoveryDraft.pullComics && !discoveryDraft.pullReadingLists)
          "
          @click="startComicDiscovery"
        >
          {{ savingDiscovery ? 'Saving and starting...' : 'Pull now' }}
        </BaseButton>
        <BaseButton v-else variant="danger-ghost" size="large" @click="$emit('stop-discovery')">
          Stop pull
        </BaseButton>
      </div>
    </section>

    <section v-if="metronComicScan" class="account-settings-panel metron-scan-panel">
      <header class="metron-scan-heading">
        <div class="metron-scan-heading-copy grid gap-1.5 max-w-prose">
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">
            Metron maintenance
          </p>
          <h3>Incomplete comic data</h3>
          <p class="muted block text-muted">
            Choose which missing fields make a comic incomplete. Issue responses also create missing
            arc and character links without extra detail calls.
          </p>
        </div>
        <label class="compact-toggle metron-scan-toggle">
          <input v-model="draft.enabled" type="checkbox" />
          <span>{{ draft.enabled ? 'Enabled' : 'Disabled' }}</span>
        </label>
      </header>

      <fieldset class="permission-scopes metron-incomplete-fields">
        <legend>Consider a comic incomplete when it has no</legend>
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
        Select at least one field before saving or running this scan.
      </p>

      <div class="metron-scan-fields">
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Schedule</span>
          <BaseSelect v-model="draft.schedule" size="large">
            <option value="daily">Daily</option>
            <option value="weekly">Specific weekdays</option>
          </BaseSelect>
        </label>
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Start time (server time)</span>
          <BaseTextInput v-model="draft.startTime" size="large" type="time" />
        </label>
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Calls per day</span>
          <BaseTextInput
            v-model.number="draft.dailyCallLimit"
            size="large"
            min="1"
            step="1"
            type="number"
          />
        </label>
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Minimum Metron interval (seconds)</span>
          <BaseTextInput
            v-model.number="draft.minIntervalSeconds"
            size="large"
            min="0"
            step="1"
            type="number"
          />
        </label>
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
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
      <p class="muted metron-scan-hint block text-muted">
        Some issues have no publisher, cover date, or synopsis on Metron itself, so they can stay
        "incomplete" no matter how often they're checked. The cooldown skips a comic for this many
        days after it was last checked, so those rows stop using up the whole daily call budget. Set
        to 0 to recheck everything every run.
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

      <div class="metron-scan-status" aria-live="polite">
        <div>
          <strong>{{ metronComicScan.callsUsedToday }} / {{ draft.dailyCallLimit }}</strong>
          <span>calls used today</span>
        </div>
        <div v-if="metronComicScan.running">
          <strong>{{ metronComicScan.updated }}</strong>
          <span
            >updated ({{ metronComicScan.failed }} failed) from
            {{ metronComicScan.scanned }} scanned</span
          >
        </div>
        <p v-else>
          Quota resets daily · {{ metronComicScan.usageDate }}
          <template v-if="metronComicScan.stopReason">
            · Last scan: {{ metronComicScan.stopReason }} ({{ metronComicScan.updated }} updated,
            {{ metronComicScan.failed }} failed)
          </template>
        </p>
      </div>
      <p v-if="metronComicScan.lastError" class="text-danger text-sm m-0">
        Last error: {{ metronComicScan.lastError }}
      </p>

      <div class="metron-scan-actions">
        <BaseButton
          variant="primary"
          size="large"
          :disabled="saving || !(draft.incompleteFields || []).length"
          @click="save"
        >
          {{ saving ? 'Saving...' : 'Save settings' }}
        </BaseButton>
        <BaseButton
          v-if="!metronComicScan.running"
          variant="neutral"
          size="large"
          :disabled="saving || !draft.enabled || !(draft.incompleteFields || []).length"
          @click="startComicScan"
        >
          {{ saving ? 'Saving and starting...' : 'Scan now' }}
        </BaseButton>
        <BaseButton v-else variant="danger-ghost" size="large" @click="$emit('stop')">
          Stop scan
        </BaseButton>
      </div>
    </section>
    <LoadingState v-else />
  </div>
</template>

<style scoped>
@reference '../../../styles.css';

.metron-scan-field :is(input, select, textarea) {
  @apply w-full;
}

.permission-scopes label {
  @apply inline-flex min-h-8 items-center gap-2 rounded border border-line bg-surface px-2.5 py-2 font-extrabold leading-ui-tight text-label;
}

.permission-scopes legend {
  @apply mb-0.5 w-full text-sm font-extrabold uppercase text-muted;
}

.metron-scan-status > div {
  @apply grid gap-px;
}

.metron-scan-status strong {
  @apply text-base text-ink;
}

.metron-scan-status :is(span, p) {
  @apply m-0 text-sm font-bold;
}

.metron-scan-status p {
  @apply ml-auto;
}

@media (width <= 720px) {
  .metron-scan-status p {
    @apply ml-0 mr-0;
  }
}

.metron-scan-heading {
  @apply flex items-start justify-between gap-6 down-mobile:items-stretch down-mobile:flex-col;
}

.compact-toggle.metron-scan-toggle {
  @apply flex-none min-w-36 justify-center border border-line rounded bg-surface py-3 px-3.5 down-mobile:self-start inline-flex items-center gap-2 min-h-8 text-label font-extrabold leading-ui-tight;
}

.metron-scan-fields {
  @apply grid grid-cols-[repeat(2,minmax(220px,360px))] gap-y-4 gap-x-6 down-mobile:grid-cols-1;
}

.permission-scopes {
  @apply border-0 p-0 m-0 grid grid-cols-[repeat(auto-fit,minmax(126px,1fr))] gap-2 min-w-0 disabled:opacity-55 down-mobile:grid-cols-1;
}

.metron-scan-status {
  @apply flex items-center gap-7 border border-line rounded bg-surface py-3 px-3.5 text-muted down-mobile:items-stretch down-mobile:flex-col;
}

.metron-scan-actions {
  @apply flex items-center flex-wrap gap-2.5 down-mobile:items-stretch down-mobile:flex-col [&_>_button]:w-40 down-mobile:[&_>_button]:w-full;
}

.account-settings-panel.metron-scan-panel {
  @apply gap-6 rounded-xl p-6 down-mobile:p-4 grid border border-line bg-surface-soft;
}

.permission-scopes.metron-discovery-types,
.permission-scopes.metron-incomplete-fields {
  @apply border-0 p-0 m-0 grid grid-cols-[repeat(auto-fit,minmax(126px,1fr))] gap-2 min-w-0 disabled:opacity-55 down-mobile:grid-cols-1;
}
</style>
