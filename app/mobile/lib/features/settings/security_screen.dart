import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/auth/biometric_lock.dart';
import 'package:opendray/core/auth/cf_access_controller.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/auth/cf_access_login_screen.dart';

// Security settings: the app lock, and this device's Cloudflare
// Access session.
//
// Both belong together because they answer the same question -- who
// is allowed to drive this gateway from a phone. Access decides
// whether the device reaches the tunnel; the app lock decides whether
// whoever is holding the phone reaches the app.
class SecurityScreen extends ConsumerStatefulWidget {
  const SecurityScreen({super.key});

  @override
  ConsumerState<SecurityScreen> createState() => _SecurityScreenState();
}

class _SecurityScreenState extends ConsumerState<SecurityScreen> {
  String? _lockError;

  Future<void> _setLock({required bool enabled}) async {
    setState(() => _lockError = null);
    final auth = ref.read(localAuthProvider);
    final controller = ref.read(biometricLockControllerProvider.notifier);

    if (!enabled) {
      await controller.setEnabled(enabled: false);
      return;
    }

    // Turning the lock on without checking would let the operator
    // enable a gate the device cannot enforce, and they would only
    // find out by never being asked for anything.
    if (!await canLockDevice(auth)) {
      if (!mounted) return;
      setState(() => _lockError = t.settings.security.appLockUnavailable);
      return;
    }
    // Prove it works before persisting, so a device that reports
    // support but fails in practice cannot lock the operator into a
    // setting they can no longer reach.
    if (!await promptUnlock(auth, t.lock.reason)) {
      if (!mounted) return;
      setState(() => _lockError = t.settings.security.appLockFailed);
      return;
    }
    await controller.setEnabled(enabled: true);
  }

  Future<void> _signInToAccess(String baseUrl) async {
    final cookie = await CfAccessLoginScreen.show(context, baseUrl);
    if (cookie == null || !mounted) return;
    await ref.read(cfAccessControllerProvider.notifier).save(cookie);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final lockEnabled = ref.watch(biometricLockEnabledProvider);
    final access = ref.watch(cfAccessControllerProvider);
    final auth = ref.watch(authControllerProvider);
    final baseUrl = switch (auth) {
      AuthLoggedOut(serverUrl: final s) => s,
      AuthLoggedIn(serverUrl: final s) => s,
      _ => '',
    };

    return Scaffold(
      appBar: AppBar(title: Text(t.settings.security.section)),
      body: SafeArea(
        bottom: false,
        child: ListView(
          padding: const EdgeInsets.symmetric(vertical: 8),
          children: [
            SwitchListTile(
              secondary: const Icon(Icons.fingerprint),
              title: Text(t.settings.security.appLock),
              subtitle: Text(
                t.settings.security.appLockSubtitle,
                style: theme.textTheme.bodySmall,
              ),
              value: lockEnabled,
              onChanged: (v) => _setLock(enabled: v),
            ),
            if (_lockError != null)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                child: Text(
                  _lockError!,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.error),
                ),
              ),
            const Divider(height: 24),
            _SectionHeader(t.access.section),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
              child: Text(
                t.access.sectionSubtitle,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.outline),
              ),
            ),
            _AccessStatusTile(state: access),
            if (baseUrl.isNotEmpty)
              ListTile(
                leading: const Icon(Icons.shield_outlined),
                title: Text(t.access.signIn),
                onTap: () => _signInToAccess(baseUrl),
              ),
            if (access is CfAccessReady)
              ListTile(
                // Not a logout icon: this forgets the cookie on this
                // device, it does not end the session at Cloudflare.
                // The identity provider may still sign you straight
                // back in without asking for anything.
                leading: const Icon(Icons.delete_outline),
                title: Text(t.access.clear),
                subtitle: Text(
                  t.access.clearSubtitle,
                  style: theme.textTheme.bodySmall,
                ),
                onTap: () => ref
                    .read(cfAccessControllerProvider.notifier)
                    .clear(baseUrl: baseUrl),
              ),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}

class _AccessStatusTile extends StatelessWidget {
  const _AccessStatusTile({required this.state});

  final CfAccessState state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final (icon, title, subtitle) = switch (state) {
      CfAccessReady(session: final s) => (
          Icons.verified_user_outlined,
          t.access.sessionActive,
          s.expiresAt == null
              ? t.access.sessionExpiryUnknown
              : t.access.sessionExpires(
                  when: s.expiresAt!.toLocal().toString().split('.').first,
                ),
        ),
      CfAccessNeedsLogin() => (
          Icons.gpp_maybe_outlined,
          t.access.sessionNone,
          t.access.signIn,
        ),
      _ => (Icons.shield_outlined, t.access.sessionNone, ''),
    };
    return ListTile(
      leading: Icon(icon),
      title: Text(title),
      subtitle: subtitle.isEmpty
          ? null
          : Text(subtitle, style: theme.textTheme.bodySmall),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader(this.label);
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 6),
      child: Text(
        label.toUpperCase(),
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.8,
          fontSize: 11,
        ),
      ),
    );
  }
}
