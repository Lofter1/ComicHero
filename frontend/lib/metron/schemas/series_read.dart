import 'package:comichero_frontend/metron/schemas/associated_series.dart';
import 'package:comichero_frontend/metron/schemas/basic_imprint.dart';
import 'package:comichero_frontend/metron/schemas/basic_publisher.dart';
import 'package:comichero_frontend/metron/schemas/genre.dart';
import 'package:comichero_frontend/metron/schemas/series_type.dart';

class SeriesRead {
  final int id;
  final String name;
  final String sortName;
  final int volume;
  final SeriesType seriesType;
  final String status;
  final BasicPublisher publisher;
  final BasicImprint imprint;
  final int yearBegan;
  final int yearEnd;
  final String desc;
  final int issueCount;
  final List<Genre> genres;
  final List<AssociatedSeries> associated;
  final int cvId;
  final int gcdId;
  final String resourceUrl;
  final String modified;

  SeriesRead({
    required this.id,
    required this.name,
    required this.sortName,
    required this.volume,
    required this.seriesType,
    required this.status,
    required this.publisher,
    required this.imprint,
    required this.yearBegan,
    required this.yearEnd,
    required this.desc,
    required this.issueCount,
    required this.genres,
    required this.associated,
    required this.cvId,
    required this.gcdId,
    required this.resourceUrl,
    required this.modified,
  });

  factory SeriesRead.fromJson(Map<String, dynamic> json) {
    return SeriesRead(
      id: json['id'],
      name: json['name'],
      sortName: json['sort_name'],
      volume: json['volume'],
      seriesType: SeriesType.fromJson(json['series_type']),
      status: json['status'],
      publisher: BasicPublisher.fromJson(json['publisher']),
      imprint: BasicImprint.fromJson(json['imprint']),
      yearBegan: json['year_began'],
      yearEnd: json['year_end'],
      desc: json['desc'],
      issueCount: json['issue_count'],
      genres: (json['genres'] as List<dynamic>)
          .map((item) => Genre.fromJson(item))
          .toList(),
      associated: (json['associated'] as List<dynamic>)
          .map((item) => AssociatedSeries.fromJson(item))
          .toList(),
      cvId: json['cv_id'],
      gcdId: json['gcd_id'],
      resourceUrl: json['resource_url'],
      modified: json['modified'],
    );
  }
}
