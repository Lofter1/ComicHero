import 'package:comichero_frontend/metron/schemas/genre.dart';
import 'package:comichero_frontend/metron/schemas/series_type.dart';

class IssueSeries {
  final int id;
  final String name;
  final String sortName;
  final int volume;
  final int yearBegan;
  final SeriesType seriesType;
  final List<Genre> genres;

  IssueSeries({
    required this.id,
    required this.name,
    required this.sortName,
    required this.volume,
    required this.yearBegan,
    required this.seriesType,
    required this.genres,
  });

  factory IssueSeries.fromJson(Map<String, dynamic> json) {
    return IssueSeries(
      id: json['id'],
      name: json['name'],
      sortName: json['sort_name'],
      volume: json['volume'],
      yearBegan: json['year_began'],
      seriesType: SeriesType.fromJson(json['series_type']),
      genres: (json['genres'] as List<dynamic>? ?? [])
          .map((e) => Genre.fromJson(e))
          .toList(),
    );
  }
}
