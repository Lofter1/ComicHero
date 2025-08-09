import 'package:comichero_frontend/metron/schemas/series_list.dart';

class PaginatedSeriesTypeList {
  final int count;
  final String next;
  final String previous;
  final List<SeriesList> results;

  PaginatedSeriesTypeList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedSeriesTypeList.fromJson(Map<String, dynamic> json) {
    return PaginatedSeriesTypeList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => SeriesList.fromJson(item))
          .toList(),
    );
  }
}
