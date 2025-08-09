import 'package:comichero_frontend/metron/metron.dart';
import 'package:comichero_frontend/metron/schemas/creator_list.dart';

class CharacterRead {
  final int id;
  final String name;
  final List<String> alias;
  final String desc;
  final String image;
  final List<CreatorList> creators;
  final List<TeamList> teams;
  final List<UniverseList> universes;
  final int cvId;
  final int gcdId;
  final String resourceUrl;
  final String modified;

  CharacterRead({
    required this.id,
    required this.name,
    required this.alias,
    required this.desc,
    required this.image,
    required this.creators,
    required this.teams,
    required this.universes,
    required this.cvId,
    required this.gcdId,
    required this.resourceUrl,
    required this.modified,
  });

  factory CharacterRead.fromJson(Map<String, dynamic> json) {
    return CharacterRead(
      id: json['id'],
      name: json['name'],
      alias: json['alias'] as List<String>,
      desc: json['desc'],
      image: json['image'],
      creators: (json['creators'] as List<dynamic>)
          .map((item) => CreatorList.fromJson(item))
          .toList(),
      teams: (json['teams'] as List<dynamic>)
          .map((item) => TeamList.fromJson(item))
          .toList(),
      universes: (json['universes'] as List<dynamic>)
          .map((item) => UniverseList.fromJson(item))
          .toList(),
      cvId: json['cv_id'],
      gcdId: json['gcd_id'],
      resourceUrl: json['resource_url'],
      modified: json['modified'],
    );
  }
}
