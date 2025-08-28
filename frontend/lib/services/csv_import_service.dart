import 'dart:convert';
import 'dart:io';

import 'package:collection/collection.dart';
import 'package:csv/csv.dart';
import 'package:http/http.dart' as http;

import 'package:comichero_frontend/app_config.dart';
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

    final comicLookup = await _fetchGroupedBySeries(
      groupedBySeries,
      onProgress: onProgress,
      cancelationToken: cancelationToken,
    );
    if (cancelationToken?.isCancelled == true) {
      return ImportResult(successes: [], failures: []);
    }

    return _createEntries(
      parsedCsvData,
      comicLookup,
      readingOrderId,
      onProgress: onProgress,
      cancelationToken: cancelationToken,
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
        failures.add(
          FailedImport(
            csvRow: row,
            reason: "Comic not found.",
            tip:
                "Check the series name, series year and the cover date. "
                "If the comic is not available on Metron, "
                "consider helping out the Metron project by adding it to the "
                "Metron database at https://metron.cloud/",
          ),
        );

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

    final totalSeries = groupedBySeries.length;
    var processedSeries = 0;

    for (final entry in groupedBySeries.entries) {
      if (cancelationToken != null && cancelationToken.isCancelled) {
        return comicLookup;
      }

      onProgress?.call(
        ImportProgress(
          step: "Fetching series - ${entry.key}",
          progress: ++processedSeries / totalSeries,
        ),
      );

      final seriesName = entry.value.first.seriesName;
      final seriesYearBegan = entry.value.first.seriesYearBegan;

      final comics = await ComicService().get(
        seriesName: seriesName,
        seriesYearBegan: seriesYearBegan,
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
        onProgress?.call(
          ImportProgress(
            step: "Fetching series - ${entry.key} (Metron)",
            progress: processedSeries / totalSeries,
          ),
        );

        final metronComics = await MetronService().getIssueList(
          seriesName: seriesName,
          seriesYearBegan: seriesYearBegan,
          loadAll: true,
        );

        final metronSearchRows = [...notFoundRows];

        for (final row in metronSearchRows) {
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
            notFoundRows.remove(row);
          }
        }
      }

      if (cancelationToken != null && cancelationToken.isCancelled) {
        return comicLookup;
      }

      if (notFoundRows.isNotEmpty) {
        onProgress?.call(
          ImportProgress(
            step: "Fetching series - ${entry.key} (GCD)",
            progress: processedSeries / totalSeries,
          ),
        );

        // TODO: put in own service
        final gcdSeriesSearchString = seriesName
            .substring(seriesName.indexOf('/') + 1)
            .trim();

        var uriString =
            "${AppConfig.apiProxyUrl}/gcd/series/name/$gcdSeriesSearchString"
            "/year/$seriesYearBegan/";

        final gcdSeriesUri = Uri.parse("$uriString?format=json");
        final seriesResponse = await http.get(
          gcdSeriesUri,
          headers: {"pb_auth": pb.authStore.token},
        );

        if (seriesResponse.statusCode != 200) {
          throw HttpException(
            "Error fetching series from gcd. "
            "Status code ${seriesResponse.statusCode}",
            uri: gcdSeriesUri,
          );
        }

        final data = jsonDecode(seriesResponse.body) as Map<String, dynamic>;
        final results = data['results'] as List<dynamic>;

        Map<String, dynamic>? foundGcdSeries;
        for (final gcdSeries in results) {
          if (_normalizeSeriesName(gcdSeries['name']) ==
              _normalizeSeriesName(seriesName)) {
            foundGcdSeries = gcdSeries;
            break;
          }
        }

        if (foundGcdSeries != null) {
          final activeIssues = foundGcdSeries['active_issues'] ?? [];
          final issueDescriptors = foundGcdSeries['issue_descriptors'] ?? [];

          for (final row in notFoundRows) {
            final issueIndex = issueDescriptors.indexWhere((desc) {
              final numberPart = desc.split(' ').first;
              return numberPart == row.issue;
            });

            if (issueIndex != -1 && issueIndex < activeIssues.length) {
              final issueUrl = activeIssues[issueIndex];

              // Fetch issue details from GCD
              final gcdIssueUri = Uri.parse(issueUrl);

              final issueResponse = await http.get(
                gcdIssueUri,
                headers: {"pb_auth": pb.authStore.token},
              );

              if (issueResponse.statusCode != 200) {
                throw HttpException(
                  "Error fetching issue from gcd. "
                  "Status code ${issueResponse.statusCode}",
                  uri: gcdIssueUri,
                );
              }

              final issueData = jsonDecode(issueResponse.body);

              final gcdMatch = Comic(
                id: '',
                seriesName: seriesName,
                issue: row.issue,
                seriesYearBegan: seriesYearBegan,
                coverUrl: issueData['cover'],
                coverDate: row.coverDate,
              );
              comicLookup[_normalizeSeriesName(gcdMatch.title)] = gcdMatch;
            }
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
  final String? tip;

  FailedImport({required this.csvRow, required this.reason, this.tip});
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
  final int seriesYearBegan;
  final String? notes;

  String get seriesKey => "$seriesName ($seriesYearBegan)";
  String get comicKey => "$seriesKey${issue.isNotEmpty ? " #$issue" : ""}";

  ParsedCsvRow({
    required this.seriesName,
    required this.issue,
    required this.position,
    required this.seriesYearBegan,
    this.coverDate,
    this.notes,
  });

  @override
  String toString() {
    var str = "$position - $seriesName - $seriesYearBegan";

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
    final seriesYearBeganStr = map[_CsvColumnDefinitions.seriesYearBegan];
    final issue = map[_CsvColumnDefinitions.issue];
    final coverDateStr = map[_CsvColumnDefinitions.coverDate];
    final coverMonthStr = map[_CsvColumnDefinitions.coverMonth];
    final coverYearStr = map[_CsvColumnDefinitions.coverYear];
    final positionStr = map[_CsvColumnDefinitions.position];
    final notes = map[_CsvColumnDefinitions.notes]; // may be null

    if (seriesName == null ||
        issue == null ||
        positionStr == null ||
        seriesYearBeganStr == null) {
      return null;
    }

    final position = int.tryParse(positionStr);
    if (position == null) return null;
    final seriesYearBegan = int.tryParse(seriesYearBeganStr);
    if (seriesYearBegan == null) return null;

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
      seriesYearBegan: seriesYearBegan,
      notes: notes,
    );
  }
}
