import 'package:flutter/cupertino.dart';

import 'package:comichero_frontend/services/auth_service.dart';

class AuthNotifier extends ValueNotifier<bool> {
  AuthNotifier() : super(AuthService().isLoggedIn());
}

final authNotifier = AuthNotifier();
