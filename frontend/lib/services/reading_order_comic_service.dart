import 'package:pocketbase/pocketbase.dart';

import 'package:comichero_frontend/models/models.dart';
import 'package:comichero_frontend/services/services.dart';

class ReadingOrderEntriesService {
  final collection = pb.collection('readingOrderEntries');

  Future<List<ReadingOrderEntry>> get({String? readingOrderId}) async {
    List<String> filters = [];
    if (readingOrderId != null) {
      filters.add("readingOrder.id = '$readingOrderId'");
    }

    return (await collection.getFullList(
      filter: filters.join("&&"),
    )).map(mapRecordToReadingOrderEntry).toList();
  }

  Future<ResultList<RecordModel>> getWithComics(
    String readingOrderId, {
    ReadingOrderEntriesListOptions? options,
    int page = 1,
  }) {
    var expand = 'comic';

    if (pb.authStore.isValid) {
      expand += '.userComics_via_comic';
    }

    List<String> filters = ["readingOrder.id = '$readingOrderId'"];

    if (options != null && options.filterString != '') {
      filters.add(options.filterString);
    }

    return collection.getList(
      expand: expand,
      sort: options?.sortString,
      filter: filters.join("&&"),
      page: page,
    );
  }

  Future<ReadingOrderEntry> create(ReadingOrderEntry entry) async {
    var newRecord = await collection.create(
      body: {
        'readingOrder': entry.readingOrderId,
        'position': entry.position,
        'notes': entry.notes,
        'comic': entry.comic?.id,
      },
    );

    var newEntry = mapRecordToReadingOrderEntry(newRecord);
    newEntry.comic = entry.comic;

    return newEntry;
  }

  Future<ReadingOrderEntry> update(ReadingOrderEntry entry) async {
    var updatedRecord = await collection.update(
      entry.id,
      body: {
        'readingOrder': entry.readingOrderId,
        'position': entry.position,
        'notes': entry.notes,
        'comic': entry.comic?.id,
      },
    );

    var updatedEntry = mapRecordToReadingOrderEntry(updatedRecord);
    updatedEntry.comic = entry.comic;

    return updatedEntry;
  }

  Future<ReadingOrderEntry> getById(String id) async {
    String expand = 'comic';

    if (pb.authStore.isValid) {
      expand += '.userComics_via_comic';
    }

    var record = await collection.getOne(id, expand: expand);
    return mapRecordToReadingOrderEntry(record);
  }

  ReadingOrderEntry mapRecordToReadingOrderEntry(RecordModel record) {
    return ReadingOrderEntry(
      id: record.id,
      readingOrderId: record.data['readingOrder'],
      position: record.data['position'],
      notes: record.data['notes'],
      comic: record.get<RecordModel?>('expand.comic', null) != null
          ? ComicService().mapRecordToComic(
              record.get<RecordModel>('expand.comic'),
            )
          : null,
    );
  }
}
