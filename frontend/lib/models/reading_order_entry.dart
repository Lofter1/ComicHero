import 'package:comichero_frontend/models/comic.dart';

class ReadingOrderEntry {
  final String id;
  final String readingOrderId;
  int position;
  Comic? comic;
  String? notes;

  ReadingOrderEntry({
    required this.id,
    required this.readingOrderId,
    required this.position,
    this.comic,
    this.notes,
  });

  ReadingOrderEntry copyWith({
    String? id,
    String? readingOrderId,
    int? position,
    Comic? comic,
    String? notes,
  }) {
    return ReadingOrderEntry(
      id: id ?? this.id,
      readingOrderId: readingOrderId ?? this.readingOrderId,
      position: position ?? this.position,
      comic: comic ?? this.comic,
      notes: notes ?? this.notes,
    );
  }
}
