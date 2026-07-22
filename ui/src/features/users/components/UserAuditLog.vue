<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import BaseButton from '@/shared/components/form/BaseButton.vue'
import BaseSelect from '@/shared/components/form/BaseSelect.vue'
import BaseTextInput from '@/shared/components/form/BaseTextInput.vue'
import EmptyState from '@/shared/components/feedback/EmptyState.vue'

const props = defineProps({
  users: { type: Array, default: () => [] },
  auditEvents: { type: Array, default: () => [] },
  auditPagination: {
    type: Object,
    default: () => ({ limit: 25, offset: 0, total: 0, hasMore: false }),
  },
  auditLoading: { type: Boolean, default: false },
})

const emit = defineEmits(['load'])

const auditQuery = ref('')
const auditUser = ref('all')
const auditMethod = ref('all')
const auditStatus = ref('all')
const auditSort = ref('occurredAt')
const auditDirection = ref('desc')
const auditPageSize = ref(props.auditPagination.limit || 25)
let auditSearchTimer

const auditRange = computed(() => {
  if (!props.auditPagination.total || !props.auditEvents.length) return '0 results'
  const start = props.auditPagination.offset + 1
  const end = Math.min(
    props.auditPagination.offset + props.auditEvents.length,
    props.auditPagination.total,
  )
  return `${start}–${end} of ${props.auditPagination.total}`
})

const auditFiltersActive = computed(
  () =>
    auditQuery.value.trim() ||
    auditUser.value !== 'all' ||
    auditMethod.value !== 'all' ||
    auditStatus.value !== 'all',
)

watch([auditUser, auditMethod, auditStatus, auditSort, auditDirection, auditPageSize], () => {
  requestAuditEvents(0)
})

watch(auditQuery, () => {
  window.clearTimeout(auditSearchTimer)
  auditSearchTimer = window.setTimeout(() => requestAuditEvents(0), 250)
})

onBeforeUnmount(() => window.clearTimeout(auditSearchTimer))

function auditParams(offset) {
  const params = {
    q: auditQuery.value.trim(),
    method: auditMethod.value === 'all' ? '' : auditMethod.value,
    status: auditStatus.value === 'all' ? '' : auditStatus.value,
    sort: auditSort.value,
    direction: auditDirection.value,
    limit: auditPageSize.value,
    offset,
  }
  if (auditUser.value === 'system') params.system = true
  else if (auditUser.value !== 'all') params.userId = auditUser.value
  return params
}

function requestAuditEvents(offset) {
  emit('load', auditParams(Math.max(0, offset)))
}

function formatTimestamp(value) {
  if (!value) return 'Unknown'
  const normalized = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(value)
    ? `${value.replace(' ', 'T')}Z`
    : value
  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}
</script>

