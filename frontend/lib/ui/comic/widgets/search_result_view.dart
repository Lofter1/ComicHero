import 'package:flutter/material.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/ui/ui.dart';

class IssueSearchResultView extends StatelessWidget {
  final List<Comic> searchResults;
  final Function(Comic comic) onComicSelected;

  const IssueSearchResultView({
    super.key,
    required this.searchResults,
    required this.onComicSelected,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 400,
      child: ListView.separated(
        itemCount: searchResults.length,
        itemBuilder: (context, index) {
          final comic = searchResults[index];
          return ListTile(
            leading: ComicCover(comic: comic, rounding: 1),
            title: Text(comic.title),
            onTap: () => onComicSelected(comic),
          );
        },
        separatorBuilder: (BuildContext context, int index) {
          return Divider();
        },
      ),
    );
  }
}
