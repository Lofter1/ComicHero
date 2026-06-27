export function emptyComic() {
  return {
    id: null,
    title: '',
    series: '',
    seriesYear: 0,
    issue: 0,
    publisher: '',
    coverDate: '',
    coverImage: '',
    description: '',
    read: false,
  }
}

export function comicPayload(comic) {
  return {
    series: comic.series,
    seriesYear: Number(comic.seriesYear),
    issue: Number(comic.issue),
    publisher: comic.publisher,
    coverDate: comic.coverDate,
    coverImage: comic.coverImage,
    description: comic.description,
    read: comic.read,
  }
}

export function comicMatchesSearch(comic, term) {
  if (!term) return true

  return [comic.title, comic.series, comic.publisher, comic.description]
    .filter(Boolean)
    .some(value => value.toLowerCase().includes(term))
}

export function comicLabel(comics, comicID) {
  const comic = comics.find(item => item.id === Number(comicID))
  if (!comic) return 'Unknown comic'
  return comic.title
}
