import 'package:flutter/foundation.dart';

import 'running_plugins_models.dart';

/// Tracks every plugin currently "open" in the app-switcher sense.
///
/// Lifecycle is user-driven: a plugin enters the set on first navigation
/// and only leaves via an explicit [close]. Navigating away marks the
/// active id as null but never disposes entries — the widgets they
/// represent stay mounted inside `RunningPluginsHost` so their state
/// (scroll, WebView DOM, in-flight futures) survives.
///
/// All mutations emit a single [notifyListeners] at the end so
/// consumers (the host, the switcher grid, the bottom-nav badge) see
/// one consistent snapshot per event.
class RunningPluginsService extends ChangeNotifier {
  RunningPluginsService();

  final List<RunningPluginEntry> _entries = [];
  String? _activeId;
  String? _previousActiveId;

  /// Read-only view of the running set, in insertion order. The switcher
  /// sorts by `lastActiveAt` at render time; callers relying on that
  /// order should sort explicitly.
  List<RunningPluginEntry> get entries => List.unmodifiable(_entries);

  /// Id of the entry currently displayed on top of the Navigator, or
  /// `null` when a non-plugin route (Sessions, Settings, the switcher
  /// itself) is showing.
  String? get activeId => _activeId;

  /// Id that was active immediately before the last transition. Used by
  /// the host to trigger a thumbnail capture on the outgoing entry
  /// exactly when [activeId] flips away from it.
  String? get previousActiveId => _previousActiveId;

  /// Registers [seed] if an entry with the same id isn't already
  /// present. Idempotent; safe to call from a post-frame callback on
  /// every rebuild of a reveal shell.
  void ensureOpened(RunningPluginEntry seed) {
    final existing = _indexOf(seed.id);
    if (existing != -1) return;
    _entries.add(seed);
    notifyListeners();
  }

  /// Focuses the entry with [id]. No-op if [id] is already active —
  /// avoids a spurious notify that would re-trigger a thumbnail capture
  /// on every reveal-shell rebuild.
  void setActive(String id) {
    if (_activeId == id) return;
    final idx = _indexOf(id);
    if (idx == -1) return;
    _previousActiveId = _activeId;
    _activeId = id;
    _entries[idx] = _entries[idx].copyWith(lastActiveAt: DateTime.now());
    notifyListeners();
  }

  /// Marks no plugin as active (e.g. the user navigated to Sessions or
  /// opened the switcher). Entries stay in the list and mounted.
  void clearActive() {
    if (_activeId == null) return;
    _previousActiveId = _activeId;
    _activeId = null;
    notifyListeners();
  }

  /// Removes [id] from the running set. This is the only path that
  /// actually disposes the plugin widget — [RunningPluginsHost] unmounts
  /// the Offstage entry, which runs the usual `State.dispose` chain.
  void close(String id) {
    final idx = _indexOf(id);
    if (idx == -1) return;
    _entries.removeAt(idx);
    if (_activeId == id) {
      _previousActiveId = _activeId;
      _activeId = null;
    }
    notifyListeners();
  }

  /// Replaces an entry's thumbnail bytes. Silently no-ops if the entry
  /// was closed between capture and store (can happen when the user
  /// taps ✕ while a capture future is still running).
  void updateThumbnail(String id, PluginThumbnail thumb) {
    final idx = _indexOf(id);
    if (idx == -1) return;
    _entries[idx] = _entries[idx].copyWith(thumbnail: thumb);
    notifyListeners();
  }

  int _indexOf(String id) {
    for (var i = 0; i < _entries.length; i++) {
      if (_entries[i].id == id) return i;
    }
    return -1;
  }
}
