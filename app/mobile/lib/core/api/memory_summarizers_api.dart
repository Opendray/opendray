import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Read-only client for GET /api/v1/memory-summarizer-providers — the
// configured summarizer providers (LM Studio / OpenAI-compat / Anthropic /
// …) a memory worker can be pinned to. Mobile only needs the list to
// populate the worker screen's "Summarizer provider" dropdown; full CRUD
// stays on web. Mirrors app/shared/src/lib/memoryAmbient.ts.

class SummarizerProvider {
  SummarizerProvider({
    required this.id,
    required this.name,
    required this.kind,
    required this.model,
    required this.isDefault,
    required this.enabled,
    required this.baseUrl,
  });

  factory SummarizerProvider.fromJson(Map<String, dynamic> json) =>
      SummarizerProvider(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        kind: json['kind'] as String? ?? '',
        model: json['model'] as String? ?? '',
        isDefault: json['is_default'] as bool? ?? false,
        enabled: json['enabled'] as bool? ?? false,
        baseUrl: json['base_url'] as String? ?? '',
      );

  final String id;
  final String name;
  final String kind;
  final String model;
  final bool isDefault;
  final bool enabled;
  // OpenAI-compatible endpoint; used to live-probe the models it serves.
  final String baseUrl;

  /// "name · model" for the dropdown label.
  String get label => model.isEmpty ? name : '$name · $model';
}

class MemorySummarizersApi {
  MemorySummarizersApi(this._dio);
  final Dio _dio;

  Future<List<SummarizerProvider>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/memory-summarizer-providers',
      );
      final raw = res.data?['providers'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(SummarizerProvider.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final memorySummarizersApiProvider = Provider<MemorySummarizersApi>((ref) {
  return MemorySummarizersApi(ref.watch(dioProvider));
});
