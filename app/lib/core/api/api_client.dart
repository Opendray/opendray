import 'dart:typed_data';
import 'package:dio/dio.dart';
import '../models/session.dart';
import '../models/provider.dart';

/// Thrown when a server endpoint returns 4xx/5xx. Carries the server's own
/// error message (from `{"error": "..."}`) so UI can show a useful reason
/// instead of a bare "status code 500".
class ApiException implements Exception {
  final int statusCode;
  final String message;
  final String path;
  ApiException(this.statusCode, this.message, this.path);
  @override
  String toString() => message.isNotEmpty ? message : 'HTTP $statusCode';
}

/// Extracts the server's error message from a DioException, falling back
/// to a sensible default when the body isn't JSON or the server is offline.
ApiException apiExceptionFrom(DioException e) {
  final status = e.response?.statusCode ?? 0;
  final path = e.requestOptions.path;
  String msg = '';
  final data = e.response?.data;
  if (data is Map && data['error'] is String) {
    msg = data['error'] as String;
  } else if (data is String && data.isNotEmpty) {
    msg = data;
  } else if (e.type == DioExceptionType.connectionTimeout ||
             e.type == DioExceptionType.receiveTimeout ||
             e.type == DioExceptionType.sendTimeout) {
    msg = 'Server timed out';
  } else if (e.type == DioExceptionType.connectionError) {
    msg = 'Can\'t reach server (${e.message ?? "network error"})';
  } else {
    msg = e.message ?? 'HTTP $status';
  }
  return ApiException(status, msg, path);
}

class ApiClient {
  final Dio _dio;
  final String baseUrl;
  final Map<String, String> extraHeaders;

  ApiClient({required this.baseUrl, this.extraHeaders = const {}})
      : _dio = Dio(BaseOptions(
          baseUrl: baseUrl,
          connectTimeout: const Duration(seconds: 10),
          receiveTimeout: const Duration(seconds: 10),
          headers: extraHeaders,
        ));

  /// Callers wrap API calls with this so DioException bodies bubble up as
  /// readable user-facing messages. Rethrows ApiException; passes other
  /// errors through unchanged.
  static Future<T> describeErrors<T>(Future<T> Function() fn) async {
    try {
      return await fn();
    } on DioException catch (e) {
      throw apiExceptionFrom(e);
    }
  }

  // ── Sessions ──────────────────────────────────────────────

  Future<List<Session>> listSessions() async {
    final res = await _dio.get('/api/sessions');
    return (res.data as List).map((e) => Session.fromJson(e)).toList();
  }

  Future<Session> createSession({
    required String cwd,
    required String sessionType,
    String name = '',
    String model = '',
    List<String> extraArgs = const [],
    String? claudeAccountId,
    String? llmProviderId,
  }) async {
    final res = await _dio.post('/api/sessions', data: {
      'cwd': cwd,
      'sessionType': sessionType,
      'name': name,
      'model': model,
      'extraArgs': extraArgs,
      if (claudeAccountId != null && claudeAccountId.isNotEmpty)
        'claudeAccountId': claudeAccountId,
      if (llmProviderId != null && llmProviderId.isNotEmpty)
        'llmProviderId': llmProviderId,
    });
    return Session.fromJson(res.data);
  }

  /// Re-binds a running or stopped Claude session to a different account —
  /// the server stops it, updates the binding, and restarts (the resume
  /// flow from [ClaudeSessionID] kicks in automatically). Pass null/empty
  /// to unbind (fall back to system keychain).
  Future<void> switchSessionAccount(String id, String? accountId) async {
    await _dio.post('/api/sessions/$id/switch-account',
        data: {'accountId': accountId ?? ''});
  }

  Future<Session> getSession(String id) async {
    final res = await _dio.get('/api/sessions/$id');
    return Session.fromJson(res.data);
  }

  Future<void> startSession(String id) async {
    await _dio.post('/api/sessions/$id/start');
  }

  Future<void> stopSession(String id) async {
    await _dio.post('/api/sessions/$id/stop');
  }

  Future<void> deleteSession(String id) async {
    await _dio.delete('/api/sessions/$id');
  }

  Future<void> sendInput(String id, String input) async {
    await _dio.post('/api/sessions/$id/input', data: {'input': input});
  }

