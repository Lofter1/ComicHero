import 'package:collection/collection.dart';
import 'package:csv/csv.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/services/services.dart';

class CsvImportService {
  Future<ImportResult> importCsv(
    String csvString,
    String readingOrderId, {
    void Function(ImportProgress)? onProgress,
    ImportCancelationToken? cancelationToken,
  }) async {
    onProgress?.call(ImportProgress(step: "Parsing CSV"));
    final parsedCsvData = _parseCsv(csvString);
    if (parsedCsvData == null || cancelationToken?.isCancelled == true) {
      return ImportResult(successes: [], failures: []);
    }

    final groupedBySeries = _groupBySeries(parsedCsvData);
    if (cancelationToken?.isCancelled == true) {
      return ImportResult(successes: [], failures: []);
    }

    onProgress?.call(ImportProgress(step: "Fetching comics from backend"));
    final comicLookup = await _fetchGroupedBySeries(
      groupedBySeries,
      onProgress: onProgress,
      cancelationToken: cancelationToken,
    );
    if (cancelationToken?.isCancelled == true) {
      return ImportResult(successes: [], failures: []);
    }

    onProgress?.call(ImportProgress(step: "Creating entries"));
    return _createEntries(
      parsedCsvData,
      comicLookup,
      readingOrderId,
      onProgress: onProgress,
    );
  }

  Future<ImportResult> _createEntries(
    List<ParsedCsvRow> parsedCsvData,
    Map<String, Comic> comicLookup,
    String readingOrderId, {
    Function(ImportProgress)? onProgress,
    ImportCancelationToken? cancelationToken,
  }) async {
    final successes = <SuccessfulImport>[];
    final failures = <FailedImport>[];

    for (final row in parsedCsvData) {
      if (cancelationToken != null && cancelationToken.isCancelled) {
        return ImportResult(successes: successes, failures: failures);
      }

      onProgress?.call(
        ImportProgress(
          step: "Creating entries - ${row.comicKey}",
          progress: (successes.length + failures.length) / parsedCsvData.length,
        ),
      );
      var comic = comicLookup[_normalizeSeriesName(row.comicKey)];
      if (comic == null) {
        failures.add(FailedImport(csvRow: row, reason: "Comic not found."));

        continue;
      }

      try {
        if (comic.id.isEmpty) {
          comic = await ComicService().create(comic);
        }

        final entry = ReadingOrderEntry(
          id: '',
          readingOrderId: readingOrderId,
          position: row.position,
          comic: comic,
          notes: row.notes,
        );

        final createdEntry = await ReadingOrderEntriesService().create(entry);
        successes.add(SuccessfulImport(csvRow: row, entry: createdEntry));
      } catch (e) {
        failures.add(
          FailedImport(csvRow: row, reason: "Failed to create entry: $e"),
        );
      }
    }

    return ImportResult(successes: successes, failures: failures);
  }

  Future<Map<String, Comic>> _fetchGroupedBySeries(
    Map<String, List<ParsedCsvRow>> groupedBySeries, {
    Function(ImportProgress)? onProgress,
    ImportCancelationToken? cancelationToken,
  }) async {
    final comicLookup = <String, Comic>{};

    for (final entry in groupedBySeries.entries) {
      if (cancelationToken != null && cancelationToken.isCancelled) {
        return comicLookup;
      }
      final comics = await ComicService().get(
        seriesName: entry.value.first.seriesName,
        seriesYearBegan: entry.value.first.seriesYearBegan,
      );

      final notFoundRows = <ParsedCsvRow>[];

      for (final row in entry.value) {
        final match = comics.firstWhereOrNull(
          (comic) =>
              _normalizeSeriesName(row.seriesName) ==
                  _normalizeSeriesName(comic.seriesName) &&
              row.seriesYearBegan == comic.seriesYearBegan &&
              row.issue == comic.issue &&
              row.coverDate?.year == comic.coverDate?.year &&
              row.coverDate?.month == comic.coverDate?.month,
        );
        if (match != null) {
          comicLookup[_normalizeSeriesName(match.title)] = match;
        } else {
          notFoundRows.add(row);
        }
      }

      if (cancelationToken != null && cancelationToken.isCancelled) {
        return comicLookup;
      }

      if (notFoundRows.isNotEmpty) {
        final metronComics = await MetronService().getIssueList(
          seriesName: entry.value.first.seriesName,
          seriesYearBegan: entry.value.first.seriesYearBegan,
          loadAll: true,
        );

        for (final row in notFoundRows) {
          final match = metronComics.firstWhereOrNull(
            (comic) =>
                _normalizeSeriesName(row.seriesName) ==
                    _normalizeSeriesName(comic.seriesName) &&
                row.seriesYearBegan == comic.seriesYearBegan &&
                row.issue == comic.issue &&
                row.coverDate?.year == comic.coverDate?.year &&
                row.coverDate?.month == comic.coverDate?.month,
          );
          if (match != null) {
            comicLookup[_normalizeSeriesName(match.title)] = match;
          }
        }
      }
    }
    return comicLookup;
  }

  Map<String, List<ParsedCsvRow>> _groupBySeries(List<ParsedCsvRow> csvData) {
    final Map<String, List<ParsedCsvRow>> groupedBySeries = {};
    for (final row in csvData) {
      groupedBySeries.putIfAbsent(row.seriesKey, () => []).add(row);
    }
    return groupedBySeries;
  }

