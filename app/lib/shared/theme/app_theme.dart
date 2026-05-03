// Opendray Design System v1 — Flutter theme
//
// Supports both dark and light. Legacy `AppColors.X` references and the
// `buildAppTheme()` builder are preserved as aliases so the existing 970+
// call sites keep compiling during the migration. Once every widget reads
// from `OpendrayTokens` via `Theme.of(context).extension<OpendrayTokens>()`,
// the AppColors class + buildAppTheme shim can be deleted.

import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

// ---------------------------------------------------------------------------
// Color scales
// ---------------------------------------------------------------------------

class _DarkPalette {
  static const bg            = Color(0xFF0B0D11);
  static const bgRaised      = Color(0xFF12151B);
  static const surface       = Color(0xFF141619);
  static const surface2      = Color(0xFF1C1F26);
  static const surface3      = Color(0xFF242832);
  static const border        = Color(0xFF2A2E38);
  static const borderStrong  = Color(0xFF3A3F4D);
  static const text          = Color(0xFFE6E8EF);
  static const textMuted     = Color(0xFF9097A3);
  static const textSubtle    = Color(0xFF6B7180);
  static const accent        = Color(0xFF6366F1);
  static const accentHover   = Color(0xFF7B7EF4);
  static const accentSoft    = Color(0x246366F1);
  static const accentBorder  = Color(0x596366F1);
  static const accentText    = Color(0xFFA5A8FF);
  static const success       = Color(0xFF22C55E);
  static const successSoft   = Color(0x2422C55E);
  static const warning       = Color(0xFFF59E0B);
  static const warningSoft   = Color(0x24F59E0B);
  static const danger        = Color(0xFFEF4444);
  static const dangerSoft    = Color(0x24EF4444);
  static const info          = Color(0xFF38BDF8);
  static const infoSoft      = Color(0x2438BDF8);
}

class _LightPalette {
  static const bg            = Color(0xFFF7F8FA);
  static const bgRaised      = Color(0xFFFFFFFF);
  static const surface       = Color(0xFFFFFFFF);
  static const surface2      = Color(0xFFF2F3F7);
  static const surface3      = Color(0xFFE7E9EF);
  static const border        = Color(0xFFE4E6EC);
  static const borderStrong  = Color(0xFFCFD3DB);
  static const text          = Color(0xFF0F172A);
  static const textMuted     = Color(0xFF52606D);
  static const textSubtle    = Color(0xFF8792A2);
  static const accent        = Color(0xFF4F46E5);
  static const accentHover   = Color(0xFF4338CA);
  static const accentSoft    = Color(0x144F46E5);
  static const accentBorder  = Color(0x4D4F46E5);
  static const accentText    = Color(0xFF4F46E5);
  static const success       = Color(0xFF16A34A);
  static const successSoft   = Color(0x1416A34A);
  static const warning       = Color(0xFFD97706);
  static const warningSoft   = Color(0x1AD97706);
  static const danger        = Color(0xFFDC2626);
  static const dangerSoft    = Color(0x14DC2626);
  static const info          = Color(0xFF0284C7);
  static const infoSoft      = Color(0x140284C7);
}

// ---------------------------------------------------------------------------
// Custom ThemeExtension for non-Material tokens
// ---------------------------------------------------------------------------

class OpendrayTokens extends ThemeExtension<OpendrayTokens> {
  final Color bg, bgRaised, surface, surface2, surface3;
  final Color border, borderStrong;
  final Color text, textMuted, textSubtle;
  final Color accent, accentHover, accentSoft, accentBorder, accentText;
  final Color success, successSoft;
  final Color warning, warningSoft;
  final Color danger, dangerSoft;
  final Color info, infoSoft;

  final double sp1, sp2, sp3, sp4, sp5, sp6, sp8, sp10, sp12, sp16;
  final double rXs, rSm, rMd, rLg, rXl;

  const OpendrayTokens({
    required this.bg, required this.bgRaised,
    required this.surface, required this.surface2, required this.surface3,
    required this.border, required this.borderStrong,
    required this.text, required this.textMuted, required this.textSubtle,
    required this.accent, required this.accentHover, required this.accentSoft,
    required this.accentBorder, required this.accentText,
    required this.success, required this.successSoft,
    required this.warning, required this.warningSoft,
    required this.danger, required this.dangerSoft,
    required this.info, required this.infoSoft,
    this.sp1 = 4, this.sp2 = 8, this.sp3 = 12, this.sp4 = 16, this.sp5 = 20,
    this.sp6 = 24, this.sp8 = 32, this.sp10 = 40, this.sp12 = 48, this.sp16 = 64,
    this.rXs = 4, this.rSm = 6, this.rMd = 8, this.rLg = 12, this.rXl = 16,
  });

