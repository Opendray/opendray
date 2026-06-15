import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'package:opendray/core/storage/secure_store.dart';

// Auth lifecycle. We deliberately keep this small: the app only
// cares about (a) is there a server URL, (b) is there a valid
// token, (c) when does the token expire. Everything richer (per-
// device tokens, biometric unlock state) is layered above this.
sealed class AuthState {
  const AuthState();
}

class AuthBootstrapping extends AuthState {
  const AuthBootstrapping();
}

class AuthOnboarding extends AuthState {
  const AuthOnboarding();
}

class AuthLoggedOut extends AuthState {
  const AuthLoggedOut(this.serverUrl);
  final String serverUrl;
}

class AuthLoggedIn extends AuthState {
  const AuthLoggedIn({
    required this.serverUrl,
    required this.token,
    required this.username,
    required this.expiresAt,
  });
  final String serverUrl;
  final String token;
  final String username;
  final DateTime expiresAt;
}

class AuthController extends StateNotifier<AuthState> {
  AuthController(this._storage) : super(const AuthBootstrapping()) {
    _bootstrap();
  }

  final FlutterSecureStorage _storage;

  Future<void> _bootstrap() async {
    try {
      final serverUrl = await _storage.read(key: SecureKeys.serverUrl);
      if (serverUrl == null || serverUrl.isEmpty) {
        state = const AuthOnboarding();
        return;
      }
      final token = await _storage.read(key: SecureKeys.token);
      final username = await _storage.read(key: SecureKeys.username);
      final expiresAtRaw = await _storage.read(key: SecureKeys.tokenExpiresAt);
      if (token == null ||
          token.isEmpty ||
          username == null ||
          expiresAtRaw == null) {
        state = AuthLoggedOut(serverUrl);
        return;
      }
      final expiresAt = DateTime.tryParse(expiresAtRaw);
      if (expiresAt == null || expiresAt.isBefore(DateTime.now())) {
        await _clearTokenOnly();
        state = AuthLoggedOut(serverUrl);
        return;
      }
      state = AuthLoggedIn(
        serverUrl: serverUrl,
        token: token,
        username: username,
        expiresAt: expiresAt,
      );
    } on Object catch (_) {
      // Secure storage can fail to decrypt on Android when the
      // Keystore key and EncryptedSharedPreferences data fall out
      // of sync (app reinstall, OS upgrade, or Auto Backup
      // restoring prefs without the key). If the exception escapes,
      // _bootstrap's future fails silently and the app is stuck on
      // the splash forever. Wipe the corrupted store and fall back
      // to onboarding so the user can always recover.
      try {
        await _storage.deleteAll();
      } on Object catch (_) {
        // Nothing more we can do; still drop to a usable screen.
      }
      state = const AuthOnboarding();
    }
  }

  Future<void> setServerUrl(String url) async {
    await _storage.write(key: SecureKeys.serverUrl, value: url);
    state = AuthLoggedOut(url);
  }

  Future<void> setLoggedIn({
    required String token,
    required String username,
    required DateTime expiresAt,
  }) async {
    final current = state;
    final serverUrl = switch (current) {
      AuthLoggedOut(serverUrl: final s) => s,
      AuthLoggedIn(serverUrl: final s) => s,
      _ => null,
    };
    if (serverUrl == null) return;
    await _storage.write(key: SecureKeys.token, value: token);
    await _storage.write(key: SecureKeys.username, value: username);
    await _storage.write(
      key: SecureKeys.tokenExpiresAt,
      value: expiresAt.toIso8601String(),
    );
    state = AuthLoggedIn(
      serverUrl: serverUrl,
      token: token,
      username: username,
      expiresAt: expiresAt,
    );
  }

  Future<void> logout() async {
    final current = state;
    final serverUrl = switch (current) {
      AuthLoggedOut(serverUrl: final s) => s,
      AuthLoggedIn(serverUrl: final s) => s,
      _ => null,
    };
    await _clearTokenOnly();
    state = serverUrl == null
        ? const AuthOnboarding()
        : AuthLoggedOut(serverUrl);
  }

  Future<void> resetServer() async {
    await _storage.deleteAll();
    state = const AuthOnboarding();
  }

  Future<void> _clearTokenOnly() async {
    await _storage.delete(key: SecureKeys.token);
    await _storage.delete(key: SecureKeys.username);
    await _storage.delete(key: SecureKeys.tokenExpiresAt);
  }
}

final authControllerProvider =
    StateNotifierProvider<AuthController, AuthState>((ref) {
  return AuthController(ref.watch(secureStorageProvider));
});
