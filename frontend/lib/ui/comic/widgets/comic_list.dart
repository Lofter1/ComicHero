import 'package:flutter/material.dart';

import 'package:comichero_frontend/ui/ui.dart';
import 'package:comichero_frontend/models/comic.dart';

class ComicList extends StatelessWidget {
  final List<Comic> comics;
  final List<PopupMenuEntry<Function(Comic)>>? entryPopupMenuItems;

  const ComicList({super.key, required this.comics, this.entryPopupMenuItems});

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: comics.length,
      separatorBuilder: (_, _) => const SizedBox(height: 24),
      itemBuilder: (context, index) {
        return ComicQuickOverview(
          key: ValueKey(comics[index].hashCode),
          comic: comics[index],
          popupMenuItems: entryPopupMenuItems,
        );
      },
    );
  }
}
