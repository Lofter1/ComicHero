export function emptyReadingOrder() {
  return {
    id: null,
    name: '',
    description: '',
    image: '',
    coverImageData: '',
    favorite: false,
    isPublic: true,
    entries: [],
    comics: [],
    childOrderIds: [],
  }
}

export function readingOrderFormFromDetail(detail) {
  const entries = (detail.entries || []).map((entry) => {
    if (entry.type === 'readingOrder' && entry.readingOrder) {
      return {
        type: 'readingOrder',
        readingOrderId: entry.readingOrder.id,
        title: entry.readingOrder.name || '',
        description: entry.readingOrder.description || '',
        comment: entry.comment || '',
      }
    }

    const comic = entry.comic
    return {
      type: 'comic',
      comicId: comic?.id,
      title: comic?.title || '',
      comment: comic?.comment || '',
      tags: comic?.tags || '',
    }
  })
  const fallbackEntries = (detail.comics || []).map((comic) => ({
    type: 'comic',
    comicId: comic.id,
    title: comic.title || '',
    comment: comic.comment || '',
    tags: comic.tags || '',
  }))

  return {
    id: detail.id,
    name: detail.name,
    description: detail.description,
    image: detail.image || '',
    coverImageData: '',
    favorite: detail.favorite,
    isPublic: detail.isPublic,
    childOrderIds: (detail.childReadingOrders || []).map((order) => order.id),
    entries: entries.length ? entries : fallbackEntries,
    comics: (detail.comics || []).map((comic) => ({
      comicId: comic.id,
      title: comic.title || '',
      comment: comic.comment || '',
      tags: comic.tags || '',
    })),
  }
}

export function readingOrderPayload(order) {
  return {
    name: order.name,
    description: order.description,
    favorite: order.favorite,
    isPublic: order.isPublic,
    coverImageData: order.coverImageData || '',
  }
}

export function readingOrderComicsPayload(order) {
  const sourceEntries = order.entries || []
  return {
    entries: sourceEntries
      .map((entry) => {
        if (entry.type === 'readingOrder') {
          return {
            type: 'readingOrder',
            readingOrderId: Number(entry.readingOrderId),
            comment: entry.comment || '',
          }
        }

        return {
          type: 'comic',
          comicId: Number(entry.comicId),
          comment: entry.comment || '',
          tags: entry.tags || '',
        }
      })
      .filter((entry) => {
        return entry.type === 'readingOrder' ? entry.readingOrderId > 0 : entry.comicId > 0
      }),
    readingOrderIds: (sourceEntries.length
      ? sourceEntries
      : (order.childOrderIds || []).map((id) => ({ type: 'readingOrder', readingOrderId: id }))
    )
      .filter((entry) => entry.type === 'readingOrder')
      .map((entry) => Number(entry.readingOrderId))
      .filter((id) => id > 0),
    comics: (sourceEntries.length
      ? sourceEntries.filter((entry) => entry.type !== 'readingOrder')
      : order.comics
    )
      .filter((comic) => Number(comic.comicId) > 0)
      .map((comic) => ({
        comicId: Number(comic.comicId),
        comment: comic.comment,
        tags: comic.tags,
      })),
  }
}

export function formatProgress(progress) {
  return `${Math.round((progress ?? 0) * 100)}%`
}

export function formatRating(rating) {
  const value = Number(rating) || 0
  if (value <= 0) return 'Unrated'
  return `${value.toFixed(1)} / 5`
}
