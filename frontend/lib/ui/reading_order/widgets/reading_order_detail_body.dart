import 'package:flutter/material.dart';

import 'package:comichero_frontend/models/models.dart';

class ReadingOrderDetailBody extends StatelessWidget {
  const ReadingOrderDetailBody({super.key, required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(15),
      child: _Description(readingOrder: readingOrder),
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
