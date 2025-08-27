import 'dart:convert';
import 'dart:io';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:comichero_frontend/providers/reading_order_entries_provider.dart';
import 'package:comichero_frontend/providers/reading_order_progress_provider.dart';
import 'package:comichero_frontend/ui/general/error_helpers.dart';
import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/services/csv_import_service.dart';
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
                  _ReadingOrderEntriesListBody(readingOrder: readingOrder),
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

class _ReadingOrderEntriesListBody extends ConsumerStatefulWidget {
  final ReadingOrder readingOrder;

  const _ReadingOrderEntriesListBody({required this.readingOrder});

  @override
  ConsumerState<_ReadingOrderEntriesListBody> createState() =>
      _ReadingOrderDetailViewBodyState();
}

class _ReadingOrderDetailViewBodyState
    extends ConsumerState<_ReadingOrderEntriesListBody> {
  bool isImporting = false;
  String importProgressText = "";
  double? importProgressPercent;
  ImportCancelationToken? importCancelationToken;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (isImporting)
          _CsvImportProgress(
            importProgressText: importProgressText,
            importProgressPercent: importProgressPercent,
            onCancel: () {
              importCancelationToken?.cancel();
              setState(() {
                isImporting = false;
                importProgressText = "";
                importProgressPercent = 0;
              });
              ScaffoldMessenger.of(
                context,
              ).showSnackBar(SnackBar(content: Text("Import canceled")));
            },
          ),

        ReadingOrderToolbar(
          onRefresh: _onRefresh,
          onAddEntry: _openAddComicPopup,
          onCsvImport: _onCsvImport,
        ),

        AuthGuard(
          loggedInView: (context) =>
              _ReadingProgress(readingOrder: widget.readingOrder),
          loggedOutView: (context) => LinearProgressIndicator(value: 0),
        ),

        ReadingOrderEntriesList(
          readingOrderId: widget.readingOrder.id,
          onEntryRemoved: _onEntryRemoved,
          onEntryEdit: _onEntryEdit,
        ),
      ],
    );
  }

  void _onRefresh() {
    ref.invalidate(entriesForReadingOrderProvider);
    ref.invalidate(readingOrderProgressProvider);
  }

  Future<void> _onEntryRemoved(ReadingOrderEntry entry) async {
    ref
        .read(entriesForReadingOrderProvider(widget.readingOrder.id).notifier)
        .removeEntry(entry);
  }

  Future<void> _onEntryEdit(ReadingOrderEntry entry) async {
    final updatedEntry = await _showEditDialog(entry);

    if (updatedEntry == null) {
      return;
    }
    _updateReadingOrderEntry(updatedEntry);
  }

  Future<void> _onCsvImport() async {
    try {
      final csvString = await _getCsvFileContent();
      if (csvString == null) return;

      importCancelationToken = ImportCancelationToken();

      final importResult = await CsvImportService().importCsv(
        csvString,
        widget.readingOrder.id,
        cancelationToken: importCancelationToken,
        onProgress: (p0) {
          setState(() {
            isImporting = true;
            importProgressText = p0.step;
            importProgressPercent = p0.progress;
          });
        },
      );

      ref.invalidate(entriesForReadingOrderProvider);
      ref.invalidate(readingOrderProgressProvider);

      if (!mounted) return;

      final successCount = importResult.successes.length;
      final failureCount = importResult.failures.length;

      final snackbarMessage = failureCount == 0
          ? "Imported $successCount entries successfully."
          : "Imported $successCount entries, $failureCount failed.";

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          duration: Duration(minutes: 20),
          content: Text(snackbarMessage),
          action: SnackBarAction(
            label: 'Details',
            onPressed: () => _showCsvImportDetailDialog(importResult),
          ),
        ),
      );
    } on Exception catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(getErrorSnackbar(e));
      }
    } finally {
      setState(() {
        isImporting = false;
        importProgressText = "";
        importProgressPercent = 0;
        importCancelationToken = null;
      });
    }
  }

  Future<void> _updateReadingOrderEntry(ReadingOrderEntry entry) async {
    try {
      final updatedEntry = await ref
          .read(entriesForReadingOrderProvider(widget.readingOrder.id).notifier)
          .updateEntry(entry);

      if (!mounted) {
        return;
      }

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          backgroundColor: Colors.green,
          content: Text(
            "Successfully updated entry ${updatedEntry.comic?.title}",
          ),
        ),
      );
    } on Exception catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(getErrorSnackbar(e));
    }
  }

  Future<void> _createReadingOrderEntry(ReadingOrderEntry entry) async {
    try {
      final newEntry = await ref
          .read(entriesForReadingOrderProvider(widget.readingOrder.id).notifier)
          .createEntry(entry);

      if (!mounted) {
        return;
      }

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          backgroundColor: Colors.green,
          content: Text("Successfully added entry ${newEntry.comic?.title}"),
        ),
      );
    } on Exception catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(getErrorSnackbar(e));
    }
  }

  Future<void> _openAddComicPopup() async {
    var comic = await _showSearchDialog();

    if (comic == null) {
      return;
    }

    final readingOrderEntry = await _showEditDialog(
      ReadingOrderEntry(
        id: '',
        readingOrderId: widget.readingOrder.id,
        position: 0,
        comic: comic,
      ),
    );

    if (readingOrderEntry == null) {
      return;
    }

    _createReadingOrderEntry(readingOrderEntry);
  }

  Future<ReadingOrderEntry?> _showEditDialog(ReadingOrderEntry entry) {
    return showDialog<ReadingOrderEntry>(
      context: context,
      builder: (BuildContext context) {
        return EntryEditDialog(
          entry: ReadingOrderEntry(
            id: entry.id,
            readingOrderId: entry.readingOrderId,
            position: entry.position,
            comic: entry.comic,
            notes: entry.notes,
          ),
        );
      },
    );
  }

  Future<Comic?> _showSearchDialog() {
    return showDialog<Comic>(
      context: context,
      builder: (context) {
        return ComicSearchDialog();
      },
    );
  }

  Future<String?> _getCsvFileContent() async {
    FilePickerResult? result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['csv'],
    );
    if (result == null || result.files.single.path == null) return null;

    if (kIsWeb) {
      final bytes = result.files.single.bytes;
      if (bytes == null) return null;
      return utf8.decode(bytes);
    }

    final file = File(result.files.single.path!);
    return file.readAsString();
  }

  void _showCsvImportDetailDialog(ImportResult importResult) {
    showDialog(
      context: context,
      builder: (context) {
        return _CsvImportDetailDialog(importResult: importResult);
      },
    );
  }
}

