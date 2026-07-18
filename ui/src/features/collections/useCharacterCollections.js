import { ref } from 'vue'
import {
  addCharacterToCollection,
  createCharacterCollection,
  deleteCharacterCollection as removeCharacterCollection,
  getCharacterCollection,
  listCharacterCollections,
  removeCharacterFromCollection,
  setCharacterCollectionStarted,
} from '@/api/client.js'

export function useCharacterCollections({ activeView, viewMode, loading, error }) {
  const collections = ref([])
  const selectedCollection = ref(null)
  const saving = ref(false)
  const startSaving = ref(false)
  const addDialogCharacter = ref(null)
  const addDialogCollections = ref([])
  const addDialogLoading = ref(false)

  async function loadCollections() {
    collections.value = await listCharacterCollections()
  }

  async function openCollection(collection) {
    if (!collection?.id) return
    loading.value = true
    error.value = ''
    try {
      activeView.value = 'collections'
      viewMode.value = 'detail'
      selectedCollection.value = await getCharacterCollection(collection.id)
    } finally {
      loading.value = false
    }
  }

  async function createCollection(name, { open = true } = {}) {
    if (saving.value) return null
    saving.value = true
    error.value = ''
    try {
      const detail = await createCharacterCollection({ name: String(name || '').trim() })
      await loadCollections()
      if (open) {
        activeView.value = 'collections'
        viewMode.value = 'detail'
        selectedCollection.value = detail
      }
      return detail
    } catch (err) {
      error.value = err.message
      return null
    } finally {
      saving.value = false
    }
  }

  async function toggleSelectedCollectionStarted() {
    if (!selectedCollection.value?.id || startSaving.value) return
    startSaving.value = true
    error.value = ''
    try {
      selectedCollection.value = await setCharacterCollectionStarted(
        selectedCollection.value.id,
        !selectedCollection.value.startedAt,
      )
      await loadCollections()
    } catch (err) {
      error.value = err.message
    } finally {
      startSaving.value = false
    }
  }

  async function deleteSelectedCollection() {
    const collection = selectedCollection.value
    if (!collection?.id || saving.value) return
    if (!confirm(`Delete ${collection.name}? This cannot be undone.`)) return
    saving.value = true
    error.value = ''
    try {
      await removeCharacterCollection(collection.id)
      selectedCollection.value = null
      await loadCollections()
      viewMode.value = 'browse'
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function addCharacter(character, collectionID = selectedCollection.value?.id) {
    if (!character?.id || !collectionID || saving.value) return false
    saving.value = true
    error.value = ''
    try {
      const detail = await addCharacterToCollection(collectionID, character.id)
      if (selectedCollection.value?.id === collectionID) selectedCollection.value = detail
      await loadCollections()
      return true
    } catch (err) {
      error.value = err.message
      return false
    } finally {
      saving.value = false
    }
  }

  async function removeCharacter(character) {
    if (!character?.id || !selectedCollection.value?.id || saving.value) return
    saving.value = true
    error.value = ''
    try {
      selectedCollection.value = await removeCharacterFromCollection(
        selectedCollection.value.id,
        character.id,
      )
      await loadCollections()
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function openAddDialog(character) {
    if (!character?.id) return
    addDialogCharacter.value = character
    addDialogCollections.value = []
    addDialogLoading.value = true
    error.value = ''
    try {
      addDialogCollections.value = await listCharacterCollections({ characterId: character.id })
    } catch (err) {
      error.value = err.message
      addDialogCharacter.value = null
    } finally {
      addDialogLoading.value = false
    }
  }

  function closeAddDialog() {
    if (saving.value) return
    addDialogCharacter.value = null
    addDialogCollections.value = []
  }

  async function addDialogCharacterTo(collection) {
    if (!collection?.id || !addDialogCharacter.value) return
    const added = await addCharacter(addDialogCharacter.value, collection.id)
    if (added) closeAddDialog()
  }

  async function createAndAddDialogCollection(name) {
    const character = addDialogCharacter.value
    if (!character) return
    const detail = await createCollection(name, { open: false })
    if (!detail) return
    const added = await addCharacter(character, detail.id)
    if (added) closeAddDialog()
  }

  return {
    collections,
    selectedCollection,
    saving,
    startSaving,
    addDialogCharacter,
    addDialogCollections,
    addDialogLoading,
    loadCollections,
    openCollection,
    createCollection,
    toggleSelectedCollectionStarted,
    deleteSelectedCollection,
    addCharacter,
    removeCharacter,
    openAddDialog,
    closeAddDialog,
    addDialogCharacterTo,
    createAndAddDialogCollection,
  }
}