<template>
  <section class="detail-panel" :aria-busy="auditLoading">
    <header class="section-heading">
      <div>
        <p class="eyebrow mt-0 mb-1.5 text-eyebrow text-xs font-bold uppercase">Audit log</p>
        <h3>Recent changes</h3>
      </div>
      <p class="m-0 text-sm font-bold text-muted">{{ auditRange }}</p>
    </header>

    <div class="audit-log-tools">
      <label class="audit-search-field col-span-2 down-compact:col-span-full">
        <span class="sr-only">Search audit log</span>
        <BaseTextInput
          v-model="auditQuery"
          type="search"
          placeholder="Search user, action, path, or status"
        />
      </label>
      <label>
        <span class="sr-only">Filter audit log by user</span>
        <BaseSelect v-model="auditUser">
          <option value="all">All users</option>
          <option value="system">System / unauthenticated</option>
          <option v-for="entry in users" :key="entry.user.id" :value="String(entry.user.id)">
            {{ entry.user.name }}
          </option>
        </BaseSelect>
      </label>
      <label>
        <span class="sr-only">Filter audit log by method</span>
        <BaseSelect v-model="auditMethod">
          <option value="all">All methods</option>
          <option value="POST">POST</option>
          <option value="PUT">PUT</option>
          <option value="PATCH">PATCH</option>
          <option value="DELETE">DELETE</option>
        </BaseSelect>
      </label>
      <label>
        <span class="sr-only">Filter audit log by status</span>
        <BaseSelect v-model="auditStatus">
          <option value="all">All statuses</option>
          <option value="1xx">1xx informational</option>
          <option value="2xx">2xx success</option>
          <option value="3xx">3xx redirect</option>
          <option value="4xx">4xx client error</option>
          <option value="5xx">5xx server error</option>
        </BaseSelect>
      </label>
      <label>
        <span class="sr-only">Sort audit log by</span>
        <BaseSelect v-model="auditSort">
          <option value="occurredAt">Sort by date</option>
          <option value="user">Sort by user</option>
          <option value="action">Sort by action</option>
          <option value="status">Sort by status</option>
        </BaseSelect>
      </label>
      <label>
        <span class="sr-only">Audit log sort direction</span>
        <BaseSelect v-model="auditDirection">
          <option value="desc">Descending</option>
          <option value="asc">Ascending</option>
        </BaseSelect>
      </label>
      <label>
        <span class="sr-only">Audit log results per page</span>
        <BaseSelect v-model.number="auditPageSize">
          <option :value="25">25 results</option>
          <option :value="50">50 results</option>
          <option :value="100">100 results</option>
          <option :value="200">200 results</option>
        </BaseSelect>
      </label>
    </div>

    <p v-if="auditLoading && auditEvents.length === 0" class="mt-4 text-muted font-bold">
      Loading audit events...
    </p>
    <EmptyState v-else-if="auditEvents.length === 0" tag="p">
      {{
        auditFiltersActive
          ? 'No audit events match the current filters.'
          : 'No state-changing actions recorded yet.'
      }}
    </EmptyState>
    <div v-else class="table-scroll audit-table-scroll" :class="{ 'opacity-60': auditLoading }">
      <table class="w-full min-w-[640px] border-collapse text-left">
        <thead>
          <tr>
            <th class="border-b border-line bg-surface-muted px-3 py-2.5">When</th>
            <th class="border-b border-line bg-surface-muted px-3 py-2.5">User</th>
            <th class="border-b border-line bg-surface-muted px-3 py-2.5">Action</th>
            <th class="border-b border-line bg-surface-muted px-3 py-2.5">Status</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="event in auditEvents" :key="event.id">
            <td class="border-b border-line px-3 py-2.5 align-top">
              {{ formatTimestamp(event.occurredAt) }}
            </td>
            <td class="border-b border-line px-3 py-2.5 align-top">
              {{ event.userName || 'System / unauthenticated' }}
            </td>
            <td class="border-b border-line px-3 py-2.5 align-top">
              <code class="whitespace-nowrap">{{ event.method }} {{ event.path }}</code>
            </td>
            <td class="border-b border-line px-3 py-2.5 align-top">{{ event.statusCode }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <footer v-if="auditEvents.length || auditPagination.offset > 0" class="audit-pagination">
      <span class="text-sm font-bold text-muted">{{ auditRange }}</span>
      <div class="flex items-center justify-end gap-2 down-mobile:grid down-mobile:grid-cols-2">
        <BaseButton
          :disabled="auditLoading || auditPagination.offset === 0"
          @click="requestAuditEvents(auditPagination.offset - auditPageSize)"
        >
          Previous
        </BaseButton>
        <BaseButton
          :disabled="auditLoading || !auditPagination.hasMore"
          @click="requestAuditEvents(auditPagination.offset + auditPageSize)"
        >
          Next
        </BaseButton>
      </div>
    </footer>
  </section>
</template>

<style scoped>
@reference '../../../styles.css';

.detail-panel {
  @apply min-w-0 max-w-full min-h-panel border border-line rounded-xl bg-panel p-5 shadow-detail down-mobile:min-h-0 down-mobile:p-3.5;
}

.section-heading {
  @apply flex items-end justify-between gap-3 down-mobile:items-start down-mobile:flex-col;
}

.audit-log-tools {
  @apply mt-4 grid min-w-0 grid-cols-[repeat(auto-fit,minmax(min(100%,140px),1fr))] gap-2;
}

.audit-pagination {
  @apply mt-3 flex items-center justify-between gap-3 down-mobile:items-stretch down-mobile:flex-col;
}

.audit-table-scroll {
  @apply mt-4 w-full max-w-full overflow-x-auto overscroll-x-contain rounded border border-line;
}
</style>
