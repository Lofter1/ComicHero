import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:infinite_scroll_pagination/infinite_scroll_pagination.dart';

import 'package:comichero_frontend/providers/reading_order_progress_provider.dart';
import 'package:comichero_frontend/providers/reading_orders_provider.dart';
import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ReadingOrderList extends ConsumerWidget {
  const ReadingOrderList({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final pagedReadingOrdersState = ref.watch(readingOrdersProvider);
    final pagedReadingOrdersNotifier = ref.read(readingOrdersProvider.notifier);

    return ScrollConfiguration(
      behavior: ScrollConfiguration.of(context).copyWith(scrollbars: false),
      child: PagedAlignedGridView<int, ReadingOrder>(
        state: pagedReadingOrdersState,
        fetchNextPage: pagedReadingOrdersNotifier.fetchNextPage,
        builderDelegate: PagedChildBuilderDelegate(
          itemBuilder: (context, item, index) {
            const padding = 10.0;
            return Card(
              key: ValueKey(item.id),
              margin: const EdgeInsets.all(10),

              child: SizedBox(
                height: 190,
                child: Column(
                  children: [
                    Padding(
                      padding: const EdgeInsets.only(
                        top: padding,
                        left: padding,
                        right: padding,
                      ),
                      child: _ReadingOrderTitle(readingOrder: item),
                    ),
                    Expanded(
                      child: Padding(
                        padding: const EdgeInsets.all(padding),
                        child:
                            item.description != null &&
                                item.description!.isNotEmpty
                            ? _ReadingOrderDescription(readingOrder: item)
                            : const SizedBox.shrink(),
                      ),
                    ),
                    Divider(height: 1),
                    Padding(
                      padding: const EdgeInsets.all(padding),

                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          _ReadingOrderProgress(
                            key: ValueKey(item.id),
                            readingOrderId: item.id,
                          ),
                          _OpenDetailViewButton(readingOrder: item),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            );
          },
        ),
        gridDelegateBuilder: (childCount) {
          return SliverSimpleGridDelegateWithMaxCrossAxisExtent(
            maxCrossAxisExtent: 550,
          );
        },
      ),
    );
  }
}

class _ReadingOrderProgress extends ConsumerWidget {
  final String readingOrderId;

  const _ReadingOrderProgress({super.key, required this.readingOrderId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return AuthGuard(
      loggedInView: (context) {
        final progressState = ref.watch(
          readingOrderProgressProvider(readingOrderId),
        );
        return progressState.when(
          data: (progressData) =>
              Text("Progress: ${progressData.read}/${progressData.total}"),
          error: (error, stacktrace) {
            return Icon(Icons.error, color: Colors.red);
          },
          loading: () => Text("Progress: -/-"),
        );
      },
    );
  }
}

class _ReadingOrderTitle extends StatelessWidget {
  const _ReadingOrderTitle({required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context) {
    return Text(
      readingOrder.name,
      style: Theme.of(context).textTheme.titleLarge,
    );
  }
}

class _OpenDetailViewButton extends StatelessWidget {
  const _OpenDetailViewButton({required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      onPressed: () {
        Navigator.push(
          context,
          MaterialPageRoute(
            builder: (_) => ReadingOrderPage(readingOrder: readingOrder),
          ),
        );
      },
      child: const Text('View Details'),
    );
  }
}

class _ReadingOrderDescription extends StatelessWidget {
  const _ReadingOrderDescription({required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context) {
    return Text(
      readingOrder.description!,
      maxLines: 3,
      overflow: TextOverflow.ellipsis,
      style: Theme.of(context).textTheme.bodyLarge,
    );
  }
}
