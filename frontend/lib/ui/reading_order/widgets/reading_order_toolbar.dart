import 'package:flutter/material.dart';

import 'package:comichero_frontend/ui/ui.dart';

class ReadingOrderToolbar extends StatelessWidget {
  final Function() onAddEntry;
  final Function() onRefresh;
  final Function() onCsvImport;

  const ReadingOrderToolbar({
    super.key,
    required this.onAddEntry,
    required this.onRefresh,
    required this.onCsvImport,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(8.0),
      child: Row(
        children: [
          _FilterButton(),
          Spacer(),
          _AddEntryButton(onAddEntry: onAddEntry),
          _RefreshButton(onRefresh: onRefresh),
          _MenuButton(onCsvImport: onCsvImport),
        ],
      ),
    );
  }
}

class _MenuButton extends StatelessWidget {
  const _MenuButton({required this.onCsvImport});

  final Function() onCsvImport;

  @override
  Widget build(BuildContext context) {
    return AuthGuard(
      loggedInView: (context) => PopupMenuButton(
        itemBuilder: (context) {
          return [
            PopupMenuItem(
              onTap: onCsvImport,
              child: Row(
                children: [Icon(Icons.file_upload), Text("Import from CSV")],
              ),
            ),
          ];
        },
        icon: Icon(Icons.more_vert),
      ),
    );
  }
}

class _RefreshButton extends StatelessWidget {
  const _RefreshButton({required this.onRefresh});

  final Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    return IconButton(onPressed: onRefresh, icon: Icon(Icons.refresh));
  }
}

class _AddEntryButton extends StatelessWidget {
  const _AddEntryButton({required this.onAddEntry});

  final Function() onAddEntry;

  @override
  Widget build(BuildContext context) {
    return AuthGuard(
      loggedInView: (_) =>
          IconButton(onPressed: onAddEntry, icon: Icon(Icons.add)),
    );
  }
}

class _FilterButton extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    void openFilterModal() {
      showModalBottomSheet(
        context: context,
        constraints: const BoxConstraints(maxWidth: double.infinity),
        builder: (context) {
          return FilterModal();
        },
      );
    }

    return TextButton.icon(
      onPressed: openFilterModal,
      icon: Icon(Icons.filter_list),
      label: Text("Filter"),
    );
  }
}
