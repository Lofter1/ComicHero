import 'dart:convert';
import 'package:comichero_frontend/metron/schemas/paginated_arc_list_list.dart';
import 'package:comichero_frontend/metron/schemas/paginated_character_list_list.dart';
import 'package:comichero_frontend/metron/schemas/paginated_creator_list_list.dart';
import 'package:comichero_frontend/metron/schemas/paginated_imprint_list_list.dart';
import 'package:comichero_frontend/metron/schemas/paginated_publisher_list_list.dart';
import 'package:comichero_frontend/metron/schemas/paginated_role_list.dart';
import 'package:comichero_frontend/metron/schemas/paginated_series_type_list.dart';
import 'package:comichero_frontend/metron/schemas/paginated_team_list_list.dart';
import 'package:comichero_frontend/metron/schemas/paginated_universe_list_list.dart';
import 'package:http/http.dart' as http;
import 'package:comichero_frontend/metron/metron.dart';

class MetronApi {
  final String baseUrl;

  late final String basicAuthHash;

  MetronApi({
    String? username,
    String? password,
    this.baseUrl = "https://metron.cloud/api",
  }) {
    basicAuthHash = base64Encode(utf8.encode('$username:$password'));
  }

  Future<PaginatedArcListList> arc(ArcParameters params) =>
      _getList('arc', params, PaginatedArcListList.fromJson);

  Future<PaginatedCharacterListList> character(CharacterParameters params) =>
      _getList('character', params, PaginatedCharacterListList.fromJson);

  Future<PaginatedCreatorListList> creator(CreatorParameters params) =>
      _getList('creator', params, PaginatedCreatorListList.fromJson);

  Future<PaginatedImprintListList> imprint(ImprintParameters params) =>
      _getList('imprint', params, PaginatedImprintListList.fromJson);

  Future<PaginatedIssueListList> issue(IssueParameters params) =>
      _getList('issue', params, PaginatedIssueListList.fromJson);

  Future<PaginatedPublisherListList> publisher(PublisherParameters params) =>
      _getList('publisher', params, PaginatedPublisherListList.fromJson);

  Future<PaginatedRoleList> role(RoleParameters params) =>
      _getList('role', params, PaginatedRoleList.fromJson);

  Future<PaginatedSeriesListList> series(SeriesParameters params) =>
      _getList('series', params, PaginatedSeriesListList.fromJson);

  Future<PaginatedSeriesTypeList> seriesType(SeriesTypeParameters params) =>
      _getList('seriesType', params, PaginatedSeriesTypeList.fromJson);

  Future<PaginatedTeamListList> team(TeamParameters params) =>
      _getList('team', params, PaginatedTeamListList.fromJson);

  Future<PaginatedUniverseListList> universe(UniverseParameters params) =>
      _getList('universe', params, PaginatedUniverseListList.fromJson);

  Future<IssueRead> issueById(int id) =>
      _getById('issue', id, IssueRead.fromJson);

  Future<T> _getById<T>(
    String path,
    int id,
    T Function(Map<String, dynamic>) fromJson,
  ) async {
    final requestUri = Uri.parse("$baseUrl/$path/$id/");
    var response = await _get(requestUri);
    return fromJson(jsonDecode(response.body));
  }

  Future<T> _getList<T>(
    String path,
    RequestParameters params,
    T Function(Map<String, dynamic>) fromJson,
  ) async {
    final requestUri = Uri.parse(
      "$baseUrl/$path/",
    ).replace(queryParameters: params.toUriParams());

    var response = await _get(requestUri);
    return fromJson(jsonDecode(response.body));
  }

  Future<http.Response> _get(Uri uri) {
    return http.get(uri, headers: {'Authorization': 'Basic $basicAuthHash'});
  }
}
