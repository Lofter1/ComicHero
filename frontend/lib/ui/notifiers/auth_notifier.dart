import 'package:flutter/cupertino.dart';

class AuthNotifier extends ValueNotifier<bool> {
  AuthNotifier() : super(false);
}

final authNotifier = AuthNotifier();
