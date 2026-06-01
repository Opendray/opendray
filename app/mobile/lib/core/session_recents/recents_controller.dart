// RecentsController — per-device "most recently opened" timestamps for
// sessions, so the sessions list can surface what you actually use on
// top. Mirrors the web zustand sessionTabs.recents map (opendray.
// sessionTabs in localStorage). Persists to shared_preferences like
// LocaleController / ThemeController so the order survives restarts.

import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _prefsKey = 'opendray.sessions.recents.v1';

class RecentsController extends StateNotifier<Map<String, int>> {
  RecentsController() : super(const {}) {
    _restore();
  }

  Future<void> _restore() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString(_prefsKey);
      if (raw == null || raw.isEmpty) return;
      final decoded = jsonDecode(raw);
      if (decoded is Map) {
        state = decoded.map(
          (k, v) => MapEntry(k as String, (v as num).toInt()),
        );
      }
    } on Object {
      // Best-effort; empty map fallback keeps the list working.
    }
  }

  // Record that session [id] was opened now (epoch millis). The web
  // store stamps on both open and tab-activate; mobile has one path
  // (tap → push), so a single mark is enough.
  Future<void> markOpened(String id) async {
    if (id.isEmpty) return;
    final next = Map<String, int>.from(state)
      ..[id] = DateTime.now().millisecondsSinceEpoch;
    state = next;
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_prefsKey, jsonEncode(next));
    } on Object {
      // Best-effort persistence; in-memory order still works this run.
    }
  }
}

final recentsProvider =
    StateNotifierProvider<RecentsController, Map<String, int>>(
  (ref) => RecentsController(),
);
