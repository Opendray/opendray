import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// Sessions endpoint surface. F2 covers full CRUD: list, fetch one,
// create, stop, start (re-spawn), delete.
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
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<SessionSummary> getById(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/sessions/$id');
      return SessionSummary.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<SessionSummary> create(CreateSessionRequest req) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/sessions',
        data: req.toJson(),
      );
      return SessionSummary.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<SessionSummary> stop(String id) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/sessions/$id/stop',
      );
      return SessionSummary.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<SessionSummary> start(String id) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/sessions/$id/start',
      );
      return SessionSummary.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/sessions/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> resize(String id, {required int cols, required int rows}) async {
    try {
      await _dio.post<void>(
        '/api/v1/sessions/$id/resize',
        data: {'cols': cols, 'rows': rows},
      );
    } on Object catch (e) {
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

final sessionByIdProvider =
    FutureProvider.autoDispose.family<SessionSummary, String>((ref, id) {
  return ref.watch(sessionsApiProvider).getById(id);
});
