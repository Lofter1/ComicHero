import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:comichero_frontend/providers/reading_order_entries_provider.dart';
import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/ui/ui.dart';
import 'package:infinite_scroll_pagination/infinite_scroll_pagination.dart';

class ReadingOrderEntriesList extends ConsumerWidget {
  final String readingOrderId;
  final Function(ReadingOrderEntry entry) onEntryRemoved;
  final Function(ReadingOrderEntry entry) onEntryEdit;

  const ReadingOrderEntriesList({
    super.key,
    required this.readingOrderId,
    required this.onEntryRemoved,
    required this.onEntryEdit,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final pagedEntriesState = ref.watch(
      entriesForReadingOrderProvider(readingOrderId),
    );
    final entriesNotifier = ref.read(
      entriesForReadingOrderProvider(readingOrderId).notifier,
    );

    return Expanded(
      child: ScrollConfiguration(
        behavior: ScrollConfiguration.of(context).copyWith(scrollbars: false),
        child: PagedListView<int, ReadingOrderEntry>(
          state: pagedEntriesState,
          fetchNextPage: entriesNotifier.fetchNextPage,
          builderDelegate: PagedChildBuilderDelegate(
            itemBuilder: (context, item, index) => ReadingOrderEntryListTile(
              key: ValueKey(item.comic.hashCode),
              entry: item,
              onEntryRemovedClick: () => onEntryRemoved(item),
              onEntryEditClick: () => onEntryEdit(item),
              onSetRead: (bool read) =>
                  entriesNotifier.comicSetRead(item.comic!, read),
              onSetSkipped: (bool skipped) =>
                  entriesNotifier.comicSetSkipped(item.comic!, skipped),
            ),
          ),
        ),
      ),
    );
  }
}
