import 'package:comichero_frontend/metron/metron.dart';

class PaginatedTeamListList {
  final int count;
  final String next;
  final String previous;
  final List<TeamList> results;

  PaginatedTeamListList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedTeamListList.fromJson(Map<String, dynamic> json) {
    return PaginatedTeamListList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => TeamList.fromJson(item))
          .toList(),
    );
  }
}
