import 'package:flutter/material.dart';

import 'package:comichero_frontend/ui/notifiers/notifiers.dart';

class AuthGuard extends StatelessWidget {
  final WidgetBuilder loggedInView;
  final WidgetBuilder? loggedOutView;
  const AuthGuard({super.key, required this.loggedInView, this.loggedOutView});

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder(
      valueListenable: authNotifier,
      builder: (context, isLoggedIn, _) {
        if (isLoggedIn) {
          return loggedInView(context);
        } else {
          return loggedOutView != null
              ? loggedOutView!(context)
              : SizedBox.shrink();
        }
      },
    );
  }
}
