class CreatorList {
  final int id;
  final String name;
  final String modified;

  CreatorList({required this.id, required this.name, required this.modified});

  factory CreatorList.fromJson(Map<String, dynamic> json) {
    return CreatorList(
      id: json['id'],
      name: json['name'],
      modified: json['modified'],
    );
  }
}
