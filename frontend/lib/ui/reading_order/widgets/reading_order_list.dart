import 'package:flutter/material.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ReadingOrderList extends StatelessWidget {
  const ReadingOrderList({super.key, required this.readingOrders});

  final List<ReadingOrder> readingOrders;

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: readingOrders.length,
      separatorBuilder: (_, _) => const SizedBox(height: 24),
      itemBuilder: (context, index) {
        return ReadingOrderQuickOverview(readingOrder: readingOrders[index]);
      },
    );
  }
}
