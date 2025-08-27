import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/services/services.dart';
import 'package:pocketbase/pocketbase.dart';
import 'package:http/http.dart' as http;
import 'package:path/path.dart' as path;
import 'package:image/image.dart' as img;

class ComicService {
  static const collectionName = 'comics';
  static RecordService collection = pb.collection(collectionName);
  static RecordService userCollection = pb.collection('userComics');

  Future<List<Comic>> get({
    String? seriesName,
    int? seriesYearBegan,
    String? issue,
    DateTime? coverDate,
  }) async {
    String expand = '';

    if (pb.authStore.isValid) {
      expand = 'userComics_via_comic';
    }

    final filters = <String>[];

    if (seriesName != null && seriesName.isNotEmpty) {
      filters.add("seriesName~'${seriesName.replaceAll("'", r"\'")}'");
    }
    if (seriesYearBegan != null) {
      filters.add("seriesYearBegan=$seriesYearBegan");
    }
    if (issue != null && issue.isNotEmpty) {
      filters.add("issue='${issue.replaceAll("'", r"\'")}'");
    }
    if (coverDate != null) {
      filters.add("coverDate~'$coverDate'");
    }

    final filter = filters.join(' && ');

    return (await collection.getFullList(
      expand: expand,
      filter: filter,
    )).map(mapRecordToComic).toList();
  }

  Future<Comic> getById(String id) async {
    String expand = '';
    if (pb.authStore.isValid) {
      expand += 'userComics_via_comic';
    }

    var record = await collection.getOne(id, expand: expand);

    return mapRecordToComic(record);
  }

  Future<Comic> create(Comic comic) async {
    final response = await http.get(
      Uri.parse(comic.coverUrl!),
      headers: {"pb_auth": pb.authStore.token},
    );
    final bytes = response.bodyBytes;
    final filename = path.basename(comic.coverUrl!);

    img.Image? original = img.decodeImage(bytes);
    if (original == null) {
      throw Exception("Failed to decode image");
    }

    // 3. Resize to 50%
    final resized = img.copyResize(
      original,
      width: (original.width / 2).round(),
      height: (original.height / 2).round(),
    );

    // 4. Encode back to bytes (JPEG or PNG depending on your original format)
    final resizedBytes = img.encodeJpg(resized); // or encodePng(resized)

    return collection
        .create(
          body: <String, dynamic>{
            "seriesName": comic.seriesName,
            "seriesYearBegan": comic.seriesYearBegan,
            "description": comic.description,
            "issue": comic.issue,
            "marvelUnlimitedUrl": comic.marvelUrl,
            "coverDate": comic.coverDate.toString(),
          },
          files: [
            http.MultipartFile.fromBytes(
              'cover',
              resizedBytes,
              filename: filename,
            ),
          ],
        )
        .then(mapRecordToComic);
  }

  Future<List<Comic>> search(String searchString) async {
    var (series, yearBegan, issue) = _splitComicName(searchString);
    final resultList = await collection.getList(
      filter:
          'seriesName~"$series" ${issue != null ? '&& issue="$issue"' : ''} ${yearBegan != null ? '&& seriesYearBegan="$yearBegan"' : ''}',
    );
    return resultList.items.map(mapRecordToComic).toList();
  }

  // Future<Comic> markRead(Comic comic) => setReadStatus(comic, true);

  // Future<Comic> markNotRead(Comic comic) => setReadStatus(comic, false);

  // Future<Comic> markSkipped(Comic comic) => setSkippedStatus(comic, true);

  // Future<Comic> markNotSkipped(Comic comic) => setSkippedStatus(comic, false);

  (String seriesName, String? seriesStartYear, String? issueNumber)
  _splitComicName(String input) {
    final regex = RegExp(r'^(.*?)(?:\s+\((\d{4})\))?(?:\s+#?(\d+))?$');
    final match = regex.firstMatch(input.trim());

    if (match != null) {
      final name = match.group(1)?.trim() ?? '';
      final year = match.group(2); // optional 4-digit year
      final issue = match.group(3); // optional issue number
      return (name, year, issue);
    }

    return (input.trim(), null, null);
  }

  Future<Comic> setReadStatus(Comic comic, bool readStatus) async {
    if (pb.authStore.isValid) {
      final body = <String, dynamic>{"read": readStatus};

      if (readStatus == true) {
        body.addAll(<String, dynamic>{"skipped": false});
      }

      if (comic.userComicId != null) {
        await userCollection.update(comic.userComicId!, body: body);
      } else {
        body.addAll(<String, dynamic>{
          "user": pb.authStore.record!.data['id'],
          "comic": comic.id,
        });
        await userCollection.create(body: body);
      }
    }

    return getById(comic.id);
  }

  Future<Comic> setSkippedStatus(Comic comic, bool skippedStatus) async {
    if (pb.authStore.isValid) {
      final body = <String, dynamic>{"skipped": skippedStatus};

      if (skippedStatus == true) {
        body.addAll(<String, dynamic>{"read": false});
      }

      if (comic.userComicId != null) {
        await userCollection.update(comic.userComicId!, body: body);
      } else {
        body.addAll(<String, dynamic>{
          "user": pb.authStore.record!.data['id'],
          "comic": comic.id,
        });
        await userCollection.create(body: body);
      }
    }

    return getById(comic.id);
  }

  String _getCoverUrlForRecord(RecordModel record) =>
      pb.files.getURL(record, record.data['cover']).toString();

  Comic mapRecordToComic(RecordModel record) {
    var comic = Comic.fromRecord(record);
    comic.coverUrl = _getCoverUrlForRecord(record);
    return comic;
  }
}
