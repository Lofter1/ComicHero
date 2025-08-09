import 'package:infinite_scroll_pagination/infinite_scroll_pagination.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/providers/reading_order_entries_list_options_provider.dart';
import 'package:comichero_frontend/providers/reading_order_progress_provider.dart';
import 'package:comichero_frontend/services/services.dart';

part 'reading_order_entries_provider.g.dart';

@riverpod
class EntriesForReadingOrder extends _$EntriesForReadingOrder {
  final _readingOrderEntriesService = ReadingOrderEntriesService();
  final _comicService = ComicService();

  bool _isFetching = false;

  @override
  PagingState<int, ReadingOrderEntry> build(String readingOrderId) {
    ref.watch(readingOrderEntriesOptionsProvider);
    return PagingState();
  }

  Future<void> fetchNextPage() async {
    if (_isFetching) return;
    _isFetching = true;

    state = state.copyWith(isLoading: true, error: null);

    try {
      final response = await _readingOrderEntriesService.getWithComics(
        readingOrderId,
        options: ref.read(readingOrderEntriesOptionsProvider),
        page: state.nextIntPageKey,
      );

      state = state.copyWith(
        isLoading: false,
        error: null,
        hasNextPage: response.totalPages != response.page,
        keys: [...?state.keys, state.nextIntPageKey],
        pages: [
          ...?state.pages,
          response.items
              .map(_readingOrderEntriesService.mapRecordToReadingOrderEntry)
              .toList(),
        ],
      );
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e);
    } finally {
      _isFetching = false;
    }
  }

  Future<ReadingOrderEntry> createEntry(ReadingOrderEntry entry) async {
    final newEntry = await _readingOrderEntriesService.create(entry);

    ref.invalidateSelf();
    ref.invalidate(readingOrderProgressProvider);
    return newEntry;
  }

  Future<ReadingOrderEntry> updateEntry(ReadingOrderEntry entry) async {
    final updatedEntry = await _readingOrderEntriesService.update(entry);

    ref.invalidateSelf();
    return updatedEntry;
  }

  Future<void> removeEntry(ReadingOrderEntry entry) async {
    if (state.pages == null) return;

    await ReadingOrderService().removeEntry(entry);

    state = state.copyWith(
      pages: state.pages!
          .map((page) => List<ReadingOrderEntry>.from(page)..remove(entry))
          .toList(),
    );
    ref.invalidate(readingOrderProgressProvider);
  }

  void updateComic(Comic updatedComic) {
    if (state.pages == null) return;

    final updatedEntries = state.pages!.map((page) {
      return page.map((entry) {
        if (entry.comic != null && entry.comic?.id == updatedComic.id) {
          return entry.copyWith(comic: updatedComic);
        }
        return entry;
      }).toList();
    }).toList();

    state = state.copyWith(pages: updatedEntries);
  }

  Future<void> comicSetRead(Comic comic, bool value) async {
    if (comic.read! == value) return;

    final updatedComic = await _comicService.setReadStatus(comic, value);

    updateComic(updatedComic);
    ref.invalidate(readingOrderProgressProvider);
  }

  Future<void> comicSetSkipped(Comic comic, bool value) async {
    if (comic.skipped! == value) return;

    final updatedComic = await _comicService.setSkippedStatus(comic, value);

    updateComic(updatedComic);

    if (comic.read! && value) {
      ref.invalidate(readingOrderProgressProvider);
    }
  }
}
