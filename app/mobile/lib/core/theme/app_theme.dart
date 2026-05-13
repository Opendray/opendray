import 'package:flutter/material.dart';

// Material 3 themes for the mobile app. Single design language with
// the web SPA: the dark palette mirrors app/web/src/index.css custom
// properties so screens look like one product across surfaces.
//
// The light variant was added in PR #51 to let operators flip the
// app to match their phone's system appearance. The same accent
// (#e6ae57) drives both variants — only the surface / text / border
// tokens swap.
class AppTheme {
  AppTheme._();

  // Shared accent. Used as primary in both variants.
  static const _accent = Color(0xFFE6AE57);
  static const _destructive = Color(0xFFE07A5F);

  // ── Dark palette ────────────────────────────────────────────
  static const _darkBg = Color(0xFF0E0F11);
  static const _darkCard = Color(0xFF161718);
  static const _darkBorder = Color(0xFF2A2C30);
  static const _darkMuted = Color(0xFF8B9098);
  static const _darkInput = Color(0xFF111214);

  // ── Light palette ───────────────────────────────────────────
  // Chosen to keep readable contrast against the same accent. Off-
  // white background (not pure #FFFFFF) reduces glare under bright
  // sunlight, common when the operator uses the phone outdoors.
  //
  // Contrast notes (vs `_lightBg = #F7F7F5`):
  //  - `_lightText`  (#111418) ≈ 16:1   — body text, easily AAA
  //  - `_lightMuted` (#4B5563) ≈ 6.5:1  — secondary text, AA-clean
  //    (was #6B7280 ≈ 4.17:1, which fell below the WCAG-AA 4.5:1
  //    minimum and read as a grey haze on cwd rows / timestamps)
  //  - `_lightBorder` (#D1D5DB) ≈ 1.5:1 — non-text, just visible
  static const _lightBg = Color(0xFFF7F7F5);
  static const _lightCard = Color(0xFFFFFFFF);
  static const _lightBorder = Color(0xFFD1D5DB);
  static const _lightMuted = Color(0xFF4B5563);
  static const _lightInput = Color(0xFFFAFAF9);
  static const _lightText = Color(0xFF111418);

  static ThemeData dark() {
    const scheme = ColorScheme.dark(
      primary: _accent,
      onPrimary: Color(0xFF1A1308),
      secondary: _accent,
      onSecondary: Color(0xFF1A1308),
      surface: _darkCard,
      onSurface: Colors.white,
      surfaceContainerHighest: _darkCard,
      error: _destructive,
      onError: Colors.white,
      outline: _darkBorder,
      outlineVariant: _darkBorder,
    );

    return _build(
      brightness: Brightness.dark,
      scheme: scheme,
      bg: _darkBg,
      card: _darkCard,
      border: _darkBorder,
      muted: _darkMuted,
      inputFill: _darkInput,
      bodyColor: Colors.white,
      appBarFg: Colors.white,
      navUnselected: _darkMuted,
    );
  }

  static ThemeData light() {
    const scheme = ColorScheme.light(
      primary: _accent,
      onPrimary: Color(0xFF1A1308),
      secondary: _accent,
      onSecondary: Color(0xFF1A1308),
      surface: _lightCard,
      onSurface: _lightText,
      surfaceContainerHighest: _lightCard,
      error: _destructive,
      onError: Colors.white,
      outline: _lightBorder,
      outlineVariant: _lightBorder,
    );

    return _build(
      brightness: Brightness.light,
      scheme: scheme,
      bg: _lightBg,
      card: _lightCard,
      border: _lightBorder,
      muted: _lightMuted,
      inputFill: _lightInput,
      bodyColor: _lightText,
      appBarFg: _lightText,
      navUnselected: _lightMuted,
    );
  }

  // _build is the shared chassis — every visual choice that doesn't
  // depend on brightness lives here so light/dark can never drift
  // structurally. Only the color tokens are parameterised.
  static ThemeData _build({
    required Brightness brightness,
    required ColorScheme scheme,
    required Color bg,
    required Color card,
    required Color border,
    required Color muted,
    required Color inputFill,
    required Color bodyColor,
    required Color appBarFg,
    required Color navUnselected,
  }) {
    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: bg,
      brightness: brightness,
      textTheme: TextTheme(
        bodyLarge: TextStyle(color: bodyColor),
        bodyMedium: TextStyle(color: bodyColor),
        bodySmall: TextStyle(color: muted),
        labelMedium: TextStyle(color: muted),
      ),
      appBarTheme: AppBarTheme(
        backgroundColor: bg,
        foregroundColor: appBarFg,
        elevation: 0,
        centerTitle: false,
      ),
      cardTheme: CardThemeData(
        color: card,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          side: BorderSide(color: border),
          borderRadius: BorderRadius.circular(12),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: inputFill,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: BorderSide(color: border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: BorderSide(color: border),
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
      bottomNavigationBarTheme: BottomNavigationBarThemeData(
        backgroundColor: card,
        selectedItemColor: _accent,
        unselectedItemColor: navUnselected,
        type: BottomNavigationBarType.fixed,
        showUnselectedLabels: true,
      ),
      dividerColor: border,
    );
  }
}
