import { computed, ref } from 'vue'
import {
  createArc,
  deleteArc as removeArc,
  getArc,
  listArcs,
  setArcStarted,
  setArcComics,
  updateArc,
} from '@/api/client.js'
import { arcComicsPayload, arcFormFromDetail, arcPayload, emptyArc } from '@/features/arcs/model.js'

export function useArcs({ activeView, viewMode, error, saving, loadComics, loadPagedList }) {
  const arcs = ref([])
  const selectedArc = ref(null)
  const quickSavingArcID = ref(null)
  const startSaving = ref(false)
  const arcForm = ref(emptyArc())

  const visibleArcs = computed(() => arcs.value)

  function arcProgress(arcComics) {
    if (arcComics.length === 0) return 0
    const readCount = arcComics.filter((comic) => comic.read).length
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
    arcs.value = arcs.value.map((arc) => {
      return arc.id === detail.id ? { ...arc, favorite: detail.favorite } : arc
    })

    if (selectedArc.value?.id === detail.id) {
      selectedArc.value = { ...selectedArc.value, favorite: detail.favorite }
    }
    if (arcForm.value?.id === detail.id) {
      arcForm.value = { ...arcForm.value, favorite: detail.favorite }
    }
  }

  async function toggleSelectedArcStarted() {
    if (!selectedArc.value?.id || startSaving.value) return
    startSaving.value = true
    error.value = ''
    try {
      const detail = await setArcStarted(selectedArc.value.id, !selectedArc.value.startedAt)
      selectedArc.value = detail
      arcs.value = arcs.value.map((arc) =>
        arc.id === detail.id ? { ...arc, startedAt: detail.startedAt || null } : arc,
      )
    } catch (err) {
      error.value = err.message
    } finally {
      startSaving.value = false
    }
  }

  function newArc() {
    error.value = ''
    activeView.value = 'arcs'
    viewMode.value = 'edit'
    selectedArc.value = null
    arcForm.value = emptyArc()
    loadComics().catch((err) => {
      error.value = err.message
    })
  }

  function editArc() {
    if (!selectedArc.value) return
    error.value = ''
    arcForm.value = arcFormFromDetail(selectedArc.value)
    viewMode.value = 'edit'
    loadComics().catch((err) => {
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
    startSaving,
    arcForm,
    visibleArcs,
    arcProgress,
    openArc,
    toggleArcFavorite,
    toggleSelectedArcStarted,
    newArc,
    editArc,
    saveArc,
    deleteArc,
    loadArcs,
  }
}
