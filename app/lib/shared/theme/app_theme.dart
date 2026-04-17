import 'package:flutter/material.dart';

class AppColors {
  static const bg = Color(0xFF0B0D11);
  static const surface = Color(0xFF141619);
  static const surfaceAlt = Color(0xFF1C1F26);
  static const border = Color(0xFF2A2E38);
  static const text = Color(0xFFE1E4ED);
  static const textMuted = Color(0xFF6B7280);
  static const accent = Color(0xFF6366F1);
  static const accentSoft = Color(0x266366F1);
  static const success = Color(0xFF22C55E);
  static const successSoft = Color(0x2622C55E);
  static const warning = Color(0xFFF59E0B);
  static const warningSoft = Color(0x26F59E0B);
  static const error = Color(0xFFEF4444);
  static const errorSoft = Color(0x26EF4444);
}

ThemeData buildAppTheme() {
  return ThemeData(
    brightness: Brightness.dark,
    scaffoldBackgroundColor: AppColors.bg,
    colorScheme: const ColorScheme.dark(
      primary: AppColors.accent,
      surface: AppColors.surface,
      error: AppColors.error,
    ),
    fontFamily: '.SF Pro Text',
    appBarTheme: const AppBarTheme(
      backgroundColor: AppColors.surface,
      elevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        color: AppColors.text,
        fontSize: 16,
        fontWeight: FontWeight.w600,
      ),
    ),
    cardTheme: CardThemeData(
      color: AppColors.surface,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: const BorderSide(color: AppColors.border),
      ),
      margin: EdgeInsets.zero,
    ),
    dividerColor: AppColors.border,
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: AppColors.surfaceAlt,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
        borderSide: const BorderSide(color: AppColors.border),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
        borderSide: const BorderSide(color: AppColors.border),
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
      hintStyle: const TextStyle(color: AppColors.textMuted, fontSize: 13),
      labelStyle: const TextStyle(color: AppColors.textMuted, fontSize: 12),
    ),
    textTheme: const TextTheme(
      bodyMedium: TextStyle(color: AppColors.text, fontSize: 14),
      bodySmall: TextStyle(color: AppColors.textMuted, fontSize: 12),
      titleMedium: TextStyle(color: AppColors.text, fontSize: 16, fontWeight: FontWeight.w600),
    ),
    bottomNavigationBarTheme: const BottomNavigationBarThemeData(
      backgroundColor: AppColors.surface,
      selectedItemColor: AppColors.accent,
      unselectedItemColor: AppColors.textMuted,
      type: BottomNavigationBarType.fixed,
      selectedLabelStyle: TextStyle(fontSize: 11),
      unselectedLabelStyle: TextStyle(fontSize: 11),
    ),
  );
}
