import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/auth/biometric_lock.dart';

// The rule these lock in: relock on a real absence, never on the
// incidental foreground losses that the Access sign-in flow itself
// causes. Cloudflare SSO routinely bounces the operator out to a mail
// app for a one-time code; relocking on the way back would make the
// two features fight each other.

void main() {
  group('shouldRelockOnResume', () {
    test('a long absence relocks', () {
      final paused = DateTime(2026, 9, 2, 10);
      expect(
        shouldRelockOnResume(
          enabled: true,
          pausedAt: paused,
          resumedAt: paused.add(const Duration(minutes: 5)),
        ),
        isTrue,
      );
    });

    test('a glance at a notification does not relock', () {
      final paused = DateTime(2026, 9, 2, 10);
      expect(
        shouldRelockOnResume(
          enabled: true,
          pausedAt: paused,
          resumedAt: paused.add(const Duration(seconds: 5)),
        ),
        isFalse,
      );
    });

    // Fetching an Access one-time code from a mail app is a normal
    // part of signing in, and it costs well over the resume-refresh
    // threshold used elsewhere in the app.
    test('a trip to a mail app for an OTP does not relock', () {
      final paused = DateTime(2026, 9, 2, 10);
      expect(
        shouldRelockOnResume(
          enabled: true,
          pausedAt: paused,
          resumedAt: paused.add(const Duration(seconds: 45)),
        ),
        isFalse,
      );
    });

    test('never relocks when the lock is off', () {
      final paused = DateTime(2026, 9, 2, 10);
      expect(
        shouldRelockOnResume(
          enabled: false,
          pausedAt: paused,
          resumedAt: paused.add(const Duration(hours: 3)),
        ),
        isFalse,
      );
    });

    test('a resume with no recorded pause does not relock', () {
      expect(
        shouldRelockOnResume(
          enabled: true,
          pausedAt: null,
          resumedAt: DateTime(2026, 9, 2, 10),
        ),
        isFalse,
      );
    });
  });

  group('biometricLockEnabledProvider', () {
    // The gate distinguishes "not read yet" (null) from "off"
    // (false); the switch in settings only needs a bool. Flattening
    // in one place keeps the two readings from drifting.
    test('flattens the not-yet-loaded state to off', () {
      final container = ProviderContainer(
        overrides: [
          biometricLockControllerProvider.overrideWith(
            (ref) => _StubLockController(value: null),
          ),
        ],
      );
      addTearDown(container.dispose);
      expect(container.read(biometricLockEnabledProvider), isFalse);
    });

    test('passes a resolved flag through', () {
      final container = ProviderContainer(
        overrides: [
          biometricLockControllerProvider.overrideWith(
            (ref) => _StubLockController(value: true),
          ),
        ],
      );
      addTearDown(container.dispose);
      expect(container.read(biometricLockEnabledProvider), isTrue);
    });
  });
}

/// Skips the secure-storage read so the test can pin an exact state,
/// including the null the real controller only holds momentarily.
class _StubLockController extends BiometricLockController {
  _StubLockController({required bool? value})
      : super(const _NullStorage()) {
    state = value;
  }
}

class _NullStorage implements FlutterSecureStorage {
  const _NullStorage();

  @override
  dynamic noSuchMethod(Invocation invocation) => Future<String?>.value();
}
