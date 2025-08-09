import 'package:flutter/material.dart';
import 'package:loader_overlay/loader_overlay.dart';
import 'package:pocketbase/pocketbase.dart';

import 'package:comichero_frontend/models/reading_order.dart';
import 'package:comichero_frontend/services/services.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ReadingOrdersListPage extends StatefulWidget {
  const ReadingOrdersListPage({super.key});

  @override
  State<ReadingOrdersListPage> createState() => _ReadingOrdersListPageState();
}

class _ReadingOrdersListPageState extends State<ReadingOrdersListPage> {
  late Future<ResultList<RecordModel>> list;

  void _loadList() {
    setState(() {
      list = ReadingOrderService().get();
    });
  }

  @override
  void initState() {
    super.initState();
    _loadList();
    authNotifier.addListener(_loadList);
  }

  @override
  void dispose() {
    super.dispose();
    authNotifier.removeListener(_loadList);
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder(
      future: list,
      builder: (context, snapshot) {
        context.loaderOverlay.snapshotLoader(snapshot);

        var readingOrders = snapshot.data?.items
            .map((item) => ReadingOrder.fromRecord(item))
            .toList();

        return Scaffold(
          appBar: ComicHeroAppBar(title: "Reading Orders"),
          body: Center(
            child: ReadingOrderList(readingOrders: readingOrders ?? []),
          ),
          floatingActionButton: AuthGuard(
            loggedInView: (_) => FloatingActionButton(
              onPressed: () async {
                var newReadingOrder = await showDialog(
                  context: context,
                  builder: (_) {
                    String name = '';
                    String description = '';
                    return AlertDialog(
                      contentPadding: EdgeInsets.all(20),
                      title: Text('Create List'),
                      content: Column(
                        mainAxisSize: MainAxisSize.min,
                        spacing: 10,
                        children: [
                          TextField(
                            decoration: InputDecoration(
                              labelText: 'Name',
                              border: OutlineInputBorder(),
                            ),
                            onChanged: (value) => name = value,
                          ),
                          TextField(
                            decoration: InputDecoration(
                              labelText: 'Description',
                              border: OutlineInputBorder(),
                            ),
                            onChanged: (value) => description = value,
                            maxLines: null,
                            minLines: 2,
                          ),
                        ],
                      ),
                      actions: [
                        TextButton(
                          onPressed: () {
                            Navigator.pop(context);
                          },
                          child: Text('Cancel'),
                        ),
                        TextButton(
                          onPressed: () {
                            Navigator.pop(
                              context,
                              ReadingOrder(
                                id: '',
                                name: name,
                                description: description,
                              ),
                            );
                          },
                          child: Text('Save'),
                        ),
                      ],
                    );
                  },
                );

                if (newReadingOrder == null) return;

                newReadingOrder = await ReadingOrderService().create(
                  newReadingOrder,
                );

                _loadList();
              },
              child: Icon(Icons.add),
            ),
          ),
        );
      },
    );
  }
}

class ReadingOrdersListPageViewModel {}
