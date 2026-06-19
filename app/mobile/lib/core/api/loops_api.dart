import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Loop Engine (autoloop) endpoint surface. The gateway returns a bare JSON
// array for list/runs and a bare object for a single loop (see
// internal/autoloop/handler.go) — no wrapper envelope.

enum LoopKind {
  interval,
  goal,
  unknown;

  static LoopKind parse(String? raw) => switch (raw) {
        'interval' => LoopKind.interval,
        'goal' => LoopKind.goal,
        _ => LoopKind.unknown,
      };

  String get wire => switch (this) {
        LoopKind.interval => 'interval',
        LoopKind.goal => 'goal',
        LoopKind.unknown => 'goal',
      };
}

enum LoopStatus {
  pending,
  running,
  paused,
  done,
  stopped,
  failed,
  escalated,
  unknown;

  static LoopStatus parse(String? raw) => switch (raw) {
        'pending' => LoopStatus.pending,
        'running' => LoopStatus.running,
        'paused' => LoopStatus.paused,
        'done' => LoopStatus.done,
        'stopped' => LoopStatus.stopped,
        'failed' => LoopStatus.failed,
        'escalated' => LoopStatus.escalated,
        _ => LoopStatus.unknown,
      };

  bool get isTerminal =>
      this == LoopStatus.done ||
      this == LoopStatus.stopped ||
      this == LoopStatus.failed ||
      this == LoopStatus.escalated;
}

class LoopSummary {
  LoopSummary({
    required this.id,
    required this.sessionId,
    required this.origin,
    required this.kind,
    required this.status,
    required this.prompt,
    required this.maxIterations,
    required this.failureCap,
    required this.iteration,
    this.goal,
    this.intervalSeconds,
    this.lastVerdict,
    this.lastReason,
    this.deadlineAt,
  });

  factory LoopSummary.fromJson(Map<String, dynamic> json) => LoopSummary(
        id: json['id'] as String? ?? '',
        sessionId: json['session_id'] as String? ?? '',
        origin: json['origin'] as String? ?? 'operator',
        kind: LoopKind.parse(json['kind'] as String?),
        status: LoopStatus.parse(json['status'] as String?),
        prompt: json['prompt'] as String? ?? '',
        maxIterations: (json['max_iterations'] as num?)?.toInt() ?? 0,
        failureCap: (json['failure_cap'] as num?)?.toInt() ?? 0,
        iteration: (json['iteration'] as num?)?.toInt() ?? 0,
        goal: json['goal'] as String?,
        intervalSeconds: (json['interval_seconds'] as num?)?.toInt(),
        lastVerdict: json['last_verdict'] as String?,
        lastReason: json['last_reason'] as String?,
        deadlineAt: DateTime.tryParse(json['deadline_at'] as String? ?? ''),
      );

  final String id;
  final String sessionId;
  final String origin;
  final LoopKind kind;
  final LoopStatus status;
  final String prompt;
  final int maxIterations;
  final int failureCap;
  final int iteration;
  final String? goal;
  final int? intervalSeconds;
  final String? lastVerdict;
  final String? lastReason;
  final DateTime? deadlineAt;

  bool get isIntegration => origin == 'integration';
}

class LoopRun {
  LoopRun({
    required this.id,
    required this.iteration,
    required this.prompt,
    this.verdict,
    this.reason,
  });

  factory LoopRun.fromJson(Map<String, dynamic> json) => LoopRun(
        id: (json['id'] as num?)?.toInt() ?? 0,
        iteration: (json['iteration'] as num?)?.toInt() ?? 0,
        prompt: json['prompt'] as String? ?? '',
        verdict: json['verdict'] as String?,
        reason: json['reason'] as String?,
      );

  final int id;
  final int iteration;
  final String prompt;
  final String? verdict;
  final String? reason;
}

class CreateLoopRequest {
  CreateLoopRequest({
    required this.sessionId,
    required this.kind,
    required this.prompt,
    required this.maxIterations,
    required this.deadlineAt,
    required this.failureCap,
    this.goal,
    this.intervalSeconds,
  });

  final String sessionId;
  final LoopKind kind;
  final String prompt;
  final int maxIterations;
  final DateTime deadlineAt;
  final int failureCap;
  final String? goal;
  final int? intervalSeconds;

  Map<String, dynamic> toJson() => {
        'session_id': sessionId,
        'kind': kind.wire,
        'prompt': prompt,
        'max_iterations': maxIterations,
        'deadline_at': deadlineAt.toUtc().toIso8601String(),
        'failure_cap': failureCap,
        if (kind == LoopKind.goal && goal != null) 'goal': goal,
        if (kind == LoopKind.interval && intervalSeconds != null)
          'interval_seconds': intervalSeconds,
      };
}

class LoopsApi {
  LoopsApi(this._dio);
  final Dio _dio;

  Future<List<LoopSummary>> list() async {
    try {
      final res = await _dio.get<List<dynamic>>('/api/v1/loops');
      final raw = res.data;
      if (raw == null) return [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(LoopSummary.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<LoopRun>> runs(String id) async {
    try {
      final res = await _dio.get<List<dynamic>>('/api/v1/loops/$id/runs');
      final raw = res.data;
      if (raw == null) return [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(LoopRun.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<LoopSummary> create(CreateLoopRequest req) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/loops',
        data: req.toJson(),
      );
      return LoopSummary.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<LoopSummary> _transition(String id, String action) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/loops/$id/$action',
      );
      return LoopSummary.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<LoopSummary> pause(String id) => _transition(id, 'pause');
  Future<LoopSummary> resume(String id) => _transition(id, 'resume');
  Future<LoopSummary> stop(String id) => _transition(id, 'stop');
}

final loopsApiProvider = Provider<LoopsApi>((ref) {
  return LoopsApi(ref.watch(dioProvider));
});

final loopsListProvider = FutureProvider.autoDispose<List<LoopSummary>>((ref) {
  return ref.watch(loopsApiProvider).list();
});

final loopRunsProvider =
    FutureProvider.autoDispose.family<List<LoopRun>, String>((ref, id) {
  return ref.watch(loopsApiProvider).runs(id);
});
