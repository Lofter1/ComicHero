import 'package:comichero_frontend/metron/schemas/imprint_list.dart';

class PaginatedImprintListList {
  final int count;
  final String next;
  final String previous;
  final List<ImprintList> results;

  PaginatedImprintListList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedImprintListList.fromJson(Map<String, dynamic> json) {
    return PaginatedImprintListList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => ImprintList.fromJson(item))
          .toList(),
    );
  }
}
