<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import CBLRepositorySettings from './CBLRepositorySettings.vue'
import MetronSchedulingSettings from './MetronSchedulingSettings.vue'
import UserAccessSettings from './UserAccessSettings.vue'

defineProps({
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

defineEmits([
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

    <UserAccessSettings
      v-show="activeSettingsTab === 'general'"
      id="settings-panel-general"
      :registration-mode="registrationMode"
      :saving-registration-mode="savingRegistrationMode"
      :public-access="publicAccess"
      :saving-public-access="savingPublicAccess"
      :invite="invite"
      :generating-invite="generatingInvite"
      @update-registration-mode="$emit('update-registration-mode', $event)"
      @update-public-access="$emit('update-public-access', $event)"
      @generate-invite="$emit('generate-invite')"
    />

    <CBLRepositorySettings
      v-if="cblRepositorySync"
      v-show="activeSettingsTab === 'cbl-repositories'"
      :cbl-repository-sync="cblRepositorySync"
      :cbl-repository-files="cblRepositoryFiles"
      :saving-cbl-repository-sync="savingCblRepositorySync"
      :loading-cbl-repository-files="loadingCblRepositoryFiles"
      @save="$emit('save-cbl-repository-sync', $event)"
      @load-files="$emit('load-cbl-repository-files', $event)"
      @trigger="$emit('trigger-cbl-repository-sync', $event)"
      @stop="$emit('stop-cbl-repository-sync')"
      @resolve-metron-issue="$emit('resolve-cbl-metron-issue', $event)"
      @resolution-pending="selectSettingsTab('cbl-repositories')"
    />

    <MetronSchedulingSettings
      v-show="activeSettingsTab === 'metron'"
      :metron-comic-scan="metronComicScan"
      :metron-comic-discovery="metronComicDiscovery"
      :saving="saving"
      :saving-discovery="savingDiscovery"
      @save="$emit('save', $event)"
      @trigger="$emit('trigger', $event)"
      @stop="$emit('stop')"
      @save-discovery="$emit('save-discovery', $event)"
      @trigger-discovery="$emit('trigger-discovery', $event)"
      @stop-discovery="$emit('stop-discovery')"
    >
      <template #metron-import>
        <slot name="metron-import"></slot>
      </template>
    </MetronSchedulingSettings>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.settings-tab-button {
  @apply min-h-10 min-w-0 rounded-[7px] border-0 bg-transparent px-3 py-2 font-extrabold text-label;
}

.settings-tab-button.active {
  @apply bg-primary text-white;
}

.settings-tabs {
  @apply grid grid-cols-3 gap-1.5 border border-line-strong rounded-lg bg-panel-soft p-1.5 down-phone:grid-cols-2;
}

.settings-tab-button:last-child {
  @apply down-phone:col-span-2;
}
</style>
