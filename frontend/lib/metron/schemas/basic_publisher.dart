class BasicPublisher {
  final int id;
  final String name;

  BasicPublisher({required this.id, required this.name});

  factory BasicPublisher.fromJson(Map<String, dynamic> json) {
    return BasicPublisher(id: json['id'], name: json['name']);
  }
}
