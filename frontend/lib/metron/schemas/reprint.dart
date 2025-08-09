class Reprint {
  final int id;
  final String issue;

  Reprint({required this.id, required this.issue});

  factory Reprint.fromJson(Map<String, dynamic> json) {
    return Reprint(id: json['id'], issue: json['issue']);
  }
}
