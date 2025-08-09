class BasicImprint {
  final int id;
  final String name;

  BasicImprint({required this.id, required this.name});

  factory BasicImprint.fromJson(Map<String, dynamic> json) {
    return BasicImprint(id: json['id'], name: json['name']);
  }
}
