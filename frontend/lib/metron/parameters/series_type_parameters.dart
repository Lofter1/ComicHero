import 'package:comichero_frontend/metron/metron.dart';

class SeriesTypeParameters implements RequestParameters {
  final String? modifiedGt;
  final String? name;
  final int? page;

  SeriesTypeParameters(this.modifiedGt, this.name, this.page);

  @override
  Map<String, String> toUriParams() {
    return <String, String>{
      if (modifiedGt != null) 'modified_gt': modifiedGt.toString(),
      if (name != null) 'name': name.toString(),
      if (page != null) 'page': page.toString(),
    };
  }
}
