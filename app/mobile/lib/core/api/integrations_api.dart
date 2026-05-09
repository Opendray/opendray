import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/integrations + /api/v1/integrations/_calls. Mobile
// More tab uses these to surface "who's calling me" — list of
// registered integrations and the per-integration call log.

class Integration {
  Integration({
    required this.id,
    required this.name,
    required this.baseUrl,
    required this.routePrefix,
    required this.scopes,
    required this.enabled,
    required this.healthStatus,
    required this.createdAt,
    required this.isSystem,
    this.version,
    this.healthLastSeen,
    this.rotatedAt,
  });

  factory Integration.fromJson(Map<String, dynamic> json) {
    final scopesRaw = json['scopes'];
    final scopes = scopesRaw is List
        ? scopesRaw.whereType<String>().toList()
        : <String>[];
    return Integration(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      baseUrl: json['base_url'] as String? ?? '',
      routePrefix: json['route_prefix'] as String? ?? '',
      scopes: scopes,
      version: json['version'] as String?,
      enabled: json['enabled'] as bool? ?? false,
      healthStatus: json['health_status'] as String? ?? 'unknown',
      healthLastSeen:
          DateTime.tryParse(json['health_last_seen'] as String? ?? '')?.toUtc(),
      createdAt: DateTime.tryParse(json['created_at'] as String? ?? '')
              ?.toUtc() ??
          DateTime.fromMillisecondsSinceEpoch(0),
      rotatedAt:
          DateTime.tryParse(json['rotated_at'] as String? ?? '')?.toUtc(),
      isSystem: json['is_system'] as bool? ?? false,
    );
  }

  final String id;
  final String name;
  final String baseUrl;
  final String routePrefix;
  final List<String> scopes;
  final String? version;
  final bool enabled;
  // unknown | healthy | degraded | unhealthy
  final String healthStatus;
  final DateTime? healthLastSeen;
  final DateTime createdAt;
  final DateTime? rotatedAt;
  // System rows are managed by opendray itself (e.g. opendray-memory MCP)
  // — not operator-mutable.
  final bool isSystem;
}

class CallEntry {
  CallEntry({
    required this.id,
    required this.timestamp,
    required this.integrationId,
    required this.direction,
    required this.method,
    required this.path,
    required this.statusCode,
    required this.durationMs,
    this.bytesWritten,
    this.requestId,
    this.resourceKind,
    this.resourceId,
  });

  factory CallEntry.fromJson(Map<String, dynamic> json) => CallEntry(
        id: (json['id'] as num?)?.toInt() ?? 0,
        timestamp:
            DateTime.tryParse(json['ts'] as String? ?? '')?.toUtc() ??
                DateTime.now().toUtc(),
        integrationId: json['integration_id'] as String? ?? '',
        direction: json['direction'] as String? ?? '',
        method: json['method'] as String? ?? '',
        path: json['path'] as String? ?? '',
        statusCode: (json['status_code'] as num?)?.toInt() ?? 0,
        durationMs: (json['duration_ms'] as num?)?.toInt() ?? 0,
        bytesWritten: (json['bytes_written'] as num?)?.toInt(),
        requestId: json['request_id'] as String?,
        resourceKind: json['resource_kind'] as String?,
        resourceId: json['resource_id'] as String?,
      );

  final int id;
  final DateTime timestamp;
  final String integrationId;
  // "inbound" — integration → opendray, "outbound" — opendray → integration.
  final String direction;
  final String method;
  final String path;
  final int statusCode;
  final int durationMs;
  final int? bytesWritten;
  final String? requestId;
  final String? resourceKind;
  final String? resourceId;
}

class CallsPage {
  CallsPage({required this.entries, required this.nextCursor});

  factory CallsPage.fromJson(Map<String, dynamic> json) {
    final raw = json['entries'];
    final entries = raw is List
        ? raw
            .whereType<Map<String, dynamic>>()
            .map(CallEntry.fromJson)
            .toList()
        : <CallEntry>[];
    return CallsPage(
      entries: entries,
      nextCursor: json['next_cursor'] as String?,
    );
  }

  final List<CallEntry> entries;
  // String form of the int cursor returned by the server (or null at end).
  final String? nextCursor;
}

class IntegrationsApi {
  IntegrationsApi(this._dio);
  final Dio _dio;

  Future<List<Integration>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/integrations');
      final raw = res.data?['integrations'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(Integration.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<Integration> get(String id) async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/integrations/$id');
      return Integration.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<CallsPage> calls({
    String? integrationId,
    String? direction,
    int? statusClass,
    DateTime? since,
    DateTime? until,
    String? cursor,
    int limit = 100,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/integrations/_calls',
        queryParameters: {
          if (integrationId != null && integrationId.isNotEmpty)
            'integration_id': integrationId,
          if (direction != null && direction.isNotEmpty) 'direction': direction,
          if (statusClass != null) 'status_class': statusClass,
          if (since != null) 'since': since.toUtc().toIso8601String(),
          if (until != null) 'until': until.toUtc().toIso8601String(),
          if (cursor != null && cursor.isNotEmpty) 'cursor': cursor,
          'limit': limit,
        },
      );
      return CallsPage.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final integrationsApiProvider = Provider<IntegrationsApi>((ref) {
  return IntegrationsApi(ref.watch(dioProvider));
});
