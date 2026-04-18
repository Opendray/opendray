import 'package:dio/dio.dart';

/// Thin client for the bootstrap-token-gated /api/setup/* endpoints.
///
/// Lives outside the main ApiClient because:
///   • No JWT yet — auth is token-in-header instead
///   • Different base URL story — on web this is same-origin; on mobile
///     it's whatever the user points ServerConfig at
///   • It's short-lived, discarded after finalize
class SetupApi {
  final Dio _dio;
  final String bootstrapToken;

  SetupApi({
    required String baseUrl,
    required this.bootstrapToken,
    Map<String, String> extraHeaders = const {},
  }) : _dio = Dio(BaseOptions(
          baseUrl: baseUrl,
          connectTimeout: const Duration(seconds: 10),
          // Probes that spin up real DB connections can take a few seconds;
          // keep the receive timeout generous.
          receiveTimeout: const Duration(seconds: 15),
          headers: {
            ...extraHeaders,
            'X-Setup-Token': bootstrapToken,
          },
          // Accept 4xx so callers can read the server's error body instead
          // of a generic DioException.
          validateStatus: (s) => s != null && s < 500,
        ));

  /// Status is the only unauthenticated endpoint — callers use it before
  /// they have a token.
  static Future<SetupStatus> status(String baseUrl, {
    Map<String, String> extraHeaders = const {},
  }) async {
    final dio = Dio(BaseOptions(
      baseUrl: baseUrl,
      connectTimeout: const Duration(seconds: 8),
      receiveTimeout: const Duration(seconds: 8),
      headers: extraHeaders,
      validateStatus: (s) => s != null && s < 500,
    ));
    final res = await dio.get('/api/setup/status');
    if (res.statusCode != 200 || res.data is! Map) {
      throw SetupApiException(res.statusCode ?? 0, 'unexpected response');
    }
    return SetupStatus.fromJson(Map<String, dynamic>.from(res.data as Map));
  }

  /// Tries the loopback-only /api/setup/token. Returns null on 403 (not
  /// loopback) or any error — caller then falls back to URL query param.
  static Future<String?> loopbackToken(String baseUrl, {
    Map<String, String> extraHeaders = const {},
  }) async {
    try {
      final dio = Dio(BaseOptions(
        baseUrl: baseUrl,
        connectTimeout: const Duration(seconds: 3),
        receiveTimeout: const Duration(seconds: 3),
        headers: extraHeaders,
        validateStatus: (s) => s != null && s < 500,
      ));
      final res = await dio.get('/api/setup/token');
      if (res.statusCode == 200 && res.data is Map) {
        final t = (res.data as Map)['token'];
        if (t is String && t.isNotEmpty) return t;
      }
    } catch (_) {}
    return null;
  }

  Future<EnvProbe> env() async {
    final res = await _dio.get('/api/setup/env');
    _ensureOk(res, 'env');
    return EnvProbe.fromJson(Map<String, dynamic>.from(res.data as Map));
  }

  Future<void> testDB({
    required String host,
    required int port,
    required String user,
    required String password,
    required String name,
    String sslmode = 'disable',
  }) async {
    final res = await _dio.post('/api/setup/db/test', data: {
      'host': host,
      'port': port,
      'user': user,
      'password': password,
      'name': name,
      'sslmode': sslmode,
    });
    _ensureOk(res, 'db/test');
  }

  Future<void> commitDBEmbedded({
    String? dataDir,
    String? cacheDir,
    int? port,
  }) async {
    final res = await _dio.post('/api/setup/db/commit', data: {
      'mode': 'embedded',
      'embedded': {
        'dataDir': ?dataDir,
        'cacheDir': ?cacheDir,
        'port': ?port,
      },
    });
    _ensureOk(res, 'db/commit');
  }

