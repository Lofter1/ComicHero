import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/services/services.dart';
import 'package:comichero_frontend/ui/ui.dart';
import 'package:flutter/material.dart';
import 'package:loader_overlay/loader_overlay.dart';

class ComicListView extends StatefulWidget {
  const ComicListView({super.key});

  @override
  State<ComicListView> createState() => _ComicListViewState();
}

class _ComicListViewState extends State<ComicListView> {
  Future<List<Comic>> listFuture = ComicService().get();

  @override
  void initState() {
    super.initState();
    authNotifier.addListener(_reload);
  }

  @override
  void dispose() {
    super.dispose();
    authNotifier.removeListener(_reload);
  }

  void _reload() {
    setState(() {
      listFuture = ComicService().get();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Align(
          alignment: Alignment.centerRight,
          child: IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Reload',
            onPressed: _reload,
          ),
        ),
        Expanded(
          child: FutureBuilder(
            future: listFuture,
            builder: (context, snapshot) {
              context.loaderOverlay.snapshotLoader(snapshot);
              if (snapshot.hasError) {
                return Text('Error: ${snapshot.error}');
              } else if (snapshot.hasData) {
                var comics = snapshot.data!;
                return ComicList(comics: comics);
              } else {
                return const Text("No data");
              }
            },
          ),
        ),
      ],
    );
  }
}