  static const dark = OpendrayTokens(
    bg: _DarkPalette.bg, bgRaised: _DarkPalette.bgRaised,
    surface: _DarkPalette.surface, surface2: _DarkPalette.surface2, surface3: _DarkPalette.surface3,
    border: _DarkPalette.border, borderStrong: _DarkPalette.borderStrong,
    text: _DarkPalette.text, textMuted: _DarkPalette.textMuted, textSubtle: _DarkPalette.textSubtle,
    accent: _DarkPalette.accent, accentHover: _DarkPalette.accentHover,
    accentSoft: _DarkPalette.accentSoft, accentBorder: _DarkPalette.accentBorder,
    accentText: _DarkPalette.accentText,
    success: _DarkPalette.success, successSoft: _DarkPalette.successSoft,
    warning: _DarkPalette.warning, warningSoft: _DarkPalette.warningSoft,
    danger:  _DarkPalette.danger,  dangerSoft:  _DarkPalette.dangerSoft,
    info:    _DarkPalette.info,    infoSoft:    _DarkPalette.infoSoft,
  );

  static const light = OpendrayTokens(
    bg: _LightPalette.bg, bgRaised: _LightPalette.bgRaised,
    surface: _LightPalette.surface, surface2: _LightPalette.surface2, surface3: _LightPalette.surface3,
    border: _LightPalette.border, borderStrong: _LightPalette.borderStrong,
    text: _LightPalette.text, textMuted: _LightPalette.textMuted, textSubtle: _LightPalette.textSubtle,
    accent: _LightPalette.accent, accentHover: _LightPalette.accentHover,
    accentSoft: _LightPalette.accentSoft, accentBorder: _LightPalette.accentBorder,
    accentText: _LightPalette.accentText,
    success: _LightPalette.success, successSoft: _LightPalette.successSoft,
    warning: _LightPalette.warning, warningSoft: _LightPalette.warningSoft,
    danger:  _LightPalette.danger,  dangerSoft:  _LightPalette.dangerSoft,
    info:    _LightPalette.info,    infoSoft:    _LightPalette.infoSoft,
  );

  @override
  OpendrayTokens copyWith() => this;

  @override
  OpendrayTokens lerp(ThemeExtension<OpendrayTokens>? other, double t) {
    if (other is! OpendrayTokens) return this;
    return OpendrayTokens(
      bg: Color.lerp(bg, other.bg, t)!,
      bgRaised: Color.lerp(bgRaised, other.bgRaised, t)!,
      surface: Color.lerp(surface, other.surface, t)!,
      surface2: Color.lerp(surface2, other.surface2, t)!,
      surface3: Color.lerp(surface3, other.surface3, t)!,
      border: Color.lerp(border, other.border, t)!,
      borderStrong: Color.lerp(borderStrong, other.borderStrong, t)!,
      text: Color.lerp(text, other.text, t)!,
      textMuted: Color.lerp(textMuted, other.textMuted, t)!,
      textSubtle: Color.lerp(textSubtle, other.textSubtle, t)!,
      accent: Color.lerp(accent, other.accent, t)!,
      accentHover: Color.lerp(accentHover, other.accentHover, t)!,
      accentSoft: Color.lerp(accentSoft, other.accentSoft, t)!,
      accentBorder: Color.lerp(accentBorder, other.accentBorder, t)!,
      accentText: Color.lerp(accentText, other.accentText, t)!,
      success: Color.lerp(success, other.success, t)!,
      successSoft: Color.lerp(successSoft, other.successSoft, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      warningSoft: Color.lerp(warningSoft, other.warningSoft, t)!,
      danger:  Color.lerp(danger,  other.danger,  t)!,
      dangerSoft:  Color.lerp(dangerSoft,  other.dangerSoft,  t)!,
      info:    Color.lerp(info,    other.info,    t)!,
      infoSoft:    Color.lerp(infoSoft,    other.infoSoft,    t)!,
    );
  }
}

// ---------------------------------------------------------------------------
// ThemeData builders
// ---------------------------------------------------------------------------

class AppTheme {
  static final dark  = _build(Brightness.dark, OpendrayTokens.dark);
  static final light = _build(Brightness.light, OpendrayTokens.light);

