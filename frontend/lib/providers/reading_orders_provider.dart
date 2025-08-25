import 'package:infinite_scroll_pagination/infinite_scroll_pagination.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'package:comichero_frontend/services/services.dart';
import 'package:comichero_frontend/models/models.dart';

part 'reading_orders_provider.g.dart';

@riverpod
class ReadingOrders extends _$ReadingOrders {
  final _readingOrderService = ReadingOrderService();

  bool _isFetching = false;

  @override
  PagingState<int, ReadingOrder> build() {
    return PagingState();
  }

  Future<void> fetchNextPage() async {
    if (_isFetching) return;
    _isFetching = true;

    state = state.copyWith(isLoading: true, error: null);

    try {
      // TODO: paginate
      final response = await _readingOrderService.get();

      state = state.copyWith(
        isLoading: false,
        error: null,
        hasNextPage: response.totalPages != response.page,
        keys: [...?state.keys, state.nextIntPageKey],
        pages: [
          ...?state.pages,
          response.items.map(ReadingOrder.fromRecord).toList(),
        ],
      );
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e);
    } finally {
      _isFetching = false;
    }
  }
}
