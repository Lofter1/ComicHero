import 'package:flutter/material.dart';

import 'package:comichero_frontend/services/services.dart';
import 'package:comichero_frontend/models/comic.dart';

class ComicCover extends StatelessWidget {
  const ComicCover({super.key, required this.comic, this.rounding = 9});

  final Comic comic;
  final double rounding;

  @override
  Widget build(BuildContext context) {
    return Image.network(
      comic.coverUrl!,
      fit: BoxFit.scaleDown,
      headers: {"pb_auth": pb.authStore.token},
    );
  }
}
