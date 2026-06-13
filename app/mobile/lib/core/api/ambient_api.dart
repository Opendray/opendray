import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps the ambient-memory runtime endpoints that govern how memories
// are captured and injected:
//   /api/v1/memory-capture-rules      — when a session is summarised
//   /api/v1/memory-injection-profiles — what context is pre-loaded
//
// Mirrors app/shared/src/lib/memoryAmbient.ts. Mobile surfaces these
// read-mostly: list + quick toggles (enable, run-now) for capture
// rules and a strategy switch for injection profiles. Full create /
// delete with trigger-config editing stays on web — too dense for a
// phone form — so this client intentionally omits create/delete.

// Trigger kinds: after_messages | on_idle | k_chars | manual.
class CaptureRule {
  CaptureRule({
    required this.id,
    required this.name,
    required this.enabled,
    required this.triggerKind,
    required this.targetScope,
    this.summarizerProviderId,
  });

  factory CaptureRule.fromJson(Map<String, dynamic> j) => CaptureRule(
        id: j['id'] as String? ?? '',
        name: j['name'] as String? ?? '',
        enabled: j['enabled'] as bool? ?? false,
        triggerKind: j['trigger_kind'] as String? ?? '',
        targetScope: j['target_scope'] as String? ?? 'project',
        summarizerProviderId: j['summarizer_provider_id'] as String?,
      );

  final String id;
  final String name;
  final bool enabled;
  final String triggerKind;
  final String targetScope;
  final String? summarizerProviderId;

  CaptureRule copyWith({bool? enabled}) => CaptureRule(
        id: id,
        name: name,
        enabled: enabled ?? this.enabled,
        triggerKind: triggerKind,
        targetScope: targetScope,
        summarizerProviderId: summarizerProviderId,
      );
}

// Strategy kinds: none | top_k_recent | top_k_relevant | on_keyword |
// manual_only | hybrid.
class InjectionProfile {
  InjectionProfile({
    required this.id,
    required this.strategyKind,
    this.sessionId,
  });

  factory InjectionProfile.fromJson(Map<String, dynamic> j) => InjectionProfile(
        id: j['id'] as String? ?? '',
        strategyKind: j['strategy_kind'] as String? ?? 'none',
        sessionId: j['session_id'] as String?,
      );

  final String id;
  final String strategyKind;
  final String? sessionId;
}

class AmbientApi {
  AmbientApi(this._dio);
  final Dio _dio;

  Future<List<CaptureRule>> listCaptureRules() async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/memory-capture-rules');
      final raw = res.data?['rules'];
      return raw is List
          ? raw
              .whereType<Map<String, dynamic>>()
              .map(CaptureRule.fromJson)
              .toList()
          : <CaptureRule>[];
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<CaptureRule> setCaptureRuleEnabled(
    String id, {
    required bool enabled,
  }) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/memory-capture-rules/$id',
        data: {'enabled': enabled},
      );
      return CaptureRule.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Returns the number of sessions the rule was invoked against.
  Future<int> runCaptureRuleNow(String id) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/memory-capture-rules/$id/run-now',
      );
      return (res.data?['sessions_invoked'] as num?)?.toInt() ?? 0;
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<InjectionProfile>> listInjectionProfiles() async {
    try {
      final res = await _dio
          .get<Map<String, dynamic>>('/api/v1/memory-injection-profiles');
      final raw = res.data?['profiles'];
      return raw is List
          ? raw
              .whereType<Map<String, dynamic>>()
              .map(InjectionProfile.fromJson)
              .toList()
          : <InjectionProfile>[];
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<InjectionProfile> setInjectionStrategy(
    String id,
    String strategyKind,
  ) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/memory-injection-profiles/$id',
        data: {'strategy_kind': strategyKind},
      );
      return InjectionProfile.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final ambientApiProvider = Provider<AmbientApi>((ref) {
  return AmbientApi(ref.watch(dioProvider));
});
