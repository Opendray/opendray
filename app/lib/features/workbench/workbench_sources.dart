/// Adapters that bridge [WorkbenchService] (T19) to the decoupled source
/// interfaces consumed by [StatusBarStrip] (T20) and [MenuSlot] (T22).
///
/// Why adapters exist:
///   - T20 and T22 were designed to depend on narrow `Listenable`-shaped
///     interfaces (`StatusBarSource`, [MenuSource]) so they could be
///     tested without standing up a full `WorkbenchService` (which in
///     turn requires an `ApiClient`).
///   - The service already satisfies the *shape* these widgets want,
///     but we still need a thin wrapper to (a) re-fire `notifyListeners`
///     onto the adapter so widget `ListenableBuilder`s can subscribe
///     without holding a service reference, and (b) coerce the
///     `Future<InvokeResult?>` return signature down to the widgets'
///     `Future<void>`.
///
/// Each adapter owns the forwarding subscription and cleans up in
/// [dispose]; the embedding `StatefulWidget` in each page is responsible
/// for calling dispose, which unwinds the listener registered on the
/// underlying service.
library;

import 'package:flutter/foundation.dart';

import 'status_bar_strip.dart';
import 'workbench_models.dart';
import 'workbench_service.dart';

/// Adapts [WorkbenchService] to [StatusBarSource] (T20's interface).
///
/// The underlying service's [Listenable] + `statusBarItems` already
/// satisfy the interface shape; this wrapper bridges the `invoke`
/// signature (T20 wants `Future<void>`, service returns
/// `Future<InvokeResult?>`).
class WorkbenchStatusBarSource extends ChangeNotifier
    implements StatusBarSource {
  WorkbenchStatusBarSource(this._service) {
    _service.addListener(_forward);
  }

  final WorkbenchService _service;

  void _forward() => notifyListeners();

  @override
  List<WorkbenchStatusBarItem> get statusBarItems => _service.statusBarItems;

  @override
  Future<void> invoke(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  }) async {
    await _service.invoke(pluginName, commandId, args: args);
  }

  @override
  void dispose() {
    _service.removeListener(_forward);
    super.dispose();
  }
}

/// Minimum interface [MenuSlot] consumes — mirrors [StatusBarSource]'s
/// decoupling pattern so widget tests don't require a real
/// `WorkbenchService`.
abstract class MenuSource implements Listenable {
  /// Returns the contributed menu entries for a named slot
  /// (e.g. `"appBar/right"`). Must return `const []` (never null) for
  /// slots with no contributions — [MenuSlot] relies on this to render
  /// `SizedBox.shrink()`.
  List<WorkbenchMenuEntry> entriesFor(String slotId);

  /// Fire-and-forget command invocation. Callers (the widget's
  /// `onSelected`) don't await — any failure is surfaced via the
  /// service's own showMessage channel.
  Future<void> invoke(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  });
}

/// Adapts [WorkbenchService] to [MenuSource].
class WorkbenchMenuSource extends ChangeNotifier implements MenuSource {
  WorkbenchMenuSource(this._service) {
    _service.addListener(_forward);
  }

  final WorkbenchService _service;

  void _forward() => notifyListeners();

  @override
  List<WorkbenchMenuEntry> entriesFor(String slotId) {
    return _service.menus[slotId] ?? const [];
  }

  @override
  Future<void> invoke(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  }) async {
    await _service.invoke(pluginName, commandId, args: args);
  }

  @override
  void dispose() {
    _service.removeListener(_forward);
    super.dispose();
  }
}
