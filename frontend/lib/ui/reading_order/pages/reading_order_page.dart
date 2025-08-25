import 'dart:convert';
import 'dart:io';
import 'package:csv/csv.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:loader_overlay/loader_overlay.dart';

import 'package:comichero_frontend/providers/reading_order_entries_provider.dart';
import 'package:comichero_frontend/providers/reading_order_progress_provider.dart';
import 'package:comichero_frontend/ui/general/error_helpers.dart';
import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/services/services.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ReadingOrderPage extends StatefulWidget {
  const ReadingOrderPage({super.key, required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  State<ReadingOrderPage> createState() => _ReadingOrderPageState();
}

class _ReadingOrderPageState extends State<ReadingOrderPage> {
  @override
  void initState() {
    super.initState();
    _loadReadingOrder();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: ComicHeroAppBar(title: widget.readingOrder.name),
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
                children: [
                  _ReadingOrderEntriesListBody(
                    readingOrder: widget.readingOrder,
                  ),
                  ReadingOrderDetailBody(readingOrder: widget.readingOrder),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _loadReadingOrder() {
    setState(() {});
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
  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ReadingOrderToolbar(
          onRefresh: _onRefresh,
          onAddEntry: _openAddComicPopup,
          onCsvImport: _onCsvImport,
        ),

        AuthGuard(
          loggedInView: (context) =>
              _Progress(readingOrder: widget.readingOrder),
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

  void _onCsvImport() async {
    await _handleCsvImport();
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

  List<_CsvReadingOrderEntry>? _parseCsv(String csvContent) {
    csvContent = csvContent.replaceAll('\n', '\r\n');
    final csv = const CsvToListConverter().convert(csvContent);
    if (csv.isEmpty || csv.length < 2) {
      return null;
    }

    var headerIndex = _getHeaderIndex(csv);
    if (headerIndex == null) return null;

    return csv
        .skip(1)
        .map(
          (row) => _CsvReadingOrderEntry(
            position: row[headerIndex['Position']!] as int,
            seriesName: row[headerIndex['SeriesName']!].toString(),
            yearBegan: row[headerIndex['YearBegin']!] is int?
                ? row[headerIndex['YearBegin']!] as int?
                : null,
            issueNumber: row[headerIndex['Issue']!].toString(),
            coverMonth: row[headerIndex['CoverMonth']!] is int?
                ? row[headerIndex['CoverMonth']!] as int?
                : null,
            coverYear: row[headerIndex['CoverYear']!] is int?
                ? row[headerIndex['CoverYear']!] as int?
                : null,
          ),
        )
        .toList();
  }

  Future<String?> _getCsvString() async {
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

  Map<String, int>? _getHeaderIndex(List<List<dynamic>> csvData) {
    final headers = csvData.first.map((h) => h.toString().trim()).toList();
    final headerIndex = {
      for (int i = 0; i < headers.length; i++) headers[i]: i,
    };

    const requiredHeaders = [
      'Position',
      'SeriesName',
      'YearBegin',
      'Issue',
      'CoverMonth',
      'CoverYear',
    ];
    for (final header in requiredHeaders) {
      if (!headerIndex.containsKey(header)) {
        return null;
      }
    }
    return headerIndex;
  }

  Map<String, List<_CsvReadingOrderEntry>> _groupByFullSeriesName(
    List<_CsvReadingOrderEntry> listToGroup,
  ) {
    final Map<String, List<_CsvReadingOrderEntry>> group = {};

    for (final entry in listToGroup) {
      group.putIfAbsent(entry.fullSeriesName, () => []).add(entry);
    }

    return group;
  }

  Future<void> _handleCsvImport() async {
    try {
      context.loaderOverlay.show();
      final contents = await _getCsvString();
      if (contents == null) return;

      var csvEntryList = _parseCsv(contents);
      if (csvEntryList == null || csvEntryList.isEmpty) return;

      List<ReadingOrderEntry> preparedReadingOrderEntriesFromDb = [];
      List<ReadingOrderEntry> preparedReadingOrderEntriesFromMetron = [];
      List<_CsvReadingOrderEntry> entriesNotFoundInDb = [];
      List<_CsvReadingOrderEntry> entriesNotFoundInMetron = [];

      await _searchComicInDb(
        searchCsvEntries: csvEntryList,
        notFoundEntries: entriesNotFoundInDb,
        foundEntries: preparedReadingOrderEntriesFromDb,
      );
      await _searchInMetron(
        searchCsvEntries: entriesNotFoundInDb,
        notFoundEntries: entriesNotFoundInMetron,
        foundEntries: preparedReadingOrderEntriesFromMetron,
      );

      if (mounted) context.loaderOverlay.hide();

      if (entriesNotFoundInMetron.isNotEmpty) {
        bool? continueImport = await _promptContinueWithMissingData(
          entriesNotFoundInMetron,
        );

        if (continueImport == null || continueImport == false) {
          if (mounted) {
            ScaffoldMessenger.of(
              context,
            ).showSnackBar(SnackBar(content: Text("Canceled CSV import")));
          }
          return;
        }
      }

      if (mounted) {
        context.loaderOverlay.show();
      }
      //TODO: improve performance
      for (final metronComicEntry in preparedReadingOrderEntriesFromMetron) {
        var newDbComic = await ComicService().create(metronComicEntry.comic!);
        metronComicEntry.comic = newDbComic;
        preparedReadingOrderEntriesFromDb.add(metronComicEntry);
      }

      for (final existignComicEntry in preparedReadingOrderEntriesFromDb) {
        await ReadingOrderEntriesService().create(existignComicEntry);
      }

      if (mounted) {
        context.loaderOverlay.hide();
      }

      ref.invalidate(entriesForReadingOrderProvider);
      ref.invalidate(readingOrderProgressProvider);

      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text("Importing CSV complete")));
      }
    } on Exception catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(getErrorSnackbar(e));
      }
    } finally {
      if (mounted) context.loaderOverlay.hide();
    }
  }

  Future<bool?> _promptContinueWithMissingData(
    List<_CsvReadingOrderEntry> missingData,
  ) async {
    return await showDialog<bool>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: Text("Not able to find all entries"),
          content: SizedBox(
            width: 700,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              spacing: 10,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('The Following entries could not be found'),
                Flexible(
                  child: SizedBox(
                    height: 400,
                    child: ListView.separated(
                      itemBuilder: (context, index) {
                        return ListTile(
                          subtitle: Text(
                            "Position: ${missingData[index].position}",
                          ),
                          title: Text(missingData[index].issueName),
                        );
                      },
                      separatorBuilder: (context, index) {
                        return Divider();
                      },
                      itemCount: missingData.length,
                    ),
                  ),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () {
                Navigator.pop(context, false);
              },
              child: Text("Cancel Import"),
            ),
            TextButton(
              onPressed: () {
                Navigator.pop(context, true);
              },
              child: Text("Import with found data"),
            ),
          ],
        );
      },
    );
  }

  Future<void> _searchInMetron({
    required List<_CsvReadingOrderEntry> searchCsvEntries,
    required List<_CsvReadingOrderEntry> notFoundEntries,
    required List<ReadingOrderEntry> foundEntries,
  }) async {
    final groupedMissingEntries = _groupByFullSeriesName(searchCsvEntries);

    for (final seriesGroup in groupedMissingEntries.values) {
      final metronSeriesContent = await MetronService().getIssueList(
        seriesName: seriesGroup.first.seriesName,
        seriesYearBegan: seriesGroup.first.yearBegan,
        loadAll: true,
      );

      if (metronSeriesContent.isEmpty) {
        notFoundEntries.addAll(seriesGroup);
        continue;
      }

      for (final entry in seriesGroup) {
        Comic? foundComic;

        var results = metronSeriesContent
            .where(
              (e) =>
                  e.title.toLowerCase().contains(
                    entry.issueName.toLowerCase(),
                  ) &&
                  (entry.coverYear == null ||
                      e.releaseDate?.year == entry.coverYear) &&
                  (entry.coverMonth == null ||
                      e.releaseDate?.month == entry.coverMonth),
            )
            .toList();

        if (results.isEmpty &&
            entry.coverYear != null &&
            entry.coverMonth != null) {
          results = await MetronService().getIssueList(
            seriesName: entry.seriesName,
            coverYear: entry.coverYear,
            coverMonth: entry.coverMonth,
            loadAll: true,
          );
        }

        if (results.length > 1) {
          if (mounted) context.loaderOverlay.hide();
          foundComic = await _promptMultipleEntriesFoundInMetron(
            entryName: entry.issueName,
            metronResults: results,
          );
          if (mounted) context.loaderOverlay.show();
        } else {
          foundComic = results.firstOrNull;
        }

        if (foundComic == null) {
          notFoundEntries.add(entry);
        } else {
          foundEntries.add(
            ReadingOrderEntry(
              id: '',
              readingOrderId: widget.readingOrder.id,
              position: entry.position,
              comic: foundComic,
            ),
          );
        }
      }
    }
  }

  Future<void> _searchComicInDb({
    required List<_CsvReadingOrderEntry> searchCsvEntries,
    required List<_CsvReadingOrderEntry> notFoundEntries,
    required List<ReadingOrderEntry> foundEntries,
  }) async {
    for (final csvEntry in searchCsvEntries) {
      Comic? foundComic;
      final results = await ComicService().get(
        seriesName: csvEntry.seriesName,
        seriesYearBegan: csvEntry.yearBegan,
        issue: csvEntry.issueNumber,
        releaseDate: csvEntry.coverYear != null && csvEntry.coverMonth != null
            ? DateTime(csvEntry.coverYear!, csvEntry.coverMonth!, 1)
            : null,
      );

      if (results.length > 1) {
        if (mounted) context.loaderOverlay.hide();

        foundComic = await _promptMultipleEntriesFoundInDb(
          entryName: csvEntry.issueName,
          dbResults: results,
        );

        if (mounted) context.loaderOverlay.show();
      } else {
        foundComic = results.firstOrNull;
      }

      if (foundComic == null) {
        notFoundEntries.add(csvEntry);
      } else {
        foundEntries.add(
          ReadingOrderEntry(
            id: '',
            readingOrderId: widget.readingOrder.id,
            position: csvEntry.position,
            comic: foundComic,
          ),
        );
      }
    }
  }

  Future<Comic?> _promptMultipleEntriesFoundInMetron({
    required String entryName,
    required List<Comic> metronResults,
  }) {
    return showDialog<Comic?>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: Text(entryName),
          content: SizedBox(
            width: 700,
            child: Column(
              spacing: 10,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text("Found multiple possible entries in metron"),
                Flexible(
                  child: IssueSearchResultView(
                    searchResults: metronResults,
                    onComicSelected: (comic) {
                      Navigator.pop(context, comic);
                    },
                  ),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: Text('Skip'),
            ),
          ],
        );
      },
    );
  }

  Future<Comic?> _promptMultipleEntriesFoundInDb({
    required String entryName,
    required List<Comic> dbResults,
  }) {
    return showDialog<Comic?>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: Text(entryName),
          content: SizedBox(
            width: 700,
            child: Column(
              spacing: 10,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text("Found multiple possible entries in database"),
                Flexible(
                  child: IssueSearchResultView(
                    searchResults: dbResults,
                    onComicSelected: (comic) {
                      Navigator.pop(context, comic);
                    },
                  ),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: Text('Skip'),
            ),
          ],
        );
      },
    );
  }
}

