import 'package:comichero_frontend/metron/schemas/publisher_list.dart';

class PaginatedPublisherListList {
  final int count;
  final String next;
  final String previous;
  final List<PublisherList> results;

  PaginatedPublisherListList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedPublisherListList.fromJson(Map<String, dynamic> json) {
    return PaginatedPublisherListList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => PublisherList.fromJson(item))
          .toList(),
    );
  }
}
