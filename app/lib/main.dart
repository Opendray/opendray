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
  if (serverConfig.isConfigured) {
    // Fire-and-forget: if the probe finishes before the first frame the
    // app opens straight to the right page; if it takes longer the router
    // stays on '/' until probe() notifies and redirect() re-runs.
    unawaited(authService.probe(
      serverConfig.effectiveUrl,
      extraHeaders: serverConfig.cfAccessHeaders,
    ));
  }
  // Re-probe whenever the user changes the server URL (setup wizard) so
  // the login prompt tracks the actual server's auth state.
  serverConfig.addListener(() {
    if (serverConfig.isConfigured) {
      unawaited(authService.probe(
        serverConfig.effectiveUrl,
        extraHeaders: serverConfig.cfAccessHeaders,
      ));
    }
  });

  runApp(OpendrayApp(
    serverConfig: serverConfig,
    l10n: l10n,
    authService: authService,
  ));
}
