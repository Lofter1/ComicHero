export function emptyComic() {
  return {
    id: null,
    title: '',
    series: '',
    seriesYear: 0,
    issue: '',
    publisher: '',
    coverDate: '',
    coverImage: '',
    description: '',
    read: false,
    skipped: false,
  }
}

export function comicPayload(comic) {
  return {
    series: comic.series,
    seriesYear: Number(comic.seriesYear),
    issue: String(comic.issue || '').trim(),
    publisher: comic.publisher,
    coverDate: comic.coverDate,
    coverImage: comic.coverImage,
    description: comic.description,
    read: comic.read,
    skipped: comic.skipped,
  }
}

export function comicLabel(comics, comicID) {
  const comic = comics.find((item) => item.id === Number(comicID))
  if (!comic) return 'Unknown comic'
  return comic.title
}
