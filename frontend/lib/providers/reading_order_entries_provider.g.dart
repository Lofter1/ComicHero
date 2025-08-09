// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'reading_order_entries_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

@ProviderFor(EntriesForReadingOrder)
const entriesForReadingOrderProvider = EntriesForReadingOrderFamily._();

final class EntriesForReadingOrderProvider
    extends
        $NotifierProvider<
          EntriesForReadingOrder,
          PagingState<int, ReadingOrderEntry>
        > {
  const EntriesForReadingOrderProvider._({
    required EntriesForReadingOrderFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'entriesForReadingOrderProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$entriesForReadingOrderHash();

  @override
  String toString() {
    return r'entriesForReadingOrderProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  EntriesForReadingOrder create() => EntriesForReadingOrder();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(PagingState<int, ReadingOrderEntry> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<PagingState<int, ReadingOrderEntry>>(
        value,
      ),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is EntriesForReadingOrderProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$entriesForReadingOrderHash() =>
    r'beb0113e08c19f09528878f6d04c31739607a949';

final class EntriesForReadingOrderFamily extends $Family
    with
        $ClassFamilyOverride<
          EntriesForReadingOrder,
          PagingState<int, ReadingOrderEntry>,
          PagingState<int, ReadingOrderEntry>,
          PagingState<int, ReadingOrderEntry>,
          String
        > {
  const EntriesForReadingOrderFamily._()
    : super(
        retry: null,
        name: r'entriesForReadingOrderProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  EntriesForReadingOrderProvider call(String readingOrderId) =>
      EntriesForReadingOrderProvider._(argument: readingOrderId, from: this);

  @override
  String toString() => r'entriesForReadingOrderProvider';
}

abstract class _$EntriesForReadingOrder
    extends $Notifier<PagingState<int, ReadingOrderEntry>> {
  late final _$args = ref.$arg as String;
  String get readingOrderId => _$args;

  PagingState<int, ReadingOrderEntry> build(String readingOrderId);
  @$mustCallSuper
  @override
  void runBuild() {
    final created = build(_$args);
    final ref =
        this.ref
            as $Ref<
              PagingState<int, ReadingOrderEntry>,
              PagingState<int, ReadingOrderEntry>
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                PagingState<int, ReadingOrderEntry>,
                PagingState<int, ReadingOrderEntry>
              >,
              PagingState<int, ReadingOrderEntry>,
              Object?,
              Object?
            >;
    element.handleValue(ref, created);
  }
}

// ignore_for_file: type=lint
// ignore_for_file: subtype_of_sealed_class, invalid_use_of_internal_member, invalid_use_of_visible_for_testing_member, deprecated_member_use_from_same_package
