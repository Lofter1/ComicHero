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
  const savingPermissionsUserId = ref(null)
  const savingAdminUserId = ref(null)
  const deletingUserId = ref(null)

  async function loadUsers() {
    const [userRows, events] = await Promise.all([listUsers(), listAuditEvents({ limit: 200 })])
    users.value = userRows
    auditEvents.value = events
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
    savingPermissionsUserId,
    savingAdminUserId,
    deletingUserId,
    loadUsers,
    savePermissions,
    saveAdmin,
    removeUser,
  }
}
