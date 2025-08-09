import 'package:flutter/material.dart';

import 'package:comichero_frontend/ui/ui.dart';

class ComicHeroHomePage extends StatelessWidget {
  const ComicHeroHomePage({super.key, required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    return ReadingOrdersListPage();
  }
}
