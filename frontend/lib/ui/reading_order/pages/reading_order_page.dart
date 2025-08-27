import 'package:comichero_frontend/ui/reading_order/widgets/reading_order_entries_list_body.dart';
import 'package:flutter/material.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ReadingOrderPage extends StatelessWidget {
  const ReadingOrderPage({super.key, required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: ComicHeroAppBar(title: readingOrder.name),
      body: DefaultTabController(
        length: 2,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.center,
          mainAxisSize: MainAxisSize.min,
          children: [
            const TabBar(
              tabs: [
                Tab(icon: Icon(Icons.list)),
                Tab(icon: Icon(Icons.info)),
              ],
            ),
            Expanded(
              child: TabBarView(
                physics: const NeverScrollableScrollPhysics(),

                children: [
                  ReadingOrderEntriesListBody(readingOrder: readingOrder),
                  ReadingOrderDetailBody(readingOrder: readingOrder),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
