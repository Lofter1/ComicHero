// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'reading_order_progress_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

@ProviderFor(readingOrderProgress)
const readingOrderProgressProvider = ReadingOrderProgressFamily._();

final class ReadingOrderProgressProvider
    extends
        $FunctionalProvider<
          AsyncValue<ReadingOrderProgress>,
          ReadingOrderProgress,
          FutureOr<ReadingOrderProgress>
        >
    with
        $FutureModifier<ReadingOrderProgress>,
        $FutureProvider<ReadingOrderProgress> {
  const ReadingOrderProgressProvider._({
    required ReadingOrderProgressFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'readingOrderProgressProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$readingOrderProgressHash();

  @override
  String toString() {
    return r'readingOrderProgressProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ReadingOrderProgress> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ReadingOrderProgress> create(Ref ref) {
    final argument = this.argument as String;
    return readingOrderProgress(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is ReadingOrderProgressProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$readingOrderProgressHash() =>
    r'5b4eeea4564d3853637eedcaf2802de05215f618';

final class ReadingOrderProgressFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ReadingOrderProgress>, String> {
  const ReadingOrderProgressFamily._()
    : super(
        retry: null,
        name: r'readingOrderProgressProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ReadingOrderProgressProvider call(String readingOrderId) =>
      ReadingOrderProgressProvider._(argument: readingOrderId, from: this);

  @override
  String toString() => r'readingOrderProgressProvider';
}

// ignore_for_file: type=lint
// ignore_for_file: subtype_of_sealed_class, invalid_use_of_internal_member, invalid_use_of_visible_for_testing_member, deprecated_member_use_from_same_package
