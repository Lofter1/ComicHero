import { computed, ref } from 'vue'
import {
  copyReadingOrder,
  createReadingOrder,
  deleteReadingOrder as removeReadingOrder,
  exportReadingOrderCBL,
  getReadingOrder,
  importReadingOrderCBL,
  rateReadingOrder,
  startReadingOrder,
  stopReadingOrder,
  setReadingOrderComics,
  updateReadingOrder,
  listReadingOrders,
} from '@/api/client.js'
import {
  emptyReadingOrder,
  readingOrderComicsPayload,
  readingOrderFormFromDetail,
  readingOrderPayload,
} from '@/features/reading-orders/model.js'

export function useReadingOrders({ activeView, viewMode, loading, error, saving, loadPagedList }) {
  const readingOrders = ref([])
  const selectedOrder = ref(null)
  const quickSavingOrderID = ref(null)
  const ratingSaving = ref(false)
  const startSaving = ref(false)
  const cblImporting = ref(false)
  const orderForm = ref(emptyReadingOrder())

  const visibleOrders = computed(() => readingOrders.value)
  function readingOrderProgress(orderComics) {
    if (orderComics.length === 0) return 0
    const readCount = orderComics.filter((comic) => comic.read).length
    return readCount / orderComics.length
  }

  async function openReadingOrder(order) {
    loading.value = true
    try {
      error.value = ''
      activeView.value = 'readingOrders'
      selectedOrder.value = null
      viewMode.value = 'detail'
      const detail = await getReadingOrder(order.id)
      selectedOrder.value = detail
      orderForm.value = readingOrderFormFromDetail(detail)
    } finally {
      loading.value = false
    }
  }

  async function refreshSelectedReadingOrderDetail() {
    if (selectedOrder.value?.id) {
      const detail = await getReadingOrder(selectedOrder.value.id)
      selectedOrder.value = detail
      orderForm.value = readingOrderFormFromDetail(detail)
    }
  }

  async function toggleReadingOrderFavorite(order) {
    if (!order?.id || order.canEdit === false || quickSavingOrderID.value) return

    quickSavingOrderID.value = order.id
    error.value = ''

    try {
      const detail = await updateReadingOrder(order.id, {
        name: order.name,
        description: order.description,
        favorite: !order.favorite,
      })
      applyReadingOrderFavoriteState(detail)
    } catch (err) {
      error.value = err.message
    } finally {
      quickSavingOrderID.value = null
    }
  }

  async function startSelectedReadingOrder() {
    if (!selectedOrder.value?.id || selectedOrder.value.startedAt || startSaving.value) return

    startSaving.value = true
    error.value = ''
    try {
      const detail = await startReadingOrder(selectedOrder.value.id)
      selectedOrder.value = detail
      readingOrders.value = readingOrders.value.map((order) =>
        order.id === detail.id ? { ...order, startedAt: detail.startedAt } : order,
      )
    } catch (err) {
      error.value = err.message
    } finally {
      startSaving.value = false
    }
  }

  async function stopSelectedReadingOrder() {
    if (!selectedOrder.value?.id || !selectedOrder.value.startedAt || startSaving.value) return

    startSaving.value = true
    error.value = ''
    try {
      const detail = await stopReadingOrder(selectedOrder.value.id)
      selectedOrder.value = detail
      readingOrders.value = readingOrders.value.map((order) =>
        order.id === detail.id ? { ...order, startedAt: null } : order,
      )
    } catch (err) {
      error.value = err.message
    } finally {
      startSaving.value = false
    }
  }

  function applyReadingOrderFavoriteState(detail) {
    readingOrders.value = readingOrders.value.map((order) => {
      return order.id === detail.id ? { ...order, favorite: detail.favorite } : order
    })

    if (selectedOrder.value?.id === detail.id) {
      selectedOrder.value = { ...selectedOrder.value, favorite: detail.favorite }
    }
    if (orderForm.value?.id === detail.id) {
      orderForm.value = { ...orderForm.value, favorite: detail.favorite }
    }
  }

  function applyReadingOrderRatingState(detail) {
    readingOrders.value = readingOrders.value.map((order) => {
      return order.id === detail.id
        ? {
            ...order,
            rating: detail.rating,
            ratingCount: detail.ratingCount,
            myRating: detail.myRating,
          }
        : order
    })

    if (selectedOrder.value?.id === detail.id) {
      selectedOrder.value = {
        ...selectedOrder.value,
        rating: detail.rating,
        ratingCount: detail.ratingCount,
        myRating: detail.myRating,
      }
    }
    if (orderForm.value?.id === detail.id) {
      orderForm.value = readingOrderFormFromDetail({
        ...selectedOrder.value,
        ...detail,
      })
    }
  }

  async function rateSelectedReadingOrder(rating) {
    if (!selectedOrder.value?.id || ratingSaving.value) return

    ratingSaving.value = true
    error.value = ''

    try {
      const detail = await rateReadingOrder(selectedOrder.value.id, rating)
      applyReadingOrderRatingState(detail)
    } catch (err) {
      error.value = err.message
    } finally {
      ratingSaving.value = false
    }
  }

  function newReadingOrder() {
    error.value = ''
    activeView.value = 'readingOrders'
    viewMode.value = 'edit'
    selectedOrder.value = null
    orderForm.value = emptyReadingOrder()
    loadReadingOrderEditorOptions().catch((err) => {
      error.value = err.message
    })
  }

  function editReadingOrder() {
    if (!selectedOrder.value) return
    if (selectedOrder.value.canEdit === false) {
      error.value = 'Only the author or an admin can edit this reading order.'
      return
    }
    error.value = ''
    orderForm.value = readingOrderFormFromDetail(selectedOrder.value)
    viewMode.value = 'edit'
    loadReadingOrderEditorOptions().catch((err) => {
      error.value = err.message
    })
  }

  async function loadReadingOrderEditorOptions() {
    await loadReadingOrders()
  }

  async function saveReadingOrder() {
    if (orderForm.value.id && selectedOrder.value?.canEdit === false) {
      error.value = 'Only the author or an admin can edit this reading order.'
      return
    }
    saving.value = true
    error.value = ''

    try {
      const payload = readingOrderPayload(orderForm.value)
      const detail = orderForm.value.id
        ? await updateReadingOrder(orderForm.value.id, payload)
        : await createReadingOrder(payload)

      selectedOrder.value = await setReadingOrderComics(
        detail.id,
        readingOrderComicsPayload(orderForm.value),
      )
      orderForm.value = readingOrderFormFromDetail(selectedOrder.value)
      await loadReadingOrders({ force: true })
      viewMode.value = 'detail'
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function deleteReadingOrder() {
    if (selectedOrder.value?.canEdit === false) {
      error.value = 'Only the author or an admin can delete this reading order.'
      return
    }
    if (!orderForm.value.id || !confirm(`Delete ${orderForm.value.name}?`)) return

    saving.value = true
    error.value = ''

    try {
      await removeReadingOrder(orderForm.value.id)
      selectedOrder.value = null
      orderForm.value = emptyReadingOrder()
      await loadReadingOrders({ force: true })
      viewMode.value = 'browse'
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function copySelectedReadingOrder() {
    if (!selectedOrder.value?.id) return

    saving.value = true
    error.value = ''

    try {
      const detail = await copyReadingOrder(selectedOrder.value.id)
      selectedOrder.value = detail
      orderForm.value = readingOrderFormFromDetail(detail)
      await loadReadingOrders({ force: true })
      activeView.value = 'readingOrders'
      viewMode.value = 'detail'
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  async function importReadingOrderCBLFile(selectedFiles) {
    const files = Array.isArray(selectedFiles) ? selectedFiles : [selectedFiles].filter(Boolean)
    if (!files.length) return null

    cblImporting.value = true
    saving.value = true
    error.value = ''

    try {
      const parts = await Promise.all(
        files.map(async (file) => ({
          filename: file.name || '',
          content: await file.text(),
        })),
      )
      const result = await importReadingOrderCBL(parts.length === 1 ? parts[0] : { parts })
      selectedOrder.value = result.readingOrder
      orderForm.value = readingOrderFormFromDetail(result.readingOrder)
      await loadReadingOrders({ force: true })
      activeView.value = 'readingOrders'
      viewMode.value = 'detail'
      return result
    } catch (err) {
      error.value = err.message
      return null
    } finally {
      cblImporting.value = false
      saving.value = false
    }
  }

  async function exportSelectedReadingOrderCBL() {
    if (!selectedOrder.value?.id) return

    saving.value = true
    error.value = ''

    try {
      const result = await exportReadingOrderCBL(selectedOrder.value.id)
      downloadTextFile(result.filename, result.content, 'application/xml;charset=utf-8')
    } catch (err) {
      error.value = err.message
    } finally {
      saving.value = false
    }
  }

  function downloadTextFile(filename, content, type) {
    if (typeof document === 'undefined' || typeof URL === 'undefined') return

    const blob = new Blob([content], { type })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename || 'reading-order.cbl'
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  }

  async function loadReadingOrders(options = {}) {
    await loadPagedList('readingOrders', readingOrders, listReadingOrders, options)
  }

  return {
    readingOrders,
    selectedOrder,
    quickSavingOrderID,
    ratingSaving,
    startSaving,
    cblImporting,
    orderForm,
    visibleOrders,
    readingOrderProgress,
    openReadingOrder,
    refreshSelectedReadingOrderDetail,
    toggleReadingOrderFavorite,
    rateSelectedReadingOrder,
    startSelectedReadingOrder,
    stopSelectedReadingOrder,
    newReadingOrder,
    editReadingOrder,
    saveReadingOrder,
    deleteReadingOrder,
    copySelectedReadingOrder,
    importReadingOrderCBLFile,
    exportSelectedReadingOrderCBL,
    loadReadingOrders,
  }
}
