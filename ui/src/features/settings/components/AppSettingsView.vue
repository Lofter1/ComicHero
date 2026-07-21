<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseSelect from '@/shared/components/form/BaseSelect.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import PanelHeader from '@/shared/components/layout/PanelHeader.vue'
import ModalShell from '@/shared/components/overlay/ModalShell.vue'

const props = defineProps({
  metronComicScan: { type: Object, default: null },
  metronComicDiscovery: { type: Object, default: null },
  cblRepositorySync: { type: Object, default: null },
  cblRepositoryFiles: { type: Array, default: () => [] },
  saving: { type: Boolean, default: false },
  savingDiscovery: { type: Boolean, default: false },
  savingCblRepositorySync: { type: Boolean, default: false },
  loadingCblRepositoryFiles: { type: Boolean, default: false },
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
  'save-discovery',
  'trigger-discovery',
  'stop-discovery',
  'save-cbl-repository-sync',
  'load-cbl-repository-files',
  'trigger-cbl-repository-sync',
  'stop-cbl-repository-sync',
  'resolve-cbl-metron-issue',
  'update-registration-mode',
  'update-public-access',
  'generate-invite',
])
const route = useRoute()
const router = useRouter()
const settingsTabs = [
  { value: 'general', label: 'General' },
  { value: 'metron', label: 'Metron' },
  { value: 'cbl-repositories', label: 'CBL repositories' },
]
const activeSettingsTab = computed(() => {
  const requested = String(route.query.tab || '')
  return settingsTabs.some((tab) => tab.value === requested) ? requested : 'general'
})
const draft = reactive({})
const discoveryDraft = reactive({})
const cblDraft = reactive({})
const repositoryText = ref('')
const cblFolderPickerOpen = ref(false)
const cblFolderSearch = ref('')
const selectedCBLFolderKeys = reactive(new Set())
const visibleCBLFolderLimit = ref(100)
const cblFilePickerOpen = ref(false)
const cblFileSearch = ref('')
const selectedCBLFileKeys = reactive(new Set())
const visibleCBLFileLimit = ref(100)
const resolveMissingCBLIssues = ref(false)
const cblFileBatchSize = 100
const weekdays = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']
const incompleteFieldOptions = [
  { value: 'comicVineId', label: 'Comic Vine ID' },
  { value: 'publisher', label: 'Publisher' },
  { value: 'coverImage', label: 'Cover image' },
  { value: 'coverDate', label: 'Cover date' },
  { value: 'description', label: 'Description' },
]
const indexedCBLRepositoryFiles = computed(() =>
  props.cblRepositoryFiles.map((file) => ({
    file,
    key: cblFileKey(file),
    searchText: `${file.repositoryUrl} ${file.path} ${file.multipartGroup || ''}`.toLowerCase(),
  })),
)
const indexedCBLRepositoryFolders = computed(() => {
  const folders = new Map()
  for (const file of props.cblRepositoryFiles) {
    const parts = String(file.path || '').split('/')
    parts.pop()
    for (let index = 1; index <= parts.length; index += 1) {
      const folder = {
        repositoryUrl: file.repositoryUrl,
        path: parts.slice(0, index).join('/'),
      }
      const key = cblFolderKey(folder)
      const existing = folders.get(key)
      if (existing) existing.fileCount += 1
      else folders.set(key, { ...folder, fileCount: 1 })
    }
  }
  return [...folders.entries()]
    .map(([key, folder]) => ({
      folder,
      key,
      searchText: `${folder.repositoryUrl} ${folder.path}`.toLowerCase(),
    }))
    .sort((left, right) => left.searchText.localeCompare(right.searchText))
})
const cblRepositoryFoldersByKey = computed(
  () => new Map(indexedCBLRepositoryFolders.value.map(({ key, folder }) => [key, folder])),
)
const filteredCBLRepositoryFolders = computed(() => {
  const query = cblFolderSearch.value.trim().toLowerCase()
  const rows = query
    ? indexedCBLRepositoryFolders.value.filter(({ searchText }) => searchText.includes(query))
    : indexedCBLRepositoryFolders.value
  return rows.map(({ folder }) => folder)
})
const visibleCBLRepositoryFolders = computed(() =>
  filteredCBLRepositoryFolders.value.slice(0, visibleCBLFolderLimit.value),
)
const remainingCBLRepositoryFolderCount = computed(() =>
  Math.max(0, filteredCBLRepositoryFolders.value.length - visibleCBLRepositoryFolders.value.length),
)
const selectedCBLRepositoryFolders = computed(() => {
  const folders = []
  for (const key of selectedCBLFolderKeys) {
    const folder = cblRepositoryFoldersByKey.value.get(key)
    if (folder) folders.push(folder)
  }
  return folders
})
const cblFolderScopeLabel = computed(() => {
  const count = Array.isArray(cblDraft.folders) ? cblDraft.folders.length : 0
  if (!count) return 'All repository folders'
  return `${count} selected ${count === 1 ? 'folder' : 'folders'}`
})
const cblRepositoryFilesByKey = computed(
  () => new Map(indexedCBLRepositoryFiles.value.map(({ key, file }) => [key, file])),
)
const cblMultipartFileKeys = computed(() => {
  const groups = new Map()
  for (const { file, key } of indexedCBLRepositoryFiles.value) {
    if (!file.multipartGroup) continue
    const groupKey = `${file.repositoryUrl}\n${file.multipartGroup}`
    const keys = groups.get(groupKey) || []
    keys.push(key)
    groups.set(groupKey, keys)
  }
  return groups
})
const filteredCBLRepositoryFiles = computed(() => {
  const query = cblFileSearch.value.trim().toLowerCase()
  const rows = query
    ? indexedCBLRepositoryFiles.value.filter(({ searchText }) => searchText.includes(query))
    : indexedCBLRepositoryFiles.value
  return rows.map(({ file }) => file)
})
const visibleCBLRepositoryFiles = computed(() =>
  filteredCBLRepositoryFiles.value.slice(0, visibleCBLFileLimit.value),
)
const remainingCBLRepositoryFileCount = computed(() =>
  Math.max(0, filteredCBLRepositoryFiles.value.length - visibleCBLRepositoryFiles.value.length),
)
const selectedCBLRepositoryFiles = computed(() => {
  const files = []
  for (const key of selectedCBLFileKeys) {
    const file = cblRepositoryFilesByKey.value.get(key)
    if (file) files.push(file)
  }
  return files
})

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

