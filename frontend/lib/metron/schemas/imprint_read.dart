class ImprintRead {
  final int id;
  final String name;
  final String modified;

  ImprintRead({required this.id, required this.name, required this.modified});

  factory ImprintRead.fromJson(Map<String, dynamic> json) {
    return ImprintRead(
      id: json['id'],
      name: json['name'],
      modified: json['modified'],
    );
  }
}
