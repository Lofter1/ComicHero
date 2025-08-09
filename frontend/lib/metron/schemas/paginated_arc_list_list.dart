import 'package:comichero_frontend/metron/metron.dart';

class PaginatedArcListList {
  final int count;
  final String next;
  final String previous;
  final List<ArcList> results;

  PaginatedArcListList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedArcListList.fromJson(Map<String, dynamic> json) {
    return PaginatedArcListList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => ArcList.fromJson(item))
          .toList(),
    );
  }
}
