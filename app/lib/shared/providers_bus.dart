import 'dart:async';

/// App-wide signal that the set of enabled providers changed (a plugin was
/// toggled, registered, configured, or removed). Pages that filter by
/// `enabled` listen and reload their data — without this, a page that
/// loaded once in initState keeps showing chips/buttons for plugins the
/// user just disabled in Settings.
///
/// Singleton on purpose: tiny coordination primitive, no DI overhead, no
/// rebuild cascade.
class ProvidersBus {
  ProvidersBus._();
  static final ProvidersBus instance = ProvidersBus._();

  final StreamController<void> _ctrl = StreamController<void>.broadcast();

  /// Listen for "the plugin set just changed" — reload your data.
  Stream<void> get changes => _ctrl.stream;

  /// Fire after a successful toggle / register / delete / config update.
  void notify() => _ctrl.add(null);
}
