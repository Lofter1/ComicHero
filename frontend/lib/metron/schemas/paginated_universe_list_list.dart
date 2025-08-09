import 'package:comichero_frontend/metron/schemas/universe_list.dart';

class PaginatedUniverseListList {
  final int count;
  final String next;
  final String previous;
  final List<UniverseList> results;

  PaginatedUniverseListList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedUniverseListList.fromJson(Map<String, dynamic> json) {
    return PaginatedUniverseListList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => UniverseList.fromJson(item))
          .toList(),
    );
  }
}
