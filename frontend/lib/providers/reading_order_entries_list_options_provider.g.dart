// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'reading_order_entries_list_options_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

@ProviderFor(ReadingOrderEntriesOptions)
const readingOrderEntriesOptionsProvider =
    ReadingOrderEntriesOptionsProvider._();

final class ReadingOrderEntriesOptionsProvider
    extends
        $NotifierProvider<
          ReadingOrderEntriesOptions,
          ReadingOrderEntriesListOptions
        > {
  const ReadingOrderEntriesOptionsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'readingOrderEntriesOptionsProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$readingOrderEntriesOptionsHash();

  @$internal
  @override
  ReadingOrderEntriesOptions create() => ReadingOrderEntriesOptions();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ReadingOrderEntriesListOptions value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ReadingOrderEntriesListOptions>(
        value,
      ),
    );
  }
}

String _$readingOrderEntriesOptionsHash() =>
    r'8d93cedadabff7ac50a69b3f95cafb3cb9a056bc';

abstract class _$ReadingOrderEntriesOptions
    extends $Notifier<ReadingOrderEntriesListOptions> {
  ReadingOrderEntriesListOptions build();
  @$mustCallSuper
  @override
  void runBuild() {
    final created = build();
    final ref =
        this.ref
            as $Ref<
              ReadingOrderEntriesListOptions,
              ReadingOrderEntriesListOptions
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                ReadingOrderEntriesListOptions,
                ReadingOrderEntriesListOptions
              >,
              ReadingOrderEntriesListOptions,
              Object?,
              Object?
            >;
    element.handleValue(ref, created);
  }
}

// ignore_for_file: type=lint
// ignore_for_file: subtype_of_sealed_class, invalid_use_of_internal_member, invalid_use_of_visible_for_testing_member, deprecated_member_use_from_same_package
