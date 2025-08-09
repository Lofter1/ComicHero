import 'package:comichero_frontend/metron/metron.dart';

class SeriesParameters implements RequestParameters {
  final int? cvId;
  final int? gcdId;
  final String? imprintName;
  final bool? missingCvId;
  final bool? missingGcdId;
  final String? modifiedGt;
  final String? name;
  final int? page;
  final int? publisherId;
  final String? publisherName;
  final String? seriesType;
  final int? seriesTypeId;
  final int? status;
  final int? volume;
  final int? yearBegan;
  final int? yearEnd;

  SeriesParameters({
    this.cvId,
    this.gcdId,
    this.imprintName,
    this.missingCvId,
    this.missingGcdId,
    this.modifiedGt,
    this.name,
    this.page,
    this.publisherId,
    this.publisherName,
    this.seriesType,
    this.seriesTypeId,
    this.status,
    this.volume,
    this.yearBegan,
    this.yearEnd,
  });

  @override
  Map<String, String> toUriParams() {
    return <String, String>{
      if (cvId != null) 'cv_id': cvId.toString(),
      if (gcdId != null) 'gcd_id': gcdId.toString(),
      if (imprintName != null) 'imprint_name': imprintName.toString(),
      if (missingCvId != null) 'missing_cv_id': missingCvId.toString(),
      if (missingGcdId != null) 'missing_gcd_id': missingGcdId.toString(),
      if (modifiedGt != null) 'modified_gt': modifiedGt.toString(),
      if (name != null) 'name': name.toString(),
      if (page != null) 'page': page.toString(),
      if (publisherId != null) 'publisher_id': publisherId.toString(),
      if (publisherName != null) 'publisher_name': publisherName.toString(),
      if (seriesType != null) 'series_type': seriesType.toString(),
      if (seriesTypeId != null) 'series_type_id': seriesTypeId.toString(),
      if (status != null) 'status': status.toString(),
      if (volume != null) 'volume': volume.toString(),
      if (yearBegan != null) 'year_began': yearBegan.toString(),
      if (yearEnd != null) 'year_end': yearEnd.toString(),
    };
  }
}
