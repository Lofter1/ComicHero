import { ref } from 'vue'
import {
  deleteUser,
  listAuditEvents,
  listUsers,
  updateUserAdmin,
  updateUserMetronPermissions,
} from '@/api/client.js'

export function useUserAdministration({ error }) {
  const users = ref([])
  const auditEvents = ref([])
  const auditPagination = ref({ limit: 25, offset: 0, total: 0, hasMore: false })
  const auditLoading = ref(false)
  const savingPermissionsUserId = ref(null)
  const savingAdminUserId = ref(null)
  const deletingUserId = ref(null)
  let auditRequestId = 0

  async function loadUsers() {
    const [userRows, auditPage] = await Promise.all([
      listUsers(),
      listAuditEvents({ limit: auditPagination.value.limit }),
    ])
    users.value = userRows
    applyAuditPage(auditPage)
  }

  function applyAuditPage(page) {
    auditEvents.value = page.items
    auditPagination.value = {
      limit: page.limit,
      offset: page.offset,
      total: page.total,
      hasMore: page.hasMore,
    }
  }

  async function loadAuditEvents(params = {}) {
    const requestId = ++auditRequestId
    auditLoading.value = true
    error.value = ''
    try {
      const page = await listAuditEvents(params)
      if (requestId === auditRequestId) applyAuditPage(page)
    } catch (err) {
      if (requestId === auditRequestId) error.value = err.message
    } finally {
      if (requestId === auditRequestId) auditLoading.value = false
    }
  }

  async function savePermissions(userId, payload) {
    await updateRow({
      userId,
      savingRef: savingPermissionsUserId,
      request: () => updateUserMetronPermissions(userId, payload),
    })
  }

  async function saveAdmin(userId, payload) {
    await updateRow({
      userId,
      savingRef: savingAdminUserId,
      request: () => updateUserAdmin(userId, payload),
    })
  }

  async function updateRow({ userId, savingRef, request }) {
    savingRef.value = userId
    error.value = ''
    try {
      const updated = await request()
      users.value = users.value.map((entry) => (entry.user.id === userId ? updated : entry))
    } catch (err) {
      error.value = err.message
    } finally {
      savingRef.value = null
    }
  }

  async function removeUser(userId) {
    if (
      typeof window !== 'undefined' &&
      !window.confirm(
        'Delete this account? Their sessions, read status, Metron permissions, and account data will be removed.',
      )
    ) {
      return
    }

    deletingUserId.value = userId
    error.value = ''
    try {
      await deleteUser(userId)
      users.value = users.value.filter((entry) => entry.user.id !== userId)
    } catch (err) {
      error.value = err.message
    } finally {
      deletingUserId.value = null
    }
  }

  return {
    users,
    auditEvents,
    auditPagination,
    auditLoading,
    savingPermissionsUserId,
    savingAdminUserId,
    deletingUserId,
    loadUsers,
    loadAuditEvents,
    savePermissions,
    saveAdmin,
    removeUser,
  }
}
