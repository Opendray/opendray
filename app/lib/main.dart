import 'dart:async';

import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'core/services/auth_service.dart';
import 'core/services/l10n.dart';
import 'core/services/server_config.dart';
import 'app.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Pre-load terminal font to ensure correct character width measurement
  await GoogleFonts.pendingFonts([
    GoogleFonts.jetBrainsMono(),
  ]);

  final serverConfig = ServerConfig();
  await serverConfig.load();

  final l10n = await L10n.load();

  final authService = AuthService();
  // Wire the probe-time auto-login hook: when the active profile has
  // "remember password" enabled and a matching entry in the keychain,
  // AuthService posts /api/auth/login silently instead of bouncing
  // the user to the login screen.
  authService.setAutoLoginProvider((url) async {
    final p = serverConfig.activeProfile;
    if (p == null || p.url != url) return null;
    if (!p.rememberPassword || p.username.isEmpty) return null;
    final pwd = await serverConfig.credentialStore.readPassword(p.id);
    if (pwd == null || pwd.isEmpty) return null;
    return AutoLoginCreds(username: p.username, password: pwd);
  });
  if (serverConfig.isConfigured) {
    // Fire-and-forget: if the probe finishes before the first frame the
    // app opens straight to the right page; if it takes longer the router
    // stays on '/' until probe() notifies and redirect() re-runs.
    unawaited(authService.probe(serverConfig.effectiveUrl));
  }
  // Re-probe whenever the user changes the server URL (setup wizard) so
  // the login prompt tracks the actual server's auth state.
  serverConfig.addListener(() {
    if (serverConfig.isConfigured) {
      unawaited(authService.probe(serverConfig.effectiveUrl));
    }
  });

  runApp(NtcApp(
    serverConfig: serverConfig,
    l10n: l10n,
    authService: authService,
  ));
}
