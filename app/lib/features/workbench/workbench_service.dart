import 'package:flutter/foundation.dart';

import '../../core/api/api_client.dart';
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

  FlatContributions get contributions => _contribs;
  List<WorkbenchCommand> get commands => _contribs.commands;
  List<WorkbenchStatusBarItem> get statusBarItems => _contribs.statusBar;
  List<WorkbenchKeybinding> get keybindings => _contribs.keybindings;
  Map<String, List<WorkbenchMenuEntry>> get menus => _contribs.menus;
  bool get isLoading => _loading;
  Object? get lastError => _lastError;

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
}
