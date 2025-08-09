import 'package:flutter/material.dart';

import 'package:comichero_frontend/ui/general/login_button.dart';

class ComicHeroAppBar extends StatelessWidget implements PreferredSizeWidget {
  final String title;
  const ComicHeroAppBar({super.key, required this.title});

  @override
  Widget build(BuildContext context) {
    return AppBar(
      actionsPadding: const EdgeInsets.all(10),
      backgroundColor: Theme.of(context).colorScheme.inversePrimary,
      title: Text(title),
      actions: const <Widget>[UserButtons()],
    );
  }

  @override
  Size get preferredSize => const Size.fromHeight(kToolbarHeight);
}
