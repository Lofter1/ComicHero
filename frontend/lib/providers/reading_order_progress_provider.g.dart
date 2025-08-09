// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'reading_order_progress_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

String _$readingOrderProgressHash() =>
    r'5b4eeea4564d3853637eedcaf2802de05215f618';

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

/// See also [readingOrderProgress].
@ProviderFor(readingOrderProgress)
const readingOrderProgressProvider = ReadingOrderProgressFamily();

/// See also [readingOrderProgress].
class ReadingOrderProgressFamily
    extends Family<AsyncValue<ReadingOrderProgress>> {
  /// See also [readingOrderProgress].
  const ReadingOrderProgressFamily();

  /// See also [readingOrderProgress].
  ReadingOrderProgressProvider call(String readingOrderId) {
    return ReadingOrderProgressProvider(readingOrderId);
  }

  @override
  ReadingOrderProgressProvider getProviderOverride(
    covariant ReadingOrderProgressProvider provider,
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
  String? get name => r'readingOrderProgressProvider';
}

/// See also [readingOrderProgress].
class ReadingOrderProgressProvider
    extends AutoDisposeFutureProvider<ReadingOrderProgress> {
  /// See also [readingOrderProgress].
  ReadingOrderProgressProvider(String readingOrderId)
    : this._internal(
        (ref) => readingOrderProgress(
          ref as ReadingOrderProgressRef,
          readingOrderId,
        ),
        from: readingOrderProgressProvider,
        name: r'readingOrderProgressProvider',
        debugGetCreateSourceHash: const bool.fromEnvironment('dart.vm.product')
            ? null
            : _$readingOrderProgressHash,
        dependencies: ReadingOrderProgressFamily._dependencies,
        allTransitiveDependencies:
            ReadingOrderProgressFamily._allTransitiveDependencies,
        readingOrderId: readingOrderId,
      );

  ReadingOrderProgressProvider._internal(
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
  Override overrideWith(
    FutureOr<ReadingOrderProgress> Function(ReadingOrderProgressRef provider)
    create,
  ) {
    return ProviderOverride(
      origin: this,
      override: ReadingOrderProgressProvider._internal(
        (ref) => create(ref as ReadingOrderProgressRef),
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
  AutoDisposeFutureProviderElement<ReadingOrderProgress> createElement() {
    return _ReadingOrderProgressProviderElement(this);
  }

  @override
  bool operator ==(Object other) {
    return other is ReadingOrderProgressProvider &&
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
mixin ReadingOrderProgressRef
    on AutoDisposeFutureProviderRef<ReadingOrderProgress> {
  /// The parameter `readingOrderId` of this provider.
  String get readingOrderId;
}

class _ReadingOrderProgressProviderElement
    extends AutoDisposeFutureProviderElement<ReadingOrderProgress>
    with ReadingOrderProgressRef {
  _ReadingOrderProgressProviderElement(super.provider);

  @override
  String get readingOrderId =>
      (origin as ReadingOrderProgressProvider).readingOrderId;
}

// ignore_for_file: type=lint
// ignore_for_file: subtype_of_sealed_class, invalid_use_of_internal_member, invalid_use_of_visible_for_testing_member, deprecated_member_use_from_same_package
