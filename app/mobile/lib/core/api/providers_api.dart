import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// /api/v1/providers — read-only list of CLI providers configured
// on the gateway (Claude Code / Codex / Gemini / etc.). Used by
// the spawn-session form to populate the provider picker.
class ProvidersApi {
  ProvidersApi(this._dio);
  final Dio _dio;

  Future<List<ProviderSummary>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/providers');
      final raw = res.data?['providers'];
      if (raw is! List) return [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(ProviderSummary.fromGatewayJson)
          .where((p) => p.id.isNotEmpty)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /providers/{id}/toggle. Server fires `provider.toggle` audit
  // event and refuses to disable the only enabled provider — handler
  // returns the patched record on success.
  Future<void> setEnabled(String id, {required bool enabled}) async {
    try {
      await _dio.patch<Map<String, dynamic>>(
        '/api/v1/providers/$id/toggle',
        data: {'enabled': enabled},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final providersApiProvider = Provider<ProvidersApi>((ref) {
  return ProvidersApi(ref.watch(dioProvider));
});

final providersListProvider =
    FutureProvider.autoDispose<List<ProviderSummary>>((ref) {
  return ref.watch(providersApiProvider).list();
});
