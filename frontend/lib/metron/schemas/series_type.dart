class SeriesType {
  final int id;
  final String name;

  SeriesType({required this.id, required this.name});

  factory SeriesType.fromJson(Map<String, dynamic> json) {
    return SeriesType(id: json['id'], name: json['name']);
  }
}
