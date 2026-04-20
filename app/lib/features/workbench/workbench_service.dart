import 'dart:async';

import 'package:flutter/foundation.dart';

import '../../core/api/api_client.dart';
import '../../shared/providers_bus.dart';
import 'workbench_models.dart';

/// Single source of truth for the workbench contribution registry.
///
/// Lifecycle:
///   - Construct once with an [ApiClient] + a host-supplied `showMessage`
///     callback (wired by `app.dart` to a ScaffoldMessenger SnackBar).
///   - Call [refresh] on app start and after any plugin install /
///     uninstall event. (M1: manual; M2: SSE-driven push updates.)
///   - Any widget can call [invoke] — transport errors become
///     user-visible toasts without the caller writing a try/catch.
class WorkbenchService extends ChangeNotifier {
  WorkbenchService({
    required ApiClient api,
    required void Function(String text, {bool isError}) showMessage,
  })  : _api = api,
        _showMessage = showMessage;

  final ApiClient _api;
  final void Function(String text, {bool isError}) _showMessage;

  FlatContributions _contribs = FlatContributions.empty;
  bool _loading = false;
  Object? _lastError;

  // SSE stream lifecycle. `_streaming` guards against overlapping starts.
  StreamSubscription<Map<String, dynamic>>? _eventSub;
  bool _streaming = false;
  bool _disposed = false;

  FlatContributions get contributions => _contribs;
  List<WorkbenchCommand> get commands => _contribs.commands;
  List<WorkbenchStatusBarItem> get statusBarItems => _contribs.statusBar;
  List<WorkbenchKeybinding> get keybindings => _contribs.keybindings;
  Map<String, List<WorkbenchMenuEntry>> get menus => _contribs.menus;
  List<WorkbenchActivityBarItem> get activityBarItems => _contribs.activityBar;
  List<WorkbenchView> get views => _contribs.views;
  List<WorkbenchPanel> get panels => _contribs.panels;
  bool get isLoading => _loading;
  Object? get lastError => _lastError;

  /// Currently-focused view id. `null` when no view is open, which tells
  /// `ViewHost` to render its fallback (the dashboard's normal body).
  ///
  /// Named `_currentViewID` (not `_currentView`) because it stores the id
  /// string, not the [WorkbenchView] object — lookup happens on read.
  String? get currentViewID => _currentViewID;
  String? _currentViewID;

  /// Focus the view with `viewID`. No-op if already focused — avoids an
  /// unnecessary notify during rapid taps on the same activity-bar icon.
  void openView(String viewID) {
    if (_currentViewID == viewID) return;
    _currentViewID = viewID;
    notifyListeners();
  }

  /// Close the currently-open view. No-op when nothing is open.
  void closeView() {
    if (_currentViewID == null) return;
    _currentViewID = null;
    notifyListeners();
  }

  /// Refetches `/api/workbench/contributions`. On error, keeps the
  /// previous snapshot intact (so an install flicker doesn't wipe the
  /// palette) and surfaces the error via [showMessage].
  Future<void> refresh() async {
    _loading = true;
    _lastError = null;
    notifyListeners();
    try {
      _contribs = await _api.getContributions();
    } catch (e) {
      _lastError = e;
      _showMessage(_describeError(e), isError: true);
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  /// Invokes a plugin command. For "notify" kinds we toast locally;
  /// everything else returns the [InvokeResult] so the caller can
  /// decide (open URL, show exec output sheet, subscribe to a task).
  ///
  /// Returns null on error after messaging the user.
  Future<InvokeResult?> invoke(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  }) async {
    try {
      final result = await _api.invokePluginCommand(
        pluginName,
        commandId,
        args: args,
      );
      if (result.kind == 'notify' && result.message.isNotEmpty) {
        _showMessage(result.message, isError: false);
      } else if (result.kind == 'openView' && result.viewId.isNotEmpty) {
        openView(result.viewId);
      }
      return result;
    } on PluginPermissionDeniedException catch (e) {
      _showMessage(
        'Permission denied: ${e.reason}',
        isError: true,
      );
      return null;
    } on PluginCommandUnavailableException catch (e) {
      final hint = e.deferred
          ? 'Command $pluginName.$commandId requires M2 (${e.reason})'
          : 'Command $pluginName.$commandId unavailable: ${e.reason}';
      _showMessage(hint, isError: true);
      return null;
    } on ApiException catch (e) {
      _showMessage(e.message.isEmpty ? 'HTTP ${e.statusCode}' : e.message,
          isError: true);
      return null;
    } catch (e) {
      _showMessage(_describeError(e), isError: true);
      return null;
    }
  }

  String _describeError(Object e) {
    if (e is ApiException) {
      return e.message.isEmpty ? 'HTTP ${e.statusCode}' : e.message;
    }
    return e.toString();
  }

  /// Opens the SSE subscription on `/api/workbench/stream` and forwards
  /// host → UI events. Idempotent — a second call while already
  /// streaming is a no-op.
  ///
  /// Handled event kinds:
  ///   - `contributionsChanged` → [refresh]
  ///   - `showMessage`          → `_showMessage(text, isError: kind=="error")`
  ///   - `openView`             → [openView]
  ///
  /// Unknown kinds are ignored (forward-compat). Stream errors trigger
  /// a 3-second backoff + retry; `dispose()` breaks the loop.
  void startListening() {
    if (_streaming || _disposed) return;
    _streaming = true;
    _subscribe();
  }

  void _subscribe() {
    if (_disposed) return;
    _eventSub = _api.workbenchEvents().listen(
      _onEvent,
      onError: (Object _) => _scheduleReconnect(),
      onDone: _scheduleReconnect,
      cancelOnError: true,
    );
  }

  void _scheduleReconnect() {
    _eventSub = null;
    if (_disposed) return;
    Future.delayed(const Duration(seconds: 3), () {
      if (_disposed) return;
      _subscribe();
    });
  }

  void _onEvent(Map<String, dynamic> event) {
    final kind = event['kind'];
    final payload = event['payload'];
    switch (kind) {
      case 'contributionsChanged':
        unawaited(refresh());
        break;
      case 'showMessage':
        if (payload is Map) {
          final text = payload['text']?.toString() ?? '';
          final msgKind = payload['kind']?.toString() ?? 'info';
          if (text.isNotEmpty) {
            _showMessage(text, isError: msgKind == 'error');
          }
        }
        break;
      case 'openView':
        if (payload is String && payload.isNotEmpty) {
          openView(payload);
        }
        break;
      case 'revocation':
        // Kill-switch fired. The showMessage event (emitted
        // alongside) already rendered a snackbar; we publish to
        // the ProvidersBus so the Plugin page's provider list
        // refreshes immediately — uninstalled plugins disappear
        // without waiting for the user to hit pull-to-refresh.
        ProvidersBus.instance.notify();
        break;
      // updateStatusBar / theme — covered by status bar source + theme
      // hook in later tasks.
    }
  }

  @override
  void dispose() {
    _disposed = true;
    _eventSub?.cancel();
    _eventSub = null;
    super.dispose();
  }
}
