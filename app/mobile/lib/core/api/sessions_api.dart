import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// Sessions endpoint surface. F1 uses listSessions only; later
// phases (F2) add create / stop / start / delete on top.
class SessionsApi {
  SessionsApi(this._dio);
  final Dio _dio;

  Future<List<SessionSummary>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/sessions');
      final raw = res.data?['sessions'];
      if (raw is! List) return [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(SessionSummary.fromJson)
          .toList();
    } catch (e) {
      throw toApiException(e);
    }
  }
}

final sessionsApiProvider = Provider<SessionsApi>((ref) {
  return SessionsApi(ref.watch(dioProvider));
});

final sessionsListProvider =
    FutureProvider.autoDispose<List<SessionSummary>>((ref) {
  return ref.watch(sessionsApiProvider).list();
});
