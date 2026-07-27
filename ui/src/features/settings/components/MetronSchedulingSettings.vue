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
const characterIncompleteFieldOptions = [
  { value: 'description', label: 'Description' },
  { value: 'image', label: 'Image' },
  { value: 'aliases', label: 'Aliases' },
]
const seriesIncompleteFieldOptions = [
  { value: 'publisher', label: 'Publisher' },
  { value: 'seriesYear', label: 'Start year' },
  { value: 'volume', label: 'Volume' },
  { value: 'yearEnd', label: 'End year' },
  { value: 'issueCount', label: 'Issue count' },
  { value: 'description', label: 'Description' },
]
const arcIncompleteFieldOptions = [
  { value: 'description', label: 'Description' },
  { value: 'image', label: 'Image' },
]
const maintenanceResourceOptions = [
  { value: 'comics', label: 'Comics' },
  { value: 'characters', label: 'Characters' },
  { value: 'series', label: 'Series' },
  { value: 'arcs', label: 'Arcs' },
]

watch(
  () => props.metronComicScan?.settings,
  (settings) => {
    Object.assign(draft, settings || {})
    const fieldDefaults = [
      ['incompleteFields', incompleteFieldOptions],
      ['characterIncompleteFields', characterIncompleteFieldOptions],
      ['seriesIncompleteFields', seriesIncompleteFieldOptions],
      ['arcIncompleteFields', arcIncompleteFieldOptions],
    ]
    for (const [field, options] of fieldDefaults) {
      if (!Array.isArray(draft[field])) {
        draft[field] = options.map((option) => option.value)
      }
    }
    const knownResources = new Set(maintenanceResourceOptions.map((option) => option.value))
    const resourceOrder = Array.isArray(draft.resourceOrder)
      ? draft.resourceOrder.filter((resource, index, values) => {
          return knownResources.has(resource) && values.indexOf(resource) === index
        })
      : []
    for (const option of maintenanceResourceOptions) {
      if (!resourceOrder.includes(option.value)) resourceOrder.push(option.value)
    }
    draft.resourceOrder = resourceOrder
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

function toggleIncompleteField(group, field, checked) {
  const selected = new Set(draft[group] || [])
  if (checked) selected.add(field)
  else selected.delete(field)
  draft[group] = [...selected]
}

function moveMaintenanceResource(index, direction) {
  const target = index + direction
  if (target < 0 || target >= draft.resourceOrder.length) return
  const order = [...draft.resourceOrder]
  const selected = order[index]
  order[index] = order[target]
  order[target] = selected
  draft.resourceOrder = order
}

function maintenanceResourceLabel(resource) {
  return maintenanceResourceOptions.find((option) => option.value === resource)?.label || resource
}

function maintenanceResourceEnabled(resource) {
  return Boolean(
    {
      comics: draft.scanComics,
      characters: draft.scanCharacters,
      series: draft.scanSeries,
      arcs: draft.scanArcs,
    }[resource],
  )
}

function maintenanceResourcePullDescription(resource) {
  if (!maintenanceResourceEnabled(resource)) return 'Disabled — skipped'
  if (resource === 'comics') return 'Missing issue metadata'
  const pullsFullList = {
    characters: draft.pullCharacterComics,
    series: draft.pullSeriesComics,
    arcs: draft.pullArcComics,
  }[resource]
  return pullsFullList ? 'Metadata, then full comic list' : 'Metadata only'
}

function maintenanceSettingsValid() {
  const groups = [
    [draft.scanComics, draft.incompleteFields],
    [draft.scanCharacters, draft.characterIncompleteFields],
    [draft.scanSeries, draft.seriesIncompleteFields],
    [draft.scanArcs, draft.arcIncompleteFields],
  ]
  return (
    groups.some(([enabled]) => enabled) &&
    groups.every(([enabled, fields]) => !enabled || fields?.length)
  )
}

function comicScanPayload() {
  return {
    enabled: Boolean(draft.enabled),
    scanComics: Boolean(draft.scanComics),
    scanCharacters: Boolean(draft.scanCharacters),
    scanSeries: Boolean(draft.scanSeries),
    scanArcs: Boolean(draft.scanArcs),
    pullCharacterComics: Boolean(draft.pullCharacterComics),
    pullSeriesComics: Boolean(draft.pullSeriesComics),
    pullArcComics: Boolean(draft.pullArcComics),
    schedule: draft.schedule || 'daily',
    weekdays: draft.schedule === 'weekly' ? draft.weekdays || [] : [],
    startTime: draft.startTime || '02:00',
    dailyCallLimit: Number(draft.dailyCallLimit) || 1,
    minIntervalSeconds: Math.max(0, Number(draft.minIntervalSeconds) || 0),
    recheckCooldownDays: Math.max(0, Number(draft.recheckCooldownDays) || 0),
    incompleteFields: draft.incompleteFields || [],
    characterIncompleteFields: draft.characterIncompleteFields || [],
    seriesIncompleteFields: draft.seriesIncompleteFields || [],
    arcIncompleteFields: draft.arcIncompleteFields || [],
    resourceOrder: draft.resourceOrder || maintenanceResourceOptions.map((option) => option.value),
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
          <h3>Incomplete Metron data</h3>
          <p class="muted block text-muted">
            Fill missing metadata for comics, characters, series, and arcs. For linked resource
            types, choose whether maintenance should also pull their complete comic lists.
          </p>
        </div>
        <label class="compact-toggle metron-scan-toggle">
          <input v-model="draft.enabled" type="checkbox" />
          <span>{{ draft.enabled ? 'Enabled' : 'Disabled' }}</span>
        </label>
      </header>

      <section class="metron-maintenance-order" aria-labelledby="metron-maintenance-order-title">
        <header>
          <div>
            <h4 id="metron-maintenance-order-title">Pull order</h4>
            <p class="muted block text-muted">
              Move resource types into the priority order used by every scheduled or manual run.
            </p>
          </div>
        </header>
        <ol>
          <li
            v-for="(resource, index) in draft.resourceOrder || []"
            :key="resource"
            :class="{ 'is-disabled': !maintenanceResourceEnabled(resource) }"
          >
            <span class="metron-order-position">{{ index + 1 }}</span>
            <span class="metron-order-copy">
              <strong>{{ maintenanceResourceLabel(resource) }}</strong>
              <small>{{ maintenanceResourcePullDescription(resource) }}</small>
            </span>
            <span class="metron-order-actions">
              <BaseButton
                variant="neutral"
                size="compact"
                :disabled="index === 0"
                :aria-label="`Move ${maintenanceResourceLabel(resource)} earlier`"
                @click="moveMaintenanceResource(index, -1)"
              >
                Move up
              </BaseButton>
              <BaseButton
                variant="neutral"
                size="compact"
                :disabled="index === draft.resourceOrder.length - 1"
                :aria-label="`Move ${maintenanceResourceLabel(resource)} later`"
                @click="moveMaintenanceResource(index, 1)"
              >
                Move down
              </BaseButton>
            </span>
          </li>
        </ol>
        <div class="metron-maintenance-order-help">
          <strong>How the order works</strong>
          <ul>
            <li>
              At the start of a run, ComicHero finds all eligible incomplete records. Disabled
              resource types and types with no eligible records are skipped.
            </li>
            <li>
              Resource types run from top to bottom. Within each type, existing records are pulled
              by local ID, oldest IDs first.
            </li>
            <li>
              Comics pull issue metadata. Characters, series, and arcs pull metadata first; when
              “full comic list” is selected, that record’s paginated comic list is pulled
              immediately afterward before moving to the next record.
            </li>
            <li>
              Each metadata request and comic-list page uses the shared call budget and request
              interval. If the budget runs out, later records and resource types wait for the next
              run. Successfully checked records observe the re-check cooldown.
            </li>
            <li>
              The eligible queues are a start-of-run snapshot, so comics newly imported from a full
              list are considered by incomplete-comic maintenance on the next run.
            </li>
          </ul>
        </div>
      </section>

      <div class="metron-maintenance-resources">
        <section class="metron-maintenance-resource">
          <header>
            <label class="compact-toggle">
              <input v-model="draft.scanComics" type="checkbox" />
              <span>Comics</span>
            </label>
          </header>
          <fieldset v-if="draft.scanComics" class="permission-scopes metron-incomplete-fields">
            <legend>Incomplete when missing</legend>
            <label v-for="option in incompleteFieldOptions" :key="option.value">
              <input
                type="checkbox"
                :checked="(draft.incompleteFields || []).includes(option.value)"
                @change="
                  toggleIncompleteField('incompleteFields', option.value, $event.target.checked)
                "
              />
              <span>{{ option.label }}</span>
            </label>
          </fieldset>
          <p v-if="draft.scanComics && !(draft.incompleteFields || []).length" class="access-note">
            Select at least one comic field.
          </p>
        </section>

        <section class="metron-maintenance-resource">
          <header>
            <label class="compact-toggle">
              <input v-model="draft.scanCharacters" type="checkbox" />
              <span>Characters</span>
            </label>
            <label v-if="draft.scanCharacters" class="metron-resource-depth">
              <span>Pull</span>
              <BaseSelect v-model="draft.pullCharacterComics">
                <option :value="false">Metadata only</option>
                <option :value="true">Metadata + full comic list</option>
              </BaseSelect>
            </label>
          </header>
          <fieldset v-if="draft.scanCharacters" class="permission-scopes metron-incomplete-fields">
            <legend>Incomplete when missing</legend>
            <label v-for="option in characterIncompleteFieldOptions" :key="option.value">
              <input
                type="checkbox"
                :checked="(draft.characterIncompleteFields || []).includes(option.value)"
                @change="
                  toggleIncompleteField(
                    'characterIncompleteFields',
                    option.value,
                    $event.target.checked,
                  )
                "
              />
              <span>{{ option.label }}</span>
            </label>
          </fieldset>
          <p
            v-if="draft.scanCharacters && !(draft.characterIncompleteFields || []).length"
            class="access-note"
          >
            Select at least one character field.
          </p>
        </section>

        <section class="metron-maintenance-resource">
          <header>
            <label class="compact-toggle">
              <input v-model="draft.scanSeries" type="checkbox" />
              <span>Series</span>
            </label>
            <label v-if="draft.scanSeries" class="metron-resource-depth">
              <span>Pull</span>
              <BaseSelect v-model="draft.pullSeriesComics">
                <option :value="false">Metadata only</option>
                <option :value="true">Metadata + full comic list</option>
              </BaseSelect>
            </label>
          </header>
          <fieldset v-if="draft.scanSeries" class="permission-scopes metron-incomplete-fields">
            <legend>Incomplete when missing</legend>
            <label v-for="option in seriesIncompleteFieldOptions" :key="option.value">
              <input
                type="checkbox"
                :checked="(draft.seriesIncompleteFields || []).includes(option.value)"
                @change="
                  toggleIncompleteField(
                    'seriesIncompleteFields',
                    option.value,
                    $event.target.checked,
                  )
                "
              />
              <span>{{ option.label }}</span>
            </label>
          </fieldset>
          <p
            v-if="draft.scanSeries && !(draft.seriesIncompleteFields || []).length"
            class="access-note"
          >
            Select at least one series field.
          </p>
        </section>

        <section class="metron-maintenance-resource">
          <header>
            <label class="compact-toggle">
              <input v-model="draft.scanArcs" type="checkbox" />
              <span>Arcs</span>
            </label>
            <label v-if="draft.scanArcs" class="metron-resource-depth">
              <span>Pull</span>
              <BaseSelect v-model="draft.pullArcComics">
                <option :value="false">Metadata only</option>
                <option :value="true">Metadata + full comic list</option>
              </BaseSelect>
            </label>
          </header>
          <fieldset v-if="draft.scanArcs" class="permission-scopes metron-incomplete-fields">
            <legend>Incomplete when missing</legend>
            <label v-for="option in arcIncompleteFieldOptions" :key="option.value">
              <input
                type="checkbox"
                :checked="(draft.arcIncompleteFields || []).includes(option.value)"
                @change="
                  toggleIncompleteField('arcIncompleteFields', option.value, $event.target.checked)
                "
              />
              <span>{{ option.label }}</span>
            </label>
          </fieldset>
          <p v-if="draft.scanArcs && !(draft.arcIncompleteFields || []).length" class="access-note">
            Select at least one arc field.
          </p>
        </section>
      </div>
      <p
        v-if="!draft.scanComics && !draft.scanCharacters && !draft.scanSeries && !draft.scanArcs"
        class="access-note"
      >
        Select at least one data type before saving or running maintenance.
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
        Some records have fields that are also blank on Metron, so they can stay "incomplete" after
        a check. The cooldown prevents those records from consuming the daily call budget on every
        run. Full comic lists use additional Metron calls and import missing comics with list
        metadata. Set the cooldown to 0 to recheck everything every run.
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
          <span v-if="metronComicScan.currentResource">
            Now pulling {{ maintenanceResourceLabel(metronComicScan.currentResource) }}
          </span>
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
          :disabled="saving || !maintenanceSettingsValid()"
          @click="save"
        >
          {{ saving ? 'Saving...' : 'Save settings' }}
        </BaseButton>
        <BaseButton
          v-if="!metronComicScan.running"
          variant="neutral"
          size="large"
          :disabled="saving || !draft.enabled || !maintenanceSettingsValid()"
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

.metron-maintenance-resources {
  @apply grid gap-3;
}

.metron-maintenance-order {
  @apply grid gap-4 rounded-lg border border-line bg-surface p-4;
}

.metron-maintenance-order h4,
.metron-maintenance-order p {
  @apply m-0;
}

.metron-maintenance-order > ol {
  @apply m-0 grid list-none gap-2 p-0;
}

.metron-maintenance-order > ol > li {
  @apply grid grid-cols-[2rem_minmax(0,1fr)_auto] items-center gap-3 rounded border border-line bg-surface-soft p-3 down-mobile:grid-cols-[2rem_minmax(0,1fr)];
}

.metron-maintenance-order > ol > li.is-disabled {
  @apply opacity-60;
}

.metron-order-position {
  @apply inline-flex size-8 items-center justify-center rounded-full bg-primary-soft text-sm font-black text-control;
}

.metron-order-copy {
  @apply grid gap-0.5;
}

.metron-order-copy small {
  @apply text-xs font-bold text-muted;
}

.metron-order-actions {
  @apply flex gap-2 down-mobile:col-span-2 down-mobile:pl-11;
}

.metron-maintenance-order-help {
  @apply rounded border border-line bg-surface-soft p-3 text-sm text-muted;
}

.metron-maintenance-order-help > strong {
  @apply text-ink;
}

.metron-maintenance-order-help ul {
  @apply mb-0 mt-2 grid gap-1.5 pl-5;
}

.metron-maintenance-resource {
  @apply grid gap-3 rounded-lg border border-line bg-surface p-4;
}

.metron-maintenance-resource > header {
  @apply flex items-center justify-between gap-4 down-mobile:items-stretch down-mobile:flex-col;
}

.metron-maintenance-resource .compact-toggle {
  @apply inline-flex min-h-10 items-center gap-2 font-extrabold text-label;
}

.metron-resource-depth {
  @apply flex min-w-72 items-center gap-2 text-sm font-extrabold text-muted down-mobile:min-w-0 down-mobile:items-stretch down-mobile:flex-col;
}

.metron-resource-depth select {
  @apply min-w-56 down-mobile:w-full;
}
</style>
