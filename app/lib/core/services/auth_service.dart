import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/api_client.dart';

/// Authentication lifecycle: probes the server for whether auth is required,
/// stores the JWT across launches, and surfaces the states the router cares
/// about:
///   • [AuthState.unknown]       — first frame, before probe completes
///   • [AuthState.setupRequired] — server is in first-run setup mode
///   • [AuthState.disabled]      — server has no JWT_SECRET set
///   • [AuthState.unauthed]      — auth required, no token / bad token
///   • [AuthState.authed]        — token present and accepted
enum AuthState { unknown, setupRequired, disabled, unauthed, authed }

class AuthService extends ChangeNotifier {
  // Tokens are scoped by server URL so pointing the app at a different
  // backend (e.g. dev LXC vs. production) doesn't send a foreign JWT to
  // a server that can't verify it.
  //
  // v3 store: a map of URL → token, so users with multiple OpenDray
  // deployments can hop between them without re-login each time.
  //     {"tokens": {"http://10.0.0.1:8640": "jwtA", ...}}
  //
  // v2 (single {url, token}) is read once on upgrade and transparently
  // migrated; v1 (bare string) is dropped (user re-logs once).
  static const _keyAuth = 'opendray.auth.v3';
  static const _keyAuthV2 = 'opendray.auth.v2'; // legacy, migrated on first read

  String? _token;
  AuthState _state = AuthState.unknown;
  String? _lastServerUrl;

  String? get token => _token;
  AuthState get state => _state;

  /// True iff a token is persisted locally AND it was issued by the URL
  /// we are currently pointed at. Used by the Settings page to decide
  /// whether to show "Sign out" even in the offline / unknown state.
  bool get hasStoredToken => _token != null && _token!.isNotEmpty;

  /// Probes `/api/auth/status` (public) to decide whether the server is in
  /// auth-required mode. Loads any stored token from disk and, if present,
  /// verifies it with a cheap authenticated call. Call whenever the server
  /// URL changes or the app starts up.
  Future<void> probe(String serverUrl, {Map<String, String> extraHeaders = const {}}) async {
    _lastServerUrl = serverUrl;
    await _loadStoredFor(serverUrl);

    final probeDio = Dio(BaseOptions(
      baseUrl: serverUrl,
      connectTimeout: const Duration(seconds: 6),
      receiveTimeout: const Duration(seconds: 6),
      headers: extraHeaders,
      // Accept 4xx AND 5xx so we can distinguish "setup mode" (503 from
      // the catch-all in setup mode) from transport errors.
      validateStatus: (s) => s != null && s < 600,
    ));

    // First, ask /api/setup/status — it's public and answered in both setup
    // and normal mode. A "needsSetup: true" answer short-circuits the
    // regular auth probe and routes the user into the wizard.
    try {
      final s = await probeDio.get('/api/setup/status');
      if (s.statusCode == 200 && s.data is Map && s.data['needsSetup'] == true) {
        _state = AuthState.setupRequired;
        notifyListeners();
        return;
      }
    } catch (_) {
      // Fall through to the auth-status probe below — older servers don't
      // have /api/setup/status at all.
    }

    bool authRequired;
    try {
      final res = await probeDio.get('/api/auth/status');
      if (res.statusCode == 503) {
        // Server is still in setup mode but /api/setup/status gave us a
        // transient error above. Treat it as setupRequired rather than
        // trapping the user in "unknown".
        _state = AuthState.setupRequired;
        notifyListeners();
        return;
      }
      authRequired = (res.data is Map && res.data['authRequired'] == true);
    } catch (_) {
      // Server unreachable. If we already hold a token from a previous
      // session, stay optimistically authed so the router doesn't strand
      // the user on a blank "unknown" screen — any real 401 will flip us
      // to unauthed via the ApiClient's interceptor once the server is
      // reachable again.
      if (_token != null && _token!.isNotEmpty) {
        _state = AuthState.authed;
      } else {
        _state = AuthState.unknown;
      }
      notifyListeners();
      return;
    }

    if (!authRequired) {
      _state = AuthState.disabled;
      notifyListeners();
      return;
    }

    // Auth required — verify any stored token by hitting a cheap protected
    // endpoint. /api/sessions is always mounted and returns JSON quickly.
    if (_token == null || _token!.isEmpty) {
      _state = AuthState.unauthed;
      notifyListeners();
      return;
    }
    try {
      final res = await probeDio.get('/api/sessions',
          options: Options(headers: {'Authorization': 'Bearer $_token'}));
      if (res.statusCode == 200) {
        _state = AuthState.authed;
      } else if (res.statusCode == 401) {
        // Token no longer valid — wipe and force re-login.
        await _clearStoredToken();
        _state = AuthState.unauthed;
      } else {
        // 4xx/5xx other than 401 usually means a transient server issue,
        // not a bad token. Keep the token and stay authed so the user
        // isn't booted out of the app on every backend hiccup.
        _state = AuthState.authed;
      }
    } catch (_) {
      // Network error mid-verify — same optimistic stance as above.
      _state = AuthState.authed;
    }
    notifyListeners();
  }

