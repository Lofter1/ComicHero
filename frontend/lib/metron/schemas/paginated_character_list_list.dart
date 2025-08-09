import 'package:comichero_frontend/metron/metron.dart';

class PaginatedCharacterListList {
  final int count;
  final String next;
  final String previous;
  final List<CharacterList> results;

  PaginatedCharacterListList({
    required this.count,
    required this.next,
    required this.previous,
    required this.results,
  });

  factory PaginatedCharacterListList.fromJson(Map<String, dynamic> json) {
    return PaginatedCharacterListList(
      count: json['count'],
      next: json['next'] ?? '',
      previous: json['previous'] ?? '',
      results: (json['results'] as List<dynamic>)
          .map((item) => CharacterList.fromJson(item))
          .toList(),
    );
  }
}
