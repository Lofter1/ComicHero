import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

class AppConfig {
  static bool isInitialized = false;

  static String backendUrl = '';
  static String apiProxyUrl = '';

  static Future load() async {
    await readConfigFile('assets/config/config.json');
    if (!kReleaseMode) {
      await readConfigFile('assets/config/config.dev.json');
    }

    isInitialized = true;
  }

  static Future readConfigFile(String filePath) async {
    final jsonString = await rootBundle.loadString(filePath);
    final jsonMap = json.decode(jsonString);
    backendUrl = jsonMap['backendUrl'] ?? backendUrl;
    apiProxyUrl = jsonMap['apiProxyUrl'] ?? apiProxyUrl;
  }
}
