import 'package:flutter/material.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ReadingOrderQuickOverview extends StatelessWidget {
  final ReadingOrder readingOrder;

  const ReadingOrderQuickOverview({super.key, required this.readingOrder});

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      spacing: 10,
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            spacing: 10,
            children: [
              _ReadingOrderTitle(readingOrder: readingOrder),
              if (readingOrder.description != null &&
                  readingOrder.description!.isNotEmpty)
                _ReadingOrderDescription(readingOrder: readingOrder),
              Row(
                spacing: 10,
                children: [_OpenDetailViewButton(readingOrder: readingOrder)],
              ),
            ],
          ),
        ),
      ],
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
            builder: (_) => ReadingOrderDetailPage(readingOrder: readingOrder),
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
      maxLines: 4,
      overflow: TextOverflow.ellipsis,
      style: Theme.of(context).textTheme.bodyMedium,
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