  static ThemeData _build(Brightness b, OpendrayTokens t) {
    final base = b == Brightness.dark
        ? ThemeData.dark(useMaterial3: true)
        : ThemeData.light(useMaterial3: true);
    final textTheme = GoogleFonts.interTextTheme(base.textTheme).apply(
      bodyColor: t.text, displayColor: t.text,
    );

    return base.copyWith(
      brightness: b,
      scaffoldBackgroundColor: t.bg,
      canvasColor: t.bg,
      cardColor: t.surface,
      dividerColor: t.border,
      hintColor: t.textSubtle,
      disabledColor: t.textSubtle,
      colorScheme: ColorScheme(
        brightness: b,
        primary: t.accent,
        onPrimary: Colors.white,
        secondary: t.accentText,
        onSecondary: t.bg,
        error: t.danger,
        onError: Colors.white,
        surface: t.surface,
        onSurface: t.text,
        surfaceContainerHighest: t.surface2,
        outline: t.border,
        outlineVariant: t.borderStrong,
      ),
      textTheme: textTheme.copyWith(
        displayLarge: textTheme.displayLarge?.copyWith(fontSize: 36, fontWeight: FontWeight.w600, letterSpacing: -0.02 * 36, color: t.text),
        displayMedium: textTheme.displayMedium?.copyWith(fontSize: 28, fontWeight: FontWeight.w600, letterSpacing: -0.02 * 28, color: t.text),
        displaySmall: textTheme.displaySmall?.copyWith(fontSize: 22, fontWeight: FontWeight.w600, letterSpacing: -0.02 * 22, color: t.text),
        headlineSmall: textTheme.headlineSmall?.copyWith(fontSize: 18, fontWeight: FontWeight.w600, color: t.text),
        titleLarge: textTheme.titleLarge?.copyWith(fontSize: 15, fontWeight: FontWeight.w500, color: t.text),
        titleMedium: textTheme.titleMedium?.copyWith(fontSize: 14, fontWeight: FontWeight.w500, color: t.text),
        bodyLarge: textTheme.bodyLarge?.copyWith(fontSize: 14, height: 1.5, color: t.text),
        bodyMedium: textTheme.bodyMedium?.copyWith(fontSize: 13, height: 1.5, color: t.textMuted),
        bodySmall: textTheme.bodySmall?.copyWith(fontSize: 12, color: t.textMuted),
        labelLarge: textTheme.labelLarge?.copyWith(fontSize: 13, fontWeight: FontWeight.w500, color: t.text),
        labelMedium: textTheme.labelMedium?.copyWith(fontSize: 12, fontWeight: FontWeight.w500, color: t.textMuted),
        labelSmall: textTheme.labelSmall?.copyWith(fontSize: 11, fontWeight: FontWeight.w600, letterSpacing: 0.06 * 11, color: t.textSubtle),
      ),
      appBarTheme: AppBarTheme(
        backgroundColor: t.bg,
        foregroundColor: t.text,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        centerTitle: false,
        titleTextStyle: TextStyle(fontFamily: GoogleFonts.inter().fontFamily, fontSize: 16, fontWeight: FontWeight.w600, color: t.text),
      ),
      cardTheme: CardThemeData(
        color: t.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(t.rLg),
          side: BorderSide(color: t.border, width: 1),
        ),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: t.accent,
          foregroundColor: Colors.white,
          elevation: 0,
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(t.rMd)),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          textStyle: const TextStyle(fontWeight: FontWeight.w500, fontSize: 13),
          minimumSize: const Size(0, 36),
        ).copyWith(
          backgroundColor: WidgetStateProperty.resolveWith((s) =>
            s.contains(WidgetState.hovered) ? t.accentHover : t.accent),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: t.text,
          backgroundColor: t.surface2,
          side: BorderSide(color: t.border),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(t.rMd)),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          minimumSize: const Size(0, 36),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(foregroundColor: t.textMuted),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: t.bgRaised,
        isDense: true,
        contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(t.rMd),
          borderSide: BorderSide(color: t.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(t.rMd),
          borderSide: BorderSide(color: t.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(t.rMd),
          borderSide: BorderSide(color: t.accent, width: 2),
        ),
        hintStyle: TextStyle(color: t.textSubtle, fontSize: 13),
        labelStyle: TextStyle(color: t.textMuted, fontSize: 12, fontWeight: FontWeight.w500),
      ),
      dividerTheme: DividerThemeData(color: t.border, thickness: 1, space: 1),
      chipTheme: ChipThemeData(
        backgroundColor: t.surface3,
        selectedColor: t.accentSoft,
        side: BorderSide(color: t.border),
        labelStyle: TextStyle(fontSize: 11, color: t.textMuted, fontWeight: FontWeight.w500),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(t.rXl)),
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      ),
      bottomNavigationBarTheme: BottomNavigationBarThemeData(
        backgroundColor: t.surface,
        selectedItemColor: t.accent,
        unselectedItemColor: t.textMuted,
        selectedLabelStyle: const TextStyle(fontSize: 11, fontWeight: FontWeight.w500),
        unselectedLabelStyle: const TextStyle(fontSize: 11),
        showUnselectedLabels: true,
        type: BottomNavigationBarType.fixed,
      ),
      navigationRailTheme: NavigationRailThemeData(
        backgroundColor: t.surface,
        selectedIconTheme: IconThemeData(color: t.accent),
        unselectedIconTheme: IconThemeData(color: t.textMuted),
        selectedLabelTextStyle: TextStyle(color: t.accentText, fontSize: 13, fontWeight: FontWeight.w500),
        unselectedLabelTextStyle: TextStyle(color: t.textMuted, fontSize: 13),
        useIndicator: true,
        indicatorColor: t.accentSoft,
      ),
      tabBarTheme: TabBarThemeData(
        labelColor: t.text,
        unselectedLabelColor: t.textMuted,
        labelStyle: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
        indicator: UnderlineTabIndicator(borderSide: BorderSide(color: t.accent, width: 2)),
        dividerColor: t.border,
      ),
      dialogTheme: DialogThemeData(
        backgroundColor: t.bgRaised,
        surfaceTintColor: Colors.transparent,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(t.rLg)),
        elevation: 16,
      ),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: t.surface2,
        contentTextStyle: TextStyle(color: t.text, fontSize: 13),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(t.rMd)),
      ),
      tooltipTheme: TooltipThemeData(
        decoration: BoxDecoration(color: t.surface3, borderRadius: BorderRadius.circular(t.rSm)),
        textStyle: TextStyle(color: t.text, fontSize: 12),
      ),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith((s) => Colors.white),
        trackColor: WidgetStateProperty.resolveWith((s) =>
          s.contains(WidgetState.selected) ? t.accent : t.surface3),
      ),
      progressIndicatorTheme: ProgressIndicatorThemeData(color: t.accent, linearTrackColor: t.surface3),
      extensions: <ThemeExtension<dynamic>>[t],
    );
  }
}

