import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:loader_overlay/loader_overlay.dart';

import 'package:comichero_frontend/app_config.dart';
import 'package:comichero_frontend/ui/ui.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await AppConfig.load();
  runApp(ProviderScope(child: const MyApp()));
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Flutter Demo',
      debugShowCheckedModeBanner: false,
      darkTheme: ThemeData.dark(),
      themeMode: ThemeMode.dark,
      home: ComicHeroHomePage(title: 'Flutter Demo Home Page'),
      builder: (context, child) => LoaderOverlay(
        overlayWidgetBuilder: (_) => Center(child: CircularProgressIndicator()),
        child: child ?? const SizedBox(),
      ),
    );
  }
}
