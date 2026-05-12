// Wraps /api/v1/admin/settings (GET / PUT) and
// /api/v1/admin/restart (POST). Mobile-side we keep the config as
// an untyped Map<String, dynamic> rather than a typed Dart class —
// the config struct has 60+ leaf fields across 11 logical
// sections, and bringing them into typed Dart would force every
// backend schema tweak through a mobile pubspec bump for no gain.
// The Map lets the field-spec table in server_settings_screen.dart
// drive everything via dot-paths instead.
//
// Sensitive fields (database.url, admin.password) come back as ""
// on GET — backend strips them so the device never sees the real
// secret. PUTting "" preserves the existing value server-side.

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

class SettingsApi {
  SettingsApi(this._dio);
  final Dio _dio;

  // GET /admin/settings — returns the full Config + the path of the
  // toml file the server loaded it from. The toml path is shown to
  // the operator so they know which file would change on a PUT.
  Future<({Map<String, dynamic> config, String configPath})> get() async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/admin/settings');
      final data = res.data ?? const <String, dynamic>{};
      final cfg = data['config'];
      return (
        config: cfg is Map<String, dynamic> ? cfg : <String, dynamic>{},
        configPath: data['config_path'] as String? ?? '',
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PUT /admin/settings replaces the full config. The web admin
  // sends the entire ServerConfig struct on every save; we match
  // that semantics here so the server doesn't need partial-merge
  // logic.
  Future<void> put(Map<String, dynamic> config) async {
    try {
      await _dio.put<void>(
        '/api/v1/admin/settings',
        data: config,
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /admin/restart asks the gateway to exec itself. Returns
  // 202 + JSON immediately; the actual exec happens ~500ms later
  // server-side so the response can flush.
  Future<void> restart() async {
    try {
      await _dio.post<void>('/api/v1/admin/restart');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final settingsApiProvider = Provider<SettingsApi>((ref) {
  return SettingsApi(ref.watch(dioProvider));
});
