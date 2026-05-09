import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// Wraps /api/v1/memory/* — pgvector-backed long-term memory store.
// The mobile Memory tab uses this for the global cross-session
// browser; the inspector's per-session view filters to the current
// session.cwd / session.id (not yet wired).
class MemoryApi {
  MemoryApi(this._dio);
  final Dio _dio;

  Future<MemoryStatus> status() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/memory/status');
      return MemoryStatus.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // GET /memory/list?scope=&scope_key=&n=
  // - scope is required by the server (defaults to 'project' when
  //   omitted there), but we pass it explicitly so empty values
  //   never get the wrong default.
  // - scope_key is required for session/project scopes, optional
  //   for global.
  Future<List<Memory>> list({
    required MemoryScope scope,
    String? scopeKey,
    int limit = 100,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/memory/list',
        queryParameters: {
          'scope': scope.wire,
          if (scopeKey != null && scopeKey.isNotEmpty) 'scope_key': scopeKey,
          'n': limit,
        },
      );
      final raw = res.data?['memories'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(Memory.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // GET /memory/scope-keys?scope=  — distinct scope_key values for
  // populating the cwd picker in the Project view.
  Future<List<String>> scopeKeys(MemoryScope scope) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/memory/scope-keys',
        queryParameters: {'scope': scope.wire},
      );
      final raw = res.data?['scope_keys'];
      if (raw is! List) return const [];
      return raw.whereType<String>().toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /memory/search — semantic search via pgvector cosine
  // similarity. Empty results = no matches above threshold; pass
  // minSimilarity=-1 to bypass the cutoff.
  Future<List<MemoryHit>> search({
    required String query,
    required MemoryScope scope,
    String? scopeKey,
    int topK = 20,
    double? minSimilarity,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/memory/search',
        data: {
          'query': query,
          'scope': scope.wire,
          if (scopeKey != null && scopeKey.isNotEmpty) 'scope_key': scopeKey,
          'top_k': topK,
          if (minSimilarity != null) 'min_similarity': minSimilarity,
        },
      );
      final raw = res.data?['hits'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(MemoryHit.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<Memory> get(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/memory/$id');
      return Memory.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /memory/store — returns the new id. scope_key required for
  // session/project; ignored (must be empty) for global.
  Future<String> store({
    required String text,
    required MemoryScope scope,
    String? scopeKey,
    Map<String, dynamic>? metadata,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/memory/store',
        data: {
          'text': text,
          'scope': scope.wire,
          if (scopeKey != null && scopeKey.isNotEmpty) 'scope_key': scopeKey,
          if (metadata != null && metadata.isNotEmpty) 'metadata': metadata,
        },
      );
      return res.data?['id'] as String? ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /memory/{id} — re-embeds before persisting.
  Future<void> update({
    required String id,
    required String text,
    Map<String, dynamic>? metadata,
  }) async {
    try {
      await _dio.patch<void>(
        '/api/v1/memory/$id',
        data: {
          'text': text,
          if (metadata != null) 'metadata': metadata,
        },
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/memory/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final memoryApiProvider = Provider<MemoryApi>((ref) {
  return MemoryApi(ref.watch(dioProvider));
});
