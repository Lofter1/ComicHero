import 'package:flutter/material.dart';
import 'package:loader_overlay/loader_overlay.dart';

import 'package:comichero_frontend/services/services.dart';
import 'package:comichero_frontend/ui/ui.dart';

class UserButtons extends StatelessWidget {
  const UserButtons({super.key});

  @override
  Widget build(BuildContext context) {
    return AuthGuard(
      loggedInView: (context) => PopupMenuButton<String>(
        icon: Icon(Icons.account_circle),
        tooltip: AuthService().getCurrentUsername(),
        onSelected: (value) {
          switch (value) {
            case 'logout':
              _logout();
              break;
            case 'manage':
              _onManageAccount(context);
              break;
          }
        },
        itemBuilder: (context) => [
          PopupMenuItem(value: 'manage', child: Text('Manage Account')),
          PopupMenuItem(value: 'logout', child: Text('Logout')),
        ],
      ),
      loggedOutView: (context) => IconButton(
        onPressed: () => _onAuthButtonPressed(context),
        icon: Icon(Icons.login),
        tooltip: 'Login or Register',
      ),
    );
  }

  void _onManageAccount(BuildContext context) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text("Manage Account (not implemented)")));
  }

  void _onAuthButtonPressed(BuildContext context) async {
    final result = await showDialog<AuthDialogResult>(
      context: context,
      builder: (_) => _AuthDialog(),
    );

    if (result != null) {
      if (context.mounted) {
        context.loaderOverlay.show();
      }
      String message;
      try {
        if (result.isLogin) {
          await AuthService().login(result.email, result.password);
        } else {
          await AuthService().register(
            result.email,
            result.name!,
            result.password,
            result.passwordConfirm!,
          );
        }

        if (AuthService().isLoggedIn()) {
          authNotifier.value = true;
          message = '${result.isLogin ? 'Login' : 'Registration'} successful!';
        } else {
          message = '${result.isLogin ? 'Login' : 'Registration'} failed.';
        }
      } catch (e) {
        message = '${result.isLogin ? 'Login' : 'Registration'} failed: $e';
      }

      if (context.mounted) {
        context.loaderOverlay.hide();
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(message)));
      }
    }
  }

  void _logout() {
    authNotifier.value = false;
    AuthService().logout();
  }
}

class AuthDialogResult {
  final String email;
  final String password;
  final bool isLogin;
  final String? name;
  final String? passwordConfirm;

  AuthDialogResult({
    required this.email,
    required this.password,
    required this.isLogin,
    this.name,
    this.passwordConfirm,
  });
}

class _AuthDialog extends StatefulWidget {
  @override
  State<_AuthDialog> createState() => _AuthDialogState();
}

class _AuthDialogState extends State<_AuthDialog> {
  String email = '';
  String password = '';
  String passwordConfirm = '';
  String name = '';
  bool isLogin = true;

  final _formKey = GlobalKey<FormState>();

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(isLogin ? 'Login' : 'Register'),
      content: Form(
        key: _formKey,
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            spacing: 10,
            children: [
              TextFormField(
                decoration: InputDecoration(labelText: 'Email'),
                autofocus: true,
                textInputAction: TextInputAction.next,
                onChanged: (value) => email = value,
                validator: (value) {
                  if (value == null || value.isEmpty) {
                    return 'Email cannot be empty';
                  }

                  return null;
                },
              ),
              if (!isLogin)
                TextFormField(
                  decoration: InputDecoration(labelText: 'Name'),
                  textInputAction: TextInputAction.next,
                  onChanged: (value) => name = value,
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Name cannot be empty';
                    }
                    return null;
                  },
                ),
              TextFormField(
                decoration: InputDecoration(labelText: 'Password'),
                textInputAction: isLogin
                    ? TextInputAction.done
                    : TextInputAction.next,
                obscureText: true,
                onChanged: (value) => password = value,
                validator: (value) {
                  if (value == null || value.isEmpty) {
                    return 'Password cannot be empty';
                  }
                  if (!isLogin && value.length < 8) {
                    return 'Password must be at least 8 characters';
                  }
                  return null;
                },
                onFieldSubmitted: (value) {
                  if (isLogin) {
                    _onLogin();
                  }
                },
              ),
              if (!isLogin)
                TextFormField(
                  decoration: InputDecoration(labelText: 'Confirm Password'),
                  textInputAction: TextInputAction.done,
                  obscureText: true,
                  onChanged: (value) => passwordConfirm = value,
                  validator: (value) {
                    if (!isLogin && value != password) {
                      return 'Passwords do not match';
                    }
                    return null;
                  },
                ),
              TextButton(
                onPressed: () => setState(() => isLogin = !isLogin),
                child: Text(
                  isLogin
                      ? "Don't have an account? Register"
                      : "Already have an account? Log in",
                ),
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text('Cancel'),
        ),
        ElevatedButton(
          onPressed: _onLogin,
          child: Text(isLogin ? 'Login' : 'Register'),
        ),
      ],
    );
  }

  void _onLogin() {
    if (_formKey.currentState?.validate() == false) {
      return;
    }
    Navigator.of(context).pop(
      AuthDialogResult(
        email: email,
        password: password,
        isLogin: isLogin,
        name: isLogin ? null : name,
        passwordConfirm: isLogin ? null : passwordConfirm,
      ),
    );
  }
}
