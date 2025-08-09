library;

import 'package:comichero_frontend/app_config.dart';
import 'package:pocketbase/pocketbase.dart';

export './comic_service.dart';
export './reading_order_service.dart';
export './reading_order_comic_service.dart';
export './metron_service.dart';
export './auth_service.dart';

export './options/options.dart';

export 'services.dart' hide pb;

PocketBase? _pb;

PocketBase get pb {
  _pb ??= PocketBase(AppConfig.backendUrl);
  return _pb!;
}
