import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:local_auth/local_auth.dart';

import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/storage/secure_store.dart';

// Optional app lock in front of everything.
//
// This exists because of what publishing the gateway costs: once the
// app works from the public internet, an unlocked phone is a live
// admin console. The bearer token and the Access cookie both sit in
// the platform keystore, so anyone holding the device is already
// past both gates. The device lock is the only thing left, and it is
// worth being able to demand it again at the app boundary.

/// How long the app may be out of the foreground before it relocks.
///
/// Deliberately much longer than the resume-refresh threshold in
/// core/lifecycle/resume_refresh.dart: the Cloudflare Access sign-in itself sends
/// the operator to a mail app for a one-time code, and relocking on
/// the way back would turn the two features into a loop.
const kBiometricRelockThreshold = Duration(seconds: 60);

/// Whether an absence that began at [pausedAt] and ended at
/// [resumedAt] should put the lock screen back up.
bool shouldRelockOnResume({
  required bool enabled,
  required DateTime? pausedAt,
  required DateTime resumedAt,
}) {
  if (!enabled) return false;
  if (pausedAt == null) return false;
  return resumedAt.difference(pausedAt) >= kBiometricRelockThreshold;
}

/// Persisted on/off flag for the lock.
///
/// The state is nullable on purpose: null means "not read yet". The
/// gate has to tell that apart from "off", because the transition it
/// locks on is the one out of null. Treating the initial false as a
/// real "off" would make turning the lock on in settings look
/// identical to app start, and prompt for a second unlock right after
/// the one that enabled it.
class BiometricLockController extends StateNotifier<bool?> {
  BiometricLockController(this._storage) : super(null) {
    _bootstrap();
  }

  final FlutterSecureStorage _storage;

  Future<void> _bootstrap() async {
    try {
      state = await _storage.read(key: SecureKeys.biometricLock) == '1';
    } on Object catch (_) {
      // A keystore we cannot read means we cannot prove the operator
      // asked for the lock. Defaulting to off keeps them out of an
      // app they can never unlock.
      state = false;
    }
  }

  Future<void> setEnabled({required bool enabled}) async {
    if (enabled) {
      await _storage.write(key: SecureKeys.biometricLock, value: '1');
    } else {
      await _storage.delete(key: SecureKeys.biometricLock);
    }
    state = enabled;
  }
}

final biometricLockControllerProvider =
    StateNotifierProvider<BiometricLockController, bool?>((ref) {
  return BiometricLockController(ref.watch(secureStorageProvider));
});

/// The flag with "still loading" flattened to "off", for callers that
/// only need to render a switch.
final biometricLockEnabledProvider = Provider<bool>((ref) {
  return ref.watch(biometricLockControllerProvider) ?? false;
});

final localAuthProvider = Provider<LocalAuthentication>((ref) {
  return LocalAuthentication();
});

/// Whether this device can actually enforce the lock. A device with
/// no screen lock enrolled would let the operator switch the setting
/// on and then never be asked for anything.
Future<bool> canLockDevice(LocalAuthentication auth) async {
  try {
    return await auth.isDeviceSupported();
  } on Object catch (_) {
    return false;
  }
}

/// Prompts for biometrics or the device passcode.
///
/// `biometricOnly: false` on purpose: a passcode fallback is what
/// keeps a failed fingerprint from locking the operator out of their
/// own gateway.
Future<bool> promptUnlock(LocalAuthentication auth, String reason) async {
  try {
    return await auth.authenticate(
      localizedReason: reason,
      options: const AuthenticationOptions(
        stickyAuth: true,
        biometricOnly: false,
      ),
    );
  } on Object catch (_) {
    return false;
  }
}

/// Puts the lock screen in front of [child] until the operator
/// authenticates, and puts it back after a real absence.
class BiometricGate extends ConsumerStatefulWidget {
  const BiometricGate({required this.child, super.key});

  final Widget child;

  @override
  ConsumerState<BiometricGate> createState() => _BiometricGateState();
}

class _BiometricGateState extends ConsumerState<BiometricGate>
    with WidgetsBindingObserver {
  bool _locked = false;
  bool _prompting = false;
  DateTime? _pausedAt;

  // True once the stored flag has been read. Until then we do not
  // know whether to lock, and locking optimistically would flash a
  // lock screen at operators who never turned it on.
  bool _initialized = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    final enabled = ref.read(biometricLockEnabledProvider);
    switch (state) {
      case AppLifecycleState.paused:
      case AppLifecycleState.detached:
      case AppLifecycleState.hidden:
        _pausedAt ??= DateTime.now();
      case AppLifecycleState.resumed:
        final pausedAt = _pausedAt;
        _pausedAt = null;
        if (shouldRelockOnResume(
          enabled: enabled,
          pausedAt: pausedAt,
          resumedAt: DateTime.now(),
        )) {
          setState(() => _locked = true);
          unawaited(_unlock());
        }
      case AppLifecycleState.inactive:
        break;
    }
  }

  Future<void> _unlock() async {
    if (_prompting) return;
    _prompting = true;
    try {
      final ok = await promptUnlock(
        ref.read(localAuthProvider),
        t.lock.reason,
      );
      if (!mounted) return;
      if (ok) setState(() => _locked = false);
    } finally {
      _prompting = false;
    }
  }

  @override
  Widget build(BuildContext context) {
    final stored = ref.watch(biometricLockControllerProvider);
    final enabled = stored ?? false;

    // Runs exactly once, on the frame the stored flag first resolves.
    // Later flips of the flag are the operator using the settings
    // switch, and those must not lock: they have just authenticated
    // to turn it on.
    if (!_initialized && stored != null) {
      _initialized = true;
      if (enabled) {
        _locked = true;
        WidgetsBinding.instance.addPostFrameCallback((_) => _unlock());
      }
    }

    return Stack(
      children: [
        widget.child,
        if (_locked && enabled)
          Positioned.fill(child: _LockScreen(onUnlock: _unlock)),
      ],
    );
  }
}

class _LockScreen extends StatelessWidget {
  const _LockScreen({required this.onUnlock});

  final Future<void> Function() onUnlock;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // Opaque, not translucent: the point is that a session transcript
    // is not readable over the lock screen's shoulder.
    return Material(
      color: theme.colorScheme.surface,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.lock_outline,
              size: 48,
              color: theme.colorScheme.outline,
            ),
            const SizedBox(height: 16),
            Text(t.lock.title, style: theme.textTheme.titleMedium),
            const SizedBox(height: 24),
            FilledButton.icon(
              onPressed: onUnlock,
              icon: const Icon(Icons.fingerprint),
              label: Text(t.lock.unlock),
            ),
          ],
        ),
      ),
    );
  }
}
