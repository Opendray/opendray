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
  // a server that can't verify it. We persist a JSON blob under one key:
  //     {"url": "<effectiveUrl>", "token": "<jwt>"}
  static const _keyAuth = 'opendray.auth.v2';

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

  /// Reads the persisted token ONLY if it was issued by [currentUrl].
  /// Legacy installs (before v2) had a bare string under the old key —
  /// those are dropped silently: the user re-logs once.
  Future<void> _loadStoredFor(String currentUrl) async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_keyAuth);
    if (raw == null || raw.isEmpty) {
      _token = null;
      return;
    }
    try {
      final data = jsonDecode(raw) as Map<String, dynamic>;
      final storedUrl = (data['url'] as String?) ?? '';
      final storedToken = (data['token'] as String?) ?? '';
      if (storedUrl == currentUrl && storedToken.isNotEmpty) {
        _token = storedToken;
      } else {
        // URL mismatch — either user pointed at a different server, or
        // their old token is from before we scoped by URL. Either way
        // we don't trust it here; keep the blob on disk so switching
        // back restores it, but don't apply it to THIS server.
        _token = null;
      }
    } catch (_) {
      _token = null;
    }
  }

  Future<void> _storeToken(String t) async {
    _token = t;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_keyAuth, jsonEncode({
      'url': _lastServerUrl ?? '',
      'token': t,
    }));
  }

  Future<void> _clearStoredToken() async {
    _token = null;
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keyAuth);
  }
}
