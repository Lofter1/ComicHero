enum SortBy { position, comicReleaseDate }

enum ReadFilter { read, notRead, none }

enum SkippedFilter { skipped, notSkipped, none }

class ReadingOrderEntriesListOptions {
  final ReadFilter filterRead;
  final SkippedFilter filterSkipped;

  final SortBy sortBy;

  ReadingOrderEntriesListOptions({
    this.filterRead = ReadFilter.none,
    this.filterSkipped = SkippedFilter.none,
    this.sortBy = SortBy.position,
  });

  String get sortString {
    switch (sortBy) {
      case SortBy.position:
        return 'position';
      case SortBy.comicReleaseDate:
        return 'comic.releaseDate';
    }
  }

  String get filterString {
    final filters = <String>[];

    if (filterRead != ReadFilter.none) {
      if (filterRead == ReadFilter.read) {
        filters.add('comic.userComics_via_comic.read = true');
      } else {
        filters.add(
          '(comic.userComics_via_comic.read = false || comic.userComics_via_comic.read = null)',
        );
      }
    }

    if (filterSkipped != SkippedFilter.none) {
      if (filterSkipped == SkippedFilter.skipped) {
        filters.add('comic.userComics_via_comic.skipped = true');
      } else {
        filters.add(
          '(comic.userComics_via_comic.skipped = false || comic.userComics_via_comic.skipped = null)',
        );
      }
    }

    return filters.join(' && ');
  }

  ReadingOrderEntriesListOptions copyWith({
    ReadFilter? filterRead,
    SkippedFilter? filterSkipped,
    SortBy? sortBy,
    int? page,
  }) {
    return ReadingOrderEntriesListOptions(
      filterRead: filterRead ?? this.filterRead,
      filterSkipped: filterSkipped ?? this.filterSkipped,
      sortBy: sortBy ?? this.sortBy,
    );
  }
}
