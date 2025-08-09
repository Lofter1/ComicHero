import 'dart:async';
import 'package:loader_overlay/loader_overlay.dart';

import 'package:flutter/widgets.dart';

extension OverlayLoaderExtension on OverlayExtensionHelper {
  static Timer? _debounceTimer;

  void snapshotLoader(
    AsyncSnapshot asyncSnapshot, {
    Duration delay = const Duration(milliseconds: 300),
  }) {
    if (asyncSnapshot.connectionState == ConnectionState.waiting) {
      _debounceTimer?.cancel();
      _debounceTimer = Timer(delay, () {
        show();
      });
    } else {
      _debounceTimer?.cancel();
      _debounceTimer = null;
      hide();
    }
  }
}
