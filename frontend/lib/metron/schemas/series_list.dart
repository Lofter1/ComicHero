class SeriesList {
  final int id;
  final String series;
  final int yearBegan;
  final int volume;
  final int issueCount;
  final String modified;

  SeriesList({
    required this.id,
    required this.series,
    required this.yearBegan,
    required this.volume,
    required this.issueCount,
    required this.modified,
  });

  factory SeriesList.fromJson(Map<String, dynamic> json) {
    return SeriesList(
      id: json['id'],
      series: json['series'],
      yearBegan: json['year_began'],
      volume: json['volume'],
      issueCount: json['issue_count'],
      modified: json['modified'],
    );
  }
}
