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

export const readingOrderEditorPageSize = 100

export function reorderReadingOrderEntry(entries, fromIndex, toIndex) {
  const source = Array.isArray(entries) ? entries : []
  if (
    !Number.isInteger(fromIndex) ||
    !Number.isInteger(toIndex) ||
    fromIndex < 0 ||
    toIndex < 0 ||
    fromIndex >= source.length ||
    toIndex >= source.length ||
    fromIndex === toIndex
  ) {
    return source
  }

  const reordered = [...source]
  const [entry] = reordered.splice(fromIndex, 1)
  reordered.splice(toIndex, 0, entry)
  return reordered
}

export function readingOrderEditorPage(
  entries,
  requestedPage,
  pageSize = readingOrderEditorPageSize,
) {
  const source = Array.isArray(entries) ? entries : []
  const size = Math.max(1, Number(pageSize) || readingOrderEditorPageSize)
  const pageCount = Math.max(1, Math.ceil(source.length / size))
  const page = Math.min(Math.max(0, Number(requestedPage) || 0), pageCount - 1)
  const start = page * size
  const end = Math.min(start + size, source.length)

  return {
    page,
    pageCount,
    start,
    end,
    entries: source.slice(start, end).map((entry, offset) => ({
      entry,
      index: start + offset,
    })),
  }
}

export function readingOrderFormFromDetail(detail) {
  const entries = (detail.entries || []).map((entry) => {
    if (entry.type === 'section' && entry.section) {
      return {
        type: 'section',
        title: entry.section.title || '',
        description: entry.section.description || '',
      }
    }

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
        if (entry.type === 'section') {
          return {
            type: 'section',
            title: String(entry.title || '').trim(),
            description: entry.description || '',
          }
        }

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
        if (entry.type === 'section') return Boolean(entry.title)
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
      ? sourceEntries.filter((entry) => entry.type === 'comic')
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

export function readingOrderDisplayComics(order) {
  const flattened = order?.comics || []
  const entries = order?.entries || []
  if (!entries.length) return flattened

  const currentState = new Map(
    flattened.map((comic) => [comic.id, { read: comic.read, skipped: comic.skipped }]),
  )
  const display = []
  let section = null
  let sectionIndex = 0
  let entryIndex = 0

  const appendComics = (comics, group) => {
    for (const comic of comics) {
      const state = currentState.get(comic.id)
      display.push({
        ...comic,
        read: state?.read ?? comic.read,
        skipped: state?.skipped ?? comic.skipped,
        section: group,
      })
    }
  }

  for (const entry of entries) {
    entryIndex += 1

    if (entry.type === 'section') {
      sectionIndex += 1
      section = {
        key: `order-${order?.id || 'draft'}-section-${sectionIndex}`,
        kind: 'section',
        label: 'Section',
        title: entry.section?.title || '',
        description: entry.section?.description || '',
      }
      continue
    }

    if (entry.type === 'readingOrder') {
      appendComics(entry.comics || [], {
        key: `order-${order?.id || 'draft'}-nested-${entryIndex}`,
        kind: 'readingOrder',
        label: 'Reading order',
        title: entry.readingOrder?.name || 'Nested reading order',
        description: entry.comment || entry.readingOrder?.description || '',
      })
      continue
    }

    appendComics(entry.comic ? [entry.comic] : [], section)
  }

  return display.length === flattened.length ? display : flattened
}

export function formatProgress(progress) {
  return `${Math.round((progress ?? 0) * 100)}%`
}

export function formatRating(rating) {
  const value = Number(rating) || 0
  if (value <= 0) return 'Unrated'
  return `${value.toFixed(1)} / 5`
}
