import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

// Single FlutterSecureStorage instance for the whole app. Persists
// the gateway URL + bearer token (and, in F2, the biometric flag).
//
// iOS: maps to Keychain Services with `first_unlock_this_device`.
// Android: maps to EncryptedSharedPreferences (AES-256-GCM with a
// per-app key in the Android Keystore).
final secureStorageProvider = Provider<FlutterSecureStorage>((ref) {
  return const FlutterSecureStorage(
    iOptions: IOSOptions(
      accessibility: KeychainAccessibility.first_unlock_this_device,
    ),
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );
});

class SecureKeys {
  static const serverUrl = 'opendray.server_url';
  static const token = 'opendray.token';
  static const username = 'opendray.username';
  static const tokenExpiresAt = 'opendray.token_expires_at';
}
