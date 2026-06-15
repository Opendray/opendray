import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/mcps + /api/v1/mcps/_secrets. Mobile exposes the
// observability + light-mutation surface (toggle, delete, edit
// secret values). Server create/edit (command / args / env /
// headers — multi-field config) stays web-only because pasting
// long shell args on a phone is a tax not worth charging.

class McpServer {
  McpServer({
    required this.id,
    required this.name,
    required this.transport,
    required this.enabled,
    this.description,
    this.command,
    this.args,
    this.env,
    this.url,
    this.headers,
    this.builtin = false,
  });

  factory McpServer.fromJson(Map<String, dynamic> json) {
    final argsRaw = json['args'];
    final args = argsRaw is List
        ? argsRaw.whereType<String>().toList()
        : <String>[];
    final envRaw = json['env'];
    final env = envRaw is Map
        ? envRaw.map((k, v) => MapEntry(k.toString(), v.toString()))
        : <String, String>{};
    final headersRaw = json['headers'];
    final headers = headersRaw is Map
        ? headersRaw.map((k, v) => MapEntry(k.toString(), v.toString()))
        : <String, String>{};
    return McpServer(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      description: json['description'] as String?,
      // stdio (default) | sse | http
      transport: (json['transport'] as String?)?.isNotEmpty ?? false
          ? json['transport'] as String
          : 'stdio',
      command: json['command'] as String?,
      args: args.isEmpty ? null : args,
      env: env.isEmpty ? null : env,
      url: json['url'] as String?,
      headers: headers.isEmpty ? null : headers,
      enabled: json['enabled'] as bool? ?? false,
      builtin: json['builtin'] as bool? ?? false,
    );
  }

  final String id;
  final String name;
  final String? description;
  final String transport;
  final String? command;
  final List<String>? args;
  final Map<String, String>? env;
  final String? url;
  final Map<String, String>? headers;
  final bool enabled;
  // Gateway-provided servers (opendray-memory) — auto-attached to every
  // session, read-only in the registry (no edit / delete / toggle).
  final bool builtin;
}

class McpSecretsState {
  McpSecretsState({
    required this.path,
    required this.present,
    required this.encrypted,
    required this.keys,
  });

  factory McpSecretsState.fromJson(Map<String, dynamic> json) {
    final raw = json['keys'];
    final keys = raw is List
        ? raw.whereType<String>().toList()
        : <String>[];
    return McpSecretsState(
      path: json['path'] as String? ?? '',
      present: json['present'] as bool? ?? false,
      encrypted: json['encrypted'] as bool? ?? false,
      keys: keys,
    );
  }

  // Absolute filesystem path of the secrets file. Sensitive — only
  // shown server-side and to admin clients; useful for "where am I
  // pointing?" debugging.
  final String path;
  // Whether the secrets file exists on disk yet.
  final bool present;
  // True when the file is AES-GCM encrypted (key in OS keychain),
  // false when it fell back to plaintext storage.
  final bool encrypted;
  // The key names only — values are NEVER returned by the API.
  final List<String> keys;
}

class McpApi {
  McpApi(this._dio);
  final Dio _dio;

  Future<List<McpServer>> list() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/mcps');
      final raw = res.data?['servers'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(McpServer.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<McpServer> get(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/mcps/$id');
      return McpServer.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /mcps creates a new server from the supplied body. id must
  // match the body's server.id. Server-side validates the schema
  // and 400s with a parseable message on malformed JSON.
  Future<void> create({
    required String id,
    required Map<String, dynamic> server,
  }) async {
    try {
      await _dio.post<void>(
        '/api/v1/mcps',
        data: {'id': id, 'server': server},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PUT /mcps/{id} replaces the whole Server object with the body.
  // Used for full edits (config JSON) — see also setEnabled for the
  // common toggle-only path.
  Future<void> replace({
    required String id,
    required Map<String, dynamic> server,
  }) async {
    try {
      await _dio.put<void>(
        '/api/v1/mcps/$id',
        data: {'id': id, 'server': server},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PUT /mcps/{id} replaces the whole Server object. Mobile uses this
  // for the common enabled-toggle path; create/edit goes through
  // the dedicated create() / replace() above.
  Future<void> setEnabled(String id, {required bool enabled}) async {
    try {
      final current = await get(id);
      final body = {
        'id': current.id,
        'server': {
          'id': current.id,
          'name': current.name,
          if (current.description != null) 'description': current.description,
          'transport': current.transport,
          if (current.command != null) 'command': current.command,
          if (current.args != null) 'args': current.args,
          if (current.env != null) 'env': current.env,
          if (current.url != null) 'url': current.url,
          if (current.headers != null) 'headers': current.headers,
          'enabled': enabled,
        },
      };
      await _dio.put<void>('/api/v1/mcps/$id', data: body);
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/mcps/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // GET /mcps/_secrets returns the vault state without any values.
  Future<McpSecretsState> secretsGet() async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/mcps/_secrets');
      return McpSecretsState.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PUT /mcps/_secrets/{key} body `{value}`. The key path keeps the
  // value out of any URL logging. Server validates key against
  // `[A-Za-z_][A-Za-z0-9_]*`.
  Future<McpSecretsState> secretsSet(String key, String value) async {
    try {
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/mcps/_secrets/$key',
        data: {'value': value},
      );
      return McpSecretsState.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> secretsDelete(String key) async {
    try {
      await _dio.delete<void>('/api/v1/mcps/_secrets/$key');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final mcpApiProvider = Provider<McpApi>((ref) {
  return McpApi(ref.watch(dioProvider));
});