  Future<void> commitDBExternal({
    required String host,
    required int port,
    required String user,
    required String password,
    required String name,
    String sslmode = 'disable',
  }) async {
    final res = await _dio.post('/api/setup/db/commit', data: {
      'mode': 'external',
      'external': {
        'host': host,
        'port': port,
        'user': user,
        'password': password,
        'name': name,
        'sslmode': sslmode,
      },
    });
    _ensureOk(res, 'db/commit');
  }

  Future<void> setAdmin({required String username, required String password}) async {
    final res = await _dio.post('/api/setup/admin', data: {
      'username': username,
      'password': password,
    });
    _ensureOk(res, 'admin');
  }

  /// Auto-generate a JWT secret on the server. Caller can also pass a
  /// custom value via [customValue] for the advanced path.
  Future<void> setJWT({String? customValue}) async {
    final body = customValue == null
        ? {'mode': 'auto'}
        : {'mode': 'custom', 'value': customValue};
    final res = await _dio.post('/api/setup/jwt', data: body);
    _ensureOk(res, 'jwt');
  }

  Future<void> finalize() async {
    final res = await _dio.post('/api/setup/finalize');
    _ensureOk(res, 'finalize');
  }

  void _ensureOk(Response res, String step) {
    if (res.statusCode == 200) return;
    final msg = (res.data is Map && (res.data as Map)['error'] is String)
        ? (res.data as Map)['error'] as String
        : 'HTTP ${res.statusCode}';
    throw SetupApiException(res.statusCode ?? 0, '$step: $msg');
  }
}

class SetupApiException implements Exception {
  final int statusCode;
  final String message;
  SetupApiException(this.statusCode, this.message);
  @override
  String toString() => message;
}

/// Mirrors kernel/setup.Status.
class SetupStatus {
  final bool needsSetup;
  final String step; // "welcome" | "db" | "admin" | "cli" | "finalize" | "completed"
  final String dbMode; // "" | "embedded" | "external"
  final bool dbTested;
  final bool adminConfigured;
  final int schemaVersion;

  SetupStatus({
    required this.needsSetup,
    required this.step,
    required this.dbMode,
    required this.dbTested,
    required this.adminConfigured,
    required this.schemaVersion,
  });

  factory SetupStatus.fromJson(Map<String, dynamic> j) => SetupStatus(
        needsSetup: j['needsSetup'] == true,
        step: (j['step'] as String?) ?? 'welcome',
        dbMode: (j['dbMode'] as String?) ?? '',
        dbTested: j['dbTested'] == true,
        adminConfigured: j['adminConfigured'] == true,
        schemaVersion: (j['schemaVersion'] as num?)?.toInt() ?? 1,
      );
}

/// Mirrors GET /api/setup/env.
class EnvProbe {
  final String os;
  final String arch;
  final Tool node;
  final Tool npm;
  final Map<String, Tool> clis; // "claude" / "codex" / "gemini"

  EnvProbe({
    required this.os,
    required this.arch,
    required this.node,
    required this.npm,
    required this.clis,
  });

  factory EnvProbe.fromJson(Map<String, dynamic> j) {
    final rawClis = (j['clis'] as Map?) ?? {};
    return EnvProbe(
      os: (j['os'] as String?) ?? '',
      arch: (j['arch'] as String?) ?? '',
      node: Tool.fromJson(Map<String, dynamic>.from((j['node'] as Map?) ?? {})),
      npm: Tool.fromJson(Map<String, dynamic>.from((j['npm'] as Map?) ?? {})),
      clis: {
        for (final e in rawClis.entries)
          e.key as String: Tool.fromJson(Map<String, dynamic>.from(e.value as Map? ?? {})),
      },
    );
  }
}

class Tool {
  final bool installed;
  final String version;
  final String path;
  Tool({required this.installed, required this.version, required this.path});

  factory Tool.fromJson(Map<String, dynamic> j) => Tool(
        installed: j['installed'] == true,
        version: (j['version'] as String?) ?? '',
        path: (j['path'] as String?) ?? '',
      );
}
