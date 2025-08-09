import 'package:comichero_frontend/services/services.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'reading_order_entries_list_options_provider.g.dart';

@riverpod
class ReadingOrderEntriesOptions extends _$ReadingOrderEntriesOptions {
  @override
  ReadingOrderEntriesListOptions build() {
    return ReadingOrderEntriesListOptions();
  }

  void setReadFilter(ReadFilter value) {
    state = state.copyWith(filterRead: value);
  }

  void setSkippedFilter(SkippedFilter value) {
    state = state.copyWith(filterSkipped: value);
  }
}
