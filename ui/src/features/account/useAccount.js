import { ref } from 'vue'
import { deleteAccount, getAccountStatistics, updateAccount } from '@/api/client.js'

export function useAccount({ error, userStatus, currentUser, activeView, viewMode, authMode }) {
  const saving = ref(false)
  const deleting = ref(false)
  const statistics = ref(null)
  const statisticsLoading = ref(false)
  const statisticsError = ref('')

  async function saveAccount(payload, validationMessage = '') {
    if (validationMessage) {
      error.value = validationMessage
      return
    }
    if (!payload) return

    saving.value = true
    error.value = ''
    try {
      userStatus.value = await updateAccount(payload)
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function deleteCurrentAccount(payload) {
    deleting.value = true
    error.value = ''
    try {
      activeView.value = 'readingOrders'
      viewMode.value = 'browse'
      userStatus.value = await deleteAccount(payload)
      authMode.value = 'login'
    } catch (err) {
      error.value = err.message
    } finally {
      deleting.value = false
    }
  }

  async function loadStatistics() {
    if (!currentUser.value) {
      statistics.value = null
      statisticsError.value = ''
      return
    }

    statisticsLoading.value = true
    statisticsError.value = ''
    try {
      statistics.value = await getAccountStatistics()
    } catch (err) {
      statisticsError.value = err.message
    } finally {
      statisticsLoading.value = false
    }
  }

  return {
    saving,
    deleting,
    statistics,
    statisticsLoading,
    statisticsError,
    saveAccount,
    deleteCurrentAccount,
    loadStatistics,
  }
}
