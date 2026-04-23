import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Per-profile password vault backed by the OS keychain (iOS Keychain,
/// Android Keystore/EncryptedSharedPreferences, macOS Keychain, Linux
/// libsecret, Windows DPAPI). On **web**, `flutter_secure_storage` falls
/// back to IndexedDB with no real encryption, so we deliberately refuse
/// to persist there — web users always retype their password. Callers
/// can check [isSupported] to disable the "remember password" checkbox
/// on those platforms up front.
///
/// Keys are namespaced `opendray.serverpwd.<profileId>`. The profile
/// id is opaque and stable across renames (see ServerProfile.newId),
/// so editing the alias or URL never orphans a stored password.
class SecureCredentialStore {
  static const _prefix = 'opendray.serverpwd.';

  final FlutterSecureStorage _storage;

  SecureCredentialStore()
      : _storage = const FlutterSecureStorage(
          aOptions: AndroidOptions(encryptedSharedPreferences: true),
          iOptions: IOSOptions(
            accessibility: KeychainAccessibility.unlocked,
          ),
        );

  /// True on platforms where the backing store is actually encrypted.
  /// On web we return false so the UI can grey out "remember password"
  /// and stop users from shipping a plaintext IndexedDB blob they
  /// didn't realize wasn't safe.
  bool get isSupported => !kIsWeb;

  String _keyFor(String profileId) => '$_prefix$profileId';

  Future<void> savePassword(String profileId, String password) async {
    if (!isSupported) return;
    if (profileId.isEmpty) return;
    try {
      await _storage.write(key: _keyFor(profileId), value: password);
    } catch (_) {
      // Secure storage can throw on locked keystores (first boot after
      // factory reset, corrupted keystore on Android 6 downgrade, etc).
      // Better to silently fail than to crash the login flow — the user
      // just retypes this once.
    }
  }

  Future<String?> readPassword(String profileId) async {
    if (!isSupported) return null;
    if (profileId.isEmpty) return null;
    try {
      return await _storage.read(key: _keyFor(profileId));
    } catch (_) {
      return null;
    }
  }

  Future<void> deletePassword(String profileId) async {
    if (!isSupported) return;
    if (profileId.isEmpty) return;
    try {
      await _storage.delete(key: _keyFor(profileId));
    } catch (_) {}
  }
}
