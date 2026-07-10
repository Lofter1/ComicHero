import { ref } from 'vue'
import {
  createComic,
  deleteComic as removeComic,
  getComic,
  listComics,
  searchMetronComics,
  updateComic,
  updateComicFromMetron,
  updateComicReadStatus,
} from '@/api/client.js'
import { comicPayload, emptyComic } from '@/domain/comics.js'

export function useComics({
  activeView,
  viewMode,
  error,
  saving,
  loadPagedList,
  refreshActiveLibraryData,
  readingOrderProgress,
  selectedOrder,
  selectedArc,
  selectedCharacter,
  selectedSeries,
  collectionProgress = readingOrderProgress,
}) {
  const comics = ref([])
  const selectedComic = ref(null)
  const quickSavingComicID = ref(null)
  const comicForm = ref(emptyComic())
  const metronMetadataOpen = ref(false)
  const metronMetadataSearching = ref(false)
  const metronMetadataApplyingID = ref(null)
  const metronMetadataStatus = ref('')
  const metronMetadataResults = ref([])

  async function loadComics(options = {}) {
    await loadPagedList('comics', comics, listComics, options)
  }

  async function openComic(comic) {
    error.value = ''
    activeView.value = 'comics'
    selectedComic.value = null
    viewMode.value = 'detail'
    resetMetronMetadata()
    const detail = await getComic(comic.id)
    selectedComic.value = detail
    comicForm.value = { ...detail }
  }

  async function openOrderComic(comic) {
    if (!comic?.id) return
    await openComic(comic)
  }

  function newComic() {
    error.value = ''
    activeView.value = 'comics'
    viewMode.value = 'edit'
    selectedComic.value = null
    comicForm.value = emptyComic()
  }

  function editComic() {
    if (!selectedComic.value) return
    error.value = ''
    comicForm.value = { ...selectedComic.value }
    viewMode.value = 'edit'
  }

  async function saveComic() {
    saving.value = true
    error.value = ''

    try {
      const payload = comicPayload(comicForm.value)
      const detail = comicForm.value.id
        ? await updateComic(comicForm.value.id, payload)
        : await createComic(payload)

      selectedComic.value = detail
      comicForm.value = { ...detail }
      await loadComics({ force: true })
      viewMode.value = 'detail'
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function deleteComic() {
    if (!comicForm.value.id || !confirm(`Delete ${comicForm.value.title}?`)) return

    saving.value = true
    error.value = ''

    try {
      await removeComic(comicForm.value.id)
      selectedComic.value = null
      comicForm.value = emptyComic()
      await loadComics({ force: true })
      viewMode.value = 'browse'
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function toggleComicRead(comic) {
    if (!comic?.id || quickSavingComicID.value) return

    quickSavingComicID.value = comic.id
    error.value = ''

    try {
      const detail = await updateComicReadStatus(comic.id, { read: !comic.read })
      applyComicReadState(detail)
      refreshActiveLibraryData().catch((err) => {
        error.value = err.message
      })
    } catch (err) {
      error.value = err.message
    } finally {
      quickSavingComicID.value = null
    }
  }

  async function toggleComicSkipped(comic) {
    if (!comic?.id || quickSavingComicID.value) return

    quickSavingComicID.value = comic.id
    error.value = ''

    try {
      const detail = await updateComicReadStatus(comic.id, { skipped: !comic.skipped })
      applyComicReadState(detail)
      refreshActiveLibraryData().catch((err) => {
        error.value = err.message
      })
    } catch (err) {
      error.value = err.message
    } finally {
      quickSavingComicID.value = null
    }
  }

  function applyComicReadState(detail) {
    comics.value = comics.value.map((comic) =>
      comic.id === detail.id ? { ...comic, read: detail.read, skipped: detail.skipped } : comic,
    )

    if (selectedComic.value?.id === detail.id) {
      selectedComic.value = { ...selectedComic.value, ...detail }
    }
    if (comicForm.value?.id === detail.id) {
      comicForm.value = { ...comicForm.value, read: detail.read, skipped: detail.skipped }
    }
    if (selectedOrder.value) {
      const orderComics = selectedOrder.value.comics.map((comic) => {
        return comic.id === detail.id
          ? { ...comic, read: detail.read, skipped: detail.skipped }
          : comic
      })
      selectedOrder.value = {
        ...selectedOrder.value,
        comics: orderComics,
        progress: readingOrderProgress(orderComics),
      }
    }
    if (selectedArc.value) {
      const arcComics = selectedArc.value.comics.map((comic) => {
        return comic.id === detail.id
          ? { ...comic, read: detail.read, skipped: detail.skipped }
          : comic
      })
      selectedArc.value = {
        ...selectedArc.value,
        comics: arcComics,
        progress: collectionProgress(arcComics),
      }
    }
    if (selectedCharacter.value) {
      const characterComics = selectedCharacter.value.comics.map((comic) => {
        return comic.id === detail.id
          ? { ...comic, read: detail.read, skipped: detail.skipped }
          : comic
      })
      selectedCharacter.value = {
        ...selectedCharacter.value,
        comics: characterComics,
        progress: readingOrderProgress(characterComics),
      }
    }
    if (selectedSeries.value?.comics) {
      const seriesComics = selectedSeries.value.comics.map((comic) => {
        return comic.id === detail.id
          ? { ...comic, read: detail.read, skipped: detail.skipped }
          : comic
      })
      selectedSeries.value = {
        ...selectedSeries.value,
        comics: seriesComics,
        readCount: seriesComics.filter((comic) => comic.read).length,
        progress: readingOrderProgress(seriesComics),
      }
    }
  }

  function applyComicDetailState(detail) {
    comics.value = comics.value.map((comic) =>
      comic.id === detail.id ? { ...comic, ...detail } : comic,
    )
    selectedComic.value = detail
    comicForm.value = { ...detail }

    if (selectedOrder.value) {
      const orderComics = selectedOrder.value.comics.map((comic) => {
        return comic.id === detail.id ? { ...comic, ...detail, comment: comic.comment } : comic
      })
      selectedOrder.value = {
        ...selectedOrder.value,
        comics: orderComics,
        progress: readingOrderProgress(orderComics),
      }
    }
    if (selectedArc.value) {
      const arcComics = selectedArc.value.comics.map((comic) => {
        return comic.id === detail.id ? { ...comic, ...detail, comment: comic.comment } : comic
      })
      selectedArc.value = {
        ...selectedArc.value,
        comics: arcComics,
        progress: collectionProgress(arcComics),
      }
    }
    if (selectedCharacter.value) {
      const characterComics = selectedCharacter.value.comics.map((comic) => {
        return comic.id === detail.id ? { ...comic, ...detail } : comic
      })
      selectedCharacter.value = {
        ...selectedCharacter.value,
        comics: characterComics,
        progress: readingOrderProgress(characterComics),
      }
    }
    if (selectedSeries.value?.comics) {
      const seriesComics = selectedSeries.value.comics.map((comic) => {
        return comic.id === detail.id ? { ...comic, ...detail } : comic
      })
      selectedSeries.value = {
        ...selectedSeries.value,
        comics: seriesComics,
        readCount: seriesComics.filter((comic) => comic.read).length,
        progress: readingOrderProgress(seriesComics),
      }
    }
  }

  function resetMetronMetadata() {
    metronMetadataOpen.value = false
    metronMetadataSearching.value = false
    metronMetadataApplyingID.value = null
    metronMetadataStatus.value = ''
    metronMetadataResults.value = []
  }

  async function searchSelectedComicMetron() {
    if (!selectedComic.value) return

    metronMetadataOpen.value = true
    metronMetadataSearching.value = true
    metronMetadataStatus.value = ''
    error.value = ''

    try {
      const { data } = await searchMetronComics({
        q: selectedComic.value.title,
        series: selectedComic.value.series,
        issue: selectedComic.value.issue,
      })
      metronMetadataResults.value = Array.isArray(data) ? data : []
      metronMetadataStatus.value = metronMetadataResults.value.length
        ? 'Choose the Metron issue that matches this comic.'
        : 'No Metron matches found.'
    } catch (err) {
      error.value = err.message
    } finally {
      metronMetadataSearching.value = false
    }
  }

  async function applyMetronMetadata(metronIssueID) {
    if (!selectedComic.value?.id || !metronIssueID) return

    metronMetadataApplyingID.value = metronIssueID
    metronMetadataStatus.value = 'Updating comic metadata from Metron...'
    error.value = ''

    try {
      const { data } = await updateComicFromMetron(selectedComic.value.id, metronIssueID)
      applyComicDetailState(data)
      await loadComics({ force: true })
      metronMetadataStatus.value = 'Metadata updated from Metron.'
      metronMetadataOpen.value = false
      metronMetadataResults.value = []
    } catch (err) {
      error.value = err.message
    } finally {
      metronMetadataApplyingID.value = null
    }
  }

  return {
    comics,
    selectedComic,
    quickSavingComicID,
    comicForm,
    metronMetadataOpen,
    metronMetadataSearching,
    metronMetadataApplyingID,
    metronMetadataStatus,
    metronMetadataResults,
    loadComics,
    openComic,
    openOrderComic,
    newComic,
    editComic,
    saveComic,
    deleteComic,
    toggleComicRead,
    toggleComicSkipped,
    resetMetronMetadata,
    searchSelectedComicMetron,
    applyMetronMetadata,
  }
}
