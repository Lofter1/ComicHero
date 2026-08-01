import { ref } from 'vue'
import { createApiToken, listApiTokens, revokeApiToken } from '@/api/client.js'

export function useApiTokens() {
  const tokens = ref([])
  const loading = ref(false)
  const error = ref('')
  const creating = ref(false)
  const revokingId = ref(null)
  // Set once, right after creation, to the { token, ...APIToken } response.
  // The raw token is never retrievable again, so this is the only chance to
  // show it - callers should render it prominently and clear it once the
  // user has copied it or dismissed it.
  const createdToken = ref(null)

  async function loadTokens() {
    loading.value = true
    error.value = ''
    try {
      tokens.value = await listApiTokens()
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function createToken(payload) {
    creating.value = true
    error.value = ''
    try {
      createdToken.value = await createApiToken(payload)
      await loadTokens()
      return true
    } catch (err) {
      error.value = err.message
      return false
    } finally {
      creating.value = false
    }
  }

  async function revokeToken(id) {
    if (
      !window.confirm('Revoke this API token? Any service using it will immediately lose access.')
    ) {
      return
    }
    revokingId.value = id
    error.value = ''
    try {
      await revokeApiToken(id)
      tokens.value = tokens.value.filter((token) => token.id !== id)
    } catch (err) {
      error.value = err.message
    } finally {
      revokingId.value = null
    }
  }

  function dismissCreatedToken() {
    createdToken.value = null
  }

  return {
    tokens,
    loading,
    error,
    creating,
    revokingId,
    createdToken,
    loadTokens,
    createToken,
    revokeToken,
    dismissCreatedToken,
  }
}
