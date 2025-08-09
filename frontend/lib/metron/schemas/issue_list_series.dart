class IssueListSeries {
  final String name;
  final int volume;
  final int yearBegan;

  IssueListSeries({
    required this.name,
    required this.volume,
    required this.yearBegan,
  });

  factory IssueListSeries.fromJson(Map<String, dynamic> json) {
    return IssueListSeries(
      name: json['name'] ?? '',
      volume: json['volume'],
      yearBegan: json['year_began'],
    );
  }
}
