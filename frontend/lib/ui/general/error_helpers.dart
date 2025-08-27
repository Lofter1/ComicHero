import 'package:flutter/material.dart';

SnackBar getErrorSnackbar(Exception exception) {
  return SnackBar(
    duration: Duration(minutes: 1),
    action: SnackBarAction(label: "Ok", onPressed: () {}),
    content: Row(
      mainAxisSize: MainAxisSize.min,
      spacing: 10,
      children: [
        Icon(Icons.error, color: Colors.red),
        Expanded(child: Text("An error occurred: $exception")),
      ],
    ),
  );
}
