import 'package:comichero_frontend/metron/schemas/series_list.dart';

class PaginatedSeriesListList {
  final int count;
  final String next;
  final String previous;
  final List<SeriesList> results;

  PaginatedSeriesListList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedSeriesListList.fromJson(Map<String, dynamic> json) {
    return PaginatedSeriesListList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => SeriesList.fromJson(item))
          .toList(),
    );
  }
}
