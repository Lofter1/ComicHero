class CharacterList {
  final int id;
  final String name;
  final String modified;

  CharacterList({required this.id, required this.name, required this.modified});

  factory CharacterList.fromJson(Map<String, dynamic> json) {
    return CharacterList(
      id: json['id'],
      name: json['name'],
      modified: json['modified'],
    );
  }
}
