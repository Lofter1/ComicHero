import 'package:comichero_frontend/services/services.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:loader_overlay/loader_overlay.dart';

import 'package:comichero_frontend/models/comic.dart';
import 'package:comichero_frontend/ui/ui.dart';

class ComicQuickOverview extends StatefulWidget {
  final Comic comic;
  final List<PopupMenuEntry<Function(Comic)>>? popupMenuItems;

  const ComicQuickOverview({
    super.key,
    required this.comic,
    this.popupMenuItems,
  });

  @override
  State<ComicQuickOverview> createState() => _ComicQuickOverviewState();
}

class _ComicQuickOverviewState extends State<ComicQuickOverview> {
  late Comic comic;

  void onReadComicButtonClicked() {
    context.loaderOverlay.show();
    ComicService().setReadStatus(comic, !comic.read!);
    context.loaderOverlay.hide();
  }

  void onSkippedComicButtonClicked() {
    context.loaderOverlay.show();
    ComicService().setSkippedStatus(comic, !comic.skipped!);
    context.loaderOverlay.hide();
  }

  void updateComic(Comic comic) {
    setState(() {
      this.comic = comic;
    });
  }

  @override
  void initState() {
    super.initState();
    comic = widget.comic;
  }

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      spacing: 10,
      children: [
        if (comic.coverUrl != null && comic.coverUrl!.isNotEmpty)
          ComicCover(comic: comic),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            spacing: 10,
            children: [
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _ComicTitleBar(comic: comic, widget: widget),
                  if (comic.coverDate != null)
                    Text(DateFormat.yMMMd().format(comic.coverDate!)),
                ],
              ),
              _ComicActionButtonRow(
                comic: comic,
                onReadComicButtonClicked: onReadComicButtonClicked,
                onSkippedComicButtonClicked: onSkippedComicButtonClicked,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _ComicActionButtonRow extends StatelessWidget {
  const _ComicActionButtonRow({
    required this.comic,
    required this.onReadComicButtonClicked,
    required this.onSkippedComicButtonClicked,
  });

  final Comic comic;
  final VoidCallback onReadComicButtonClicked;
  final VoidCallback onSkippedComicButtonClicked;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      children: [
        AuthGuard(
          loggedInView: (context) => StatusButtons(
            onReadComicButtonClicked: onReadComicButtonClicked,
            onSkippedComicButtonClicked: onSkippedComicButtonClicked,
          ),
        ),
      ],
    );
  }
}

class _ComicTitleBar extends StatelessWidget {
  const _ComicTitleBar({required this.comic, required this.widget});

  final Comic comic;
  final ComicQuickOverview widget;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        _ComicTitle(comic: comic),
        if (widget.popupMenuItems != null)
          PopupMenuButton<Function(Comic)>(
            onSelected: (action) => action(comic),
            itemBuilder: (context) => widget.popupMenuItems!,
          ),
      ],
    );
  }
}

class ComicDescription extends StatelessWidget {
  const ComicDescription({super.key, required this.comic});

  final Comic comic;

  @override
  Widget build(BuildContext context) {
    return Text(
      comic.description!,
      maxLines: 4,
      overflow: TextOverflow.ellipsis,
      style: Theme.of(context).textTheme.bodyMedium,
    );
  }
}

class _ComicTitle extends StatelessWidget {
  const _ComicTitle({required this.comic});

  final Comic comic;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Row(
        spacing: 10,
        children: [
          Flexible(
            fit: FlexFit.loose,
            child: Text(
              comic.title,
              style: Theme.of(context).textTheme.titleLarge,
            ),
          ),
          AuthGuard(
            loggedInView: (context) {
              if (comic.read != null && comic.read!) {
                return Icon(Icons.check_circle, color: Colors.green);
              }
              if (comic.skipped != null && comic.skipped!) {
                return Icon(Icons.skip_next, color: Colors.amber);
              }
              return SizedBox.shrink();
            },
          ),
        ],
      ),
    );
  }
}
