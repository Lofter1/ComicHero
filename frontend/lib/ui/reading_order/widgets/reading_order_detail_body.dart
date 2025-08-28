import 'package:comichero_frontend/providers/reading_order_entries_provider.dart';
import 'package:comichero_frontend/providers/reading_order_progress_provider.dart';
import 'package:comichero_frontend/ui/general/auth_guard.dart';
import 'package:comichero_frontend/ui/general/error_helpers.dart';
import 'package:flutter/material.dart';

import 'package:comichero_frontend/services/services.dart';
import 'package:comichero_frontend/models/models.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ReadingOrderDetailBody extends StatelessWidget {
  const ReadingOrderDetailBody({super.key, required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(15),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _Description(readingOrder: readingOrder),
          _ReadingOrderActionButtons(readingOrder: readingOrder),
        ],
      ),
    );
  }
}

class _ReadingOrderActionButtons extends StatelessWidget {
  const _ReadingOrderActionButtons({required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context) {
    return AuthGuard(
      loggedInView: (context) => Card(
        child: Padding(
          padding: const EdgeInsets.all(8.0),
          child: Column(
            children: [
              Text(
                'Danger Zone',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              Row(
                children: [_ClearAllEntriesButton(readingOrder: readingOrder)],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ClearAllEntriesButton extends ConsumerWidget {
  const _ClearAllEntriesButton({required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return TextButton(
      style: TextButton.styleFrom(foregroundColor: Colors.red),
      onPressed: () async {
        try {
          await ReadingOrderService().clearAllEntries(readingOrder.id);

          if (context.mounted) {
            ScaffoldMessenger.of(
              context,
            ).showSnackBar(SnackBar(content: Text("Cleared Reading Order")));
          }
        } on Exception catch (e) {
          if (context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(getErrorSnackbar(e));
          }
        } finally {
          if (context.mounted) {
            ref.invalidate(entriesForReadingOrderProvider(readingOrder.id));
            ref.invalidate(readingOrderProgressProvider(readingOrder.id));
          }
        }
      },
      child: Text("Clear all entries"),
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
