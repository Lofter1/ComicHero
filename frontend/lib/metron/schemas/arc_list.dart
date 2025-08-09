class ArcList {
  final int id;
  final String name;
  final String modified;

  ArcList({required this.id, required this.name, required this.modified});

  factory ArcList.fromJson(Map<String, dynamic> json) {
    return ArcList(
      id: json['id'],
      name: json['name'],
      modified: json['modified'],
    );
  }
}
