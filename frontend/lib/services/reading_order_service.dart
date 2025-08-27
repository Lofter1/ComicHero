import 'package:comichero_frontend/models/models.dart';
import 'package:pocketbase/pocketbase.dart';
import 'package:comichero_frontend/services/services.dart';

class ReadingOrderService {
  final _collection = pb.collection('readingOrders');
  final _comicOrderCollection = pb.collection('readingOrderEntries');

  Future<ResultList<RecordModel>> get() {
    return _collection.getList();
  }

  Future<ReadingOrderProgress> getReadingOrderProgress(
    String readingOrderId,
  ) async {
    final result = await pb
        .collection('readingOrderProgress')
        .getFirstListItem('readingOrderId="$readingOrderId"');

    return ReadingOrderProgress(
      readingOrderId: result.getStringValue('readingOrderId'),
      read: result.getIntValue('readComics'),
      total: result.getIntValue('entryCount'),
    );
  }

  Future<ReadingOrder> create(ReadingOrder readingOrder) async {
    final newRecord = await _collection.create(
      body: <String, dynamic>{
        "name": readingOrder.name,
        "description": readingOrder.description,
      },
    );
    return getById(newRecord.id);
  }

  Future<void> removeEntry(ReadingOrderEntry entry) async {
    await _comicOrderCollection.delete(entry.id);
  }

  Future<void> delete(ReadingOrder readingOrder) async {
    await _collection.delete(readingOrder.id);
  }

  Future<ReadingOrder> getById(String id) async {
    //await Future.delayed(const Duration(seconds: 5));
    final record = await _collection.getOne(id);
    return _mapRecordToReadingOrder(record);
  }

  ReadingOrder _mapRecordToReadingOrder(RecordModel record) {
    return ReadingOrder(
      id: record.id,
      name: record.data['name'],
      description: record.data['description'],
    );
  }
}
