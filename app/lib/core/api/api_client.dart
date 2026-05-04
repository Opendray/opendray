import 'dart:convert';
import 'dart:typed_data';
import 'package:dio/dio.dart';
import '../models/session.dart';
import '../models/provider.dart';
import '../../features/workbench/workbench_models.dart';

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

/// Thrown when a consent-lookup endpoint returns 404 — the plugin has no
/// consent row on file (never installed, or already fully revoked). Callers
/// render an empty state rather than surfacing this as an error banner.
class PluginConsentNotFoundException implements Exception {
  final String pluginName;
  PluginConsentNotFoundException(this.pluginName);
  @override
  String toString() => 'No consent for plugin $pluginName';
}

/// Snapshot of a plugin's install-time consent grant as returned by
/// `GET /api/plugins/{name}/consents`.
///
/// [perms] mirrors the raw PermissionsV1 shape (see
/// `plugin/manifest.go`) — heterogeneous per-capability values:
///   • `storage` / `secret` / `telegram` / `llm` → bool
///   • `session` / `clipboard` / `git`           → string ("" = none)
///   • `fs` / `exec` / `http`                    → bool | list | object
///   • `events`                                  → `List<String>`
///
/// Kept as a raw map (rather than a typed 11-field struct) so additions
/// to PermissionsV1 don't require a client release — the UI reads via
/// [isCapGranted], which encodes the rule matrix in one place.
class PluginConsents {
  final String pluginName;
  final Map<String, dynamic> perms;
  /// The install-time PermissionsV1 block from the plugin's manifest.
  /// Empty when the plugin declared none or isn't installed. Used by
  /// the consent settings page to offer a re-grant toggle on any cap
  /// that was revoked — without this, the UI would have to force a
  /// reinstall.
  final Map<String, dynamic> manifestPerms;
  final DateTime? grantedAt;
  final DateTime? updatedAt;

  const PluginConsents({
    required this.pluginName,
    required this.perms,
    this.manifestPerms = const <String, dynamic>{},
    this.grantedAt,
    this.updatedAt,
  });

  factory PluginConsents.fromJson(
    Map<String, dynamic> json, {
    required String pluginName,
  }) {
    final rawPerms = json['perms'];
    final perms = rawPerms is Map
        ? Map<String, dynamic>.from(rawPerms)
        : const <String, dynamic>{};
    final rawManifest = json['manifestPerms'];
    final manifestPerms = rawManifest is Map
        ? Map<String, dynamic>.from(rawManifest)
        : const <String, dynamic>{};
    return PluginConsents(
      pluginName: pluginName,
      perms: perms,
      manifestPerms: manifestPerms,
      grantedAt: _parseTs(json['grantedAt']),
      updatedAt: _parseTs(json['updatedAt']),
    );
  }

  static DateTime? _parseTs(Object? v) {
    if (v is String && v.isNotEmpty) return DateTime.tryParse(v);
    return null;
  }

  /// Returns true if the named capability is currently granted.
  /// Rule matrix:
  ///
  ///   storage / secret / telegram / llm   → bool
  ///   session / clipboard / git           → non-empty string
  ///   fs / exec / http                    → any non-null/non-empty value
  ///   events                              → non-empty array
  ///
  /// Unknown cap keys return false (defensive). Keep this in lock-step
  /// with `plugin/manifest.go` PermissionsV1 and the revoke-cap allowlist
  /// in `gateway/plugins_consents.go`.
  bool isCapGranted(String cap) {
    final v = perms[cap];
    switch (cap) {
      case 'storage':
      case 'secret':
      case 'telegram':
      case 'llm':
        return v == true;
      case 'session':
      case 'clipboard':
      case 'git':
        return v is String && v.isNotEmpty;
      case 'fs':
      case 'exec':
      case 'http':
        if (v == null) return false;
        if (v is bool) return v;
        if (v is List) return v.isNotEmpty;
        if (v is Map) return v.isNotEmpty;
        if (v is String) return v.isNotEmpty;
        return true;
      case 'events':
        return v is List && v.isNotEmpty;
      default:
        return false;
    }
  }

  /// Returns true when the plugin manifest declared a non-zero value for
  /// [cap]. Caps that aren't declared can't be re-granted from the UI
  /// because we'd have nothing to PATCH back in — the switch renders
  /// disabled.
  bool isCapDeclared(String cap) {
    final v = manifestPerms[cap];
    switch (cap) {
      case 'storage':
      case 'secret':
      case 'telegram':
      case 'llm':
        return v == true;
      case 'session':
      case 'clipboard':
      case 'git':
        return v is String && v.isNotEmpty;
      case 'fs':
      case 'exec':
      case 'http':
        if (v == null) return false;
        if (v is bool) return v;
        if (v is List) return v.isNotEmpty;
        if (v is Map) return v.isNotEmpty;
        if (v is String) return v.isNotEmpty;
        return true;
      case 'events':
        return v is List && v.isNotEmpty;
      default:
        return false;
    }
  }
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
  final String Function()? _tokenProvider;
  final void Function()? _onUnauthorized;

