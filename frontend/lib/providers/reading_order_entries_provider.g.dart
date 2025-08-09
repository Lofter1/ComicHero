// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'reading_order_entries_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

String _$entriesForReadingOrderHash() =>
    r'5fa8eb1b9c64f277cf55dc6b3bda367394d2bd88';

/// Copied from Dart SDK
class _SystemHash {
  _SystemHash._();

  static int combine(int hash, int value) {
    // ignore: parameter_assignments
    hash = 0x1fffffff & (hash + value);
    // ignore: parameter_assignments
    hash = 0x1fffffff & (hash + ((0x0007ffff & hash) << 10));
    return hash ^ (hash >> 6);
  }

  static int finish(int hash) {
    // ignore: parameter_assignments
    hash = 0x1fffffff & (hash + ((0x03ffffff & hash) << 3));
    // ignore: parameter_assignments
    hash = hash ^ (hash >> 11);
    return 0x1fffffff & (hash + ((0x00003fff & hash) << 15));
  }
}

abstract class _$EntriesForReadingOrder
    extends BuildlessAutoDisposeNotifier<PagingState<int, ReadingOrderEntry>> {
  late final String readingOrderId;

  PagingState<int, ReadingOrderEntry> build(String readingOrderId);
}

/// See also [EntriesForReadingOrder].
@ProviderFor(EntriesForReadingOrder)
const entriesForReadingOrderProvider = EntriesForReadingOrderFamily();

/// See also [EntriesForReadingOrder].
class EntriesForReadingOrderFamily
    extends Family<PagingState<int, ReadingOrderEntry>> {
  /// See also [EntriesForReadingOrder].
  const EntriesForReadingOrderFamily();

  /// See also [EntriesForReadingOrder].
  EntriesForReadingOrderProvider call(String readingOrderId) {
    return EntriesForReadingOrderProvider(readingOrderId);
  }

  @override
  EntriesForReadingOrderProvider getProviderOverride(
    covariant EntriesForReadingOrderProvider provider,
  ) {
    return call(provider.readingOrderId);
  }

  static const Iterable<ProviderOrFamily>? _dependencies = null;

  @override
  Iterable<ProviderOrFamily>? get dependencies => _dependencies;

  static const Iterable<ProviderOrFamily>? _allTransitiveDependencies = null;

  @override
  Iterable<ProviderOrFamily>? get allTransitiveDependencies =>
      _allTransitiveDependencies;

  @override
  String? get name => r'entriesForReadingOrderProvider';
}

/// See also [EntriesForReadingOrder].
class EntriesForReadingOrderProvider
    extends
        AutoDisposeNotifierProviderImpl<
          EntriesForReadingOrder,
          PagingState<int, ReadingOrderEntry>
        > {
  /// See also [EntriesForReadingOrder].
  EntriesForReadingOrderProvider(String readingOrderId)
    : this._internal(
        () => EntriesForReadingOrder()..readingOrderId = readingOrderId,
        from: entriesForReadingOrderProvider,
        name: r'entriesForReadingOrderProvider',
        debugGetCreateSourceHash: const bool.fromEnvironment('dart.vm.product')
            ? null
            : _$entriesForReadingOrderHash,
        dependencies: EntriesForReadingOrderFamily._dependencies,
        allTransitiveDependencies:
            EntriesForReadingOrderFamily._allTransitiveDependencies,
        readingOrderId: readingOrderId,
      );

  EntriesForReadingOrderProvider._internal(
    super._createNotifier, {
    required super.name,
    required super.dependencies,
    required super.allTransitiveDependencies,
    required super.debugGetCreateSourceHash,
    required super.from,
    required this.readingOrderId,
  }) : super.internal();

  final String readingOrderId;

  @override
  PagingState<int, ReadingOrderEntry> runNotifierBuild(
    covariant EntriesForReadingOrder notifier,
  ) {
    return notifier.build(readingOrderId);
  }

  @override
  Override overrideWith(EntriesForReadingOrder Function() create) {
    return ProviderOverride(
      origin: this,
      override: EntriesForReadingOrderProvider._internal(
        () => create()..readingOrderId = readingOrderId,
        from: from,
        name: null,
        dependencies: null,
        allTransitiveDependencies: null,
        debugGetCreateSourceHash: null,
        readingOrderId: readingOrderId,
      ),
    );
  }

  @override
  AutoDisposeNotifierProviderElement<
    EntriesForReadingOrder,
    PagingState<int, ReadingOrderEntry>
  >
  createElement() {
    return _EntriesForReadingOrderProviderElement(this);
  }

  @override
  bool operator ==(Object other) {
    return other is EntriesForReadingOrderProvider &&
        other.readingOrderId == readingOrderId;
  }

  @override
  int get hashCode {
    var hash = _SystemHash.combine(0, runtimeType.hashCode);
    hash = _SystemHash.combine(hash, readingOrderId.hashCode);

    return _SystemHash.finish(hash);
  }
}

@Deprecated('Will be removed in 3.0. Use Ref instead')
// ignore: unused_element
mixin EntriesForReadingOrderRef
    on AutoDisposeNotifierProviderRef<PagingState<int, ReadingOrderEntry>> {
  /// The parameter `readingOrderId` of this provider.
  String get readingOrderId;
}

class _EntriesForReadingOrderProviderElement
    extends
        AutoDisposeNotifierProviderElement<
          EntriesForReadingOrder,
          PagingState<int, ReadingOrderEntry>
        >
    with EntriesForReadingOrderRef {
  _EntriesForReadingOrderProviderElement(super.provider);

  @override
  String get readingOrderId =>
      (origin as EntriesForReadingOrderProvider).readingOrderId;
}

// ignore_for_file: type=lint
// ignore_for_file: subtype_of_sealed_class, invalid_use_of_internal_member, invalid_use_of_visible_for_testing_member, deprecated_member_use_from_same_package
