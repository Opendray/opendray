import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

/// Local persistence for directory-picker quick access lists.
///
/// Two parallel lists are tracked:
///   • `recent` — LRU, capped at [_recentMax]. Written on picker confirm so
///     users can re-select previous working directories with one tap.
///   • `favorites` — user-curated, tap-to-navigate. No cap; users manage it
///     explicitly via the star toggle.
class CwdPrefs {
  static const _keyRecent = 'cwd_recent';
  static const _keyFavorites = 'cwd_favorites';
  static const _recentMax = 8;

  static Future<List<String>> getRecent() => _readList(_keyRecent);
  static Future<List<String>> getFavorites() => _readList(_keyFavorites);

  /// Prepends [path] as the most-recent entry. Deduplicates and caps length.
  static Future<void> addRecent(String path) async {
    final trimmed = path.trim();
    if (trimmed.isEmpty) return;
    final list = await _readList(_keyRecent);
    list.removeWhere((e) => e == trimmed);
    list.insert(0, trimmed);
    if (list.length > _recentMax) list.removeRange(_recentMax, list.length);
    await _writeList(_keyRecent, list);
  }

  static Future<bool> isFavorite(String path) async {
    final list = await _readList(_keyFavorites);
    return list.contains(path.trim());
  }

  /// Adds or removes [path] from favorites. Returns the new state (true=in).
  static Future<bool> toggleFavorite(String path) async {
    final trimmed = path.trim();
    if (trimmed.isEmpty) return false;
    final list = await _readList(_keyFavorites);
    final had = list.remove(trimmed);
    if (!had) list.insert(0, trimmed);
    await _writeList(_keyFavorites, list);
    return !had;
  }

  static Future<void> removeRecent(String path) async {
    final list = await _readList(_keyRecent);
    list.removeWhere((e) => e == path.trim());
    await _writeList(_keyRecent, list);
  }

  static Future<List<String>> _readList(String key) async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(key);
    if (raw == null || raw.isEmpty) return <String>[];
    try {
      final decoded = jsonDecode(raw);
      if (decoded is List) {
        return decoded.whereType<String>().toList();
      }
    } catch (_) {
      // Corrupt payload — treat as empty and let the next write repair it.
    }
    return <String>[];
  }

  static Future<void> _writeList(String key, List<String> list) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(key, jsonEncode(list));
  }
}
