import 'package:comichero_frontend/metron/schemas/role.dart';

class CreditRead {
  final int id;
  final String creator;
  final List<Role> role;

  CreditRead({required this.id, required this.creator, required this.role});

  factory CreditRead.fromJson(Map<String, dynamic> json) {
    return CreditRead(
      id: json['id'],
      creator: json['creator'],
      role: (json['role'] as List<dynamic>? ?? [])
          .map((e) => Role.fromJson(e))
          .toList(),
    );
  }
}
