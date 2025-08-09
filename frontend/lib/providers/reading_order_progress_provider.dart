import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/services/services.dart';

part 'reading_order_progress_provider.g.dart';

@riverpod
Future<ReadingOrderProgress> readingOrderProgress(
  Ref ref,
  String readingOrderId,
) {
  return ReadingOrderService().getReadingOrderProgress(readingOrderId);
}
