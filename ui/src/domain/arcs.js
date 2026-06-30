export function emptyArc() {
  return {
    id: null,
    name: '',
    description: '',
    favorite: false,
    comics: [],
  }
}

export function arcFormFromDetail(detail) {
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

export function arcPayload(arc) {
  return {
    name: arc.name,
    description: arc.description,
    favorite: arc.favorite,
  }
}

export function arcComicsPayload(arc) {
  return {
    comics: arc.comics
      .filter(comic => Number(comic.comicId) > 0)
      .map(comic => ({
        comicId: Number(comic.comicId),
        comment: comic.comment,
      })),
  }
}
