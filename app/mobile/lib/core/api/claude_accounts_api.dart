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
}

final claudeAccountsApiProvider = Provider<ClaudeAccountsApi>((ref) {
  return ClaudeAccountsApi(ref.watch(dioProvider));
});

final claudeAccountsListProvider =
    FutureProvider.autoDispose<List<ClaudeAccountSummary>>((ref) {
  return ref.watch(claudeAccountsApiProvider).list();
});
