import 'package:comichero_frontend/services/services.dart';

class AuthService {
  Future<void> login(String email, String password) async {
    await pb.collection('users').authWithPassword(email, password);
  }

  Future<void> register(
    String email,
    String name,
    String password,
    String passwordConfirm,
  ) async {
    await pb
        .collection('users')
        .create(
          body: {
            'email': email,
            'name': name,
            'password': password,
            'passwordConfirm': passwordConfirm,
          },
        );
    await login(email, password);
  }

  bool isLoggedIn() {
    return pb.authStore.isValid;
  }

  String getCurrentUsername() {
    return pb.authStore.record?.data['name'];
  }

  void logout() {
    pb.authStore.clear();
  }
}