class _CsvReadingOrderEntry {
  final int position;
  final String seriesName;
  final int? yearBegan;
  final String issueNumber;
  final int? coverMonth;
  final int? coverYear;

  _CsvReadingOrderEntry({
    required this.position,
    required this.seriesName,
    required this.issueNumber,
    this.yearBegan,
    this.coverMonth,
    this.coverYear,
  });

  String get fullSeriesName =>
      "$seriesName${yearBegan != null ? " ($yearBegan)" : ""}";
  String get issueName => "$fullSeriesName #$issueNumber";
}

class _Progress extends ConsumerWidget {
  const _Progress({required this.readingOrder});

  final ReadingOrder readingOrder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final progress = ref.watch(readingOrderProgressProvider(readingOrder.id));

    // final percentFormat = NumberFormat.decimalPercentPattern(
    //   locale: Intl.defaultLocale,
    //   decimalDigits: 2,
    // );

    return progress.when(
      data: (progress) {
        final percentage = progress.total == 0
            ? 0.0
            : progress.read / progress.total;
        // final progressText =
        //     '${progress.read} / ${progress.total} (${percentFormat.format(percentage)})';

        return Row(
          spacing: 10,
          children: [
            // Text(progressText),
            Expanded(child: LinearProgressIndicator(value: percentage)),
          ],
        );
      },
      error: (error, stacktrace) => Text('An error occured: $error'),
      loading: () => Center(child: LinearProgressIndicator()),
    );
  }
}
