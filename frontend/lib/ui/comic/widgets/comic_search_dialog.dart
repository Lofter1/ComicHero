import 'package:flutter/material.dart';
import 'package:loader_overlay/loader_overlay.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/services/services.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ComicSearchDialog extends StatelessWidget {
  const ComicSearchDialog({super.key});

  @override
  Widget build(BuildContext context) {
    return AlertDialog(insetPadding: EdgeInsets.all(10), content: SearchView());
  }
}

class SearchView extends StatefulWidget {
  final String? initialSearchQuery;

  const SearchView({super.key, this.initialSearchQuery});

  @override
  State<SearchView> createState() => _SearchViewState();
}

class _SearchViewState extends State<SearchView> {
  late final TextEditingController _searchController;

  List<Comic> _searchResults = [];
  bool searchMetron = false;

  @override
  void initState() {
    super.initState();
    _searchController = TextEditingController(text: widget.initialSearchQuery);
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 700,
      child: Column(
        spacing: 10,
        mainAxisSize: MainAxisSize.min,
        children: [
          TextField(
            controller: _searchController,
            decoration: InputDecoration(
              labelText: 'Search Comics',
              suffixIcon: IconButton(
                icon: Icon(Icons.search),
                onPressed: () => _onSearchComics(_searchController.text),
              ),
            ),
            onSubmitted: _onSearchComics,
          ),
          Tooltip(
            message: "Use Metron for search",
            child: Row(
              children: [
                Switch(
                  value: searchMetron,
                  onChanged: (val) {
                    setState(() {
                      searchMetron = val;
                    });
                  },
                ),
                Text("Search Metron"),
              ],
            ),
          ),
          Flexible(
            child: IssueSearchResultView(
              searchResults: _searchResults,
              onComicSelected: _onComicSelected,
            ),
          ),
        ],
      ),
    );
  }

  Future<List<Comic>> _searchComics(String searchQuery) {
    if (searchMetron) {
      return MetronService().searchComic(searchQuery);
    } else {
      return ComicService().search(searchQuery);
    }
  }

  void _onSearchComics(String query) async {
    if (query.isEmpty) {
      setState(() => _searchResults = []);
      return;
    }

    context.loaderOverlay.show();
    final result = await _searchComics(query);

    if (mounted) {
      context.loaderOverlay.hide();
      setState(() => _searchResults = result);
    }
  }

  Future<void> _onComicSelected(Comic comic) async {
    context.loaderOverlay.show();

    if (searchMetron) {
      comic = await MetronService().getComicById(int.parse(comic.id));
      comic = await ComicService().create(comic);
    }

    if (mounted) {
      context.loaderOverlay.hide();
      Navigator.pop(context, comic);
    }
  }
}