// Monospace text style helper (for terminal-adjacent UI chrome, not xterm.js).
TextStyle mono({double size = 13, Color? color, FontWeight? weight}) =>
    GoogleFonts.jetBrainsMono(
      fontSize: size,
      color: color,
      fontWeight: weight ?? FontWeight.w400,
    );

// ---------------------------------------------------------------------------
// Backwards-compat: legacy AppColors class + buildAppTheme builder.
// 970+ widgets reference AppColors.X today; the migration to OpendrayTokens
// happens incrementally in later PRs. Each constant maps to the dark palette
// (matches v0.3.3 visual behaviour, which only had a dark mode).
// ---------------------------------------------------------------------------

class AppColors {
  static const bg          = _DarkPalette.bg;
  static const surface     = _DarkPalette.surface;
  static const surfaceAlt  = _DarkPalette.surface2;
  static const border      = _DarkPalette.border;
  static const text        = _DarkPalette.text;
  static const textMuted   = _DarkPalette.textMuted;
  static const accent      = _DarkPalette.accent;
  static const accentSoft  = _DarkPalette.accentSoft;
  static const success     = _DarkPalette.success;
  static const successSoft = _DarkPalette.successSoft;
  static const warning     = _DarkPalette.warning;
  static const warningSoft = _DarkPalette.warningSoft;
  static const error       = _DarkPalette.danger;
  static const errorSoft   = _DarkPalette.dangerSoft;
}

ThemeData buildAppTheme() => AppTheme.dark;
