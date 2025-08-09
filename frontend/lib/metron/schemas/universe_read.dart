import 'package:comichero_frontend/metron/schemas/basic_publisher.dart';

class UniverseRead {
  final int id;
  final BasicPublisher publisher;
  final String name;
  final String designation;
  final String desc;
  final int gcdId;
  final String image;
  final String resourceUrl;
  final String modified;

  UniverseRead({
    required this.id,
    required this.publisher,
    required this.name,
    required this.designation,
    required this.desc,
    required this.gcdId,
    required this.image,
    required this.resourceUrl,
    required this.modified,
  });

  factory UniverseRead.fromJson(Map<String, dynamic> json) {
    return UniverseRead(
      id: json['id'],
      publisher: BasicPublisher.fromJson(json['publisher']),
      name: json['name'],
      designation: json['designation'],
      desc: json['desc'],
      gcdId: json['gcdId'],
      image: json['image'],
      resourceUrl: json['resourceUrl'],
      modified: json['modified'],
    );
  }
}
