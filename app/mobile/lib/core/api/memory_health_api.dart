import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// /api/v1/memory/health — M-PA memory health dashboard, mobile
// mirror of app/shared/src/lib/memoryHealth.ts. One aggregate read
// crossing both memory subsystems for a given project cwd.

class MemoryHealthSnapshot {
  MemoryHealthSnapshot({
    required this.cwd,
    required this.generatedAt,
    required this.lookbackDays,
    required this.newFactsCount,
    required this.totalFactsCount,
    required this.zeroHitFactsCount,
    required this.topHitFactText,
    required this.topHitFactHits,
    required this.captureFires,
    required this.captureFactsExtracted,
    required this.captureFactsStored,
    required this.captureFactsDeduped,
    required this.captureFailedFires,
    required this.newJournalCount,
    required this.totalJournalCount,
    required this.pendingProposals,
    required this.oldestPendingDays,
    required this.planDriftProposals,
    this.planLastUpdatedAt,
    this.goalLastUpdatedAt,
  });

  factory MemoryHealthSnapshot.fromJson(Map<String, dynamic> j) {
    DateTime? parseTs(Object? v) {
      if (v is String && v.isNotEmpty) return DateTime.tryParse(v);
      return null;
    }

    return MemoryHealthSnapshot(
      cwd: (j['cwd'] as String?) ?? '',
      generatedAt: parseTs(j['generated_at']) ?? DateTime.now().toUtc(),
      lookbackDays: (j['lookback_days'] as int?) ?? 7,
      newFactsCount: (j['new_facts_count'] as int?) ?? 0,
      totalFactsCount: (j['total_facts_count'] as int?) ?? 0,
      zeroHitFactsCount: (j['zero_hit_facts_count'] as int?) ?? 0,
      topHitFactText: (j['top_hit_fact_text'] as String?) ?? '',
      topHitFactHits: (j['top_hit_fact_hits'] as int?) ?? 0,
      captureFires: (j['capture_fires'] as int?) ?? 0,
      captureFactsExtracted: (j['capture_facts_extracted'] as int?) ?? 0,
      captureFactsStored: (j['capture_facts_stored'] as int?) ?? 0,
      captureFactsDeduped: (j['capture_facts_deduped'] as int?) ?? 0,
      captureFailedFires: (j['capture_failed_fires'] as int?) ?? 0,
      newJournalCount: (j['new_journal_count'] as int?) ?? 0,
      totalJournalCount: (j['total_journal_count'] as int?) ?? 0,
      pendingProposals: (j['pending_proposals'] as int?) ?? 0,
      oldestPendingDays: (j['oldest_pending_days'] as int?) ?? 0,
      planDriftProposals: (j['plan_drift_proposals'] as int?) ?? 0,
      planLastUpdatedAt: parseTs(j['plan_last_updated_at']),
      goalLastUpdatedAt: parseTs(j['goal_last_updated_at']),
    );
  }

  final String cwd;
  final DateTime generatedAt;
  final int lookbackDays;

  // Layer 5 — discrete facts.
  final int newFactsCount;
  final int totalFactsCount;
  final int zeroHitFactsCount;
  final String topHitFactText;
  final int topHitFactHits;

  // Capture engine activity.
  final int captureFires;
  final int captureFactsExtracted;
  final int captureFactsStored;
  final int captureFactsDeduped;
  final int captureFailedFires;

  // Layer 4 — journal.
  final int newJournalCount;
  final int totalJournalCount;

  // Layers 2-3 — operator-owned docs.
  final DateTime? planLastUpdatedAt;
  final DateTime? goalLastUpdatedAt;
  final int pendingProposals;
  final int oldestPendingDays;
  final int planDriftProposals;
}

class MemoryHealthApi {
  MemoryHealthApi(this._dio);
  final Dio _dio;

  Future<MemoryHealthSnapshot> get(String cwd) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/memory/health',
        queryParameters: {'cwd': cwd},
      );
      return MemoryHealthSnapshot.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final memoryHealthApiProvider = Provider<MemoryHealthApi>((ref) {
  return MemoryHealthApi(ref.watch(dioProvider));
});
