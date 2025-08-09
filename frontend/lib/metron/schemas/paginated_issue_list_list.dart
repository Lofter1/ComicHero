import 'package:comichero_frontend/metron/schemas/issue_list.dart';

class PaginatedIssueListList {
  final int count;
  final String next;
  final String previous;
  final List<IssueList> results;

  PaginatedIssueListList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedIssueListList.fromJson(Map<String, dynamic> json) {
    return PaginatedIssueListList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => IssueList.fromJson(item))
          .toList(),
    );
  }
}