  /// POST /api/auth/login. On success stores the token and moves to authed.
  /// Returns a human-readable error message on failure; null on success.
  Future<String?> login({
    required String serverUrl,
    required String username,
    required String password,
    Map<String, String> extraHeaders = const {},
  }) async {
    final dio = Dio(BaseOptions(
      baseUrl: serverUrl,
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 10),
      headers: extraHeaders,
      validateStatus: (s) => s != null && s < 500,
    ));
    try {
      final res = await dio.post('/api/auth/login', data: {
        'username': username,
        'password': password,
      });
      if (res.statusCode == 200 && res.data is Map && res.data['token'] is String) {
        final t = res.data['token'] as String;
        if (t == 'no-auth-configured') {
          // Shouldn't happen on the login path if probe said auth is required,
          // but handle it defensively.
          _state = AuthState.disabled;
          notifyListeners();
          return null;
        }
        await _storeToken(t);
        _state = AuthState.authed;
        notifyListeners();
        return null;
      }
      if (res.data is Map && res.data['error'] is String) {
        return res.data['error'] as String;
      }
      return 'Login failed (HTTP ${res.statusCode})';
    } on DioException catch (e) {
      return apiExceptionFrom(e).toString();
    }
  }

  /// Swaps in a token issued after a successful credential change. The
  /// server returns it from `/api/auth/change-credentials` along with the
  /// new username.
  Future<void> acceptNewToken(String newToken) async {
    await _storeToken(newToken);
    _state = AuthState.authed;
    notifyListeners();
  }

  /// Clears the stored token and transitions to unauthed. Safe to call from
  /// an interceptor on 401 — the router redirect will kick in.
  Future<void> logout() async {
    await _clearStoredToken();
    if (_lastServerUrl != null) {
      // Re-probe so we correctly show "unauthed" vs "disabled" if server
      // config changed while logged in.
      _state = AuthState.unauthed;
    } else {
      _state = AuthState.unknown;
    }
    notifyListeners();
  }

  /// Reads the persisted token for [currentUrl] from the URL→token map.
  /// Migrates a v2 single-entry blob into the new map on first run.
  Future<void> _loadStoredFor(String currentUrl) async {
    final prefs = await SharedPreferences.getInstance();
    final tokens = await _loadTokenMap(prefs);
    final t = tokens[currentUrl];
    _token = (t != null && t.isNotEmpty) ? t : null;
  }

  Future<Map<String, String>> _loadTokenMap(SharedPreferences prefs) async {
    final raw = prefs.getString(_keyAuth);
    if (raw != null && raw.isNotEmpty) {
      try {
        final data = jsonDecode(raw) as Map<String, dynamic>;
        final tokens = data['tokens'];
        if (tokens is Map) {
          return {
            for (final e in tokens.entries)
              e.key.toString(): e.value.toString(),
          };
        }
      } catch (_) {}
    }
    // v2 migration — single {url, token}. Fold it into the map and
    // drop the old key so subsequent reads hit the fast path.
    final v2 = prefs.getString(_keyAuthV2);
    if (v2 != null && v2.isNotEmpty) {
      try {
        final data = jsonDecode(v2) as Map<String, dynamic>;
        final url = (data['url'] as String?) ?? '';
        final tok = (data['token'] as String?) ?? '';
        if (url.isNotEmpty && tok.isNotEmpty) {
          final migrated = {url: tok};
          await _saveTokenMap(prefs, migrated);
          await prefs.remove(_keyAuthV2);
          return migrated;
        }
      } catch (_) {}
      await prefs.remove(_keyAuthV2);
    }
    return {};
  }

  Future<void> _saveTokenMap(SharedPreferences prefs, Map<String, String> tokens) async {
    await prefs.setString(_keyAuth, jsonEncode({'tokens': tokens}));
  }

  Future<void> _storeToken(String t) async {
    _token = t;
    final url = _lastServerUrl ?? '';
    if (url.isEmpty) return;
    final prefs = await SharedPreferences.getInstance();
    final tokens = await _loadTokenMap(prefs);
    tokens[url] = t;
    await _saveTokenMap(prefs, tokens);
  }

  /// Drops the token for the CURRENT server URL only. Tokens for other
  /// servers in the map stay intact, so switching back auto-restores
  /// their session.
  Future<void> _clearStoredToken() async {
    _token = null;
    final url = _lastServerUrl ?? '';
    if (url.isEmpty) return;
    final prefs = await SharedPreferences.getInstance();
    final tokens = await _loadTokenMap(prefs);
    if (tokens.remove(url) != null) {
      if (tokens.isEmpty) {
        await prefs.remove(_keyAuth);
      } else {
        await _saveTokenMap(prefs, tokens);
      }
    }
  }
}