  ApiClient({
    required this.baseUrl,
    String Function()? tokenProvider,
    void Function()? onUnauthorized,
  })  : _tokenProvider = tokenProvider,
        _onUnauthorized = onUnauthorized,
        _dio = Dio(BaseOptions(
          baseUrl: baseUrl,
          connectTimeout: const Duration(seconds: 10),
          receiveTimeout: const Duration(seconds: 10),
        )) {
    // Inject Authorization header on every request when a token is available,
    // and trap 401 responses so AuthService can tear down state and route
    // back to the login page.
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) {
        final t = _tokenProvider?.call();
        if (t != null && t.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $t';
        }
        handler.next(options);
      },
      onError: (e, handler) {
        if (e.response?.statusCode == 401) {
          _onUnauthorized?.call();
        }
        handler.next(e);
      },
    ));
  }

  /// Current bearer token (if any) — WsClient appends it as `?token=` since
  /// browsers can't set the Authorization header on a WebSocket handshake.
  String? get token => _tokenProvider?.call();

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

  /// Recent chats observed by the running Telegram bot — drives the
  /// setup wizard's "Detect chat" picker. Empty until the user pastes
  /// a token AND messages the bot at least once.
  Future<Map<String, dynamic>> telegramRecentChats() async {
    final res = await _dio.get('/api/telegram/recent-chats');
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
    final t = token;
    if (t != null && t.isNotEmpty) qp['token'] = t;
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

  // Legacy database endpoints removed — the pg-browser v1 plugin
  // installed from the marketplace exposes list/query via the
  // standard plugin command pipeline (invokePluginCommand).

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
    final t = token;
    final q = (t != null && t.isNotEmpty) ? '?token=${Uri.encodeQueryComponent(t)}' : '';
    return Uri.parse('$scheme://$host/api/tasks/$plugin/run/$runId/ws$q');
  }

  // ── Simulator stream ──────────────────────────────────────

  /// WebSocket URI for live simulator streaming (JPEG frames + input).
  Uri simulatorStreamWsUri({String platform = 'android', String device = ''}) {
    final u = Uri.parse(baseUrl);
    final wsScheme = u.scheme == 'https' ? 'wss' : 'ws';
    final qp = <String, String>{'platform': platform};
    if (device.isNotEmpty) qp['device'] = device;
    final t = token;
    if (t != null && t.isNotEmpty) qp['token'] = t;
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

  // ── Source Control (unified SCM plugin) ─────────────────
  //
  // Wraps /api/source-control/{plugin}/* — the merged successor to the
  // git-viewer + git-forge surfaces. Local-git calls take ?repo=<path>;
  // forge calls take {id} + ?repo=owner/name so one forge instance can
  // answer for many repositories.

  Future<List<Map<String, dynamic>>> scRepos(String plugin) async {
    final res = await _dio.get('/api/source-control/$plugin/repos');
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> scBookmarkAdd(String plugin, String path) async {
    final res = await _dio.post('/api/source-control/$plugin/bookmarks',
        data: {'path': path});
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> scBookmarkRemove(
      String plugin, String path) async {
    final res = await _dio.delete('/api/source-control/$plugin/bookmarks',
        data: {'path': path});
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> scStatus(String plugin, {String repo = ''}) async {
    final res = await _dio.get('/api/source-control/$plugin/status',
        queryParameters: {if (repo.isNotEmpty) 'repo': repo});
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<List<Map<String, dynamic>>> scLog(String plugin,
      {String repo = '', int limit = 0}) async {
    final res = await _dio.get('/api/source-control/$plugin/log',
        queryParameters: {
          if (repo.isNotEmpty) 'repo': repo,
          if (limit > 0) 'limit': limit,
        });
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> scBranches(String plugin,
      {String repo = ''}) async {
    final res = await _dio.get('/api/source-control/$plugin/branches',
        queryParameters: {if (repo.isNotEmpty) 'repo': repo});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  /// Multi-file diff. [mode] is "unstaged" | "staged" | "baseline" | "commit".
  /// [sessionId] is only consulted in baseline mode when [since] is empty.
  /// [commit] is required when mode is "commit" — the SHA to inspect.
  /// Returns {repo, mode, files[]} where each file carries path, oldPath,
  /// status, add, del, isBinary, patch, and previewHtml (.md only).
  Future<Map<String, dynamic>> scDiff(String plugin, {
    required String repo,
    String mode = 'unstaged',
    String since = '',
    String commit = '',
    bool full = false,
    String sessionId = '',
  }) async {
    final res = await _dio.get('/api/source-control/$plugin/diff',
        queryParameters: {
          'repo': repo,
          'mode': mode,
          if (since.isNotEmpty) 'since': since,
          if (commit.isNotEmpty) 'commit': commit,
          if (full) 'full': '1',
          if (sessionId.isNotEmpty) 'sessionId': sessionId,
        });
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> scBaselineSet(String plugin,
      {required String sessionId, required String repo}) async {
    final res = await _dio.post('/api/source-control/$plugin/baseline',
        data: {'sessionId': sessionId, 'repo': repo});
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>?> scBaselineGet(String plugin,
      {required String sessionId, String repo = ''}) async {
    final res = await _dio.get('/api/source-control/$plugin/baseline',
        queryParameters: {
          'sessionId': sessionId,
          if (repo.isNotEmpty) 'repo': repo,
        });
    if (res.data == null) return null;
    if (res.data is Map) return Map<String, dynamic>.from(res.data as Map);
    return null;
  }

  Future<void> scBaselineDelete(String plugin,
      {required String sessionId, required String repo}) async {
    await _dio.delete('/api/source-control/$plugin/baseline', queryParameters: {
      'sessionId': sessionId,
      'repo': repo,
    });
  }

  // ── Source Control: forge instances ─────────────────────

  Future<List<Map<String, dynamic>>> scForgesList(String plugin) async {
    final res = await _dio.get('/api/source-control/$plugin/forges');
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> scForgeCreate(String plugin, {
    required String name,
    required String type,
    required String baseUrl,
    String token = '',
  }) async {
    final res = await _dio.post('/api/source-control/$plugin/forges', data: {
      'name': name,
      'type': type,
      'baseUrl': baseUrl,
      if (token.isNotEmpty) 'token': token,
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<void> scForgeUpdate(String plugin, String id, {
    String name = '',
    String type = '',
    String baseUrl = '',
    String token = '',
  }) async {
    await _dio.put('/api/source-control/$plugin/forges/$id', data: {
      if (name.isNotEmpty) 'name': name,
      if (type.isNotEmpty) 'type': type,
      if (baseUrl.isNotEmpty) 'baseUrl': baseUrl,
      if (token.isNotEmpty) 'token': token,
    });
  }

  Future<void> scForgeDelete(String plugin, String id) async {
    await _dio.delete('/api/source-control/$plugin/forges/$id');
  }

  Future<List<Map<String, dynamic>>> scForgeRepos(
      String plugin, String id, {int limit = 0}) async {
    final res = await _dio.get('/api/source-control/$plugin/forges/$id/repos',
        queryParameters: {if (limit > 0) 'limit': limit});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  // ── Source Control: saved (bookmarked) repos per forge ─────────
  //
  // Backed by /forges/{id}/saved-repos — plugin_kv persistence. The
  // Flutter picker uses this so the user doesn't re-type owner/name
  // on every session, and server-side lastUsedAt bumps surface recents
  // on top automatically.

  Future<List<Map<String, dynamic>>> scSavedReposList(
      String plugin, String forgeId) async {
    final res = await _dio.get(
        '/api/source-control/$plugin/forges/$forgeId/saved-repos');
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> scSavedReposAdd(
      String plugin, String forgeId, {
    required String fullName,
    String description = '',
  }) async {
    final res = await _dio.post(
        '/api/source-control/$plugin/forges/$forgeId/saved-repos',
        data: {
          'fullName': fullName,
          if (description.isNotEmpty) 'description': description,
        });
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> scSavedReposRemove(
      String plugin, String forgeId, {required String fullName}) async {
    final res = await _dio.delete(
        '/api/source-control/$plugin/forges/$forgeId/saved-repos',
        data: {'fullName': fullName});
    return Map<String, dynamic>.from(res.data as Map);
  }

  // ── Source Control: pull requests under a forge instance ────

  Future<List<Map<String, dynamic>>> scPulls(String plugin, String forgeId, {
    required String repo,
    String state = '',
    int limit = 0,
  }) async {
    final res = await _dio.get(
        '/api/source-control/$plugin/forges/$forgeId/pulls',
        queryParameters: {
          'repo': repo,
          if (state.isNotEmpty) 'state': state,
          if (limit > 0) 'limit': limit,
        });
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> scPullDetail(
      String plugin, String forgeId, int number, {required String repo}) async {
    final res = await _dio.get(
        '/api/source-control/$plugin/forges/$forgeId/pulls/$number',
        queryParameters: {'repo': repo});
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<List<Map<String, dynamic>>> scPullDiff(
      String plugin, String forgeId, int number, {required String repo}) async {
    final res = await _dio.get(
        '/api/source-control/$plugin/forges/$forgeId/pulls/$number/diff',
        queryParameters: {'repo': repo});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> scPullComments(
      String plugin, String forgeId, int number, {required String repo}) async {
    final res = await _dio.get(
        '/api/source-control/$plugin/forges/$forgeId/pulls/$number/comments',
        queryParameters: {'repo': repo});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> scPullReviews(
      String plugin, String forgeId, int number, {required String repo}) async {
    final res = await _dio.get(
        '/api/source-control/$plugin/forges/$forgeId/pulls/$number/reviews',
        queryParameters: {'repo': repo});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> scPullReviewComments(
      String plugin, String forgeId, int number, {required String repo}) async {
    final res = await _dio.get(
        '/api/source-control/$plugin/forges/$forgeId/pulls/$number/review-comments',
        queryParameters: {'repo': repo});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> scPullChecks(
      String plugin, String forgeId, int number, {required String repo}) async {
    final res = await _dio.get(
        '/api/source-control/$plugin/forges/$forgeId/pulls/$number/checks',
        queryParameters: {'repo': repo});
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  // ── PostgreSQL (pg-browser plugin) ───────────────────────
  //
  // Every call accepts an optional `database` to override the
  // plugin's configured default without forcing a trip back to
  // Configure. The gateway swaps cfg.Database before the pool lookup;
  // Manager.pool() already keys by database so the override
  // transparently produces an independent connection.

  Future<Map<String, dynamic>> pgQuery(String plugin, String sql,
      {String database = ''}) async {
    final res = await _dio.post('/api/pg/$plugin/query', data: {
      'sql': sql,
      if (database.isNotEmpty) 'database': database,
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Map<String, dynamic>> pgExecute(String plugin, String sql,
      {String database = ''}) async {
    final res = await _dio.post('/api/pg/$plugin/execute', data: {
      'sql': sql,
      if (database.isNotEmpty) 'database': database,
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  /// Every database the PG server hosts (minus templates). Used to
  /// populate the plugin's database picker on first load.
  Future<List<String>> pgDatabases(String plugin) async {
    final res = await _dio.get('/api/pg/$plugin/databases');
    return (res.data as List).cast<String>();
  }

  Future<List<String>> pgSchemas(String plugin, {String database = ''}) async {
    final res = await _dio.get('/api/pg/$plugin/schemas',
        queryParameters: {if (database.isNotEmpty) 'database': database});
    return (res.data as List).cast<String>();
  }

  Future<List<Map<String, dynamic>>> pgTables(String plugin,
      {String schema = '', String database = ''}) async {
    final res = await _dio.get('/api/pg/$plugin/tables', queryParameters: {
      if (schema.isNotEmpty) 'schema': schema,
      if (database.isNotEmpty) 'database': database,
    });
    return (res.data as List).cast<Map<String, dynamic>>();
  }

  Future<List<Map<String, dynamic>>> pgColumns(String plugin,
      {String schema = '',
      required String table,
      String database = ''}) async {
    final res = await _dio.get('/api/pg/$plugin/columns', queryParameters: {
      if (schema.isNotEmpty) 'schema': schema,
      'table': table,
      if (database.isNotEmpty) 'database': database,
    });
    return (res.data as List).cast<Map<String, dynamic>>();
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

  /// Probe the host: which CLIs are installed, where credentials live,
  /// what's already imported. Backs the first-run wizard.
  /// Returns the full HostFacts JSON shape from gateway/host_facts.go.
  Future<Map<String, dynamic>> hostFacts() async {
    final res = await _dio.get('/api/host/facts');
    return Map<String, dynamic>.from(res.data as Map);
  }

  /// Server-side install of the Claude CLI via npm. Admin-gated implicitly
  /// by the existing JWT-protected route group. Returns
  /// `{ installed: bool, path?: string, version?: string, output: string }`.
  Future<Map<String, dynamic>> hostInstallClaudeCLI() async {
    final res = await _dio.post('/api/host/install-claude-cli');
    return Map<String, dynamic>.from(res.data as Map);
  }

  /// Import an existing on-disk .credentials.json as a claude_accounts
  /// row so the user doesn't have to re-OAuth what they already have.
  /// Returns `{ accountId, name, profile }`.
  Future<Map<String, dynamic>> hostImportClaudeCreds({
    required String path,
    String name = '',
  }) async {
    final res = await _dio.post('/api/host/import-claude-creds', data: {
      'path': path,
      if (name.isNotEmpty) 'name': name,
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  /// Reports whether the in-app OAuth flow is usable on this server.
  /// Returns `{ available: bool, version?: string, path?: string,
  /// installHint?: string }`. Call this before showing the "Sign in
  /// with Claude" button so the UI can fall back to manual setup or
  /// surface an install hint when the CLI is missing.
  Future<Map<String, dynamic>> claudeOAuthPreflight() async {
    final res = await _dio.get('/api/claude-accounts/oauth/preflight');
    return Map<String, dynamic>.from(res.data as Map);
  }

  /// Starts an in-app OAuth flow. The server spawns the official Claude
  /// CLI, captures its authorization URL, and returns it. The caller
  /// then opens the URL in the user's browser, collects the auth code,
  /// and POSTs it back via [claudeOAuthComplete].
  ///
  /// `name` and `displayName` are optional — both are auto-derived from
  /// the OAuth profile (email) on completion if absent.
  ///
  /// Returns `{ flowId, authorizationUrl, expiresInSec }`.
  Future<Map<String, dynamic>> claudeOAuthStart({
    String name = '',
    String displayName = '',
  }) async {
    final res = await _dio.post('/api/claude-accounts/oauth/start', data: {
      if (name.isNotEmpty) 'name': name,
      if (displayName.isNotEmpty) 'displayName': displayName,
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  /// Submits the auth code the user pasted from the Anthropic redirect.
  /// On success returns `{ accountId, profile, name, displayName }` and
  /// the new account is registered + enabled. On failure (invalid code,
  /// timeout) returns a 4xx with an error message ready to show.
  Future<Map<String, dynamic>> claudeOAuthComplete(
      String flowId, String code) async {
    final res = await _dio.post(
      '/api/claude-accounts/oauth/$flowId/complete',
      data: {'code': code},
    );
    return Map<String, dynamic>.from(res.data as Map);
  }

  /// Cancels an in-progress OAuth flow. Idempotent — returns 200 even
  /// if the flow already expired or completed. Always called when the
  /// user closes the modal without finishing.
  Future<void> claudeOAuthCancel(String flowId) async {
    await _dio.post('/api/claude-accounts/oauth/$flowId/cancel');
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

  // ── Auth ──────────────────────────────────────────────────

  /// Changes the admin username/password. On success the server returns a
  /// fresh token issued under the new username so the client can swap in
  /// the new credentials without re-login.
  Future<Map<String, dynamic>> changeCredentials({
    required String currentPassword,
    required String newUsername,
    required String newPassword,
  }) async {
    final res = await _dio.post('/api/auth/change-credentials', data: {
      'currentPassword': currentPassword,
      'newUsername': newUsername,
      'newPassword': newPassword,
    });
    return Map<String, dynamic>.from(res.data as Map);
  }

  // ── Health ────────────────────────────────────────────────

  Future<Map<String, dynamic>> health() async {
    final res = await _dio.get('/api/health');
    return res.data;
  }

  // ── Workbench (plugin platform, M1) ───────────────────────

  /// Pulls the current [FlatContributions] snapshot. Safe to call at
  /// app start + after plugin install/uninstall events.
  Future<FlatContributions> getContributions() async {
    final res = await _dio.get('/api/workbench/contributions');
    final data = res.data;
    if (data is Map) {
      return FlatContributions.fromJson(Map<String, dynamic>.from(data));
    }
    return FlatContributions.empty;
  }

  /// Subscribes to `GET /api/workbench/stream` (SSE) and yields each
  /// decoded event envelope (`{kind, plugin?, payload}`). The stream
  /// stays open until cancel; caller handles retry on error/onDone.
  ///
  /// Heartbeat lines (`:`) are silently consumed. Malformed `data:`
  /// frames are dropped without killing the stream.
  Stream<Map<String, dynamic>> workbenchEvents() async* {
    final response = await _dio.get<ResponseBody>(
      '/api/workbench/stream',
      options: Options(
        responseType: ResponseType.stream,
        headers: {'Accept': 'text/event-stream'},
        // No receive timeout — SSE streams are long-lived by design.
        receiveTimeout: Duration.zero,
      ),
    );
    final body = response.data;
    if (body == null) return;

    final buffer = StringBuffer();
    await for (final chunk in body.stream) {
      buffer.write(utf8.decode(chunk, allowMalformed: true));
      while (true) {
        final text = buffer.toString();
        final idx = text.indexOf('\n\n');
        if (idx < 0) break;
        final frame = text.substring(0, idx);
        buffer
          ..clear()
          ..write(text.substring(idx + 2));

        for (final line in frame.split('\n')) {
          if (!line.startsWith('data: ')) continue;
          final jsonStr = line.substring(6);
          try {
            final decoded = jsonDecode(jsonStr);
            if (decoded is Map) {
              yield Map<String, dynamic>.from(decoded);
            }
          } catch (_) {
            // Malformed frame — skip but keep the stream alive.
          }
        }
      }
    }
  }

  /// Invokes a plugin command via the T11 HTTP endpoint. Maps server
  /// error codes to typed exceptions so UI code can branch without
  /// string-matching.
  ///
  /// Throws:
  ///   - [PluginPermissionDeniedException] on 403 EPERM
  ///   - [PluginCommandUnavailableException] on 404 ENOTFOUND (not found)
  ///     or 501 ENOTIMPL (run kind deferred to M2/M3)
  ///   - [ApiException] for anything else (malformed body, 5xx, network)
  Future<InvokeResult> invokePluginCommand(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  }) async {
    try {
      final res = await _dio.post(
        '/api/plugins/$pluginName/commands/$commandId/invoke',
        data: {'args': args ?? const <String, dynamic>{}},
      );
      final data = res.data;
      if (data is Map) {
        return InvokeResult.fromJson(Map<String, dynamic>.from(data));
      }
      return const InvokeResult(kind: '');
    } on DioException catch (e) {
      final status = e.response?.statusCode ?? 0;
      final body = e.response?.data;
      final code = body is Map ? body['code']?.toString() ?? '' : '';
      final msg = body is Map ? body['msg']?.toString() ?? '' : '';
      if (status == 403 && code == 'EPERM') {
        throw PluginPermissionDeniedException(pluginName, commandId,
            msg.isEmpty ? 'permission denied' : msg);
      }
      if (status == 404 && code == 'ENOTFOUND') {
        throw PluginCommandUnavailableException(pluginName, commandId,
            msg.isEmpty ? 'command not found' : msg);
      }
      if (status == 501 && code == 'ENOTIMPL') {
        throw PluginCommandUnavailableException(pluginName, commandId,
            msg.isEmpty ? 'run kind deferred to M2/M3' : msg,
            deferred: true);
      }
      throw apiExceptionFrom(e);
    }
  }

  // ── Plugin consents (T12/T21) ─────────────────────────────
  //
  // Runtime consent surface. GET returns the raw PermissionsV1 shape;
  // the two DELETE endpoints drive hot-revoke — the server fires
  // InvalidateConsent synchronously so WS subs terminate before the
  // HTTP response flushes (the 200 ms SLO asserted in T12).

  /// Fetches the current consent grant for [pluginName]. Throws
  /// [PluginConsentNotFoundException] when the server has no row on
  /// file (treated as an empty state rather than an error banner in
  /// the settings UI).
  Future<PluginConsents> getPluginConsents(String pluginName) async {
    try {
      final res = await _dio.get('/api/plugins/$pluginName/consents');
      final data = res.data;
      if (data is Map) {
        return PluginConsents.fromJson(
          Map<String, dynamic>.from(data),
          pluginName: pluginName,
        );
      }
      return PluginConsents(pluginName: pluginName, perms: const {});
    } on DioException catch (e) {
      if (e.response?.statusCode == 404) {
        throw PluginConsentNotFoundException(pluginName);
      }
      throw apiExceptionFrom(e);
    }
  }

  /// Revokes a single capability on the plugin's install-time grant.
  /// Fires server-side InvalidateConsent so live WS subs get their
  /// EPERM terminal envelope before this future completes.
  ///
  /// Throws:
  ///   - [PluginConsentNotFoundException] on 404 ENOCONSENT
  ///   - [ApiException] on 400 EINVAL (unknown cap) and 5xx
  Future<void> revokePluginCapability(String pluginName, String cap) async {
    try {
      await _dio.delete('/api/plugins/$pluginName/consents/$cap');
    } on DioException catch (e) {
      if (e.response?.statusCode == 404) {
        throw PluginConsentNotFoundException(pluginName);
      }
      throw apiExceptionFrom(e);
    }
  }

  /// Revokes every capability on the plugin and deletes the consent
  /// row. Idempotent — a missing row also returns 200. The server
  /// fires one InvalidateConsent broadcast per previously-granted cap.
  Future<void> revokeAllPluginConsents(String pluginName) async {
    try {
      await _dio.delete('/api/plugins/$pluginName/consents');
    } on DioException catch (e) {
      if (e.response?.statusCode == 404) {
        throw PluginConsentNotFoundException(pluginName);
      }
      throw apiExceptionFrom(e);
    }
  }

  /// Patches a subset of capability grants (M3 T20). `patch` is a
  /// partial PermissionsV1 — only keys present in the map replace
  /// the stored value. Typical use from the Flutter UI: shrink
  /// `fs.read` by one glob, or toggle `storage` off without touching
  /// `events`.
  ///
  /// Fires InvalidateConsent server-side for every touched cap so
  /// active bridge WS subs terminate with EPERM within the 200 ms SLO.
  ///
  /// Throws:
  ///   - [PluginConsentNotFoundException] on 404 ENOENT
  ///   - [ApiException] on 400 EINVAL (unknown cap, bad body) and 5xx
  Future<void> patchPluginConsents(
    String pluginName,
    Map<String, dynamic> patch,
  ) async {
    try {
      await _dio.patch('/api/plugins/$pluginName/consents', data: patch);
    } on DioException catch (e) {
      if (e.response?.statusCode == 404) {
        throw PluginConsentNotFoundException(pluginName);
      }
      throw apiExceptionFrom(e);
    }
  }

  // ── Marketplace / install flow ────────────────────────────

  /// Fetches the plugin marketplace catalog. Empty list when the
  /// server has no catalog configured — the Hub page renders a
  /// "nothing here yet" state rather than an error banner.
  Future<List<MarketplaceEntry>> listMarketplace() async {
    try {
      final res = await _dio.get('/api/marketplace/plugins');
      final data = res.data;
      if (data is Map && data['entries'] is List) {
        return [
          for (final raw in (data['entries'] as List))
            if (raw is Map) MarketplaceEntry.fromJson(Map<String, dynamic>.from(raw)),
        ];
      }
      return const [];
    } on DioException catch (e) {
      throw apiExceptionFrom(e);
    }
  }

  /// Stages an install from a marketplace ref. Returns the pending
  /// install token + the permissions block the user is about to grant
  /// — the caller shows that in a consent dialog before calling
  /// [confirmPluginInstall].
  Future<PendingInstall> installPluginFromMarketplace(String ref) async {
    try {
      final res = await _dio.post(
        '/api/plugins/install',
        data: {'src': 'marketplace://$ref'},
      );
      final data = Map<String, dynamic>.from(res.data as Map);
      return PendingInstall.fromJson(data);
    } on DioException catch (e) {
      throw apiExceptionFrom(e);
    }
  }

  /// Confirms a staged install by echoing the token. Returns the
  /// installed plugin's canonical name.
  Future<String> confirmPluginInstall(String token) async {
    try {
      final res = await _dio.post(
        '/api/plugins/install/confirm',
        data: {'token': token},
      );
      final data = Map<String, dynamic>.from(res.data as Map);
      return (data['name'] as String?) ?? '';
    } on DioException catch (e) {
      throw apiExceptionFrom(e);
    }
  }

  // ── Built-in plugin restore ───────────────────────────────

  /// Lists every manifest bundled in the server binary + its current
  /// state (installed / disabled / uninstalled). Drives the
  /// Settings → Built-in Plugins page.
  Future<List<BuiltinInfo>> listBuiltins() async {
    try {
      final res = await _dio.get('/api/plugins/builtins');
      final data = res.data;
      if (data is Map && data['builtins'] is List) {
        return [
          for (final raw in (data['builtins'] as List))
            if (raw is Map)
              BuiltinInfo.fromJson(Map<String, dynamic>.from(raw)),
        ];
      }
      return const [];
    } on DioException catch (e) {
      throw apiExceptionFrom(e);
    }
  }

  /// Restores a previously-uninstalled built-in plugin by clearing its
  /// tombstone and re-seeding the manifest from embed.FS. Returns the
  /// reinstated Provider so the caller can update its local cache
  /// without re-fetching the full list. Throws on 404 (not a builtin)
  /// or 409 (already installed).
  Future<Provider> restoreBuiltin(String name) async {
    try {
      final res = await _dio.post('/api/plugins/builtins/$name/restore');
      final data = Map<String, dynamic>.from(res.data as Map);
      final provJson = Map<String, dynamic>.from(data['provider'] as Map);
      return Provider.fromJson(provJson);
    } on DioException catch (e) {
      throw apiExceptionFrom(e);
    }
  }

  // ── Plugin user-config (configSchema-backed) ──────────────

  /// Fetches schema + current values for the named plugin. Secret
  /// fields render as `__set__` when stored and `""` otherwise — the
  /// form widget renders accordingly.
  Future<PluginConfig> getPluginConfig(String pluginName) async {
    try {
      final res = await _dio.get('/api/plugins/$pluginName/config');
      final data = Map<String, dynamic>.from(res.data as Map);
      return PluginConfig.fromJson(data);
    } on DioException catch (e) {
      throw apiExceptionFrom(e);
    }
  }

  /// Persists user-edited config values for the named plugin. The
  /// server rewrites plugin_kv / plugin_secret rows and restarts the
  /// sidecar so the new values take effect on the next invoke.
  Future<void> putPluginConfig(
    String pluginName,
    Map<String, dynamic> values,
  ) async {
    try {
      await _dio.put(
        '/api/plugins/$pluginName/config',
        data: {'values': values},
      );
    } on DioException catch (e) {
      throw apiExceptionFrom(e);
    }
  }
}

/// Response shape of GET /api/plugins/{name}/config.
///
/// [values] is always a flat string map — numbers and booleans come
/// back encoded as strings the form widget parses per field type. The
/// secret-field sentinel [kSecretSet] stands in for a stored password
/// the user has not chosen to retype.
class PluginConfig {
  static const String kSecretSet = '__set__';

  final List<PluginConfigField> schema;
  final Map<String, String> values;

  const PluginConfig({required this.schema, required this.values});

  factory PluginConfig.fromJson(Map<String, dynamic> json) {
    final rawSchema = json['schema'];
    final rawValues = json['values'];
    return PluginConfig(
      schema: rawSchema is List
          ? [
              for (final f in rawSchema)
                if (f is Map)
                  PluginConfigField.fromJson(Map<String, dynamic>.from(f)),
            ]
          : const [],
      values: rawValues is Map
          ? {
              for (final e in rawValues.entries)
                e.key.toString(): e.value?.toString() ?? '',
            }
          : const {},
    );
  }

  /// Returns a value map suitable for PUT: string fields verbatim,
  /// numbers parsed, booleans parsed, secret sentinel preserved so
  /// the server knows "don't overwrite".
  Map<String, dynamic> toPutBody(Map<String, String> drafts) {
    final out = <String, dynamic>{};
    for (final f in schema) {
      final raw = drafts[f.key] ?? '';
      switch (f.type) {
        case 'number':
          final n = num.tryParse(raw);
          if (n != null) out[f.key] = n;
          break;
        case 'bool':
        case 'boolean':
          out[f.key] = raw == 'true';
          break;
        case 'secret':
          // Empty input → leave existing value alone by sending the
          // sentinel; non-empty → real new value.
          out[f.key] = raw.isEmpty ? kSecretSet : raw;
          break;
        default:
          out[f.key] = raw;
      }
    }
    return out;
  }
}

// ─── Built-in plugins (Settings → Built-in Plugins) ──────────────────

/// One bundled manifest returned by GET /api/plugins/builtins, paired
/// with its current state. Drives the Settings → Built-in Plugins
/// page so users can see what ships with OpenDray and restore
/// anything they previously uninstalled.
class BuiltinInfo {
  /// Full manifest — same shape the /api/providers endpoint returns
  /// for installed plugins, so the card can reuse existing rendering.
  final Provider provider;

  /// One of "installed" | "disabled" | "uninstalled". "disabled" means
  /// present in the runtime but toggled off by the user (Switch off);
  /// "uninstalled" means it was removed entirely via Uninstall and is
  /// a candidate for the Restore action.
  final String state;

  const BuiltinInfo({required this.provider, required this.state});

  factory BuiltinInfo.fromJson(Map<String, dynamic> json) {
    final provRaw = json['provider'];
    return BuiltinInfo(
      provider: provRaw is Map
          ? Provider.fromJson(Map<String, dynamic>.from(provRaw))
          : const Provider(
              name: '',
              displayName: '',
              description: '',
              icon: '',
              version: '',
              type: '',
              capabilities: Capabilities(),
            ),
      state: (json['state'] as String?) ?? 'uninstalled',
    );
  }

  bool get isInstalled => state == 'installed';
  bool get isDisabled => state == 'disabled';
  bool get isUninstalled => state == 'uninstalled';
}

// ─── Marketplace models ──────────────────────────────────────────────

/// One installable plugin returned by GET /api/marketplace/plugins.
/// Keep this as a plain value type — the Hub card reads fields directly
/// and the consent preview dialog reads [permissions] verbatim.
class MarketplaceEntry {
  final String name;
  final String version;
  final String publisher;
  final String displayName;
  final String description;
  final String icon;
  final String form;
  final List<String> tags;
  final Map<String, dynamic> permissions;
  /// Trust level: "official" / "verified" / "community". Empty
  /// defaults to "community" per spec. Rendered as a badge on the
  /// Hub card.
  final String trust;
  // i18n overlays — see provider.dart ConfigField.labelZh.
  final String displayNameZh;
  final String descriptionZh;

  const MarketplaceEntry({
    required this.name,
    required this.version,
    required this.publisher,
    this.displayName = '',
    this.description = '',
    this.icon = '',
    this.form = '',
    this.tags = const [],
    this.permissions = const {},
    this.trust = 'community',
    this.displayNameZh = '',
    this.descriptionZh = '',
    List<PluginConfigField>? configSchema,
  }) : _rawConfigSchema = configSchema;

  factory MarketplaceEntry.fromJson(Map<String, dynamic> json) {
    final rawTags = json['tags'];
    final rawPerms = json['permissions'];
    final rawSchema = json['configSchema'];
    return MarketplaceEntry(
      name: (json['name'] as String?) ?? '',
      version: (json['version'] as String?) ?? '',
      publisher: (json['publisher'] as String?) ?? '',
      displayName: (json['displayName'] as String?) ?? '',
      description: (json['description'] as String?) ?? '',
      icon: (json['icon'] as String?) ?? '',
      form: (json['form'] as String?) ?? '',
      tags: rawTags is List
          ? [for (final t in rawTags) t.toString()]
          : const [],
      permissions: rawPerms is Map
          ? Map<String, dynamic>.from(rawPerms)
          : const {},
      trust: (json['trust'] as String?) ?? 'community',
      displayNameZh: (json['displayName_zh'] as String?) ?? '',
      descriptionZh: (json['description_zh'] as String?) ?? '',
      configSchema: rawSchema is List
          ? [
              for (final f in rawSchema)
                if (f is Map)
                  PluginConfigField.fromJson(Map<String, dynamic>.from(f)),
            ]
          : null,
    );
  }

  /// `NAME@VERSION` — used both as the marketplace ref and as a
  /// stable display key when name collisions with installed plugins
  /// need disambiguation.
  String get ref => '$name@$version';

  /// Parsed ConfigSchema carried on the marketplace catalog row so the
  /// Hub can render the install-time config form without a second
  /// manifest fetch. Empty list = no user-facing config.
  List<PluginConfigField> get configSchema {
    final raw = _rawConfigSchema;
    if (raw == null) return const [];
    return raw;
  }

  /// Raw config schema stashed at construction time. Null when the
  /// catalog row omitted configSchema, which is treated as "no form"
  /// upstream.
  final List<PluginConfigField>? _rawConfigSchema;
}

/// One user-editable field in a plugin's configSchema. Shape mirrors
/// [plugin.ConfigField] on the server so the Hub form and the Plugin-
/// page re-configure flow share a single renderer.
class PluginConfigField {
  final String key;
  final String label;
  final String type; // string | number | bool | select | secret
  final String description;
  final String placeholder;
  final dynamic defaultValue;
  final List<String> options;
  final bool required;
  final String group;
  // i18n overlays — see provider.dart ConfigField.labelZh.
  final String labelZh;
  final String descriptionZh;
  final String placeholderZh;

  const PluginConfigField({
    required this.key,
    required this.label,
    required this.type,
    this.description = '',
    this.placeholder = '',
    this.defaultValue,
    this.options = const [],
    this.required = false,
    this.group = '',
    this.labelZh = '',
    this.descriptionZh = '',
    this.placeholderZh = '',
  });

  factory PluginConfigField.fromJson(Map<String, dynamic> json) {
    final rawOpts = json['options'];
    return PluginConfigField(
      key: (json['key'] as String?) ?? '',
      label: (json['label'] as String?) ?? '',
      type: (json['type'] as String?) ?? 'string',
      description: (json['description'] as String?) ?? '',
      placeholder: (json['placeholder'] as String?) ?? '',
      defaultValue: json['default'],
      options: rawOpts is List
          ? [for (final o in rawOpts) o.toString()]
          : const [],
      required: (json['required'] as bool?) ?? false,
      group: (json['group'] as String?) ?? '',
      labelZh: (json['label_zh'] as String?) ?? '',
      descriptionZh: (json['description_zh'] as String?) ?? '',
      placeholderZh: (json['placeholder_zh'] as String?) ?? '',
    );
  }

  /// True for password-style fields. The GET /config response
  /// returns `__set__` for these when a value is already stored; the
  /// form widget treats that sentinel as "keep existing — leave blank
  /// to change".
  bool get isSecret => type == 'secret';
}

/// Response of POST /api/plugins/install — a staged install waiting
/// for user confirmation. [permissions] mirrors the PermissionsV1 shape
/// so the consent dialog can reuse the same renderer as
/// [PluginConsents.perms].
class PendingInstall {
  final String token;
  final String name;
  final String version;
  final Map<String, dynamic> permissions;
  final String manifestHash;
  final DateTime? expiresAt;

  const PendingInstall({
    required this.token,
    required this.name,
    required this.version,
    this.permissions = const {},
    this.manifestHash = '',
    this.expiresAt,
  });

  factory PendingInstall.fromJson(Map<String, dynamic> json) {
    final rawPerms = json['perms'];
    final rawExp = json['expiresAt'];
    return PendingInstall(
      token: (json['token'] as String?) ?? '',
      name: (json['name'] as String?) ?? '',
      version: (json['version'] as String?) ?? '',
      permissions: rawPerms is Map
          ? Map<String, dynamic>.from(rawPerms)
          : const {},
      manifestHash: (json['manifestHash'] as String?) ?? '',
      expiresAt: rawExp is String ? DateTime.tryParse(rawExp) : null,
    );
  }
}
