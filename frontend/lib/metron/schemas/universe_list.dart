class UniverseList {
  final int id;
  final String name;
  final String modified;

  UniverseList({required this.id, required this.name, required this.modified});

  factory UniverseList.fromJson(Map<String, dynamic> json) {
    return UniverseList(
      id: json['id'],
      name: json['name'],
      modified: json['modified'],
    );
  }
}