  List<ParsedCsvRow>? _parseCsv(String csvString) {
    csvString = _normaliseCsv(csvString);

    final rows = const CsvToListConverter().convert(
      csvString,
      shouldParseNumbers: false,
    );

    if (rows.isEmpty) return null;

    final headers = rows.first.map((h) => h.toString().trim()).toList();

    for (final header in _CsvColumnDefinitions.required) {
      if (!headers.contains(header)) {
        throw FormatException("Missing expected column: $header");
      }
    }

    final hasCoverDate = _CsvColumnDefinitions.coverDateSet.every(
      headers.contains,
    );
    final hasCoverMonthYear = _CsvColumnDefinitions.coverMonthYearSet.every(
      headers.contains,
    );

    if (!hasCoverDate && !hasCoverMonthYear) {
      throw FormatException(
        "Missing expected cover information: need either CoverDate OR (CoverMonth and CoverYear)",
      );
    }

    final dataRows = rows.skip(1);
    final parsedRowList = <ParsedCsvRow>[];

    for (final row in dataRows) {
      final rowMap = _toRowMap(headers, row);
      final parsedRow = ParsedCsvRow.fromMap(rowMap);

      if (parsedRow != null) {
        parsedRowList.add(parsedRow);
      }
    }
    return parsedRowList;
  }

  Map<String, String> _toRowMap(List<String> headers, List<dynamic> row) {
    final map = <String, String>{};
    for (int i = 0; i < headers.length && i < row.length; i++) {
      map[headers[i]] = row[i].toString();
    }
    return map;
  }

  static String _normalizeSeriesName(String name) {
    var normalized = name.trim().toLowerCase();
    if (normalized.startsWith("the ")) {
      normalized = normalized.substring(4); // remove leading "the "
    }
    return normalized;
  }

  static String _normaliseCsv(String csv) {
    return csv.replaceAll('\n', '\r\n');
  }
}

class ImportCancelationToken {
  bool isCancelled = false;

  void cancel() {
    isCancelled = true;
  }
}

class ImportProgress {
  final String step;
  final double? progress;

  ImportProgress({required this.step, this.progress});
}

class ImportResult {
  final List<SuccessfulImport> successes;
  final List<FailedImport> failures;

  ImportResult({required this.successes, required this.failures});
}

class SuccessfulImport {
  final ParsedCsvRow csvRow;
  final ReadingOrderEntry entry;

  SuccessfulImport({required this.csvRow, required this.entry});
}

class FailedImport {
  final ParsedCsvRow csvRow;
  final String reason;

  FailedImport({required this.csvRow, required this.reason});
}

class _CsvColumnDefinitions {
  static const seriesName = "SeriesName";
  static const seriesYearBegan = "SeriesYearBegan";
  static const issue = "Issue";
  static const coverDate = "CoverDate";
  static const coverMonth = "CoverMonth";
  static const coverYear = "CoverYear";
  static const position = "Position";
  static const notes = "Notes";

  static const required = [seriesName, seriesYearBegan, issue, position];

  static const coverDateSet = [coverDate];
  static const coverMonthYearSet = [coverMonth, coverYear];
}

class ParsedCsvRow {
  final String seriesName;
  final String issue;
  final int position;
  final DateTime? coverDate;
  final int? seriesYearBegan;
  final String? notes;

  String get seriesKey =>
      "$seriesName${seriesYearBegan != null ? " ($seriesYearBegan)" : ""}";
  String get comicKey => "$seriesKey${issue.isNotEmpty ? " #$issue" : ""}";

  ParsedCsvRow({
    required this.seriesName,
    required this.issue,
    required this.position,
    this.coverDate,
    this.seriesYearBegan,
    this.notes,
  });

  @override
  String toString() {
    var str = "$position - $seriesName";

    if (seriesYearBegan != null) {
      str += " - $seriesYearBegan";
    }

    if (issue.isNotEmpty) {
      str += " - $issue";
    }

    if (coverDate != null) {
      final month = coverDate!.month.toString().padLeft(2, '0');
      final year = coverDate!.year;
      str += " - $month/$year";
    }

    if (notes != null && notes!.isNotEmpty) {
      str += " - Notes: $notes";
    }

    return str;
  }

  static ParsedCsvRow? fromMap(Map<String, String> map) {
    final seriesName = map[_CsvColumnDefinitions.seriesName];
    final seriesYearBegan = map[_CsvColumnDefinitions.seriesYearBegan];
    final issue = map[_CsvColumnDefinitions.issue];
    final coverDateStr = map[_CsvColumnDefinitions.coverDate];
    final coverMonthStr = map[_CsvColumnDefinitions.coverMonth];
    final coverYearStr = map[_CsvColumnDefinitions.coverYear];
    final positionStr = map[_CsvColumnDefinitions.position];
    final notes = map[_CsvColumnDefinitions.notes]; // may be null

    if (seriesName == null || issue == null || positionStr == null) {
      return null;
    }

    final position = int.tryParse(positionStr);
    if (position == null) return null;

    DateTime? coverDate;
    if (coverDateStr != null && coverDateStr.isNotEmpty) {
      coverDate = DateTime.tryParse(coverDateStr);
    } else if (coverMonthStr != null && coverYearStr != null) {
      final month = int.tryParse(coverMonthStr);
      final year = int.tryParse(coverYearStr);
      if (month != null && year != null) {
        coverDate = DateTime(year, month);
      }
    }

    return ParsedCsvRow(
      seriesName: seriesName,
      issue: issue,
      position: position,
      coverDate: coverDate,
      seriesYearBegan: int.tryParse(seriesYearBegan ?? ""),
      notes: notes,
    );
  }
}
