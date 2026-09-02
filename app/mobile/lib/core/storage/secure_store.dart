import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

// Single FlutterSecureStorage instance for the whole app. Persists
// the gateway URL, the bearer token, the Cloudflare Access cookie
// and the biometric-lock flag.
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

  // Cloudflare Access cookie lifted out of the SSO WebView. Stored
  // next to the bearer token because it is exactly as sensitive:
  // together they are what lets this device reach the gateway from
  // the public internet.
  static const cfAccessCookie = 'opendray.cf_access_cookie';

  // "1" once the operator turns on the biometric app lock.
  static const biometricLock = 'opendray.biometric_lock';
}
