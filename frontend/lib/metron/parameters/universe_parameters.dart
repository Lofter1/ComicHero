import 'package:comichero_frontend/metron/metron.dart';

class UniverseParameters implements RequestParameters {
  final String? designation;
  final String? modifiedGt;
  final String? name;
  final int? page;

  UniverseParameters(this.designation, this.modifiedGt, this.name, this.page);

  @override
  Map<String, String> toUriParams() {
    return <String, String>{
      if (designation != null) 'designation': designation.toString(),
      if (modifiedGt != null) 'modified_gt': modifiedGt.toString(),
      if (name != null) 'name': name.toString(),
      if (page != null) 'page': page.toString(),
    };
  }
}
