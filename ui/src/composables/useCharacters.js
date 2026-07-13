import { computed, ref } from 'vue'
import {
  deleteCharacter as removeCharacter,
  getCharacter,
  importMetronCharacterAppearances,
  listCharacters,
  setCharacterStarted,
  updateCharacterFavorite,
} from '@/api/client.js'
import { formatProgress } from '@/domain/readingOrders.js'

export function useCharacters({
  activeView,
  viewMode,
  error,
  loadPagedList,
  metronImportJobs,
  trackMetronImportJob,
}) {
  const characters = ref([])
  const selectedCharacter = ref(null)
  const quickSavingCharacterID = ref(null)
  const startSaving = ref(false)
  const deleting = ref(false)

  const visibleCharacters = computed(() => characters.value)
  const favoriteVisibleCharacters = computed(() =>
    characters.value.filter((character) => character.favorite),
  )
  const remainingVisibleCharacters = computed(() =>
    characters.value.filter((character) => !character.favorite),
  )
  const characterBrowseSections = computed(() => {
    if (!favoriteVisibleCharacters.value.length) {
      return [{ key: 'all', title: 'All Characters', characters: characters.value }]
    }
    return [
      { key: 'favorites', title: 'Favorites', characters: favoriteVisibleCharacters.value },
      { key: 'other', title: 'Other Characters', characters: remainingVisibleCharacters.value },
    ].filter((section) => section.characters.length)
  })
  async function openCharacter(character) {
    error.value = ''
    activeView.value = 'characters'
    selectedCharacter.value = null
    viewMode.value = 'detail'
    const detail = await getCharacter(character.id)
    selectedCharacter.value = detail
  }

  async function toggleCharacterFavorite(character) {
    if (!character?.id || quickSavingCharacterID.value) return

    quickSavingCharacterID.value = character.id
    error.value = ''

    try {
      const detail = await updateCharacterFavorite(character.id, !character.favorite)
      applyCharacterFavoriteState(detail)
    } catch (err) {
      error.value = err.message
    } finally {
      quickSavingCharacterID.value = null
    }
  }

  function applyCharacterFavoriteState(detail) {
    characters.value = characters.value.map((character) => {
      return character.id === detail.id ? { ...character, favorite: detail.favorite } : character
    })

    if (selectedCharacter.value?.id === detail.id) {
      selectedCharacter.value = { ...selectedCharacter.value, favorite: detail.favorite }
    }
  }

  async function toggleSelectedCharacterStarted() {
    if (!selectedCharacter.value?.id || startSaving.value) return
    startSaving.value = true
    error.value = ''
    try {
      const detail = await setCharacterStarted(
        selectedCharacter.value.id,
        !selectedCharacter.value.startedAt,
      )
      selectedCharacter.value = detail
      characters.value = characters.value.map((character) =>
        character.id === detail.id
          ? { ...character, startedAt: detail.startedAt || null }
          : character,
      )
    } catch (err) {
      error.value = err.message
    } finally {
      startSaving.value = false
    }
  }

  function characterProgress(character) {
    return formatProgress(character?.progress ?? 0)
  }

  async function importSelectedCharacterAppearances() {
    if (
      !selectedCharacter.value?.metronCharacterId ||
      characterImportRunning(selectedCharacter.value)
    )
      return

    error.value = ''
    try {
      const { data: job } = await importMetronCharacterAppearances(
        selectedCharacter.value.metronCharacterId,
      )
      trackMetronImportJob({ ...job, displayName: selectedCharacter.value.name })
    } catch (err) {
      error.value = err.message
    }
  }

  function characterImportRunning(character) {
    if (!character?.metronCharacterId) return false
    return metronImportJobs.value.some((job) => {
      return (
        job.type === 'character' &&
        job.metronId === character.metronCharacterId &&
        (job.status === 'queued' || job.status === 'running' || job.status === 'canceling')
      )
    })
  }

  async function refreshSelectedCharacterDetail() {
    if (selectedCharacter.value?.id) {
      selectedCharacter.value = await getCharacter(selectedCharacter.value.id)
    }
  }

  async function deleteSelectedCharacter() {
    if (!selectedCharacter.value?.id || deleting.value) return
    if (!confirm(`Delete ${selectedCharacter.value.name}? This cannot be undone.`)) return

    deleting.value = true
    error.value = ''
    try {
      await removeCharacter(selectedCharacter.value.id)
      selectedCharacter.value = null
      await loadCharacters({ force: true })
      viewMode.value = 'browse'
    } catch (err) {
      error.value = err.message
    } finally {
      deleting.value = false
    }
  }

  async function loadCharacters(options = {}) {
    await loadPagedList('characters', characters, listCharacters, options)
  }

  return {
    characters,
    selectedCharacter,
    quickSavingCharacterID,
    startSaving,
    deleting,
    visibleCharacters,
    characterBrowseSections,
    openCharacter,
    toggleCharacterFavorite,
    toggleSelectedCharacterStarted,
    characterProgress,
    importSelectedCharacterAppearances,
    characterImportRunning,
    refreshSelectedCharacterDetail,
    deleteSelectedCharacter,
    loadCharacters,
  }
}
