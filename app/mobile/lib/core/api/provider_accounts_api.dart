import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/api/models.dart';

// Read-only listing of /api/v1/{kind}-accounts for the providers whose
// accounts are managed on the web only (grok, codex, opencode). Mobile
// needs just the list, to drive the in-session account switcher; adding,
// logging in and editing those accounts stays on the web Providers page.
// `kind` is the provider id, which is also the URL prefix.
class ProviderAccountsApi {
  ProviderAccountsApi(this._dio);
  final Dio _dio;

  Future<List<ProviderAccountSummary>> list(String kind) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/$kind-accounts',
      );
      final raw = res.data?['accounts'];
      if (raw is! List) return [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(ProviderAccountSummary.fromJson)
          .where((a) => a.id.isNotEmpty)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final providerAccountsApiProvider = Provider<ProviderAccountsApi>((ref) {
  return ProviderAccountsApi(ref.watch(dioProvider));
});

// Keyed by provider id: 'grok', 'codex' or 'opencode'.
final providerAccountsListProvider = FutureProvider.autoDispose
    .family<List<ProviderAccountSummary>, String>((ref, kind) {
      return ref.watch(providerAccountsApiProvider).list(kind);
    });
