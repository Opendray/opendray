import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/routing/app_router.dart';
import 'package:opendray/core/theme/app_theme.dart';
import 'package:opendray/core/theme/theme_controller.dart';

void main() {
  runApp(const ProviderScope(child: OpendrayApp()));
}

class OpendrayApp extends ConsumerWidget {
  const OpendrayApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    // Watch the controller so a picker change in Settings rebuilds
    // MaterialApp with the new themeMode immediately — no app
    // restart needed.
    final themeMode = ref.watch(themeControllerProvider);
    return MaterialApp.router(
      title: 'opendray',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      themeMode: themeMode,
      routerConfig: router,
    );
  }
}
