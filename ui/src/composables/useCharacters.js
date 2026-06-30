import { computed, ref } from 'vue'
import {
  getCharacter,
  importMetronCharacterAppearances,
  listCharacters,
  updateCharacterFavorite,
} from '@/api/client.js'
import { formatProgress } from '@/domain/readingOrders.js'

export function useCharacters({ activeView, viewMode, error, loadPagedList, metronImportJobs, trackMetronImportJob }) {
  const characters = ref([])
  const selectedCharacter = ref(null)
  const quickSavingCharacterID = ref(null)

  const visibleCharacters = computed(() => characters.value)
  const favoriteVisibleCharacters = computed(() => characters.value.filter(character => character.favorite))
  const remainingVisibleCharacters = computed(() => characters.value.filter(character => !character.favorite))
  const characterBrowseSections = computed(() => {
    if (!favoriteVisibleCharacters.value.length) {
      return [{ key: 'all', title: 'All Characters', characters: characters.value }]
    }
    return [
      { key: 'favorites', title: 'Favorites', characters: favoriteVisibleCharacters.value },
      { key: 'other', title: 'Other Characters', characters: remainingVisibleCharacters.value },
    ].filter(section => section.characters.length)
  })
  const favoriteCharacterCount = computed(() => characters.value.filter(character => character.favorite).length)
  const currentCharacterIndex = computed(() => {
    return visibleCharacters.value.findIndex(character => character.id === selectedCharacter.value?.id)
  })

  async function openCharacter(character) {
    error.value = ''
    activeView.value = 'characters'
    selectedCharacter.value = null
    viewMode.value = 'detail'
    const detail = await getCharacter(character.id)
    selectedCharacter.value = detail
  }

  async function openAdjacentCharacter(offset) {
    const nextCharacter = visibleCharacters.value[currentCharacterIndex.value + offset]
    if (nextCharacter) {
      await openCharacter(nextCharacter)
    }
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
    characters.value = characters.value.map(character => {
      return character.id === detail.id ? { ...character, favorite: detail.favorite } : character
    })

    if (selectedCharacter.value?.id === detail.id) {
      selectedCharacter.value = { ...selectedCharacter.value, favorite: detail.favorite }
    }
  }

  function characterProgress(character) {
    return formatProgress(character?.progress ?? 0)
  }

  async function importSelectedCharacterAppearances() {
    if (!selectedCharacter.value?.metronCharacterId || characterImportRunning(selectedCharacter.value)) return

    error.value = ''
    try {
      const { data: job } = await importMetronCharacterAppearances(selectedCharacter.value.metronCharacterId)
      trackMetronImportJob({ ...job, displayName: selectedCharacter.value.name })
    } catch (err) {
      error.value = err.message
    }
  }

  function characterImportRunning(character) {
    if (!character?.metronCharacterId) return false
    return metronImportJobs.value.some(job => {
      return job.type === 'character'
        && job.metronId === character.metronCharacterId
        && (job.status === 'queued' || job.status === 'running' || job.status === 'canceling')
    })
  }

  async function refreshSelectedCharacterDetail() {
    if (selectedCharacter.value?.id) {
      selectedCharacter.value = await getCharacter(selectedCharacter.value.id)
    }
  }

  async function loadCharacters(options = {}) {
    await loadPagedList('characters', characters, listCharacters, { all: true, ...options })
  }

  return {
    characters,
    selectedCharacter,
    quickSavingCharacterID,
    visibleCharacters,
    characterBrowseSections,
    favoriteCharacterCount,
    currentCharacterIndex,
    openCharacter,
    openAdjacentCharacter,
    toggleCharacterFavorite,
    applyCharacterFavoriteState,
    characterProgress,
    importSelectedCharacterAppearances,
    characterImportRunning,
    refreshSelectedCharacterDetail,
    loadCharacters,
  }
}
