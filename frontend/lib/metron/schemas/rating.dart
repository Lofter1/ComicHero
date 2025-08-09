class Rating {
  final int id;
  final String name;

  Rating({required this.id, required this.name});

  factory Rating.fromJson(Map<String, dynamic> json) {
    return Rating(id: json['id'], name: json['name']);
  }
}