  Future<void> resizeSession(String id, int rows, int cols) async {
    await _dio.post('/api/sessions/$id/resize', data: {'rows': rows, 'cols': cols});
  }

  /// Uploads an image to the session and returns `{path, name, size}`.
  /// The server stores the file outside the session's cwd. The client is
  /// responsible for deciding whether (and when) to inject the path into the
  /// terminal via [sendInput] — no automatic injection.
  Future<Map<String, dynamic>> attachImage(
    String sessionId,
    Uint8List bytes, {
    String mimeType = 'image/png',
  }) async {
    final res = await _dio.post(
      '/api/sessions/$sessionId/image',
      data: Stream.fromIterable([bytes]),
      options: Options(
        contentType: mimeType,
        headers: {'Content-Length': bytes.length.toString()},
        responseType: ResponseType.json,
      ),
    );
    return Map<String, dynamic>.from(res.data as Map);
  }

  // ── Providers ─────────────────────────────────────────────

  Future<List<ProviderInfo>> listProviders() async {
    final res = await _dio.get('/api/providers');
    return (res.data as List).map((e) => ProviderInfo.fromJson(e)).toList();
  }

  Future<void> toggleProvider(String name, bool enabled) async {
    await _dio.patch('/api/providers/$name/toggle', data: {'enabled': enabled});
  }

  Future<void> updateProviderConfig(String name, Map<String, dynamic> config) async {
    await _dio.put('/api/providers/$name/config', data: config);
  }

  Future<void> deleteProvider(String name) async {
    await _dio.delete('/api/providers/$name');
  }

  Future<List<ModelDef>> detectModels(String name) async {
    final res = await _dio.get('/api/providers/$name/models');
    return (res.data as List).map((e) => ModelDef.fromJson(e)).toList();
  }

  // ── Docs ───────────────────────────────────────────────────

