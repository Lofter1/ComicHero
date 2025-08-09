class TeamList {
  final int id;
  final String name;
  final String modified;

  TeamList({required this.id, required this.name, required this.modified});

  factory TeamList.fromJson(Map<String, dynamic> json) {
    return TeamList(
      id: json['id'],
      name: json['name'],
      modified: json['modified'],
    );
  }
}
