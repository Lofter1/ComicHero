export function emptyReadingOrder() {
  return {
    id: null,
    name: '',
    description: '',
    favorite: false,
    comics: [],
  }
}

export function readingOrderFormFromDetail(detail) {
  return {
    id: detail.id,
    name: detail.name,
    description: detail.description,
    favorite: detail.favorite,
    comics: (detail.comics || []).map(comic => ({
      comicId: comic.id,
      comment: comic.comment || '',
    })),
  }
}

export function readingOrderPayload(order) {
  return {
    name: order.name,
    description: order.description,
    favorite: order.favorite,
  }
}

export function readingOrderComicsPayload(order) {
  return {
    comics: order.comics
      .filter(comic => Number(comic.comicId) > 0)
      .map(comic => ({
        comicId: Number(comic.comicId),
        comment: comic.comment,
      })),
  }
}

export function readingOrderMatchesSearch(order, term) {
  if (!term) return true

  return [order.name, order.description]
    .filter(Boolean)
    .some(value => value.toLowerCase().includes(term))
}

export function formatProgress(progress) {
  return `${Math.round((progress ?? 0) * 100)}%`
}