  Future<List<Map<String, dynamic>>> docsTree(String plugin, {String path = ''}) async {
    final res = await _dio.get('/api/docs/$plugin/tree', queryParameters: {'path': path});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> docsFile(String plugin, String path) async {
    final res = await _dio.get('/api/docs/$plugin/file', queryParameters: {'path': path});
    return res.data;
  }

  Future<List<Map<String, dynamic>>> docsSearch(String plugin, String query) async {
    final res = await _dio.get('/api/docs/$plugin/search', queryParameters: {'q': query});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  // ── Files ──────────────────────────────────────────────────

  Future<List<Map<String, dynamic>>> filesTree(String plugin, {String? path}) async {
    final res = await _dio.get('/api/files/$plugin/tree', queryParameters: {'path': path ?? ''});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> filesFile(String plugin, String path) async {
    final res = await _dio.get('/api/files/$plugin/file', queryParameters: {'path': path});
    return res.data;
  }

  Future<List<Map<String, dynamic>>> filesSearch(String plugin, String query, {String? basePath}) async {
    final params = <String, dynamic>{'q': query};
    if (basePath != null) params['path'] = basePath;
    final res = await _dio.get('/api/files/$plugin/search', queryParameters: params);
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  // ── Telegram ──────────────────────────────────────────────

  Future<Map<String, dynamic>> telegramStatus() async {
    final res = await _dio.get('/api/telegram/status');
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> telegramTest({String? text}) async {
    final res = await _dio.post('/api/telegram/test',
        data: {'text': ?text});
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<List<Map<String, dynamic>>> telegramLinks() async {
    final res = await _dio.get('/api/telegram/links');
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> telegramUnlink(int chatId) async {
    final res = await _dio.post('/api/telegram/unlink',
        data: {'chatId': chatId});
    return Map<String, dynamic>.from(res.data as Map);
  }

  // ── Logs ───────────────────────────────────────────────────

  /// Lists log files / directories under [path]. Empty path returns the
  /// configured allowed roots as virtual entries.
  Future<List<Map<String, dynamic>>> logsList(String plugin,
      {String path = ''}) async {
    final res = await _dio.get('/api/logs/$plugin/list',
        queryParameters: {'path': path});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  /// WebSocket URI that streams the tail of [path] — server sends each
  /// completed line as a text frame. Pass [grep] to filter server-side.
  Uri logsTailWsUri(String plugin, String path, {String grep = ''}) {
    final u = Uri.parse(baseUrl);
    final wsScheme = u.scheme == 'https' ? 'wss' : 'ws';
    final qp = <String, String>{'path': path};
    if (grep.isNotEmpty) qp['grep'] = grep;
    return Uri(
      scheme: wsScheme,
      host: u.host,
      port: u.hasPort ? u.port : null,
      path: '/api/logs/$plugin/tail/ws',
      queryParameters: qp,
    );
  }

  // ── Files (continued) ─────────────────────────────────────

  /// Creates a new directory `name` inside `parent`. Returns the absolute path.
  Future<String> filesMkdir(String plugin, String parent, String name) async {
    final res = await _dio.post(
      '/api/files/$plugin/mkdir',
      data: {'parent': parent, 'name': name},
    );
    return (res.data as Map)['path'] as String;
  }

  // ── Database ──────────────────────────────────────────────

  Future<List<Map<String, dynamic>>> dbDatabases(String plugin) async {
    final res = await _dio.get('/api/database/$plugin/databases');
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> dbSchemas(String plugin, {String? db}) async {
    final res = await _dio.get('/api/database/$plugin/schemas',
      queryParameters: db != null && db.isNotEmpty ? {'db': db} : null);
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> dbTables(String plugin, {String schema = '', String? db}) async {
    final params = <String, dynamic>{'schema': schema};
    if (db != null && db.isNotEmpty) params['db'] = db;
    final res = await _dio.get('/api/database/$plugin/tables', queryParameters: params);
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> dbColumns(String plugin, String schema, String table, {String? db}) async {
    final params = <String, dynamic>{'schema': schema, 'table': table};
    if (db != null && db.isNotEmpty) params['db'] = db;
    final res = await _dio.get('/api/database/$plugin/columns', queryParameters: params);
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> dbPreview(String plugin, String schema, String table, {int limit = 100, String? db}) async {
    final params = <String, dynamic>{'schema': schema, 'table': table, 'limit': limit};
    if (db != null && db.isNotEmpty) params['db'] = db;
    final res = await _dio.get('/api/database/$plugin/preview', queryParameters: params);
    return res.data;
  }

  Future<Map<String, dynamic>> dbQuery(String plugin, String sql, {String? db}) async {
    final res = await _dio.post('/api/database/$plugin/query',
      queryParameters: db != null && db.isNotEmpty ? {'db': db} : null,
      data: {'sql': sql});
    return res.data;
  }

  // ── Tasks ─────────────────────────────────────────────────

  Future<List<Map<String, dynamic>>> tasksList(String plugin, {String? path}) async {
    final res = await _dio.get('/api/tasks/$plugin/list',
        queryParameters: {'path': path ?? ''});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> tasksRun(String plugin, String taskId, {String? path}) async {
    final res = await _dio.post('/api/tasks/$plugin/run', data: {
      'taskId': taskId,
      'path': path ?? '',
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<List<Map<String, dynamic>>> tasksRuns(String plugin) async {
    final res = await _dio.get('/api/tasks/$plugin/runs');
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> tasksRunGet(String plugin, String runId) async {
    final res = await _dio.get('/api/tasks/$plugin/run/$runId');
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<void> tasksRunStop(String plugin, String runId) async {
    await _dio.post('/api/tasks/$plugin/run/$runId/stop');
  }

  /// Builds a `ws(s)://…/api/tasks/{plugin}/run/{runId}/ws` URL for the
  /// caller to feed into a WebSocketChannel. Centralised here so the
  /// http→ws scheme rewrite stays identical to the session WS path.
  Uri tasksRunWsUri(String plugin, String runId) {
    final scheme = baseUrl.startsWith('https') ? 'wss' : 'ws';
    final host = baseUrl.replaceAll(RegExp(r'^https?://'), '');
    return Uri.parse('$scheme://$host/api/tasks/$plugin/run/$runId/ws');
  }

  // ── Simulator stream ──────────────────────────────────────

  /// WebSocket URI for live simulator streaming (JPEG frames + input).
  Uri simulatorStreamWsUri({String platform = 'android', String device = ''}) {
    final u = Uri.parse(baseUrl);
    final wsScheme = u.scheme == 'https' ? 'wss' : 'ws';
    final qp = <String, String>{'platform': platform};
    if (device.isNotEmpty) qp['device'] = device;
    return Uri(
      scheme: wsScheme,
      host: u.host,
      port: u.hasPort ? u.port : null,
      path: '/api/simulator/stream/ws',
      queryParameters: qp,
    );
  }

  // ── Preview ───────────────────────────────────────────────

  Future<List<Map<String, dynamic>>> previewDiscover() async {
    final res = await _dio.get('/api/preview/discover');
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  // ── Simulator ─────────────────────────────────────────────

  Future<Uint8List> simulatorScreenshot({
    required String platform,
    String device = '',
  }) async {
    final params = <String, dynamic>{'platform': platform};
    if (device.isNotEmpty) params['device'] = device;
    final res = await _dio.get<List<int>>(
      '/api/simulator/screenshot',
      queryParameters: params,
      options: Options(responseType: ResponseType.bytes),
    );
    return Uint8List.fromList(res.data!);
  }

  Future<void> simulatorInput({
    required String platform,
    String device = '',
    required String action,
    int x = 0,
    int y = 0,
    int x2 = 0,
    int y2 = 0,
    int duration = 300,
    String key = '',
    String text = '',
  }) async {
    await _dio.post('/api/simulator/input', data: {
      'platform': platform,
      'device': device,
      'action': action,
      'x': x, 'y': y, 'x2': x2, 'y2': y2,
      'duration': duration,
      'key': key,
      'text': text,
    });
  }

  // ── MCP ───────────────────────────────────────────────────

  Future<List<Map<String, dynamic>>> mcpServers() async {
    final res = await _dio.get('/api/mcp/servers');
    return ((res.data as Map)['servers'] as List? ?? [])
        .cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> mcpCreateServer(Map<String, dynamic> body) async {
    final res = await _dio.post('/api/mcp/servers', data: body);
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> mcpUpdateServer(String id, Map<String, dynamic> body) async {
    final res = await _dio.put('/api/mcp/servers/$id', data: body);
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<void> mcpToggleServer(String id, bool enabled) async {
    await _dio.patch('/api/mcp/servers/$id/toggle', data: {'enabled': enabled});
  }

  Future<void> mcpDeleteServer(String id) async {
    await _dio.delete('/api/mcp/servers/$id');
  }

  Future<List<String>> mcpAgents() async {
    final res = await _dio.get('/api/mcp/agents');
    return ((res.data as Map)['agents'] as List? ?? []).cast<String>();
  }

  // ── Git panel ─────────────────────────────────────────────

  Future<Map<String, dynamic>> gitStatus(String plugin, {String path = ''}) async {
    final res = await _dio.get('/api/git/$plugin/status',
        queryParameters: {'path': path});
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<String> gitDiff(String plugin, {
    String path = '',
    bool staged = false,
    String since = '',
    String file = '',
  }) async {
    final res = await _dio.get('/api/git/$plugin/diff', queryParameters: {
      'path': path,
      if (staged) 'staged': 'true',
      if (since.isNotEmpty) 'since': since,
      if (file.isNotEmpty) 'file': file,
    });
    return ((res.data as Map)['diff'] ?? '') as String;
  }

  Future<List<Map<String, dynamic>>> gitLog(String plugin,
      {String path = '', int limit = 0}) async {
    final res = await _dio.get('/api/git/$plugin/log', queryParameters: {
      'path': path,
      if (limit > 0) 'limit': limit,
    });
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> gitBranches(String plugin,
      {String path = ''}) async {
    final res = await _dio.get('/api/git/$plugin/branches',
        queryParameters: {'path': path});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<void> gitStage(String plugin, String path, List<String> files) async {
    await _dio.post('/api/git/$plugin/stage',
        data: {'path': path, 'files': files});
  }

  Future<void> gitUnstage(String plugin, String path, List<String> files) async {
    await _dio.post('/api/git/$plugin/unstage',
        data: {'path': path, 'files': files});
  }

  Future<void> gitDiscard(String plugin, String path, List<String> files) async {
    await _dio.post('/api/git/$plugin/discard',
        data: {'path': path, 'files': files});
  }

  Future<Map<String, dynamic>> gitCommit(String plugin, String path, String message) async {
    final res = await _dio.post('/api/git/$plugin/commit',
        data: {'path': path, 'message': message});
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> gitSessionSnapshot(
      String plugin, String sessionId, {String path = ''}) async {
    final res = await _dio.post('/api/git/$plugin/session/snapshot',
        data: {'sessionId': sessionId, if (path.isNotEmpty) 'path': path});
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<String> gitSessionDiff(String plugin, String sessionId) async {
    final res = await _dio.get('/api/git/$plugin/session/diff',
        queryParameters: {'sessionId': sessionId});
    return ((res.data as Map)['diff'] ?? '') as String;
  }

  // ── Claude Multi-Account ─────────────────────────────────

  Future<List<Map<String, dynamic>>> claudeAccounts() async {
    final res = await _dio.get('/api/claude-accounts');
    return ((res.data as Map)['accounts'] as List? ?? [])
        .cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> claudeAccountCreate({
    required String name,
    String displayName = '',
    String configDir = '',
    String tokenPath = '',
    String token = '',
    String description = '',
    bool enabled = true,
  }) async {
    final res = await _dio.post('/api/claude-accounts', data: {
      'name': name,
      'displayName': displayName,
      'configDir': configDir,
      'tokenPath': tokenPath,
      'token': token,
      'description': description,
      'enabled': enabled,
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> claudeAccountUpdate(
    String id,
    Map<String, dynamic> body,
  ) async {
    final res = await _dio.put('/api/claude-accounts/$id', data: body);
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<void> claudeAccountToggle(String id, bool enabled) async {
    await _dio.patch('/api/claude-accounts/$id/toggle',
        data: {'enabled': enabled});
  }

  Future<void> claudeAccountSetToken(String id, String token) async {
    await _dio.put('/api/claude-accounts/$id/token', data: {'token': token});
  }

  Future<void> claudeAccountDelete(String id) async {
    await _dio.delete('/api/claude-accounts/$id');
  }

  /// Scans `~/.claude-accounts/tokens/` on the server host and creates
  /// account rows for any *.token files not already imported. Returns a
  /// `{imported: [...], skipped: [...]}` summary.
  Future<Map<String, dynamic>> claudeAccountImportLocal() async {
    final res = await _dio.post('/api/claude-accounts/import-local');
    return Map<String, dynamic>.from(res.data as Map);
  }

  // ── LLM Providers ────────────────────────────────────────
  //
  // Address book of OpenAI-compatible endpoints. Sessions (OpenCode
  // and any other OpenAI-native agent) bind to one of these at creation
  // time; the gateway injects OPENAI_BASE_URL / OPENAI_API_KEY / model
  // into the CLI at spawn. The /models endpoint probes upstream
  // /v1/models; on failure the UI falls back to free-text model entry.

  Future<List<Map<String, dynamic>>> llmProviders() async {
    final res = await _dio.get('/api/llm-providers');
    return ((res.data as Map)['providers'] as List? ?? [])
        .cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> llmProviderCreate({
    required String name,
    String displayName = '',
    String providerType = 'openai-compat',
    required String baseUrl,
    String apiKeyEnv = '',
    String description = '',
    bool enabled = true,
  }) async {
    final res = await _dio.post('/api/llm-providers', data: {
      'name': name,
      'displayName': displayName,
      'providerType': providerType,
      'baseUrl': baseUrl,
      'apiKeyEnv': apiKeyEnv,
      'description': description,
      'enabled': enabled,
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> llmProviderUpdate(
    String id,
    Map<String, dynamic> body,
  ) async {
    final res = await _dio.put('/api/llm-providers/$id', data: body);
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<void> llmProviderToggle(String id, bool enabled) async {
    await _dio.patch('/api/llm-providers/$id/toggle',
        data: {'enabled': enabled});
  }

  Future<void> llmProviderDelete(String id) async {
    await _dio.delete('/api/llm-providers/$id');
  }

  /// Probes the provider's /v1/models. Returns null if the upstream
  /// can't be reached — callers should surface "flip to manual entry"
  /// to the user.
  Future<List<String>?> llmProviderModels(String id) async {
    try {
      final res = await _dio.get('/api/llm-providers/$id/models');
      final models = (res.data as Map)['models'] as List? ?? [];
      return models.map((e) => e.toString()).toList();
    } catch (_) {
      return null;
    }
  }

  // ── Health ────────────────────────────────────────────────

  Future<Map<String, dynamic>> health() async {
    final res = await _dio.get('/api/health');
    return res.data;
  }
}
