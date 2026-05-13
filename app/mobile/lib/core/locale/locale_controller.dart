// LocaleController — single source of truth for the user's language
// pick. Mirrors ThemeController: persists to shared_preferences so
// the choice survives restarts, broadcasts via Riverpod so
// MaterialApp.router rebuilds whenever the value flips.
//
// Three states (System / English / Chinese) decouple "what the user
// asked for" from "what slang ended up using" — useful because the
// system locale can be anything and slang resolves it via fallback.

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:shared_preferences/shared_preferences.dart';

enum LocalePreference { system, english, chinese }

const _prefsKey = 'opendray.locale.preference.v1';

class LocaleController extends StateNotifier<LocalePreference> {
  LocaleController() : super(LocalePreference.system) {
    _restore();
  }

  Future<void> _restore() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final parsed = _parse(prefs.getString(_prefsKey));
      if (parsed != null && parsed != state) {
        state = parsed;
      }
      _apply(state);
    } on Object {
      _apply(state);
    }
  }

  Future<void> setPreference(LocalePreference pref) async {
    state = pref;
    _apply(pref);
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_prefsKey, _serialise(pref));
    } on Object {
      // Best-effort persistence; in-memory pick still works.
    }
  }

  static void _apply(LocalePreference pref) {
    switch (pref) {
      case LocalePreference.system:
        LocaleSettings.useDeviceLocale();
      case LocalePreference.english:
        LocaleSettings.setLocale(AppLocale.en);
      case LocalePreference.chinese:
        LocaleSettings.setLocale(AppLocale.zh);
    }
  }

  static String _serialise(LocalePreference pref) => switch (pref) {
        LocalePreference.system => 'system',
        LocalePreference.english => 'english',
        LocalePreference.chinese => 'chinese',
      };

  static LocalePreference? _parse(String? raw) => switch (raw) {
        'system' => LocalePreference.system,
        'english' => LocalePreference.english,
        'chinese' => LocalePreference.chinese,
        _ => null,
      };
}

final localeControllerProvider =
    StateNotifierProvider<LocaleController, LocalePreference>(
  (ref) => LocaleController(),
);
