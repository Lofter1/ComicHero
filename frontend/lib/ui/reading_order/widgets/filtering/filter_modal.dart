import 'package:flutter/material.dart';

import 'package:comichero_frontend/ui/ui.dart';

class FilterModal extends StatelessWidget {
  const FilterModal({super.key});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        spacing: 10,
        children: [
          // Row(
          //   spacing: 8,
          //   mainAxisAlignment: MainAxisAlignment.center,
          //   children: [
          //     Icon(Icons.filter_list),
          //     Text('Filters', style: Theme.of(context).textTheme.titleLarge),
          //   ],
          // ),
          // const Divider(),
          AuthGuard(loggedInView: (context) => _StatusFilterSection()),
        ],
      ),
    );
  }
}

class _StatusFilterSection extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      spacing: 10,
      children: [
        Row(
          spacing: 10,
          children: [
            Icon(Icons.auto_stories_outlined),
            Text('Status', style: TextTheme.of(context).titleLarge),
          ],
        ),
        Align(alignment: Alignment.centerLeft, child: StatusFilterChips()),
      ],
    );
  }
}
