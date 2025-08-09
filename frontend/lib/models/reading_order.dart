import 'package:pocketbase/pocketbase.dart';

class ReadingOrder {
  final String id;
  final String name;
  final String? description;

  ReadingOrder({required this.id, required this.name, this.description});

  factory ReadingOrder.fromRecord(RecordModel record) =>
      ReadingOrder.fromJson(record.toJson());

  factory ReadingOrder.fromJson(Map<String, dynamic> json) {
    return ReadingOrder(
      id: json['readingOrderId'] ?? json['id'],
      name: json['name'],
      description: json['description'],
    );
  }
}
