import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/auth/cf_access_controller.dart';
import 'package:opendray/features/auth/cf_access_login_screen.dart';

// Surfaces the Cloudflare Access sign-in as soon as the edge bounces
// a request, from wherever in the app that happened.
//
// Sits above the router rather than inside it because an Access
// challenge is orthogonal to opendray's own auth: it can land on the
// login screen, on a session terminal, or mid-way through onboarding,
// and the recovery is the same everywhere.
//
// Drawn as an overlay on a Stack, not as a replacement for [child],
// so the screen underneath keeps its state -- an open session with a
// half-typed prompt must still be there afterwards.
class AccessGate extends ConsumerWidget {
  const AccessGate({required this.child, super.key});

  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final access = ref.watch(cfAccessControllerProvider);
    final auth = ref.watch(authControllerProvider);
    final baseUrl = switch (auth) {
      AuthLoggedOut(serverUrl: final s) => s,
      AuthLoggedIn(serverUrl: final s) => s,
      _ => '',
    };

    // No server URL yet means onboarding has not run; there is
    // nothing to authenticate against, and onboarding drives its own
    // Access flow inline.
    final showLogin = access is CfAccessNeedsLogin && baseUrl.isNotEmpty;

    return Stack(
      children: [
        child,
        if (showLogin)
          Positioned.fill(
            child: CfAccessLoginScreen(
              baseUrl: baseUrl,
              onResult: (cookie) {
                final notifier = ref.read(cfAccessControllerProvider.notifier);
                if (cookie == null) {
                  notifier.dismiss();
                } else {
                  notifier.save(cookie);
                }
              },
            ),
          ),
      ],
    );
  }
}
