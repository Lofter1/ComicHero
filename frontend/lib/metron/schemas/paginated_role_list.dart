import 'package:comichero_frontend/metron/schemas/role.dart';

class PaginatedRoleList {
  final int count;
  final String next;
  final String previous;
  final List<Role> results;

  PaginatedRoleList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedRoleList.fromJson(Map<String, dynamic> json) {
    return PaginatedRoleList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => Role.fromJson(item))
          .toList(),
    );
  }
}
