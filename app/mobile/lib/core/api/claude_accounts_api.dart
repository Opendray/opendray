import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// /api/v1/claude-accounts — multi-account support is Claude-only on
// the gateway today (see internal/cliacct). Other providers spawn
// against env-var / system credentials and don't have an account
// concept the spawn form needs to expose.
class ClaudeAccountsApi {
  ClaudeAccountsApi(this._dio);
  final Dio _dio;

  Future<List<ClaudeAccountSummary>> list() async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/claude-accounts');
      final raw = res.data?['accounts'];
      if (raw is! List) return [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(ClaudeAccountSummary.fromJson)
          .where((a) => a.id.isNotEmpty)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /claude-accounts/{id}/toggle. Disables/enables an individual
  // OAuth account without touching the token; spawn picker filters
  // out disabled accounts.
  Future<void> setEnabled(String id, {required bool enabled}) async {
    try {
      await _dio.patch<Map<String, dynamic>>(
        '/api/v1/claude-accounts/$id/toggle',
        data: {'enabled': enabled},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /claude-accounts. Token is optional — when omitted the row
  // is created in "empty" state (token_filled=false) and the operator
  // pastes the OAuth JSON later via setToken.
  Future<void> create({
    required String name,
    String? displayName,
    String? token,
    bool enabled = true,
  }) async {
    try {
      await _dio.post<Map<String, dynamic>>(
        '/api/v1/claude-accounts',
        data: {
          'name': name,
          if (displayName != null && displayName.isNotEmpty)
            'display_name': displayName,
          if (token != null && token.isNotEmpty) 'token': token,
          'enabled': enabled,
        },
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PUT /claude-accounts/{id}. Pointer-style patch: pass null to
  // leave a field untouched. Most common mobile edit is renaming
  // display_name; the rest are rarely tweaked from a phone.
  Future<void> update(
    String id, {
    String? displayName,
    String? description,
    bool? enabled,
  }) async {
    try {
      await _dio.put<Map<String, dynamic>>(
        '/api/v1/claude-accounts/$id',
        data: {
          if (displayName != null) 'display_name': displayName,
          if (description != null) 'description': description,
          if (enabled != null) 'enabled': enabled,
        },
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PUT /claude-accounts/{id}/token replaces the OAuth blob. Mobile
  // accepts either the raw JSON pasted in (Claude exports it as a
  // single-object JSON) or the bare access_token; the server treats
  // the field as opaque text.
  Future<void> setToken(String id, String token) async {
    try {
      await _dio.put<Map<String, dynamic>>(
        '/api/v1/claude-accounts/$id/token',
        data: {'token': token},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/claude-accounts/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /claude-accounts/import-local — gateway scans
  // ~/.claude-accounts/ on its own host filesystem and registers any
  // directory it finds that doesn't already have a matching DB row.
  // Returns the count of newly-created accounts so the UI can toast
  // "synced N" vs "already in sync".
  //
  // The mobile app calls this when the operator wants to surface an
  // account they just created via host-shell `claude login` without
  // waiting for the gateway's filesystem watcher to tick.
  Future<int> importLocal() async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/claude-accounts/import-local',
      );
      final c = res.data?['count'];
      if (c is num) return c.toInt();
      return 0;
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final claudeAccountsApiProvider = Provider<ClaudeAccountsApi>((ref) {
  return ClaudeAccountsApi(ref.watch(dioProvider));
});

final claudeAccountsListProvider =
    FutureProvider.autoDispose<List<ClaudeAccountSummary>>((ref) {
  return ref.watch(claudeAccountsApiProvider).list();
});
