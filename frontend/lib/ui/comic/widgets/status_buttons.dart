import 'package:flutter/material.dart';

class StatusButtons extends StatelessWidget {
  final void Function() onReadComicButtonClicked;
  final void Function() onSkippedComicButtonClicked;

  const StatusButtons({
    super.key,
    required this.onReadComicButtonClicked,
    required this.onSkippedComicButtonClicked,
  });

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8.0,
      runSpacing: 8.0,
      children: [
        ElevatedButton(
          onPressed: onReadComicButtonClicked,
          child: Text("Toggle read"),
        ),
        ElevatedButton(
          onPressed: onSkippedComicButtonClicked,
          child: Text("Toggle skipped"),
        ),
      ],
    );
  }
}
