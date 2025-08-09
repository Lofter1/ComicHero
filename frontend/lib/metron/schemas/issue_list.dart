import 'package:comichero_frontend/metron/schemas/issue_list_series.dart';

class IssueList {
  final int id;
  final IssueListSeries series;
  final String number;
  final String issue;
  final String coverDate;
  final String storeDate;
  final String image;
  final String coverHash;
  final String modified;

  IssueList({
    required this.id,
    required this.series,
    required this.number,
    required this.issue,
    required this.coverDate,
    required this.storeDate,
    required this.image,
    required this.coverHash,
    required this.modified,
  });

  factory IssueList.fromJson(Map<String, dynamic> json) {
    return IssueList(
      id: json['id'],
      series: IssueListSeries.fromJson(json['series']),
      number: json['number'] ?? '',
      issue: json['issue'] ?? '',
      coverDate: json['cover_date'] ?? '',
      storeDate: json['store_date'] ?? '',
      image: json['image'] ?? '',
      coverHash: json['cover_hash'] ?? '',
      modified: json['modified'] ?? '',
    );
  }
}
