import 'package:flutter/material.dart';

import 'package:comichero_frontend/models/models.dart';

class EntryEditDialog extends StatefulWidget {
  final ReadingOrderEntry entry;

  const EntryEditDialog({super.key, required this.entry});

  @override
  State<EntryEditDialog> createState() => _EntryEditDialogState();
}

class _EntryEditDialogState extends State<EntryEditDialog> {
  late final TextEditingController positionController;
  late final TextEditingController notesController;

  @override
  void initState() {
    super.initState();
    positionController = TextEditingController(
      text: widget.entry.position.toString(),
    );
    notesController = TextEditingController(text: widget.entry.notes);
  }

  @override
  void dispose() {
    super.dispose();
    positionController.dispose();
    notesController.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('${widget.entry.comic?.title}'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        spacing: 10,
        children: [
          TextField(
            controller: positionController,
            decoration: InputDecoration(labelText: 'Position'),
          ),
          TextField(
            controller: notesController,
            decoration: InputDecoration(labelText: 'Notes'),
          ),
        ],
      ),
      actions: [
        TextButton(onPressed: _onCancel, child: Text('Cancel')),
        TextButton(onPressed: _onSave, child: Text('Save')),
      ],
    );
  }

  void _onSave() {
    widget.entry.position = int.tryParse(positionController.text) ?? 0;
    widget.entry.notes = notesController.text;
    Navigator.pop(context, widget.entry);
  }

  void _onCancel() {
    Navigator.pop(context);
  }
}
