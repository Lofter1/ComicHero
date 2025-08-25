// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'reading_orders_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

@ProviderFor(ReadingOrders)
const readingOrdersProvider = ReadingOrdersProvider._();

final class ReadingOrdersProvider
    extends $NotifierProvider<ReadingOrders, PagingState<int, ReadingOrder>> {
  const ReadingOrdersProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'readingOrdersProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$readingOrdersHash();

  @$internal
  @override
  ReadingOrders create() => ReadingOrders();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(PagingState<int, ReadingOrder> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<PagingState<int, ReadingOrder>>(
        value,
      ),
    );
  }
}

String _$readingOrdersHash() => r'3ef4f7845cd29316011691c18ee6c5d882d1789b';

abstract class _$ReadingOrders
    extends $Notifier<PagingState<int, ReadingOrder>> {
  PagingState<int, ReadingOrder> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final created = build();
    final ref =
        this.ref
            as $Ref<
              PagingState<int, ReadingOrder>,
              PagingState<int, ReadingOrder>
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                PagingState<int, ReadingOrder>,
                PagingState<int, ReadingOrder>
              >,
              PagingState<int, ReadingOrder>,
              Object?,
              Object?
            >;
    element.handleValue(ref, created);
  }
}

// ignore_for_file: type=lint
// ignore_for_file: subtype_of_sealed_class, invalid_use_of_internal_member, invalid_use_of_visible_for_testing_member, deprecated_member_use_from_same_package
