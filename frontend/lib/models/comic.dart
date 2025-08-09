import 'package:pocketbase/pocketbase.dart';

class Comic {
  final String id;
  final String seriesName;
  final int seriesYearBegan;
  final String issue;
  final DateTime? releaseDate;
  final String? description;
  final String? marvelUrl;
  final String? userComicId;
  final bool? read;
  final bool? skipped;
  late String? coverUrl;

  Comic({
    required this.id,
    required this.seriesName,
    required this.issue,
    required this.seriesYearBegan,
    this.releaseDate,
    this.description,
    this.coverUrl,
    this.marvelUrl,
    this.userComicId,
    this.read,
    this.skipped,
  });

  String get title =>
      "$seriesName ($seriesYearBegan) ${issue.isNotEmpty ? "#$issue" : ""}";

  factory Comic.fromRecord(RecordModel record) =>
      Comic.fromJson(record.toJson());

  factory Comic.fromJson(Map<String, dynamic> json) {
    final userComics = json['expand']?['userComics_via_comic'];
    final userComicsId =
        (userComics != null && userComics is List && userComics.isNotEmpty)
        ? userComics[0]['id']
        : null;
    final read =
        (userComics != null && userComics is List && userComics.isNotEmpty)
        ? userComics[0]['read'] ?? false
        : false;
    final skipped =
        (userComics != null && userComics is List && userComics.isNotEmpty)
        ? userComics[0]['skipped'] ?? false
        : false;

    return Comic(
      id: json['id'],
      seriesName: json['seriesName'],
      seriesYearBegan: json['seriesYearBegan'],
      issue: json['issue'],
      releaseDate: DateTime.tryParse(json['releaseDate']),
      description: json['description'],
      marvelUrl: json['marvelUnlimitedUrl'],
      userComicId: userComicsId,
      read: read,
      skipped: skipped,
    );
  }
}
