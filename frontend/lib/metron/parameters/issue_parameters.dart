import 'package:comichero_frontend/metron/metron.dart';

/// Parameter options for the Issue API endpoint
class IssueParameters implements RequestParameters {
  /// Alternate Number
  final String? altNumber;

  /// Cover Hash
  final String? coverHash;

  /// Cover Month
  final int? coverMonth;

  /// Cover Year
  final int? coverYear;

  /// Comic Vine ID
  final int? cvId;
  final String? focDate;
  final String? focDateRangeAfter;
  final String? focDateRangeBefore;

  /// Grand Comics Database ID
  final int? gcdId;

  /// Imprint Metron ID
  final int? imprintId;

  /// Imprint Name
  final String? imprintName;
  final bool? missingCvId;
  final bool? missingGcdId;

  /// Greater than Modified DateTime
  final String? modifiedGt;

  /// Issue Number
  final String? number;

  /// A page number within the paginated result set.
  final int? page;

  /// Publisher Metron ID
  final int? publisherId;

  /// Publisher Name
  final String? publisherName;

  /// Rating
  final String? rating;

  /// Series Metron ID
  final int? seriesId;

  /// Series Name
  final String? seriesName;

  /// Series Volume Number
  final int? seriesVolume;

  /// Series Beginning Year
  final int? seriesYearBegan;

  /// Distributor SKU
  final String? sku;
  final String? storeDate;
  final String? storeDateRangeAfter;
  final String? storeDateRangeBefore;

  /// UPC Code
  final String? upc;

  IssueParameters({
    this.altNumber,
    this.coverHash,
    this.coverMonth,
    this.coverYear,
    this.cvId,
    this.focDate,
    this.focDateRangeAfter,
    this.focDateRangeBefore,
    this.gcdId,
    this.imprintId,
    this.imprintName,
    this.missingCvId,
    this.missingGcdId,
    this.modifiedGt,
    this.number,
    this.page,
    this.publisherId,
    this.publisherName,
    this.rating,
    this.seriesId,
    this.seriesName,
    this.seriesVolume,
    this.seriesYearBegan,
    this.sku,
    this.storeDate,
    this.storeDateRangeAfter,
    this.storeDateRangeBefore,
    this.upc,
  });

  /// Return a map of URI parameters based on the set options
  @override
  Map<String, String> toUriParams() {
    return <String, String>{
      if (altNumber != null) 'alt_number': altNumber!,
      if (coverHash != null) 'cover_hash': coverHash!,
      if (coverMonth != null) 'cover_month': coverMonth.toString(),
      if (coverYear != null) 'cover_year': coverYear.toString(),
      if (cvId != null) 'cv_id': cvId.toString(),
      if (focDate != null) 'foc_date': focDate!,
      if (focDateRangeAfter != null) 'foc_date_range_after': focDateRangeAfter!,
      if (focDateRangeBefore != null)
        'foc_date_range_before': focDateRangeBefore!,
      if (gcdId != null) 'gcd_id': gcdId.toString(),
      if (imprintId != null) 'imprint_id': imprintId.toString(),
      if (imprintName != null) 'imprint_name': imprintName!,
      if (missingCvId != null) 'missing_cv_id': missingCvId.toString(),
      if (missingGcdId != null) 'missing_gcd_id': missingGcdId.toString(),
      if (modifiedGt != null) 'modified_gt': modifiedGt!,
      if (number != null) 'number': number!,
      if (page != null) 'page': page.toString(),
      if (publisherId != null) 'publisher_id': publisherId.toString(),
      if (publisherName != null) 'publisher_name': publisherName!,
      if (rating != null) 'rating': rating!,
      if (seriesId != null) 'series_id': seriesId.toString(),
      if (seriesName != null) 'series_name': seriesName!,
      if (seriesVolume != null) 'series_volume': seriesVolume.toString(),
      if (seriesYearBegan != null)
        'series_year_began': seriesYearBegan.toString(),
      if (sku != null) 'sku': sku!,
      if (storeDate != null) 'store_date': storeDate!,
      if (storeDateRangeAfter != null)
        'store_date_range_after': storeDateRangeAfter!,
      if (storeDateRangeBefore != null)
        'store_date_range_before': storeDateRangeBefore!,
      if (upc != null) 'upc': upc!,
    };
  }
}