class _CsvImportDetailDialog extends StatelessWidget {
  const _CsvImportDetailDialog({required this.importResult});

  final ImportResult importResult;

  @override
  Widget build(BuildContext context) {
    final successCount = importResult.successes.length;
    final failureCount = importResult.failures.length;
    final hasFailures = failureCount > 0;
    final tabCount = hasFailures ? 2 : 1;

    return AlertDialog(
      title: const Text("CSV Import Details"),
      content: SizedBox(
        width: 700,
        height: 400, // scrollable box
        child: DefaultTabController(
          length: tabCount,
          child: Column(
            children: [
              TabBar(
                tabs: [
                  Tab(text: "Successful: $successCount"),

                  if (hasFailures) Tab(text: "Failed: $failureCount"),
                ],
              ),
              Expanded(
                child: TabBarView(
                  children: [
                    _CsvImportDetailSuccessList(
                      successCount: successCount,
                      importResult: importResult,
                    ),

                    if (hasFailures)
                      _CsvImportDetailFailureList(
                        failureCount: failureCount,
                        importResult: importResult,
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text("Close"),
        ),
      ],
    );
  }
}

class _CsvImportDetailFailureList extends StatelessWidget {
  const _CsvImportDetailFailureList({
    required this.failureCount,
    required this.importResult,
  });

  final int failureCount;
  final ImportResult importResult;

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      itemCount: failureCount,
      itemBuilder: (context, index) {
        final f = importResult.failures[index];
        final csvData = f.csvRow;

        return ListTile(
          title: Text(csvData.toString(), softWrap: true),
          subtitle: Text(f.reason, softWrap: true),
        );
      },
    );
  }
}

class _CsvImportDetailSuccessList extends StatelessWidget {
  const _CsvImportDetailSuccessList({
    required this.successCount,
    required this.importResult,
  });

  final int successCount;
  final ImportResult importResult;

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      itemCount: successCount,
      itemBuilder: (context, index) {
        final s = importResult.successes[index];
        final entry = s.entry;
        final csvData = s.csvRow;

        return ListTile(
          title: Text(
            "${entry.position}: ${entry.comic?.title}",
            softWrap: true,
          ),
          subtitle: Text("CSV Data: $csvData", softWrap: true),
        );
      },
    );
  }
}

class _CsvImportProgress extends StatelessWidget {
  const _CsvImportProgress({
    required this.importProgressText,
    this.importProgressPercent,
    this.onCancel,
  });

  final String importProgressText;
  final double? importProgressPercent;
  final Function? onCancel;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(8.0),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  "Importing from CSV - $importProgressText",
                  overflow: TextOverflow.ellipsis,
                  maxLines: 1,
                ),
              ),
              if (onCancel != null)
                TextButton(
                  onPressed: () => onCancel!(),
                  child: const Text('Cancel Import'),
                ),
            ],
          ),
        ),
        LinearProgressIndicator(value: importProgressPercent),
      ],
    );
  }
}

class _ReadingProgress extends ConsumerWidget {
  const _ReadingProgress({required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final progress = ref.watch(readingOrderProgressProvider(readingOrder.id));

    return progress.when(
      data: (progress) {
        final percentage = progress.total == 0
            ? 0.0
            : progress.read / progress.total;

        return Row(
          spacing: 10,
          children: [
            Expanded(child: LinearProgressIndicator(value: percentage)),
          ],
        );
      },
      error: (error, stacktrace) => Text('An error occured: $error'),
      loading: () => Center(child: LinearProgressIndicator()),
    );
  }
}
