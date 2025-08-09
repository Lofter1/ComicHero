import 'package:comichero_frontend/metron/schemas/creator_list.dart';

class PaginatedCreatorListList {
  final int count;
  final String next;
  final String previous;
  final List<CreatorList> results;

  PaginatedCreatorListList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedCreatorListList.fromJson(Map<String, dynamic> json) {
    return PaginatedCreatorListList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => CreatorList.fromJson(item))
          .toList(),
    );
  }
}
