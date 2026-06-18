import 'dart:convert';

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
    this.defaultProviderId = '',
    this.defaultModel = '',
    this.defaultClaudeAccountId = '',
    this.mcpServers = const [],
    this.systemPrompt = '',
    this.bypassPermissions = false,
  });

  factory Integration.fromJson(Map<String, dynamic> json) {
    final scopesRaw = json['scopes'];
    final scopes = scopesRaw is List
        ? scopesRaw.whereType<String>().toList()
        : <String>[];
    final mcpRaw = json['mcp_servers'];
    final mcpServers = mcpRaw is List ? List<dynamic>.from(mcpRaw) : <dynamic>[];
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
      defaultProviderId: json['default_provider_id'] as String? ?? '',
      defaultModel: json['default_model'] as String? ?? '',
      defaultClaudeAccountId:
          json['default_claude_account_id'] as String? ?? '',
      mcpServers: mcpServers,
      systemPrompt: json['system_prompt'] as String? ?? '',
      bypassPermissions: (json['permission_mode'] as String?) == 'bypass',
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
  // Spawn defaults applied to sessions this integration creates when the
  // request omits the field (the request always wins). Empty = no default.
  final String defaultProviderId;
  final String defaultModel;
  final String defaultClaudeAccountId;
  // Spawn-profile extras applied to sessions this integration creates.
  // mcpServers is a JSON array of MCP server specs (round-tripped raw).
  final List<dynamic> mcpServers;
  final String systemPrompt;
  final bool bypassPermissions;
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

// RegisterResult is returned from POST /integrations and POST
// /integrations/{id}/rotate-key. The plaintext APIKey is the only
// time the operator can see it — the server only retains a bcrypt
// hash afterwards. UI must reveal it once and warn before dismissal.
class RegisterResult {
  RegisterResult({required this.integration, required this.apiKey});

  factory RegisterResult.fromJson(Map<String, dynamic> json) {
    final raw = json['integration'];
    final integration = raw is Map<String, dynamic>
        ? Integration.fromJson(raw)
        : Integration.fromJson({});
    return RegisterResult(
      integration: integration,
      apiKey: json['api_key'] as String? ?? '',
    );
  }

  final Integration integration;
  final String apiKey;
}

// ProxyResponse is the raw upstream result of a reverse-proxy request
// made through the console — captured for ANY status code (a 404/500 is
// a legitimate result to display, not an error to swallow).
class ProxyResponse {
  ProxyResponse({
    required this.status,
    required this.durationMs,
    required this.headers,
    required this.body,
    this.contentType,
  });

  final int status;
  final int durationMs;
  final List<MapEntry<String, String>> headers;
  final String body;
  final String? contentType;
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

  // POST /integrations — server returns the freshly-minted plaintext
  // API key (only chance to see it) plus the persisted Integration.
  Future<RegisterResult> register({
    required String name,
    required String baseUrl,
    required String routePrefix,
    List<String>? scopes,
    String? version,
    String? defaultProviderId,
    String? defaultModel,
    String? defaultClaudeAccountId,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/integrations',
        data: {
          'name': name,
          'base_url': baseUrl,
          'route_prefix': routePrefix,
          if (scopes != null && scopes.isNotEmpty) 'scopes': scopes,
          if (version != null && version.isNotEmpty) 'version': version,
          if (defaultProviderId != null && defaultProviderId.isNotEmpty)
            'default_provider_id': defaultProviderId,
          if (defaultModel != null && defaultModel.isNotEmpty)
            'default_model': defaultModel,
          if (defaultClaudeAccountId != null &&
              defaultClaudeAccountId.isNotEmpty)
            'default_claude_account_id': defaultClaudeAccountId,
        },
      );
      return RegisterResult.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /integrations/{id}. Each field optional — pass null to
  // leave it untouched.
  Future<Integration> update(
    String id, {
    String? baseUrl,
    List<String>? scopes,
    String? version,
    bool? enabled,
    String? defaultProviderId,
    String? defaultModel,
    String? defaultClaudeAccountId,
    Object? mcpServers,
    String? systemPrompt,
    bool? bypassPermissions,
  }) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/integrations/$id',
        data: {
          if (baseUrl != null) 'base_url': baseUrl,
          if (scopes != null) 'scopes': scopes,
          if (version != null) 'version': version,
          if (enabled != null) 'enabled': enabled,
          // Non-null means "set" — empty string clears the default.
          if (defaultProviderId != null)
            'default_provider_id': defaultProviderId,
          if (defaultModel != null) 'default_model': defaultModel,
          if (defaultClaudeAccountId != null)
            'default_claude_account_id': defaultClaudeAccountId,
          if (mcpServers != null) 'mcp_servers': mcpServers,
          if (systemPrompt != null) 'system_prompt': systemPrompt,
          if (bypassPermissions != null)
            'permission_mode': bypassPermissions ? 'bypass' : 'default',
        },
      );
      return Integration.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/integrations/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Sends a request through /api/v1/proxy/{prefix}/* and returns the raw
  // upstream response. Uses a bare Dio clone (same base URL + auth header
  // as the shared client) so it can surface ANY status code — the shared
  // dioProvider interceptor rejects non-2xx, which would hide a perfectly
  // valid 404/500 the operator wants to inspect.
  Future<ProxyResponse> proxy({
    required String routePrefix,
    required String method,
    required String path,
    Map<String, String> extraHeaders = const {},
    String? body,
  }) async {
    final hasBody = method != 'GET' &&
        method != 'DELETE' &&
        body != null &&
        body.trim().isNotEmpty;
    final headers = <String, dynamic>{
      ..._dio.options.headers, // Accept + Authorization from the session
      ...extraHeaders,
    };
    if (hasBody &&
        !headers.keys.any((k) => k.toLowerCase() == 'content-type')) {
      headers['Content-Type'] = 'application/json';
    }
    final raw = Dio(
      BaseOptions(
        baseUrl: _dio.options.baseUrl,
        connectTimeout: const Duration(seconds: 8),
        receiveTimeout: const Duration(seconds: 30),
        headers: headers,
        validateStatus: (_) => true,
        responseType: ResponseType.plain,
      ),
    );
    final p = path.startsWith('/') ? path : '/$path';
    final url = '/api/v1/proxy/$routePrefix$p';
    final sw = Stopwatch()..start();
    try {
      final res = await raw.request<String>(
        url,
        data: hasBody ? body : null,
        options: Options(method: method),
      );
      sw.stop();
      final out = <MapEntry<String, String>>[];
      res.headers.forEach(
        (name, values) => out.add(MapEntry(name, values.join(', '))),
      );
      var bodyText = res.data ?? '';
      final ct = res.headers.value('content-type');
      if (ct != null && ct.contains('application/json')) {
        try {
          bodyText =
              const JsonEncoder.withIndent('  ').convert(jsonDecode(bodyText));
        } on Object {
          // Not JSON despite the header — leave the raw text as-is.
        }
      }
      return ProxyResponse(
        status: res.statusCode ?? 0,
        durationMs: sw.elapsedMilliseconds,
        headers: out,
        body: bodyText,
        contentType: ct,
      );
    } on DioException catch (e) {
      sw.stop();
      throw toApiException(e);
    }
  }

  // POST /integrations/{id}/rotate-key — invalidates the previous key
  // and returns the freshly-minted plaintext for one-time display.
  // Server returns 403 for is_system integrations.
  Future<RegisterResult> rotateKey(String id) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/integrations/$id/rotate-key',
      );
      return RegisterResult.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final integrationsApiProvider = Provider<IntegrationsApi>((ref) {
  return IntegrationsApi(ref.watch(dioProvider));
});
