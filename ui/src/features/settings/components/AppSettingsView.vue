<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LoadingState from '@/shared/components/feedback/LoadingState.vue'

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
  <section class="browse-view app-settings-view">
    <nav class="settings-tabs" role="tablist" aria-label="App settings sections">
      <button
        v-for="tab in settingsTabs"
        :id="`settings-tab-${tab.value}`"
        :key="tab.value"
        type="button"
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

    <section
      v-if="cblRepositorySync"
      v-show="activeSettingsTab === 'cbl-repositories'"
      id="settings-panel-cbl-repositories"
      class="account-settings-panel metron-scan-panel settings-tab-panel"
      role="tabpanel"
      aria-labelledby="settings-tab-cbl-repositories"
    >
      <header class="metron-scan-heading">
        <div class="metron-scan-heading-copy">
          <p class="eyebrow">CBL repositories</p>
          <h3>Automatic reading-list imports</h3>
          <p class="muted">
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
        <button
          type="button"
          class="secondary-action"
          :disabled="loadingCblRepositoryFiles || !repositoryText.trim()"
          @click="openCBLFolderPicker"
        >
          {{ loadingCblRepositoryFiles ? 'Loading folders...' : 'Choose folders' }}
        </button>
      </div>

      <label class="compact-toggle cbl-auto-sync-toggle">
        <input v-model="cblDraft.autoSync" type="checkbox" />
        <span>Regularly check repositories for new and updated files</span>
      </label>

      <div class="metron-scan-fields">
        <label class="metron-scan-field">
          <span>Schedule</span>
          <select v-model="cblDraft.schedule" :disabled="!cblDraft.autoSync">
            <option value="daily">Daily</option>
            <option value="weekly">Specific weekdays</option>
          </select>
        </label>
        <label class="metron-scan-field">
          <span>Start time (server time)</span>
          <input v-model="cblDraft.startTime" type="time" :disabled="!cblDraft.autoSync" />
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
        <button
          type="button"
          class="primary-button"
          :disabled="savingCblRepositorySync || !repositoryText.trim()"
          @click="saveCBLRepositorySync"
        >
          {{ savingCblRepositorySync ? 'Saving...' : 'Save settings' }}
        </button>
        <button
          v-if="!cblRepositorySync.running"
          type="button"
          class="secondary-action"
          :disabled="savingCblRepositorySync || !cblDraft.enabled || !repositoryText.trim()"
          @click="startCBLRepositorySync"
        >
          {{ savingCblRepositorySync ? 'Saving and starting...' : 'Import now' }}
        </button>
        <button
          v-if="!cblRepositorySync.running"
          type="button"
          class="secondary-action"
          :disabled="
            savingCblRepositorySync ||
            loadingCblRepositoryFiles ||
            !cblDraft.enabled ||
            !repositoryText.trim()
          "
          @click="openCBLFilePicker"
        >
          {{ loadingCblRepositoryFiles ? 'Loading files...' : 'Choose files' }}
        </button>
        <button
          v-else
          type="button"
          class="danger-text-button"
          @click="$emit('stop-cbl-repository-sync')"
        >
          Stop import
        </button>
      </div>

      <div v-if="cblFolderPickerOpen" class="modal-backdrop" @click.self="closeCBLFolderPicker">
        <section
          class="cbl-file-picker"
          role="dialog"
          aria-modal="true"
          aria-labelledby="cbl-folder-picker-title"
        >
          <header class="cbl-file-picker-header">
            <div>
              <strong id="cbl-folder-picker-title">Choose repository folders</strong>
              <small>
                Choose one or more folders. Clear the selection to use the entire repository.
              </small>
            </div>
            <button
              class="icon-button"
              type="button"
              aria-label="Close CBL folder picker"
              @click="closeCBLFolderPicker"
            >
              ×
            </button>
          </header>

          <div class="cbl-file-picker-tools">
            <input v-model="cblFolderSearch" type="search" placeholder="Search folders..." />
            <button
              type="button"
              class="secondary-action"
              :disabled="!selectedCBLFolderKeys.size"
              @click="clearSelectedCBLFolders"
            >
              Use all folders
            </button>
          </div>

          <LoadingState v-if="loadingCblRepositoryFiles" compact />
          <p v-else-if="!filteredCBLRepositoryFolders.length" class="muted cbl-picker-empty">
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
                  type="checkbox"
                  :checked="selectedCBLFolderKeys.has(cblFolderKey(folder))"
                  @change="toggleCBLFolder(folder, $event.target.checked)"
                />
                <span class="cbl-file-picker-path">
                  <strong>{{ folder.path }}</strong>
                  <small>
                    {{ repositoryLabel(folder.repositoryUrl) }} · {{ folder.fileCount }}
                    {{ folder.fileCount === 1 ? 'CBL file' : 'CBL files' }}
                  </small>
                </span>
              </label>
            </div>
            <button
              v-if="remainingCBLRepositoryFolderCount"
              type="button"
              class="secondary-button cbl-file-picker-more"
              @click="showMoreCBLRepositoryFolders"
            >
              Show {{ Math.min(cblFileBatchSize, remainingCBLRepositoryFolderCount) }} more
              <small>{{ remainingCBLRepositoryFolderCount }} matches not shown</small>
            </button>
          </div>

          <footer class="cbl-file-picker-actions">
            <span>
              {{
                selectedCBLRepositoryFolders.length
                  ? `${selectedCBLRepositoryFolders.length} folders selected`
                  : 'Entire repositories selected'
              }}
            </span>
            <button type="button" class="secondary-button" @click="closeCBLFolderPicker">
              Cancel
            </button>
            <button
              type="button"
              class="primary-button"
              :disabled="loadingCblRepositoryFiles"
              @click="applySelectedCBLRepositoryFolders"
            >
              Apply scope
            </button>
          </footer>
        </section>
      </div>

      <div v-if="cblFilePickerOpen" class="modal-backdrop" @click.self="closeCBLFilePicker">
        <section
          class="cbl-file-picker"
          role="dialog"
          aria-modal="true"
          aria-labelledby="cbl-file-picker-title"
        >
          <header class="cbl-file-picker-header">
            <div>
              <strong id="cbl-file-picker-title">Choose CBL files</strong>
              <small
                >Select one part on its own, or use “Select all parts” for the full list.</small
              >
            </div>
            <button
              class="icon-button"
              type="button"
              aria-label="Close CBL file picker"
              @click="closeCBLFilePicker"
            >
              ×
            </button>
          </header>

          <div class="cbl-file-picker-tools">
            <input v-model="cblFileSearch" type="search" placeholder="Search paths..." />
            <button
              type="button"
              class="secondary-action"
              :disabled="!selectedCBLFileKeys.size"
              @click="clearSelectedCBLFiles"
            >
              Clear
            </button>
          </div>

          <LoadingState v-if="loadingCblRepositoryFiles" compact />
          <p v-else-if="!filteredCBLRepositoryFiles.length" class="muted">
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
                  type="checkbox"
                  :checked="selectedCBLFileKeys.has(cblFileKey(file))"
                  @change="toggleCBLFile(file, $event.target.checked)"
                />
                <span class="cbl-file-picker-path">
                  <strong>{{ file.path }}</strong>
                  <small>
                    {{ repositoryLabel(file.repositoryUrl) }} · {{ fileSizeLabel(file.size) }}
                    <template v-if="file.multipartGroup">
                      · {{ file.multipartGroup }}, part {{ file.part }}
                    </template>
                  </small>
                </span>
              </label>
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
            <button
              v-if="remainingCBLRepositoryFileCount"
              type="button"
              class="secondary-button cbl-file-picker-more"
              @click="showMoreCBLRepositoryFiles"
            >
              Show {{ Math.min(cblFileBatchSize, remainingCBLRepositoryFileCount) }} more
              <small>{{ remainingCBLRepositoryFileCount }} matches not shown</small>
            </button>
          </div>

          <footer class="cbl-file-picker-actions">
            <span>{{ selectedCBLRepositoryFiles.length }} files selected</span>
            <button type="button" class="secondary-button" @click="closeCBLFilePicker">
              Cancel
            </button>
            <button
              type="button"
              class="primary-button"
              :disabled="!selectedCBLRepositoryFiles.length || savingCblRepositorySync"
              @click="startSelectedCBLRepositoryFiles"
            >
              Import selected
            </button>
          </footer>
        </section>
      </div>

      <div v-if="cblRepositorySync.pendingResolution" class="modal-backdrop">
        <section
          class="cbl-file-picker cbl-metron-issue-picker"
          role="dialog"
          aria-modal="true"
          aria-labelledby="cbl-metron-issue-picker-title"
        >
          <header class="cbl-file-picker-header">
            <div>
              <strong id="cbl-metron-issue-picker-title">Choose the matching Metron issue</strong>
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
            </div>
          </header>

          <div class="cbl-metron-candidate-list">
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
            <button type="button" class="secondary-button" @click="chooseCBLMetronIssue(0)">
              Use CBL data only
            </button>
          </footer>
        </section>
      </div>
    </section>

    <div
      v-show="activeSettingsTab === 'metron'"
      id="settings-panel-metron"
      class="settings-tab-panel settings-metron-panel"
      role="tabpanel"
      aria-labelledby="settings-tab-metron"
    >
      <slot name="metron-import"></slot>

      <section v-if="metronComicDiscovery" class="account-settings-panel metron-scan-panel">
        <header class="metron-scan-heading">
          <div class="metron-scan-heading-copy">
            <p class="eyebrow">Metron discovery</p>
            <h3>Automatic new Metron data</h3>
            <p class="muted">
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
          <label class="metron-scan-field">
            <span>Schedule</span>
            <select v-model="discoveryDraft.schedule">
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
            </select>
          </label>
          <label class="metron-scan-field">
            <span>Start time (server time)</span>
            <input v-model="discoveryDraft.startTime" type="time" />
          </label>
          <label class="metron-scan-field">
            <span>Publisher name filter</span>
            <input
              v-model="discoveryDraft.publisherName"
              type="text"
              placeholder="All publishers"
              :disabled="!discoveryDraft.pullComics"
            />
          </label>
          <label class="metron-scan-field">
            <span>Series name filter</span>
            <input
              v-model="discoveryDraft.seriesName"
              type="text"
              placeholder="All series"
              :disabled="!discoveryDraft.pullComics"
            />
          </label>
          <label v-if="discoveryDraft.schedule === 'monthly'" class="metron-scan-field">
            <span>Day of month</span>
            <input v-model.number="discoveryDraft.monthDay" type="number" min="1" max="31" />
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
          <button
            type="button"
            class="primary-button"
            :disabled="
              savingDiscovery || (!discoveryDraft.pullComics && !discoveryDraft.pullReadingLists)
            "
            @click="saveDiscovery"
          >
            {{ savingDiscovery ? 'Saving...' : 'Save settings' }}
          </button>
          <button
            v-if="!metronComicDiscovery.running"
            type="button"
            class="secondary-action"
            :disabled="
              savingDiscovery ||
              !discoveryDraft.enabled ||
              (!discoveryDraft.pullComics && !discoveryDraft.pullReadingLists)
            "
            @click="startComicDiscovery"
          >
            {{ savingDiscovery ? 'Saving and starting...' : 'Pull now' }}
          </button>
          <button v-else type="button" class="danger-text-button" @click="$emit('stop-discovery')">
            Stop pull
          </button>
        </div>
      </section>

      <section v-if="metronComicScan" class="account-settings-panel metron-scan-panel">
        <header class="metron-scan-heading">
          <div class="metron-scan-heading-copy">
            <p class="eyebrow">Metron maintenance</p>
            <h3>Incomplete comic data</h3>
            <p class="muted">
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
          <label class="metron-scan-field">
            <span>Re-check cooldown (days)</span>
            <input v-model.number="draft.recheckCooldownDays" min="0" step="1" type="number" />
          </label>
        </div>
        <p class="muted metron-scan-hint">
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

        <div class="metron-scan-actions">
          <button
            type="button"
            class="primary-button"
            :disabled="saving || !(draft.incompleteFields || []).length"
            @click="save"
          >
            {{ saving ? 'Saving...' : 'Save settings' }}
          </button>
          <button
            v-if="!metronComicScan.running"
            type="button"
            class="secondary-action"
            :disabled="saving || !draft.enabled || !(draft.incompleteFields || []).length"
            @click="startComicScan"
          >
            {{ saving ? 'Saving and starting...' : 'Scan now' }}
          </button>
          <button v-else type="button" class="danger-text-button" @click="$emit('stop')">
            Stop scan
          </button>
        </div>
      </section>
      <LoadingState v-else />
    </div>
  </section>
</template>
