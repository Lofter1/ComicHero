import { computed, ref } from 'vue'
import {
  copyReadingOrder,
  createReadingOrder,
  deleteReadingOrder as removeReadingOrder,
  exportReadingOrderCBL,
  getReadingOrder,
  importReadingOrderCBL,
  setReadingOrderComics,
  updateReadingOrder,
  listReadingOrders,
} from '@/api/client.js'
import {
  emptyReadingOrder,
  readingOrderComicsPayload,
  readingOrderFormFromDetail,
  readingOrderPayload,
} from '@/domain/readingOrders.js'

export function useReadingOrders({
  activeView,
  viewMode,
  error,
  saving,
  loadComics,
  loadPagedList,
}) {
  const readingOrders = ref([])
  const selectedOrder = ref(null)
  const quickSavingOrderID = ref(null)
  const cblImporting = ref(false)
  const orderForm = ref(emptyReadingOrder())

  const visibleOrders = computed(() => readingOrders.value)
  const favoriteVisibleOrders = computed(() =>
    readingOrders.value.filter((order) => order.favorite),
  )
  const remainingVisibleOrders = computed(() =>
    readingOrders.value.filter((order) => !order.favorite),
  )
  const readingOrderBrowseSections = computed(() => {
    if (!favoriteVisibleOrders.value.length) {
      return [{ key: 'all', title: 'All Orders', orders: readingOrders.value }]
    }
    return [
      { key: 'favorites', title: 'Favorites', orders: favoriteVisibleOrders.value },
      { key: 'other', title: 'Other Orders', orders: remainingVisibleOrders.value },
    ].filter((section) => section.orders.length)
  })
  function readingOrderProgress(orderComics) {
    if (orderComics.length === 0) return 0
    const readCount = orderComics.filter((comic) => comic.read).length
    return readCount / orderComics.length
  }

  async function openReadingOrder(order) {
    error.value = ''
    activeView.value = 'readingOrders'
    selectedOrder.value = null
    viewMode.value = 'detail'
    const detail = await getReadingOrder(order.id)
    selectedOrder.value = detail
    orderForm.value = readingOrderFormFromDetail(detail)
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
    await Promise.all([loadComics(), loadReadingOrders()])
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

  async function importReadingOrderCBLFile(file) {
    if (!file) return null

    cblImporting.value = true
    saving.value = true
    error.value = ''

    try {
      const result = await importReadingOrderCBL({
        filename: file.name || '',
        content: await file.text(),
      })
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
    cblImporting,
    orderForm,
    visibleOrders,
    readingOrderBrowseSections,
    readingOrderProgress,
    openReadingOrder,
    refreshSelectedReadingOrderDetail,
    toggleReadingOrderFavorite,
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
