import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/routing/app_router.dart';
import 'package:opendray/core/theme/app_theme.dart';

void main() {
  runApp(const ProviderScope(child: OpendrayApp()));
}

class OpendrayApp extends ConsumerWidget {
  const OpendrayApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    return MaterialApp.router(
      title: 'opendray',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.dark(),
      darkTheme: AppTheme.dark(),
      themeMode: ThemeMode.dark,
      routerConfig: router,
    );
  }
}
