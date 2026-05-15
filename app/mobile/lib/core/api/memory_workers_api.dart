import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/memory/workers/* — the M25 per-task worker
// configuration + metrics surface. Mirrors web's
// app/shared/src/lib/memoryWorkers.ts.

// Mobile-side mirror of internal/memory/worker.TaskKind. Stays in
// lockstep with that file — adding a 4th/5th/6th/7th touchpoint
// here is purely a UI surface, the backend rows already exist
// after the M-PA / M-PC / M-PE migrations.
enum WorkerTaskKind {
  gatekeeper,
  cleaner,
  gitactivity,
  transcript,
  planDrift,
  conflictDetector,
  capture,
}

extension WorkerTaskKindX on WorkerTaskKind {
  // Persisted wire string. Matches memory_workers.task CHECK
  // constraint values; uses snake_case where the camelCase Dart
  // enum can't represent the spelling directly. Exhaustive switch
  // (no default) so Dart will flag any future enum addition.
  String get wire {
    switch (this) {
      case WorkerTaskKind.gatekeeper:
        return 'gatekeeper';
      case WorkerTaskKind.cleaner:
        return 'cleaner';
      case WorkerTaskKind.gitactivity:
        return 'gitactivity';
      case WorkerTaskKind.transcript:
        return 'transcript';
      case WorkerTaskKind.planDrift:
        return 'plan_drift';
      case WorkerTaskKind.conflictDetector:
        return 'conflict_detector';
      case WorkerTaskKind.capture:
        return 'capture';
    }
  }

  String get label {
    switch (this) {
      case WorkerTaskKind.gatekeeper:
        return 'Gatekeeper';
      case WorkerTaskKind.cleaner:
        return 'Cleaner librarian';
      case WorkerTaskKind.gitactivity:
        return 'Git activity summariser';
      case WorkerTaskKind.transcript:
        return 'Session transcript summariser';
      case WorkerTaskKind.planDrift:
        return 'Plan drift detector';
      case WorkerTaskKind.conflictDetector:
        return 'Cross-layer conflict detector';
      case WorkerTaskKind.capture:
        return 'Capture engine';
    }
  }

  String get description {
    switch (this) {
      case WorkerTaskKind.gatekeeper:
        return 'Pre-write filter on every memory_store. High frequency (<500ms target) — summarizer-only.';
      case WorkerTaskKind.cleaner:
        return 'Periodic LLM librarian. Judges aged memories as keep / stale / duplicate.';
      case WorkerTaskKind.gitactivity:
        return 'git log → 2-3 paragraph narrative every 24h. Naturally fits an agent worker.';
      case WorkerTaskKind.transcript:
        return 'Session-end "what did the agent do" summary. Naturally fits an agent worker.';
      case WorkerTaskKind.planDrift:
        return 'After each session ends, checks whether the project plan needs updating and files a proposal. Fits an agent worker for richer reasoning.';
      case WorkerTaskKind.conflictDetector:
        return 'Daily scan that finds contradictions between facts / plan / goal / journal. Higher-quality model = fewer false positives.';
      case WorkerTaskKind.capture:
        return 'Per-trigger fact extraction from session transcripts. Agent mode gives noticeably better facts on long sessions; summarizer mode is cheap and local.';
    }
  }

  // Gatekeeper stays summarizer-only (<500ms latency budget on the
  // memory_store hot path). Every other task supports Agent mode.
  bool get agentSupported => this != WorkerTaskKind.gatekeeper;
}

enum WorkerKind { summarizer, agent }

extension WorkerKindX on WorkerKind {
  String get wire => name;
  static WorkerKind parse(String s) =>
      s == 'agent' ? WorkerKind.agent : WorkerKind.summarizer;
}

class WorkerConfig {
  WorkerConfig({
    required this.task,
    required this.kind,
    required this.enabled,
    required this.updatedAt,
    this.summarizerId,
    this.providerId,
    this.accountId,
  });

  factory WorkerConfig.fromJson(Map<String, dynamic> j) {
    WorkerTaskKind parseTask(String s) {
      switch (s) {
        case 'gatekeeper':
          return WorkerTaskKind.gatekeeper;
        case 'cleaner':
          return WorkerTaskKind.cleaner;
        case 'gitactivity':
          return WorkerTaskKind.gitactivity;
        case 'transcript':
          return WorkerTaskKind.transcript;
        case 'plan_drift':
          return WorkerTaskKind.planDrift;
        case 'conflict_detector':
          return WorkerTaskKind.conflictDetector;
        case 'capture':
          return WorkerTaskKind.capture;
        default:
          return WorkerTaskKind.transcript;
      }
    }

    return WorkerConfig(
      task: parseTask(j['Task'] as String? ?? j['task'] as String? ?? ''),
      kind: WorkerKindX.parse(
          j['Kind'] as String? ?? j['kind'] as String? ?? 'summarizer'),
      summarizerId: (j['SummarizerID'] ?? j['summarizer_id']) as String?,
      providerId: (j['ProviderID'] ?? j['provider_id']) as String?,
      accountId: (j['AccountID'] ?? j['account_id']) as String?,
      enabled: (j['Enabled'] ?? j['enabled']) as bool? ?? true,
      updatedAt: DateTime.tryParse(
              (j['UpdatedAt'] ?? j['updated_at']) as String? ?? '') ??
          DateTime.now(),
    );
  }

  final WorkerTaskKind task;
  final WorkerKind kind;
  final String? summarizerId;
  final String? providerId;
  final String? accountId;
  final bool enabled;
  final DateTime updatedAt;
}

class CallSummary {
  CallSummary({
    required this.id,
    required this.task,
    required this.workerKind,
    required this.providerId,
    required this.accountId,
    required this.startedAt,
    required this.durationMs,
    required this.success,
    required this.errorMessage,
  });

  factory CallSummary.fromJson(Map<String, dynamic> j) {
    WorkerTaskKind parseTask(String s) {
      switch (s) {
        case 'gatekeeper':
          return WorkerTaskKind.gatekeeper;
        case 'cleaner':
          return WorkerTaskKind.cleaner;
        case 'gitactivity':
          return WorkerTaskKind.gitactivity;
        case 'transcript':
          return WorkerTaskKind.transcript;
        case 'plan_drift':
          return WorkerTaskKind.planDrift;
        case 'conflict_detector':
          return WorkerTaskKind.conflictDetector;
        case 'capture':
          return WorkerTaskKind.capture;
        default:
          return WorkerTaskKind.transcript;
      }
    }

    return CallSummary(
      id: (j['ID'] ?? j['id']) as int? ?? 0,
      task: parseTask((j['Task'] ?? j['task']) as String? ?? ''),
      workerKind: WorkerKindX.parse(
          (j['WorkerKind'] ?? j['worker_kind']) as String? ?? 'summarizer'),
      providerId: (j['ProviderID'] ?? j['provider_id']) as String? ?? '',
      accountId: (j['AccountID'] ?? j['account_id']) as String? ?? '',
      startedAt: DateTime.tryParse(
              (j['StartedAt'] ?? j['started_at']) as String? ?? '') ??
          DateTime.now(),
      durationMs: (j['DurationMS'] ?? j['duration_ms']) as int? ?? 0,
      success: (j['Success'] ?? j['success']) as bool? ?? false,
      errorMessage:
          (j['ErrorMessage'] ?? j['error_message']) as String? ?? '',
    );
  }

  final int id;
  final WorkerTaskKind task;
  final WorkerKind workerKind;
  final String providerId;
  final String accountId;
  final DateTime startedAt;
  final int durationMs;
  final bool success;
  final String errorMessage;
}

class TestResult {
  TestResult({
    required this.ok,
    required this.durationMs,
    this.workerKind,
    this.providerId,
    this.preview,
    this.error,
  });

  factory TestResult.fromJson(Map<String, dynamic> j) => TestResult(
        ok: j['ok'] as bool? ?? false,
        durationMs: j['duration_ms'] as int? ?? 0,
        workerKind: j['worker_kind'] as String?,
        providerId: j['provider_id'] as String?,
        preview: j['preview'] as String?,
        error: j['error'] as String?,
      );

  final bool ok;
  final int durationMs;
  final String? workerKind;
  final String? providerId;
  final String? preview;
  final String? error;
}

class MemoryWorkersApi {
  MemoryWorkersApi(this._dio);
  final Dio _dio;

  Future<List<WorkerConfig>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/memory/workers/',
      );
      final raw = res.data?['workers'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(WorkerConfig.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<WorkerConfig> upsert({
    required WorkerTaskKind task,
    required WorkerKind kind,
    String? summarizerId,
    String? providerId,
    String? accountId,
    bool enabled = true,
  }) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/memory/workers/${task.wire}',
        data: {
          'kind': kind.wire,
          if (summarizerId != null) 'summarizer_id': summarizerId,
          if (providerId != null) 'provider_id': providerId,
          if (accountId != null) 'account_id': accountId,
          'enabled': enabled,
        },
      );
      return WorkerConfig.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<TestResult> test(WorkerTaskKind task) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/memory/workers/${task.wire}/test',
      );
      return TestResult.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<CallSummary>> calls({
    WorkerTaskKind? task,
    int limit = 100,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/memory/workers/calls',
        queryParameters: {
          if (task != null) 'task': task.wire,
          'n': limit,
        },
      );
      final raw = res.data?['calls'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(CallSummary.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final memoryWorkersApiProvider = Provider<MemoryWorkersApi>((ref) {
  return MemoryWorkersApi(ref.watch(dioProvider));
});
