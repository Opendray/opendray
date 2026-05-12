// ThemeController — the single source of truth for the mobile
// app's appearance choice. Persists to shared_preferences so the
// operator's pick survives app restarts; broadcasts via Riverpod
// so MaterialApp.router rebuilds whenever the value flips.
//
// Three states (System / Light / Dark) map directly to
// flutter's ThemeMode. We don't expose any half-way "accent
// override" or font-size choice yet — keep the picker simple
// until there's a real need.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

// _prefsKey is the SharedPreferences slot we own. Versioned (`v1`)
// in case a future change to the persistence format needs to
// invalidate old values without overwriting them in place.
const _prefsKey = 'opendray.appearance.theme_mode.v1';

class ThemeController extends StateNotifier<ThemeMode> {
  ThemeController() : super(ThemeMode.system) {
    _restore();
  }

  // _restore reads the stashed pick and applies it. Failures fall
  // back to system (the constructor default) — we'd rather pick
  // up the operator's OS appearance than crash because a malformed
  // shared_preferences entry got written by a buggy older build.
  Future<void> _restore() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString(_prefsKey);
      final parsed = _parse(raw);
      if (parsed != null && parsed != state) {
        state = parsed;
      }
    } on Object {
      // Best-effort: stay on the default system theme.
    }
  }

  // setMode flips the live state and fires off the persistence
  // write. Persistence is async but the UI doesn't wait — the
  // pick takes effect instantly via state notification.
  Future<void> setMode(ThemeMode mode) async {
    state = mode;
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_prefsKey, _serialise(mode));
    } on Object {
      // Persistence is best-effort — if shared_preferences is
      // unavailable, the in-memory pick still works for this
      // session.
    }
  }

  static String _serialise(ThemeMode mode) => switch (mode) {
        ThemeMode.system => 'system',
        ThemeMode.light => 'light',
        ThemeMode.dark => 'dark',
      };

  static ThemeMode? _parse(String? raw) => switch (raw) {
        'system' => ThemeMode.system,
        'light' => ThemeMode.light,
        'dark' => ThemeMode.dark,
        _ => null,
      };
}

final themeControllerProvider =
    StateNotifierProvider<ThemeController, ThemeMode>(
  (ref) => ThemeController(),
);
