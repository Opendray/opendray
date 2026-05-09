import 'package:flutter/material.dart';

// Material 3 dark theme, single design language with the web SPA.
// Color palette mirrors app/web/src/index.css custom properties so
// screens look like one product across surfaces.
class AppTheme {
  AppTheme._();

  static const _accent = Color(0xFFE6AE57); // #e6ae57 — opendray accent
  static const _bg = Color(0xFF0E0F11);
  static const _card = Color(0xFF161718);
  static const _border = Color(0xFF2A2C30);
  static const _muted = Color(0xFF8B9098);
  static const _destructive = Color(0xFFE07A5F);

  static ThemeData dark() {
    const scheme = ColorScheme.dark(
      primary: _accent,
      onPrimary: Color(0xFF1A1308),
      secondary: _accent,
      onSecondary: Color(0xFF1A1308),
      surface: _card,
      onSurface: Colors.white,
      surfaceContainerHighest: _card,
      error: _destructive,
      onError: Colors.white,
      outline: _border,
      outlineVariant: _border,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: _bg,
      brightness: Brightness.dark,
      textTheme: const TextTheme(
        bodyLarge: TextStyle(color: Colors.white),
        bodyMedium: TextStyle(color: Colors.white),
        bodySmall: TextStyle(color: _muted),
        labelMedium: TextStyle(color: _muted),
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: _bg,
        foregroundColor: Colors.white,
        elevation: 0,
        centerTitle: false,
      ),
      cardTheme: CardThemeData(
        color: _card,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          side: const BorderSide(color: _border),
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: const Color(0xFF111214),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: _border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: _border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: _accent),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 14,
          vertical: 14,
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: _accent,
          foregroundColor: const Color(0xFF1A1308),
          padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 18),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(10),
          ),
          textStyle: const TextStyle(
            fontSize: 15,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      bottomNavigationBarTheme: const BottomNavigationBarThemeData(
        backgroundColor: _card,
        selectedItemColor: _accent,
        unselectedItemColor: _muted,
        type: BottomNavigationBarType.fixed,
        showUnselectedLabels: true,
      ),
      dividerColor: _border,
    );
  }
}