watch(
  () => props.cblRepositorySync?.settings,
  (settings) => {
    Object.assign(cblDraft, settings || {})
    cblDraft.folders = Array.isArray(settings?.folders)
      ? settings.folders.map((folder) => ({ ...folder }))
      : []
    repositoryText.value = (settings?.repositories || []).join('\n')
  },
  { immediate: true },
)

watch(
  () => [props.cblRepositorySync?.running, props.cblRepositorySync?.resolveMissingOnMetron],
  ([running, enabled]) => {
    if (running) resolveMissingCBLIssues.value = Boolean(enabled)
  },
  { immediate: true },
)

watch(cblFileSearch, () => {
  visibleCBLFileLimit.value = cblFileBatchSize
})

watch(cblFolderSearch, () => {
  visibleCBLFolderLimit.value = cblFileBatchSize
})

watch(
  () => props.cblRepositorySync?.pendingResolution?.id,
  (resolutionID) => {
    if (resolutionID) selectSettingsTab('cbl-repositories')
  },
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

function toggleCBLWeekday(day, checked) {
  const selected = new Set(cblDraft.weekdays || [])
  if (checked) selected.add(day)
  else selected.delete(day)
  cblDraft.weekdays = [...selected]
}

function cblRepositorySyncPayload() {
  const repositories = repositoryText.value
    .split('\n')
    .map((value) => value.trim())
    .filter(Boolean)
  const repositoryKeys = new Set(repositories.map(normalizedRepositoryKey))
  return {
    enabled: Boolean(cblDraft.enabled),
    repositories,
    folders: (cblDraft.folders || [])
      .filter((folder) => repositoryKeys.has(normalizedRepositoryKey(folder.repositoryUrl)))
      .map((folder) => ({ repositoryUrl: folder.repositoryUrl, path: folder.path })),
    autoSync: Boolean(cblDraft.autoSync),
    schedule: cblDraft.schedule || 'daily',
    weekdays: cblDraft.schedule === 'weekly' ? cblDraft.weekdays || [] : [],
    startTime: cblDraft.startTime || '04:00',
  }
}

function openCBLFolderPicker() {
  cblFolderPickerOpen.value = true
  cblFolderSearch.value = ''
  selectedCBLFolderKeys.clear()
  for (const folder of cblDraft.folders || []) selectedCBLFolderKeys.add(cblFolderKey(folder))
  visibleCBLFolderLimit.value = cblFileBatchSize
  emit('load-cbl-repository-files', cblRepositorySyncPayload())
}

function closeCBLFolderPicker() {
  cblFolderPickerOpen.value = false
}

function cblFolderKey(folder) {
  return `${folder.repositoryUrl}\n${folder.path}`
}

function toggleCBLFolder(folder, checked) {
  const key = cblFolderKey(folder)
  if (checked) selectedCBLFolderKeys.add(key)
  else selectedCBLFolderKeys.delete(key)
}

function clearSelectedCBLFolders() {
  selectedCBLFolderKeys.clear()
}

function showMoreCBLRepositoryFolders() {
  visibleCBLFolderLimit.value += cblFileBatchSize
}

function applySelectedCBLRepositoryFolders() {
  cblDraft.folders = selectedCBLRepositoryFolders.value.map((folder) => ({
    repositoryUrl: folder.repositoryUrl,
    path: folder.path,
  }))
  closeCBLFolderPicker()
}

function saveCBLRepositorySync() {
  emit('save-cbl-repository-sync', cblRepositorySyncPayload())
}

function startCBLRepositorySync() {
  emit('trigger-cbl-repository-sync', {
    settings: cblRepositorySyncPayload(),
    resolveMissingOnMetron: resolveMissingCBLIssues.value,
  })
}

function openCBLFilePicker() {
  cblFilePickerOpen.value = true
  cblFileSearch.value = ''
  selectedCBLFileKeys.clear()
  visibleCBLFileLimit.value = cblFileBatchSize
  emit('load-cbl-repository-files', cblRepositorySyncPayload())
}

function closeCBLFilePicker() {
  cblFilePickerOpen.value = false
}

function cblFileKey(file) {
  return `${file.repositoryUrl}\n${file.path}`
}

function multipartCBLFileKeys(file) {
  if (!file.multipartGroup) return [cblFileKey(file)]
  return (
    cblMultipartFileKeys.value.get(`${file.repositoryUrl}\n${file.multipartGroup}`) || [
      cblFileKey(file),
    ]
  )
}

function toggleCBLFile(file, checked) {
  const key = cblFileKey(file)
  if (checked) selectedCBLFileKeys.add(key)
  else selectedCBLFileKeys.delete(key)
}

function selectAllCBLParts(file) {
  for (const key of multipartCBLFileKeys(file)) selectedCBLFileKeys.add(key)
}

function multipartPartCount(file) {
  return multipartCBLFileKeys(file).length
}

function allMultipartPartsSelected(file) {
  return multipartCBLFileKeys(file).every((key) => selectedCBLFileKeys.has(key))
}

function clearSelectedCBLFiles() {
  selectedCBLFileKeys.clear()
}

function showMoreCBLRepositoryFiles() {
  visibleCBLFileLimit.value += cblFileBatchSize
}

function startSelectedCBLRepositoryFiles() {
  if (!selectedCBLRepositoryFiles.value.length) return
  emit('trigger-cbl-repository-sync', {
    settings: cblRepositorySyncPayload(),
    resolveMissingOnMetron: resolveMissingCBLIssues.value,
    files: selectedCBLRepositoryFiles.value.map((file) => ({
      repositoryUrl: file.repositoryUrl,
      path: file.path,
    })),
  })
  closeCBLFilePicker()
}

function chooseCBLMetronIssue(metronIssueId) {
  const resolutionId = props.cblRepositorySync?.pendingResolution?.id
  if (!resolutionId) return
  emit('resolve-cbl-metron-issue', { resolutionId, metronIssueId })
}

function repositoryLabel(value) {
  return String(value || '').replace(/^https:\/\/github\.com\//i, '')
}

function normalizedRepositoryKey(value) {
  return String(value || '')
    .trim()
    .replace(/\.git\/?$/i, '')
    .replace(/\/$/, '')
    .toLowerCase()
}

function fileSizeLabel(bytes) {
  const value = Number(bytes) || 0
  if (value < 1024) return `${value} B`
  return `${Math.round(value / 1024)} KB`
}

function registrationModeLabel(mode) {
  return mode === 'open' ? 'Open registration' : 'Invite only'
}

function selectSettingsTab(tab) {
  router.replace({
    name: 'settings',
    query: tab === 'general' ? {} : { tab },
  })
}
</script>

<template>
  <section class="browse-view app-settings-view grid gap-5 max-w-content pt-4 min-w-0 w-full">
    <nav class="settings-tabs" role="tablist" aria-label="App settings sections">
      <!-- Native buttons: these are stateful tabs, not standard form actions. -->
      <button
        v-for="tab in settingsTabs"
        :id="`settings-tab-${tab.value}`"
        :key="tab.value"
        type="button"
        class="settings-tab-button"
        role="tab"
        :class="{ active: activeSettingsTab === tab.value }"
        :aria-selected="activeSettingsTab === tab.value"
        :aria-controls="`settings-panel-${tab.value}`"
        @click="selectSettingsTab(tab.value)"
      >
        {{ tab.label }}
      </button>
    </nav>

    <section
      v-show="activeSettingsTab === 'general'"
      id="settings-panel-general"
      class="user-access-panel settings-tab-panel"
      role="tabpanel"
      aria-labelledby="settings-tab-general"
    >
      <div class="user-registration-panel">
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Registration</p>
          <h3>{{ registrationModeLabel(registrationMode) }}</h3>
          <p class="muted block text-muted">
            {{
              registrationMode === 'open'
                ? 'Anyone who can reach this server can register without an invite, then verify their email.'
                : 'New accounts need a single-use invite token to register.'
            }}
          </p>
        </div>

        <div class="registration-mode-toggle" role="group" aria-label="Registration mode">
          <!-- Native buttons: this segmented control has its own selected-state treatment. -->
          <button
            type="button"
            class="access-mode-button"
            :class="{ active: registrationMode === 'invite_only' }"
            :disabled="savingRegistrationMode"
            @click="$emit('update-registration-mode', 'invite_only')"
          >
            Invite only
          </button>
          <button
            type="button"
            class="access-mode-button"
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
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Invites</p>
          <h3>Invite a user</h3>
          <p class="muted block text-muted">
            {{
              registrationMode === 'open'
                ? 'Open registration is enabled, so invite tokens are optional right now.'
                : 'Generate a single-use token for a new account.'
            }}
          </p>
        </div>
        <BaseButton
          class="mt-auto justify-self-start"
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

      <div class="user-public-panel">
        <div>
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Public access</p>
          <h3>{{ publicAccess ? 'Read-only visitors' : 'Private library' }}</h3>
          <p class="muted block text-muted">
            {{
              publicAccess
                ? 'Anonymous visitors can browse and export reading orders as CBL.'
                : 'Anonymous visitors must log in before seeing the library.'
            }}
          </p>
        </div>
        <div class="registration-mode-toggle" role="group" aria-label="Public access">
          <!-- Native buttons: this segmented control has its own selected-state treatment. -->
          <button
            type="button"
            class="access-mode-button"
            :class="{ active: !publicAccess }"
            :disabled="savingPublicAccess"
            @click="$emit('update-public-access', false)"
          >
            Private
          </button>
          <button
            type="button"
            class="access-mode-button"
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

    <section
      v-if="cblRepositorySync"
      v-show="activeSettingsTab === 'cbl-repositories'"
      id="settings-panel-cbl-repositories"
      class="account-settings-panel metron-scan-panel settings-tab-panel"
      role="tabpanel"
      aria-labelledby="settings-tab-cbl-repositories"
    >
      <header class="metron-scan-heading">
        <div class="metron-scan-heading-copy grid gap-1.5 max-w-prose">
          <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">
            CBL repositories
          </p>
          <h3>Automatic reading-list imports</h3>
          <p class="muted block text-muted">
            Import public GitHub repositories of CBL files. Changed files update their existing
            reading orders, and multipart files combine into one reading order with a section for
            each part.
          </p>
        </div>
        <label class="compact-toggle metron-scan-toggle">
          <input v-model="cblDraft.enabled" type="checkbox" />
          <span>{{ cblDraft.enabled ? 'Enabled' : 'Disabled' }}</span>
        </label>
      </header>

      <label class="metron-scan-field cbl-repository-list-field">
        <span>Repositories (one GitHub URL per line)</span>
        <textarea
          v-model="repositoryText"
          rows="4"
          spellcheck="false"
          placeholder="https://github.com/DieselTech/CBL-ReadingLists"
        ></textarea>
      </label>

      <div class="cbl-folder-scope">
        <div>
          <strong>Repository scope</strong>
          <span>{{ cblFolderScopeLabel }}</span>
          <small>
            Applies to regular checks and Import now. Only selected folders and their nested folders
            are downloaded.
          </small>
        </div>
        <BaseButton
          variant="neutral"
          :disabled="loadingCblRepositoryFiles || !repositoryText.trim()"
          @click="openCBLFolderPicker"
        >
          {{ loadingCblRepositoryFiles ? 'Loading folders...' : 'Choose folders' }}
        </BaseButton>
      </div>

      <label class="compact-toggle cbl-auto-sync-toggle">
        <input v-model="cblDraft.autoSync" type="checkbox" />
        <span>Regularly check repositories for new and updated files</span>
      </label>

      <div class="metron-scan-fields">
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Schedule</span>
          <BaseSelect v-model="cblDraft.schedule" size="large" :disabled="!cblDraft.autoSync">
            <option value="daily">Daily</option>
            <option value="weekly">Specific weekdays</option>
          </BaseSelect>
        </label>
        <label class="metron-scan-field grid gap-2 text-label font-extrabold">
          <span>Start time (server time)</span>
          <BaseTextInput
            v-model="cblDraft.startTime"
            size="large"
            type="time"
            :disabled="!cblDraft.autoSync"
          />
        </label>
      </div>

      <fieldset
        v-if="cblDraft.autoSync && cblDraft.schedule === 'weekly'"
        class="permission-scopes"
      >
        <legend>Run on</legend>
        <label v-for="day in weekdays" :key="`cbl-${day}`">
          <input
            type="checkbox"
            :checked="(cblDraft.weekdays || []).includes(day)"
            @change="toggleCBLWeekday(day, $event.target.checked)"
          />
          <span>{{ day.charAt(0).toUpperCase() + day.slice(1) }}</span>
        </label>
      </fieldset>

      <div class="cbl-manual-metron-option">
        <label class="compact-toggle">
          <input
            v-model="resolveMissingCBLIssues"
            type="checkbox"
            :disabled="cblRepositorySync.running"
          />
          <span>Search Metron for issues missing from this library</span>
        </label>
        <small>
          Used only by Import now and Choose files. Comic Vine IDs are tried first, then the series
          name and issue number.
        </small>
      </div>

      <div class="metron-scan-status" aria-live="polite">
        <div>
          <strong>{{ cblRepositorySync.filesFound }}</strong>
          <span>CBL files found</span>
        </div>
        <div>
          <strong>{{ cblRepositorySync.imported }}</strong>
          <span>new lists imported</span>
        </div>
        <div>
          <strong>{{ cblRepositorySync.updated }}</strong>
          <span>existing lists updated</span>
        </div>
        <div v-if="cblRepositorySync.unchanged">
          <strong>{{ cblRepositorySync.unchanged }}</strong>
          <span>unchanged</span>
        </div>
        <p>
          <template v-if="cblRepositorySync.pendingResolution">
            Waiting for a Metron issue selection
          </template>
          <template v-else-if="cblRepositorySync.running">
            Importing {{ cblRepositorySync.currentFile || cblRepositorySync.currentRepository }}
          </template>
          <template v-else-if="cblRepositorySync.stopReason">
            Last import: {{ cblRepositorySync.stopReason }}
            <template v-if="cblRepositorySync.failed">
              · {{ cblRepositorySync.failed }} failed
            </template>
          </template>
          <template v-else>Not run yet</template>
        </p>
        <p v-if="cblRepositorySync.lastError" class="access-note">
          Last error: {{ cblRepositorySync.lastError }}
        </p>
      </div>

      <div class="metron-scan-actions">
        <BaseButton
          variant="primary"
          size="large"
          :disabled="savingCblRepositorySync || !repositoryText.trim()"
          @click="saveCBLRepositorySync"
        >
          {{ savingCblRepositorySync ? 'Saving...' : 'Save settings' }}
        </BaseButton>
        <BaseButton
          v-if="!cblRepositorySync.running"
          variant="neutral"
          size="large"
          :disabled="savingCblRepositorySync || !cblDraft.enabled || !repositoryText.trim()"
          @click="startCBLRepositorySync"
        >
          {{ savingCblRepositorySync ? 'Saving and starting...' : 'Import now' }}
        </BaseButton>
        <BaseButton
          v-if="!cblRepositorySync.running"
          variant="neutral"
          size="large"
          :disabled="
            savingCblRepositorySync ||
            loadingCblRepositoryFiles ||
            !cblDraft.enabled ||
            !repositoryText.trim()
          "
          @click="openCBLFilePicker"
        >
          {{ loadingCblRepositoryFiles ? 'Loading files...' : 'Choose files' }}
        </BaseButton>
        <BaseButton
          v-else
          variant="danger-ghost"
          size="large"
          @click="$emit('stop-cbl-repository-sync')"
        >
          Stop import
        </BaseButton>
      </div>

      <ModalShell
        v-if="cblFolderPickerOpen"
        v-slot="{ titleId }"
        class="cbl-file-picker"
        size="extra-wide"
        structured
        @close="closeCBLFolderPicker"
      >
        <PanelHeader
          class="cbl-file-picker-header"
          title="Choose repository folders"
          :title-id="titleId"
          divided
          closable
          close-label="Close CBL folder picker"
          @close="closeCBLFolderPicker"
        >
          <template #description>
            <small>
              Choose one or more folders. Clear the selection to use the entire repository.
            </small>
          </template>
        </PanelHeader>

        <div class="cbl-file-picker-tools">
          <BaseTextInput
            v-model="cblFolderSearch"
            class="down-mobile:col-auto"
            type="search"
            placeholder="Search folders..."
          />
          <BaseButton
            variant="neutral"
            :disabled="!selectedCBLFolderKeys.size"
            @click="clearSelectedCBLFolders"
          >
            Use all folders
          </BaseButton>
        </div>

        <LoadingState v-if="loadingCblRepositoryFiles" compact />
        <p
          v-else-if="!filteredCBLRepositoryFolders.length"
          class="muted cbl-picker-empty p-4 text-muted font-ui-semibold block"
        >
          No folders containing CBL files match this search.
        </p>
        <div v-else class="cbl-file-picker-list">
          <div
            v-for="folder in visibleCBLRepositoryFolders"
            :key="cblFolderKey(folder)"
            v-memo="[selectedCBLFolderKeys.has(cblFolderKey(folder))]"
          >
            <label>
              <input
                class="mt-1"
                type="checkbox"
                :checked="selectedCBLFolderKeys.has(cblFolderKey(folder))"
                @change="toggleCBLFolder(folder, $event.target.checked)"
              />
              <span class="cbl-file-picker-path min-w-0 grid gap-1">
                <strong>{{ folder.path }}</strong>
                <small>
                  {{ repositoryLabel(folder.repositoryUrl) }} · {{ folder.fileCount }}
                  {{ folder.fileCount === 1 ? 'CBL file' : 'CBL files' }}
                </small>
              </span>
            </label>
          </div>
          <BaseButton
            v-if="remainingCBLRepositoryFolderCount"
            class="justify-self-center min-w-[min(280px,100%)] my-3 mx-2"
            variant="secondary-stacked"
            @click="showMoreCBLRepositoryFolders"
          >
            Show {{ Math.min(cblFileBatchSize, remainingCBLRepositoryFolderCount) }} more
            <small>{{ remainingCBLRepositoryFolderCount }} matches not shown</small>
          </BaseButton>
        </div>

        <footer class="cbl-file-picker-actions">
          <span>
            {{
              selectedCBLRepositoryFolders.length
                ? `${selectedCBLRepositoryFolders.length} folders selected`
                : 'Entire repositories selected'
            }}
          </span>
          <BaseButton variant="secondary" @click="closeCBLFolderPicker"> Cancel </BaseButton>
          <BaseButton
            variant="primary"
            :disabled="loadingCblRepositoryFiles"
            @click="applySelectedCBLRepositoryFolders"
          >
            Apply scope
          </BaseButton>
        </footer>
      </ModalShell>

      <ModalShell
        v-if="cblFilePickerOpen"
        v-slot="{ titleId }"
        class="cbl-file-picker"
        size="extra-wide"
        structured
        @close="closeCBLFilePicker"
      >
        <PanelHeader
          class="cbl-file-picker-header"
          title="Choose CBL files"
          :title-id="titleId"
          divided
          closable
          close-label="Close CBL file picker"
          @close="closeCBLFilePicker"
        >
          <template #description>
            <small>Select one part on its own, or use “Select all parts” for the full list.</small>
          </template>
        </PanelHeader>

        <div class="cbl-file-picker-tools">
          <BaseTextInput
            v-model="cblFileSearch"
            class="down-mobile:col-auto"
            type="search"
            placeholder="Search paths..."
          />
          <BaseButton
            variant="neutral"
            :disabled="!selectedCBLFileKeys.size"
            @click="clearSelectedCBLFiles"
          >
            Clear
          </BaseButton>
        </div>

        <LoadingState v-if="loadingCblRepositoryFiles" compact />
        <p v-else-if="!filteredCBLRepositoryFiles.length" class="muted block text-muted">
          No CBL files match this search.
        </p>
        <div v-else class="cbl-file-picker-list">
          <div
            v-for="file in visibleCBLRepositoryFiles"
            :key="cblFileKey(file)"
            v-memo="[
              selectedCBLFileKeys.has(cblFileKey(file)),
              file.multipartGroup ? allMultipartPartsSelected(file) : false,
            ]"
          >
            <label>
              <input
                class="mt-1"
                type="checkbox"
                :checked="selectedCBLFileKeys.has(cblFileKey(file))"
                @change="toggleCBLFile(file, $event.target.checked)"
              />
              <span class="cbl-file-picker-path min-w-0 grid gap-1">
                <strong>{{ file.path }}</strong>
                <small>
                  {{ repositoryLabel(file.repositoryUrl) }} · {{ fileSizeLabel(file.size) }}
                  <template v-if="file.multipartGroup">
                    · {{ file.multipartGroup }}, part {{ file.part }}
                  </template>
                </small>
              </span>
            </label>
            <!-- Native button: multipart selection is an inline text command, not a form action. -->
            <button
              v-if="file.multipartGroup"
              type="button"
              class="cbl-select-parts-button"
              :disabled="allMultipartPartsSelected(file)"
              @click="selectAllCBLParts(file)"
            >
              {{
                allMultipartPartsSelected(file)
                  ? `All ${multipartPartCount(file)} parts selected`
                  : `Select all ${multipartPartCount(file)} parts`
              }}
            </button>
          </div>
          <BaseButton
            v-if="remainingCBLRepositoryFileCount"
            class="justify-self-center min-w-[min(280px,100%)] my-3 mx-2"
            variant="secondary-stacked"
            @click="showMoreCBLRepositoryFiles"
          >
            Show {{ Math.min(cblFileBatchSize, remainingCBLRepositoryFileCount) }} more
            <small>{{ remainingCBLRepositoryFileCount }} matches not shown</small>
          </BaseButton>
        </div>

        <footer class="cbl-file-picker-actions">
          <span>{{ selectedCBLRepositoryFiles.length }} files selected</span>
          <BaseButton variant="secondary" @click="closeCBLFilePicker"> Cancel </BaseButton>
          <BaseButton
            variant="primary"
            :disabled="!selectedCBLRepositoryFiles.length || savingCblRepositorySync"
            @click="startSelectedCBLRepositoryFiles"
          >
            Import selected
          </BaseButton>
        </footer>
      </ModalShell>

      <ModalShell
        v-if="cblRepositorySync.pendingResolution"
        v-slot="{ titleId }"
        class="cbl-file-picker cbl-metron-issue-picker"
        size="wide"
        structured
      >
        <PanelHeader
          class="cbl-file-picker-header"
          title="Choose the matching Metron issue"
          :title-id="titleId"
          divided
        >
          <template #description>
            <small>
              {{ cblRepositorySync.pendingResolution.series }}
              #{{ cblRepositorySync.pendingResolution.number }}
              <template v-if="cblRepositorySync.pendingResolution.volume">
                · volume {{ cblRepositorySync.pendingResolution.volume }}
              </template>
              <template v-if="cblRepositorySync.pendingResolution.year">
                · {{ cblRepositorySync.pendingResolution.year }}
              </template>
            </small>
          </template>
        </PanelHeader>

        <div class="cbl-metron-candidate-list min-h-0 overflow-auto grid content-start gap-2.5 p-4">
          <!-- Native button: each result row is a full-card selection target. -->
          <button
            v-for="candidate in cblRepositorySync.pendingResolution.candidates"
            :key="candidate.id"
            type="button"
            class="cbl-metron-candidate"
            @click="chooseCBLMetronIssue(candidate.id)"
          >
            <img v-if="candidate.coverImage" :src="candidate.coverImage" alt="" />
            <span class="cbl-metron-candidate-copy">
              <strong>
                {{ candidate.series }}
                <template v-if="candidate.seriesYear">({{ candidate.seriesYear }})</template>
                #{{ candidate.number }}
              </strong>
              <small>
                {{ candidate.publisher || 'Unknown publisher' }}
                <template v-if="candidate.coverDate"> · {{ candidate.coverDate }}</template>
              </small>
              <small>
                Metron #{{ candidate.id }}
                <template v-if="candidate.comicVineId">
                  · Comic Vine #{{ candidate.comicVineId }}
                </template>
              </small>
            </span>
          </button>
        </div>

        <footer class="cbl-file-picker-actions">
          <span>Select the issue that should be added to this reading order.</span>
          <BaseButton variant="secondary" @click="chooseCBLMetronIssue(0)">
            Use CBL data only
          </BaseButton>
        </footer>
      </ModalShell>
    </section>

    <div
      v-show="activeSettingsTab === 'metron'"
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
              Choose which missing fields make a comic incomplete. Issue responses also create
              missing arc and character links without extra detail calls.
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
          days after it was last checked, so those rows stop using up the whole daily call budget.
          Set to 0 to recheck everything every run.
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
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.metron-scan-field :is(input, select, textarea) {
  @apply w-full;
}

.cbl-repository-list-field textarea {
  @apply min-h-24 resize-y font-ui-semibold;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
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

.cbl-file-picker-header > div {
  @apply grid gap-1;
}

.cbl-file-picker-header small,
.cbl-file-picker-path small,
.cbl-file-picker-actions span {
  @apply text-sm font-bold text-muted;
}

.settings-tab-button {
  @apply min-h-10 min-w-0 rounded-[7px] border-0 bg-transparent px-3 py-2 font-extrabold text-label;
}

.settings-tab-button.active {
  @apply bg-primary text-white;
}

.access-mode-button {
  @apply min-h-10 min-w-0 rounded-[7px] border-0 bg-transparent px-2.5 py-2 text-sm leading-tight font-black whitespace-normal text-control;
  overflow-wrap: anywhere;
}

.access-mode-button.active {
  @apply bg-primary text-(--primary-ink);
}

.cbl-file-picker-path strong {
  overflow-wrap: anywhere;
}

.cbl-file-picker-actions span {
  @apply mr-auto;
}

@media (width <= 720px) {
  .metron-scan-status p,
  .cbl-file-picker-actions span {
    @apply ml-0 mr-0;
  }
}

.settings-tabs {
  @apply grid grid-cols-3 gap-1.5 border border-line-strong rounded-lg bg-panel-soft p-1.5;
}

.user-access-panel.settings-tab-panel {
  @apply min-w-0 grid grid-cols-3 gap-4 items-stretch [&_>_div]:h-full down-tablet:grid-cols-1;
}

.user-registration-panel {
  @apply grid gap-4 content-start border border-line rounded-xl bg-surface-soft p-5 shadow-float [&_>_.registration-mode-toggle]:mt-auto;
}

.registration-mode-toggle {
  @apply grid grid-cols-2 gap-1 border border-line rounded bg-surface p-1;
}

.access-note {
  @apply m-0 border border-warning-border rounded bg-warning-soft text-warning py-2.5 px-3 text-sm font-bold leading-ui;
}

.user-invite-panel {
  @apply grid gap-4 content-start border border-line rounded-xl bg-surface-soft p-5 shadow-float;
}

.invite-token-box {
  @apply grid gap-1 border border-line rounded bg-surface p-3 [&_span]:text-muted [&_span]:text-sm [&_span]:font-bold [&_small]:text-muted [&_small]:text-sm [&_small]:font-bold [&_code]:text-(--heading) [&_code]:font-extrabold;
}

.user-public-panel {
  @apply grid gap-4 content-start border border-line rounded-xl bg-surface-soft p-5 shadow-float [&_>_.registration-mode-toggle]:mt-auto;
}

.account-settings-panel.metron-scan-panel.settings-tab-panel {
  @apply min-w-0 gap-6 rounded-xl p-6 down-mobile:p-4 grid border border-line bg-surface-soft;
}

.metron-scan-heading {
  @apply flex items-start justify-between gap-6 down-mobile:items-stretch down-mobile:flex-col;
}

.compact-toggle.metron-scan-toggle {
  @apply flex-none min-w-36 justify-center border border-line rounded bg-surface py-3 px-3.5 down-mobile:self-start inline-flex items-center gap-2 min-h-8 text-label font-extrabold leading-ui-tight;
}

.metron-scan-field.cbl-repository-list-field {
  @apply grid gap-2 text-label font-extrabold max-w-prose;
}

.cbl-folder-scope {
  @apply max-w-prose flex items-center justify-between gap-4 border border-line rounded bg-surface py-3 px-3.5 down-mobile:items-stretch down-mobile:flex-col [&_>_div]:min-w-0 [&_>_div]:grid [&_>_div]:gap-1 [&_span]:font-extrabold [&_small]:text-muted [&_small]:font-ui-semibold down-mobile:[&_>_button]:w-full;
}

.compact-toggle.cbl-auto-sync-toggle {
  @apply justify-self-start inline-flex items-center gap-2 min-h-8 border border-line rounded bg-surface text-label py-2 px-2.5 font-extrabold leading-ui-tight;
}

.metron-scan-fields {
  @apply grid grid-cols-[repeat(2,minmax(220px,360px))] gap-y-4 gap-x-6 down-mobile:grid-cols-1;
}

.permission-scopes {
  @apply border-0 p-0 m-0 grid grid-cols-[repeat(auto-fit,minmax(126px,1fr))] gap-2 min-w-0 disabled:opacity-55 down-mobile:grid-cols-1;
}

.cbl-manual-metron-option {
  @apply grid justify-items-start gap-1 [border-left:3px_solid_var(--line-strong)] pl-3 [&_small]:max-w-prose [&_small]:text-muted [&_small]:font-ui-semibold;
}

.compact-toggle {
  @apply inline-flex items-center gap-2 min-h-8 border border-line rounded bg-surface text-label py-2 px-2.5 font-extrabold leading-ui-tight;
}

.metron-scan-status {
  @apply flex items-center gap-7 border border-line rounded bg-surface py-3 px-3.5 text-muted down-mobile:items-stretch down-mobile:flex-col;
}

.metron-scan-actions {
  @apply flex items-center flex-wrap gap-2.5 down-mobile:items-stretch down-mobile:flex-col [&_>_button]:w-40 down-mobile:[&_>_button]:w-full;
}

.cbl-file-picker {
  @apply w-[min(920px,100%)] max-h-[min(820px,calc(100vh-36px))] grid grid-rows-[auto_auto_minmax(0,1fr)_auto] border border-line-strong rounded-lg bg-panel shadow-overlay overflow-hidden;
}

.cbl-file-picker-header {
  @apply border-b border-line flex items-center justify-between gap-3 py-3.5 px-4;
}

.cbl-file-picker-tools {
  @apply grid grid-cols-[minmax(180px,1fr)_auto] gap-2.5 border-b border-line py-3 px-4 down-mobile:grid-cols-1;
}

.cbl-file-picker-list {
  @apply min-h-0 overflow-auto grid content-start py-1.5 px-4 [&_>_div]:min-w-0 [&_>_div]:flex [&_>_div]:items-center [&_>_div]:gap-2.5 [&_>_div]:border-b [&_>_div]:border-line [&_>_div]:py-2.5 [&_>_div]:px-0.5 [&_>_div_>_label]:min-w-0 [&_>_div_>_label]:flex-auto [&_>_div_>_label]:flex [&_>_div_>_label]:items-start [&_>_div_>_label]:gap-2.5 [&_>_div_>_label]:cursor-pointer [&_>_div:last-of-type]:border-b-0 down-mobile:[&_>_div]:items-stretch down-mobile:[&_>_div]:flex-col;
}

.cbl-file-picker-actions {
  @apply justify-end border-t border-line down-mobile:items-stretch down-mobile:flex-col flex items-center gap-3 py-3.5 px-4;
}

.cbl-select-parts-button {
  @apply flex-none border-0 bg-transparent text-accent py-1.5 px-2 text-xs font-ui-extrabold down-mobile:self-start disabled:text-muted;
}

.cbl-file-picker.cbl-metron-issue-picker {
  @apply max-h-[min(820px,calc(100vh-36px))] grid border border-line-strong rounded-lg bg-panel shadow-overlay overflow-hidden w-[min(760px,100%)] grid-rows-[auto_minmax(0,1fr)_auto];
}

.cbl-metron-candidate {
  @apply w-full flex items-center gap-3.5 border border-line rounded bg-surface p-2.5 text-inherit text-left down-mobile:items-start hover:border-(--accent) focus-visible:border-(--accent) [&_img]:w-14 [&_img]:h-20 [&_img]:flex-none [&_img]:rounded-sm [&_img]:object-cover;
}

.cbl-metron-candidate-copy {
  @apply min-w-0 grid gap-1 [&_small]:text-muted [&_small]:font-bold;
}

.account-settings-panel.metron-scan-panel {
  @apply gap-6 rounded-xl p-6 down-mobile:p-4 grid border border-line bg-surface-soft;
}

.permission-scopes.metron-discovery-types {
  @apply border-0 p-0 m-0 grid grid-cols-[repeat(auto-fit,minmax(126px,1fr))] gap-2 min-w-0 disabled:opacity-55 down-mobile:grid-cols-1;
}

.permission-scopes.metron-incomplete-fields {
  @apply border-0 p-0 m-0 grid grid-cols-[repeat(auto-fit,minmax(126px,1fr))] gap-2 min-w-0 disabled:opacity-55 down-mobile:grid-cols-1;
}

.invite-token-box code {
  overflow-wrap: anywhere;
}

.cbl-metron-candidate-copy strong {
  overflow-wrap: anywhere;
}
</style>
