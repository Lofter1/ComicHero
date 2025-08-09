import 'package:comichero_frontend/providers/reading_order_progress_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ReadingOrderDetailBox extends StatelessWidget {
  const ReadingOrderDetailBox({super.key, required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: EdgeInsets.all(10),
      child: Padding(
        padding: const EdgeInsets.all(15),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _Description(readingOrder: readingOrder),
            AuthGuard(
              loggedInView: (_) => Column(
                children: [
                  Divider(),
                  _Progress(readingOrder: readingOrder),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _Progress extends ConsumerWidget {
  const _Progress({required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final progress = ref.watch(readingOrderProgressProvider(readingOrder.id));

    final percentFormat = NumberFormat.decimalPercentPattern(
      locale: Intl.defaultLocale,
      decimalDigits: 2,
    );

    return progress.when(
      data: (progress) {
        final percentage = progress.total == 0
            ? 0.0
            : progress.read / progress.total;
        final progressText =
            '${progress.read} / ${progress.total} (${percentFormat.format(percentage)})';

        return Row(
          spacing: 10,
          children: [
            Text(progressText),
            Expanded(child: LinearProgressIndicator(value: percentage)),
          ],
        );
      },
      error: (error, stacktrace) => Text('An error occured: $error'),
      loading: () => Center(child: LinearProgressIndicator()),
    );
  }
}

class _Description extends StatelessWidget {
  const _Description({required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Description', style: Theme.of(context).textTheme.headlineSmall),
        Text(
          readingOrder.description!,
          style: Theme.of(context).textTheme.bodyLarge,
        ),
      ],
    );
  }
}
