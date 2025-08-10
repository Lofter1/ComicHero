import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import 'package:comichero_frontend/services/services.dart';
import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ReadingOrderEntryListTile extends StatelessWidget {
  final ReadingOrderEntry entry;
  final Function() onEntryRemovedClick;
  final Function() onEntryEditClick;
  final Function(bool read) onSetRead;
  final Function(bool skipped) onSetSkipped;

  const ReadingOrderEntryListTile({
    super.key,
    required this.entry,
    required this.onEntryRemovedClick,
    required this.onEntryEditClick,
    required this.onSetRead,
    required this.onSetSkipped,
  });

  @override
  Widget build(BuildContext context) {
    if (entry.comic == null) {
      return ListTile(
        title: Text("Comic not found"),
        subtitle: Text("ID: ${entry.id}"),
      );
    }

    return ListTile(
      leading: entry.comic!.coverUrl != null
          ? Image.network(
              entry.comic!.coverUrl!,
              headers: {"pb_auth": pb.authStore.token},
            )
          : null,
      title: Text(entry.comic!.title),
      subtitle: entry.comic!.releaseDate != null
          ? Text(DateFormat.yMMMd().format(entry.comic!.releaseDate!))
          : null,
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          _ComicUserStatusIconButton(
            comic: entry.comic!,
            onStatusButtonPressed: _toggleRead,
          ),
          _ComicPopupMenuButton(
            onEntryRemoveClick: onEntryRemovedClick,
            onToggleComicEntryRead: _toggleRead,
            onToggleComicEntrySkipped: _toggleSkipped,
            onEntryEditClick: onEntryEditClick,
          ),
        ],
      ),
    );
  }

  void _toggleRead() {
    if (entry.comic != null || entry.comic?.read != null) {
      onSetRead(!entry.comic!.read!);
    }
  }

  void _toggleSkipped() {
    if (entry.comic != null || entry.comic?.skipped != null) {
      onSetSkipped(!entry.comic!.skipped!);
    }
  }
}

class _ComicUserStatusIconButton extends StatelessWidget {
  final Comic comic;
  final Function() onStatusButtonPressed;

  const _ComicUserStatusIconButton({
    required this.comic,
    required this.onStatusButtonPressed,
  });

  @override
  Widget build(BuildContext context) {
    return AuthGuard(
      loggedInView: (context) {
        return IconButton(
          icon: Icon(
            comic.read == true
                ? Icons.check_circle
                : comic.skipped == true
                ? Icons.skip_next
                : Icons.radio_button_unchecked,
            color: comic.read == true
                ? Colors.green
                : comic.skipped == true
                ? Colors.amber
                : Colors.grey,
          ),
          tooltip: "Set status",
          onPressed: onStatusButtonPressed,
        );
      },
    );
  }
}

class _ComicPopupMenuButton extends StatelessWidget {
  final Function() onToggleComicEntryRead;
  final Function() onToggleComicEntrySkipped;
  final Function() onEntryEditClick;
  final Function() onEntryRemoveClick;

  const _ComicPopupMenuButton({
    required this.onToggleComicEntryRead,
    required this.onToggleComicEntrySkipped,
    required this.onEntryEditClick,
    required this.onEntryRemoveClick,
  });

  @override
  Widget build(BuildContext context) {
    return AuthGuard(
      loggedInView: (context) => PopupMenuButton(
        itemBuilder: (context) {
          return [
            PopupMenuItem(
              child: Row(
                spacing: 5,
                children: [Icon(Icons.edit), Text('Edit')],
              ),
              onTap: () => onEntryEditClick(),
            ),
            PopupMenuItem(
              child: Row(
                spacing: 5,
                children: [Icon(Icons.check), Text("Toggle read")],
              ),
              onTap: () => onToggleComicEntryRead(),
            ),
            PopupMenuItem(
              child: Row(
                spacing: 5,
                children: [Icon(Icons.skip_next), Text("Toggle skipped")],
              ),
              onTap: () => onToggleComicEntrySkipped(),
            ),
            PopupMenuItem(
              child: Row(
                spacing: 5,
                children: [
                  Icon(Icons.delete, color: Colors.red),
                  Text('Remove', style: TextStyle(color: Colors.red)),
                ],
              ),
              onTap: () => onEntryRemoveClick(),
            ),
          ];
        },
      ),
    );
  }
}
