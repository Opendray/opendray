import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/memory/cleanup/* — the M13 LLM librarian. Backs the
// "Cleanup" tab in the Project screen.
class MemoryCleanupApi {
  MemoryCleanupApi(this._dio);
  final Dio _dio;

  /// Trigger a one-shot cleanup pass for one (scope, scope_key).
  /// Server-side this calls the configured summarizer LLM with up
  /// to BatchSize aged-eligible memories and writes one
  /// memory_cleanup_decisions row per memory.
  Future<CleanupRunResult> run({
    required String scope,
    required String scopeKey,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/memory/cleanup/run',
        data: {'scope': scope, 'scope_key': scopeKey},
      );
      return CleanupRunResult.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<CleanupDecision>> list({
    String? status,
    String? scope,
    String? scopeKey,
    int limit = 100,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/memory/cleanup/decisions',
        queryParameters: {
          if (status != null && status.isNotEmpty) 'status': status,
          if (scope != null && scope.isNotEmpty) 'scope': scope,
          if (scopeKey != null && scopeKey.isNotEmpty) 'scope_key': scopeKey,
          'n': limit,
        },
      );
      final raw = res.data?['decisions'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(CleanupDecision.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Approve atomically marks the decision approved + executes the
  /// delete/merge. Returns the updated decision with status=executed
  /// (or status=expired if the executor couldn't apply).
  Future<CleanupDecision> approve(String id) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/memory/cleanup/decisions/$id/approve',
      );
      return CleanupDecision.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Reject closes the decision without touching the memory store.
  Future<void> reject(String id) async {
    try {
      await _dio.post<void>('/api/v1/memory/cleanup/decisions/$id/reject');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

class CleanupRunResult {
  CleanupRunResult({
    required this.runId,
    required this.scope,
    required this.scopeKey,
    required this.memoriesIn,
    required this.decisionsOut,
  });

  factory CleanupRunResult.fromJson(Map<String, dynamic> j) => CleanupRunResult(
        runId: j['run_id']?.toString() ?? '',
        scope: j['scope']?.toString() ?? '',
        scopeKey: j['scope_key']?.toString() ?? '',
        memoriesIn: (j['memories_in'] is num)
            ? (j['memories_in'] as num).toInt()
            : 0,
        decisionsOut: (j['decisions_out'] is num)
            ? (j['decisions_out'] as num).toInt()
            : 0,
      );

  final String runId;
  final String scope;
  final String scopeKey;
  final int memoriesIn;
  final int decisionsOut;
}

class CleanupDecision {
  CleanupDecision({
    required this.id,
    required this.memoryId,
    required this.memoryScope,
    required this.memoryScopeKey,
    required this.memoryTextSnapshot,
    required this.verdict,
    required this.reason,
    required this.mergeInto,
    required this.runId,
    required this.status,
    required this.summarizerProviderId,
    required this.createdAt,
    this.decidedAt,
    this.executedAt,
  });

  factory CleanupDecision.fromJson(Map<String, dynamic> j) => CleanupDecision(
        id: j['id']?.toString() ?? '',
        memoryId: j['memory_id']?.toString() ?? '',
        memoryScope: j['memory_scope']?.toString() ?? '',
        memoryScopeKey: j['memory_scope_key']?.toString() ?? '',
        memoryTextSnapshot: j['memory_text_snapshot']?.toString() ?? '',
        verdict: j['verdict']?.toString() ?? '',
        reason: j['reason']?.toString() ?? '',
        mergeInto: j['merge_into']?.toString() ?? '',
        runId: j['run_id']?.toString() ?? '',
        status: j['status']?.toString() ?? '',
        summarizerProviderId:
            j['summarizer_provider_id']?.toString() ?? '',
        createdAt: DateTime.tryParse(j['created_at']?.toString() ?? '') ??
            DateTime.now(),
        decidedAt: j['decided_at'] != null
            ? DateTime.tryParse(j['decided_at'].toString())
            : null,
        executedAt: j['executed_at'] != null
            ? DateTime.tryParse(j['executed_at'].toString())
            : null,
      );

  final String id;
  final String memoryId;
  final String memoryScope;
  final String memoryScopeKey;
  final String memoryTextSnapshot;
  final String verdict;
  final String reason;
  final String mergeInto;
  final String runId;
  final String status;
  final String summarizerProviderId;
  final DateTime createdAt;
  final DateTime? decidedAt;
  final DateTime? executedAt;

  bool get isPending => status == 'pending';
}

final memoryCleanupApiProvider = Provider<MemoryCleanupApi>((ref) {
  return MemoryCleanupApi(ref.watch(dioProvider));
});
