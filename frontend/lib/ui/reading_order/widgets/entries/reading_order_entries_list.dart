import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:comichero_frontend/providers/reading_order_entries_provider.dart';
import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/ui/ui.dart';
import 'package:infinite_scroll_pagination/infinite_scroll_pagination.dart';

class ReadingOrderEntriesList extends ConsumerStatefulWidget {
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
  ConsumerState<ReadingOrderEntriesList> createState() =>
      _ReadingOrderEntriesListState();
}

class _ReadingOrderEntriesListState
    extends ConsumerState<ReadingOrderEntriesList>
    with AutomaticKeepAliveClientMixin {
  @override
  Widget build(BuildContext context) {
    super.build(context);
    final pagedEntriesState = ref.watch(
      entriesForReadingOrderProvider(widget.readingOrderId),
    );
    final entriesNotifier = ref.read(
      entriesForReadingOrderProvider(widget.readingOrderId).notifier,
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
              onEntryRemovedClick: () => widget.onEntryRemoved(item),
              onEntryEditClick: () => widget.onEntryEdit(item),
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

  @override
  bool get wantKeepAlive => true;
}
