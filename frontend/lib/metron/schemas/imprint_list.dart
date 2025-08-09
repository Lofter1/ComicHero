class ImprintList {
  final int id;
  final String name;
  final String modified;

  ImprintList({required this.id, required this.name, required this.modified});

  factory ImprintList.fromJson(Map<String, dynamic> json) {
    return ImprintList(
      id: json['id'],
      name: json['name'],
      modified: json['modified'],
    );
  }
}
