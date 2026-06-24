import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// /api/v1/antigravity-accounts — multi-account support for the
// Antigravity (agy) provider. Unlike Claude, agy keys all of its state
// off $HOME, so an "account" is just a per-account HOME directory the
// operator logs into on the gateway host. There is no token-paste flow:
// rows are surfaced by import-local (the gateway scans
// ~/.antigravity-accounts/) and removed/toggled from here. Mirrors web
// app/shared/src/lib/antigravityAccounts.ts.
class AntigravityAccountsApi {
  AntigravityAccountsApi(this._dio);
  final Dio _dio;

  Future<List<AntigravityAccountSummary>> list() async {
    try {
      final res = await _dio
          .get<Map<String, dynamic>>('/api/v1/antigravity-accounts');
      final raw = res.data?['accounts'];
      if (raw is! List) return [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(AntigravityAccountSummary.fromJson)
          .where((a) => a.id.isNotEmpty)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /antigravity-accounts/{id}/toggle. Enables/disables an account
  // without touching its on-disk HOME; the spawn picker and switcher
  // filter out disabled accounts.
  Future<void> setEnabled(String id, {required bool enabled}) async {
    try {
      await _dio.patch<Map<String, dynamic>>(
        '/api/v1/antigravity-accounts/$id/toggle',
        data: {'enabled': enabled},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/antigravity-accounts/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /antigravity-accounts/import-local — gateway scans
  // ~/.antigravity-accounts/ (and the gateway user's ~) on its own host
  // and registers any logged-in account dir that doesn't already have a
  // matching DB row. Returns the count of newly-created accounts so the
  // UI can toast "imported N" vs "already in sync".
  Future<int> importLocal() async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/antigravity-accounts/import-local',
      );
      final c = res.data?['count'];
      if (c is num) return c.toInt();
      return 0;
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final antigravityAccountsApiProvider = Provider<AntigravityAccountsApi>((ref) {
  return AntigravityAccountsApi(ref.watch(dioProvider));
});

final antigravityAccountsListProvider =
    FutureProvider.autoDispose<List<AntigravityAccountSummary>>((ref) {
  return ref.watch(antigravityAccountsApiProvider).list();
});
