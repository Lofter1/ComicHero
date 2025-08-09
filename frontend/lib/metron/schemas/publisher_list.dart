class PublisherList {
  final int id;
  final String name;
  final String modified;

  PublisherList({required this.id, required this.name, required this.modified});

  factory PublisherList.fromJson(Map<String, dynamic> json) {
    return PublisherList(
      id: json['id'],
      name: json['name'],
      modified: json['modified'],
    );
  }
}
