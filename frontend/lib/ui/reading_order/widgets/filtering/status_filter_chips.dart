import 'package:comichero_frontend/providers/reading_order_entries_list_options_provider.dart';
import 'package:comichero_frontend/services/services.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class StatusFilterChips extends ConsumerWidget {
  const StatusFilterChips({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final options = ref.watch(readingOrderEntriesOptionsProvider);
    final optionsNotifier = ref.read(
      readingOrderEntriesOptionsProvider.notifier,
    );

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        spacing: 10,
        children: [
          FilterChip(
            label: Text('Read'),
            selected: options.filterRead == ReadFilter.read,
            onSelected: (bool selected) {
              if (selected) {
                optionsNotifier.setReadFilter(ReadFilter.read);
              } else {
                optionsNotifier.setReadFilter(ReadFilter.none);
              }
            },
          ),
          FilterChip(
            label: Text('Not Read'),
            selected: options.filterRead == ReadFilter.notRead,
            onSelected: (bool selected) {
              if (selected) {
                optionsNotifier.setReadFilter(ReadFilter.notRead);
              } else {
                optionsNotifier.setReadFilter(ReadFilter.none);
              }
            },
          ),
          FilterChip(
            label: Text('Skipped'),
            selected: options.filterSkipped == SkippedFilter.skipped,
            onSelected: (bool selected) {
              if (selected) {
                optionsNotifier.setSkippedFilter(SkippedFilter.skipped);
              } else {
                optionsNotifier.setSkippedFilter(SkippedFilter.none);
              }
            },
          ),
          FilterChip(
            label: Text('Not Skipped'),
            selected: options.filterSkipped == SkippedFilter.notSkipped,
            onSelected: (bool selected) {
              if (selected) {
                optionsNotifier.setSkippedFilter(SkippedFilter.notSkipped);
              } else {
                optionsNotifier.setSkippedFilter(SkippedFilter.none);
              }
            },
          ),
        ],
      ),
    );
  }
}
