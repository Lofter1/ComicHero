import 'package:comichero_frontend/metron/metron.dart';

class IssueRead {
  final int id;
  final BasicPublisher? publisher;
  final BasicImprint? imprint;
  final IssueSeries? series;
  final String number;
  final String altNumber;
  final String title;
  final List<String> name;
  final String coverDate;
  final String storeDate;
  final String focDate;
  final String price;
  final Rating? rating;
  final String sku;
  final String isbn;
  final String upc;
  final int page;
  final String desc;
  final String image;
  final String coverHash;
  final List<ArcList> arcs;
  final List<CreditRead> credits;
  final List<CharacterList> characters;
  final List<TeamList> teams;
  final List<UniverseList> universes;
  final List<Reprint> reprints;
  final List<VariantIssue> variants;
  final int cvId;
  final int gcdId;
  final String resourceUrl;
  final String modified;

  IssueRead({
    required this.id,
    required this.publisher,
    required this.imprint,
    required this.series,
    required this.number,
    required this.altNumber,
    required this.title,
    required this.name,
    required this.coverDate,
    required this.storeDate,
    required this.focDate,
    required this.price,
    required this.rating,
    required this.sku,
    required this.isbn,
    required this.upc,
    required this.page,
    required this.desc,
    required this.image,
    required this.coverHash,
    required this.arcs,
    required this.credits,
    required this.characters,
    required this.teams,
    required this.universes,
    required this.reprints,
    required this.variants,
    required this.cvId,
    required this.gcdId,
    required this.resourceUrl,
    required this.modified,
  });

  factory IssueRead.fromJson(Map<String, dynamic> json) {
    return IssueRead(
      id: json['id'] ?? 0,
      publisher: json['publisher'] != null
          ? BasicPublisher.fromJson(json['publisher'])
          : null,
      imprint: json['imprint'] != null
          ? BasicImprint.fromJson(json['imprint'])
          : null,
      series: json['series'] != null
          ? IssueSeries.fromJson(json['series'])
          : null,
      number: json['number'] ?? '',
      altNumber: json['alt_number'] ?? '',
      title: json['title'] ?? '',
      name: (json['name'] as List<dynamic>? ?? []).cast<String>(),
      coverDate: json['cover_date'] ?? '',
      storeDate: json['store_date'] ?? '',
      focDate: json['foc_date'] ?? '',
      price: json['price'] ?? '',
      rating: json['rating'] != null ? Rating.fromJson(json['rating']) : null,
      sku: json['sku'] ?? '',
      isbn: json['isbn'] ?? '',
      upc: json['upc'] ?? '',
      page: json['page'] ?? 0,
      desc: json['desc'] ?? '',
      image: json['image'] ?? '',
      coverHash: json['cover_hash'] ?? '',
      arcs: (json['arcs'] as List<dynamic>? ?? [])
          .map((e) => ArcList.fromJson(e))
          .toList(),
      credits: (json['credits'] as List<dynamic>? ?? [])
          .map((e) => CreditRead.fromJson(e))
          .toList(),
      characters: (json['characters'] as List<dynamic>? ?? [])
          .map((e) => CharacterList.fromJson(e))
          .toList(),
      teams: (json['teams'] as List<dynamic>? ?? [])
          .map((e) => TeamList.fromJson(e))
          .toList(),
      universes: (json['universes'] as List<dynamic>? ?? [])
          .map((e) => UniverseList.fromJson(e))
          .toList(),
      reprints: (json['reprints'] as List<dynamic>? ?? [])
          .map((e) => Reprint.fromJson(e))
          .toList(),
      variants: (json['variants'] as List<dynamic>? ?? [])
          .map((e) => VariantIssue.fromJson(e))
          .toList(),
      cvId: json['cv_id'] ?? 0,
      gcdId: json['gcd_id'] ?? 0,
      resourceUrl: json['resource_url'] ?? '',
      modified: json['modified'] ?? '',
    );
  }
}
