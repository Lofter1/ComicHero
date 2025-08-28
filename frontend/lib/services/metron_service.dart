import 'package:comichero_frontend/app_config.dart';
import 'package:comichero_frontend/metron/metron.dart';
import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/services/services.dart';

class MetronService {
  late MetronApi _metronClient;

  MetronService() {
    final metronUri = Uri.parse(
      AppConfig.apiProxyUrl.endsWith('/')
          ? '${AppConfig.apiProxyUrl}metron'
          : '${AppConfig.apiProxyUrl}/metron',
    );

    _metronClient = MetronApi(
      baseUrl: metronUri.toString(),
      customHeaders: <String, String>{"pb_auth": pb.authStore.token},
      requestDelaySeconds: 0,
    );
  }

  Future<PaginatedSeriesListList> searchSeries(String query) {
    final (seriesName, startingYear) = _splitSeriesName(query);
    return _metronClient.series(
      SeriesParameters(
        name: seriesName,
        yearBegan: int.tryParse(startingYear ?? ''),
      ),
    );
  }

  Future<List<Comic>> searchComic(String searchQuery) async {
    final (seriesName, startingYear, issueNumber) = _splitComicName(
      searchQuery,
    );
    return (await _metronClient.issue(
      IssueParameters(
        seriesName: seriesName,
        number: issueNumber,
        seriesYearBegan: int.tryParse(startingYear ?? ''),
      ),
    )).results.map(_mapIssueListToComic).toList();
  }

  Future<List<Comic>> getIssueList({
    String? seriesName,
    int? seriesYearBegan,
    String? issue,
    int? coverMonth,
    int? coverYear,
    bool loadAll = false,
    int page = 1,
  }) async {
    PaginatedIssueListList result;
    List<Comic> comicList = [];

    do {
      result = await _metronClient.issue(
        IssueParameters(
          seriesName: seriesName,
          seriesYearBegan: seriesYearBegan,
          number: issue,
          page: page,
          coverMonth: coverMonth,
          coverYear: coverYear,
        ),
      );
      comicList.addAll(result.results.map(_mapIssueListToComic));
      page++;
    } while (loadAll && result.next.isNotEmpty);

    return comicList;
  }

  Future<Comic> getComicById(int id, {bool includeMetronId = false}) async {
    final metronIssue = await _metronClient.issueById(id);
    return _mapIssueReadToComic(metronIssue, includeMetronId: includeMetronId);
  }

  Comic _mapIssueReadToComic(
    IssueRead metronIssue, {
    bool includeMetronId = false,
  }) {
    return Comic(
      id: includeMetronId ? metronIssue.id.toString() : "",
      seriesName: metronIssue.series!.name,
      seriesYearBegan: metronIssue.series!.yearBegan,
      issue: metronIssue.number,
      coverUrl: metronIssue.image,
      description: metronIssue.desc,
      coverDate: DateTime.tryParse(metronIssue.coverDate),
    );
  }

  Comic _mapIssueListToComic(IssueList c) {
    return Comic(
      id: '',
      seriesName: c.series.name,
      seriesYearBegan: c.series.yearBegan,
      issue: c.number,
      coverUrl: c.image,
      coverDate: DateTime.tryParse(c.coverDate),
    );
  }

  (String seriesName, String? seriesStartYear, String? issueNumber)
  _splitComicName(String input) {
    final regex = RegExp(r'^(.*?)(?:\s+\((\d{4})\))?(?:\s+#?(\d+))?$');
    final match = regex.firstMatch(input.trim());

    if (match != null) {
      final name = match.group(1)?.trim() ?? '';
      final year = match.group(2); // optional 4-digit year
      final issue = match.group(3); // optional issue number
      return (name, year, issue);
    }

    return (input.trim(), null, null);
  }

  (String seriesName, String? yearBegan) _splitSeriesName(String input) {
    final regex = RegExp(r'^(.*?)(?:\s*\((\d{4})\))?$');
    final match = regex.firstMatch(input);

    if (match != null) {
      final title = match.group(1)?.trim() ?? '';
      final year = match.group(2); // might be null
      return (title, year);
    }

    // fallback: just return whole string as title
    return (input, null);
  }
}
