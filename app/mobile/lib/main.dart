import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/locale/locale_controller.dart';
import 'package:opendray/core/routing/app_router.dart';
import 'package:opendray/core/theme/app_theme.dart';
import 'package:opendray/core/theme/theme_controller.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  LocaleSettings.useDeviceLocale();
  runApp(
    TranslationProvider(
      child: const ProviderScope(child: OpendrayApp()),
    ),
  );
}

class OpendrayApp extends ConsumerWidget {
  const OpendrayApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    final themeMode = ref.watch(themeControllerProvider);
    // Watching the locale controller rebuilds MaterialApp so the
    // Material/Cupertino delegates re-resolve and Flutter's own
    // built-in widgets (DatePicker, dialog buttons, etc.) follow
    // the user's pick.
    ref.watch(localeControllerProvider);
    return MaterialApp.router(
      title: 'opendray',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      themeMode: themeMode,
      locale: TranslationProvider.of(context).flutterLocale,
      supportedLocales: AppLocaleUtils.instance.supportedLocales,
      localizationsDelegates: GlobalMaterialLocalizations.delegates,
      routerConfig: router,
    );
  }
}
