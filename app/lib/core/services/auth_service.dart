import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/api_client.dart';

/// Authentication lifecycle: probes the server for whether auth is required,
/// stores the JWT across launches, and surfaces three states the router
/// cares about:
///   • [AuthState.unknown]     — first frame, before probe completes
///   • [AuthState.disabled]    — server has no JWT_SECRET set
///   • [AuthState.unauthed]    — auth required, no token / bad token
///   • [AuthState.authed]      — token present and accepted
enum AuthState { unknown, disabled, unauthed, authed }

class AuthService extends ChangeNotifier {
  static const _keyToken = 'opendray.auth.token';

  String? _token;
  AuthState _state = AuthState.unknown;
  String? _lastServerUrl;

  String? get token => _token;
  AuthState get state => _state;

  /// True iff a token is persisted locally, regardless of whether the
  /// server has been reached yet. Used by the Settings page to decide
  /// whether to show "Sign out" even in the offline / unknown state —
  /// otherwise the user would have no way back to the login page when
  /// the server is temporarily unreachable.
  bool get hasStoredToken => _token != null && _token!.isNotEmpty;

  /// Probes `/api/auth/status` (public) to decide whether the server is in
  /// auth-required mode. Loads any stored token from disk and, if present,
  /// verifies it with a cheap authenticated call. Call whenever the server
  /// URL changes or the app starts up.
  Future<void> probe(String serverUrl, {Map<String, String> extraHeaders = const {}}) async {
    _lastServerUrl = serverUrl;
    final prefs = await SharedPreferences.getInstance();
    _token = prefs.getString(_keyToken);

    final probeDio = Dio(BaseOptions(
      baseUrl: serverUrl,
      connectTimeout: const Duration(seconds: 6),
      receiveTimeout: const Duration(seconds: 6),
      headers: extraHeaders,
      // Accept 4xx so we can inspect status without try/catch.
      validateStatus: (s) => s != null && s < 500,
    ));

    bool authRequired;
    try {
      final res = await probeDio.get('/api/auth/status');
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

  Future<void> _storeToken(String t) async {
    _token = t;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_keyToken, t);
  }

  Future<void> _clearStoredToken() async {
    _token = null;
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keyToken);
  }
}
