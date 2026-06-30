import { computed, ref } from 'vue'
import {
    createReadingOrder,
    deleteReadingOrder as removeReadingOrder,
    getReadingOrder,
    setReadingOrderComics,
    updateReadingOrder,
    listReadingOrders
} from '@/api/client.js'
import {
    emptyReadingOrder,
    readingOrderComicsPayload,
    readingOrderFormFromDetail,
    readingOrderPayload,
} from '@/domain/readingOrders.js'


export function useReadingOrders({ activeView, viewMode, error, saving, loadComics, loadPagedList }) {
    const readingOrders = ref([])
    const selectedOrder = ref(null)
    const quickSavingOrderID = ref(null)
    const orderForm = ref(emptyReadingOrder())


    const visibleOrders = computed(() => readingOrders.value)
    const favoriteVisibleOrders = computed(() => readingOrders.value.filter(order => order.favorite))
    const remainingVisibleOrders = computed(() => readingOrders.value.filter(order => !order.favorite))
    const readingOrderBrowseSections = computed(() => {
        if (!favoriteVisibleOrders.value.length) {
            return [{ key: 'all', title: 'All Orders', orders: readingOrders.value }]
        }
        return [
            { key: 'favorites', title: 'Favorites', orders: favoriteVisibleOrders.value },
            { key: 'other', title: 'Other Orders', orders: remainingVisibleOrders.value },
        ].filter(section => section.orders.length)
    })
    const favoriteOrderCount = computed(() => readingOrders.value.filter(order => order.favorite).length)
    const currentOrderIndex = computed(() => {
        return visibleOrders.value.findIndex(order => order.id === selectedOrder.value?.id)
    })

    function readingOrderProgress(orderComics) {
        if (orderComics.length === 0) return 0
        const readCount = orderComics.filter(comic => comic.read).length
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

    async function openAdjacentReadingOrder(offset) {
        const nextOrder = visibleOrders.value[currentOrderIndex.value + offset]
        if (nextOrder) {
            await openReadingOrder(nextOrder)
        }
    }

    async function toggleReadingOrderFavorite(order) {
        if (!order?.id || quickSavingOrderID.value) return

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
        readingOrders.value = readingOrders.value.map(order => {
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
        loadComics().catch(err => {
            error.value = err.message
        })
    }

    function editReadingOrder() {
        if (!selectedOrder.value) return
        error.value = ''
        orderForm.value = readingOrderFormFromDetail(selectedOrder.value)
        viewMode.value = 'edit'
        loadComics().catch(err => {
            error.value = err.message
        })
    }

    async function saveReadingOrder() {
        saving.value = true
        error.value = ''

        try {
            const payload = readingOrderPayload(orderForm.value)
            const detail = orderForm.value.id
                ? await updateReadingOrder(orderForm.value.id, payload)
                : await createReadingOrder(payload)

            selectedOrder.value = await setReadingOrderComics(detail.id, readingOrderComicsPayload(orderForm.value))
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
    async function loadReadingOrders(options = {}) {
        await loadPagedList('readingOrders', readingOrders, listReadingOrders, options)
    }

    return {
        readingOrders,
        selectedOrder,
        quickSavingOrderID,
        orderForm,
        visibleOrders,
        favoriteVisibleOrders,
        remainingVisibleOrders,
        readingOrderBrowseSections,
        favoriteOrderCount,
        currentOrderIndex,
        readingOrderProgress,
        openReadingOrder,
        openAdjacentReadingOrder,
        toggleReadingOrderFavorite,
        applyReadingOrderFavoriteState,
        newReadingOrder,
        editReadingOrder,
        saveReadingOrder,
        deleteReadingOrder,
        loadReadingOrders
    }
}
