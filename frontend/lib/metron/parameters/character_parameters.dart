import 'package:comichero_frontend/metron/metron.dart';

class CharacterParameters implements RequestParameters {
  final int? cvId;
  final int? gcdId;
  final String? modifiedGt;
  final String? name;
  final int? page;

  CharacterParameters({
    this.cvId,
    this.gcdId,
    this.modifiedGt,
    this.name,
    this.page,
  });

  @override
  Map<String, String> toUriParams() {
    return <String, String>{
      if (cvId != null) 'cv_id': cvId.toString(),
      if (gcdId != null) 'gc_id': gcdId.toString(),
      if (modifiedGt != null) 'modified_gt': modifiedGt.toString(),
      if (name != null) 'name': name.toString(),
      if (page != null) 'page': page.toString(),
    };
  }
}
