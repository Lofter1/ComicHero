class AssociatedSeries {
  final int id;
  final String series;

  AssociatedSeries({required this.id, required this.series});

  factory AssociatedSeries.fromJson(Map<String, dynamic> json) {
    return AssociatedSeries(id: json['id'], series: json['series']);
  }
}
