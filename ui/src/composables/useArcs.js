import { computed, ref } from 'vue'
import {
  createArc,
  deleteArc as removeArc,
  getArc,
  listArcs,
  setArcComics,
  updateArc,
} from '@/api/client.js'
import {
  arcComicsPayload,
  arcFormFromDetail,
  arcPayload,
  emptyArc,
} from '@/domain/arcs.js'

export function useArcs({ activeView, viewMode, error, saving, loadComics, loadPagedList }) {
  const arcs = ref([])
  const selectedArc = ref(null)
  const quickSavingArcID = ref(null)
  const arcForm = ref(emptyArc())

  const visibleArcs = computed(() => arcs.value)
  const favoriteVisibleArcs = computed(() => arcs.value.filter(arc => arc.favorite))
  const remainingVisibleArcs = computed(() => arcs.value.filter(arc => !arc.favorite))
  const arcBrowseSections = computed(() => {
    if (!favoriteVisibleArcs.value.length) {
      return [{ key: 'all', title: 'All Arcs', arcs: arcs.value }]
    }
    return [
      { key: 'favorites', title: 'Favorites', arcs: favoriteVisibleArcs.value },
      { key: 'other', title: 'Other Arcs', arcs: remainingVisibleArcs.value },
    ].filter(section => section.arcs.length)
  })
  const favoriteArcCount = computed(() => arcs.value.filter(arc => arc.favorite).length)

  function arcProgress(arcComics) {
    if (arcComics.length === 0) return 0
    const readCount = arcComics.filter(comic => comic.read).length
    return readCount / arcComics.length
  }

  async function openArc(arc) {
    if (!arc?.id) return
    error.value = ''
    activeView.value = 'arcs'
    selectedArc.value = null
    viewMode.value = 'detail'
    const detail = await getArc(arc.id)
    selectedArc.value = detail
    arcForm.value = arcFormFromDetail(detail)
  }

  async function toggleArcFavorite(arc) {
    if (!arc?.id || quickSavingArcID.value) return

    quickSavingArcID.value = arc.id
    error.value = ''

    try {
      const detail = await updateArc(arc.id, {
        name: arc.name,
        description: arc.description,
        favorite: !arc.favorite,
      })
      applyArcFavoriteState(detail)
    } catch (err) {
      error.value = err.message
    } finally {
      quickSavingArcID.value = null
    }
  }

  function applyArcFavoriteState(detail) {
    arcs.value = arcs.value.map(arc => {
      return arc.id === detail.id ? { ...arc, favorite: detail.favorite } : arc
    })

    if (selectedArc.value?.id === detail.id) {
      selectedArc.value = { ...selectedArc.value, favorite: detail.favorite }
    }
    if (arcForm.value?.id === detail.id) {
      arcForm.value = { ...arcForm.value, favorite: detail.favorite }
    }
  }

  function newArc() {
    error.value = ''
    activeView.value = 'arcs'
    viewMode.value = 'edit'
    selectedArc.value = null
    arcForm.value = emptyArc()
    loadComics().catch(err => {
      error.value = err.message
    })
  }

  function editArc() {
    if (!selectedArc.value) return
    error.value = ''
    arcForm.value = arcFormFromDetail(selectedArc.value)
    viewMode.value = 'edit'
    loadComics().catch(err => {
      error.value = err.message
    })
  }

  async function saveArc() {
    saving.value = true
    error.value = ''

    try {
      const payload = arcPayload(arcForm.value)
      const detail = arcForm.value.id
        ? await updateArc(arcForm.value.id, payload)
        : await createArc(payload)

      selectedArc.value = await setArcComics(detail.id, arcComicsPayload(arcForm.value))
      arcForm.value = arcFormFromDetail(selectedArc.value)
      await loadArcs({ force: true })
      viewMode.value = 'detail'
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function deleteArc() {
    if (!arcForm.value.id || !confirm(`Delete ${arcForm.value.name}?`)) return

    saving.value = true
    error.value = ''

    try {
      await removeArc(arcForm.value.id)
      selectedArc.value = null
      arcForm.value = emptyArc()
      await loadArcs({ force: true })
      viewMode.value = 'browse'
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function loadArcs(options = {}) {
    await loadPagedList('arcs', arcs, listArcs, options)
  }

  return {
    arcs,
    selectedArc,
    quickSavingArcID,
    arcForm,
    visibleArcs,
    arcBrowseSections,
    favoriteArcCount,
    arcProgress,
    openArc,
    toggleArcFavorite,
    applyArcFavoriteState,
    newArc,
    editArc,
    saveArc,
    deleteArc,
    loadArcs,
  }
}
