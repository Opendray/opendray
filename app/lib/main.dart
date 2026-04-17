import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
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

  runApp(NtcApp(serverConfig: serverConfig, l10n: l10n));
}
