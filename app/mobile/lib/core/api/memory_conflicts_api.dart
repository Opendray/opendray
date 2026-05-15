import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// /api/v1/memory/conflicts — M-PC cross-layer conflict inbox.
// Mobile mirror of app/shared/src/lib/memoryConflicts.ts.

enum ConflictLayer { fact, plan, goal, journal }

extension ConflictLayerX on ConflictLayer {
  String get wire => name;
  static ConflictLayer parse(String s) {
    switch (s) {
      case 'plan':
        return ConflictLayer.plan;
      case 'goal':
        return ConflictLayer.goal;
      case 'journal':
        return ConflictLayer.journal;
      default:
        return ConflictLayer.fact;
    }
  }
}

enum ConflictSeverity { low, medium, high }

extension ConflictSeverityX on ConflictSeverity {
  static ConflictSeverity parse(String s) {
    switch (s) {
      case 'low':
        return ConflictSeverity.low;
      case 'high':
        return ConflictSeverity.high;
      default:
        return ConflictSeverity.medium;
    }
  }
}

enum ConflictStatus { pending, accepted, dismissed, expired }

extension ConflictStatusX on ConflictStatus {
  String get wire => name;
  static ConflictStatus parse(String s) {
    switch (s) {
      case 'accepted':
        return ConflictStatus.accepted;
      case 'dismissed':
        return ConflictStatus.dismissed;
      case 'expired':
        return ConflictStatus.expired;
      default:
        return ConflictStatus.pending;
    }
  }
}

class MemoryConflict {
  MemoryConflict({
    required this.id,
    required this.cwd,
    required this.layerA,
    required this.refA,
    required this.layerB,
    required this.refB,
    required this.evidence,
    required this.severity,
    required this.status,
    required this.detectedAt,
    this.decidedAt,
    this.decidedBy,
  });

  factory MemoryConflict.fromJson(Map<String, dynamic> j) {
    DateTime parseTs(Object? v) =>
        v is String ? (DateTime.tryParse(v) ?? DateTime.now()) : DateTime.now();
    DateTime? parseTsNullable(Object? v) =>
        v is String && v.isNotEmpty ? DateTime.tryParse(v) : null;
    return MemoryConflict(
      id: (j['id'] as String?) ?? '',
      cwd: (j['cwd'] as String?) ?? '',
      layerA: ConflictLayerX.parse((j['layer_a'] as String?) ?? 'fact'),
      refA: (j['ref_a'] as String?) ?? '',
      layerB: ConflictLayerX.parse((j['layer_b'] as String?) ?? 'fact'),
      refB: (j['ref_b'] as String?) ?? '',
      evidence: (j['evidence'] as String?) ?? '',
      severity:
          ConflictSeverityX.parse((j['severity'] as String?) ?? 'medium'),
      status: ConflictStatusX.parse((j['status'] as String?) ?? 'pending'),
      detectedAt: parseTs(j['detected_at']),
      decidedAt: parseTsNullable(j['decided_at']),
      decidedBy: j['decided_by'] as String?,
    );
  }

  final String id;
  final String cwd;
  final ConflictLayer layerA;
  final String refA;
  final ConflictLayer layerB;
  final String refB;
  final String evidence;
  final ConflictSeverity severity;
  final ConflictStatus status;
  final DateTime detectedAt;
  final DateTime? decidedAt;
  final String? decidedBy;
}

class MemoryConflictsApi {
  MemoryConflictsApi(this._dio);
  final Dio _dio;

  Future<List<MemoryConflict>> list({
    required String cwd,
    ConflictStatus? status,
    int limit = 50,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/memory/conflicts',
        queryParameters: {
          'cwd': cwd,
          if (status != null) 'status': status.wire,
          'n': limit,
        },
      );
      final raw = res.data?['conflicts'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(MemoryConflict.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Marks the conflict accepted ("operator agrees a contradiction
  // exists and will fix it manually") or dismissed ("detector got
  // it wrong"). action must match ConflictStatus.{accepted|dismissed}.
  Future<void> decide(String id, String action) async {
    try {
      await _dio.post<Map<String, dynamic>>(
        '/api/v1/memory/conflicts/$id/$action',
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Forces a fresh sweep against the configured conflict_detector
  // worker. Returns the number of new conflicts written.
  Future<int> detect(String cwd) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/memory/conflicts/detect',
        queryParameters: {'cwd': cwd},
      );
      return (res.data?['detected'] as int?) ?? 0;
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final memoryConflictsApiProvider = Provider<MemoryConflictsApi>((ref) {
  return MemoryConflictsApi(ref.watch(dioProvider));
});
