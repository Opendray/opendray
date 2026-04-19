/// The Workbench view host container (T18 — M2 plugin platform).
///
/// Swaps the dashboard's main content with a plugin-contributed view
/// while one is focused (`WorkbenchService.currentViewID` is set). When
/// no view is focused, the [fallback] (the dashboard body) renders
/// unchanged — tapping the activity-bar X closes the view and restores
/// the fallback.
///
/// Rendering rules:
///   - `view.render == "webview"`    → [PluginWebView] (T16)
///   - `view.render == "declarative"` → placeholder card pointing to M5
///   - id set but view not found (plugin uninstalled mid-session) → we
///     auto-call `service.closeView()` and render [fallback]
///
/// A thin top bar shows the view title + a close button. The bar is part
/// of this widget (not the plugin) so plugins can't hide the escape hatch.
library;

import 'package:flutter/material.dart';

import 'webview_host.dart';
import 'workbench_models.dart';
import 'workbench_service.dart';

/// Test seam: widget tests can inject a stand-in for [PluginWebView] so
/// they don't have to pump a real platform WebView.
@visibleForTesting
typedef PluginWebViewBuilder = Widget Function(WorkbenchView view);

class ViewHost extends StatelessWidget {
  const ViewHost({
    required this.service,
    required this.baseUrl,
    required this.bearerToken,
    required this.fallback,
    @visibleForTesting PluginWebViewBuilder? webViewBuilder,
    super.key,
  }) : _webViewBuilder = webViewBuilder;

  final PluginWebViewBuilder? _webViewBuilder;

  /// Workbench service — provides `currentViewID` + the `views` list.
  final WorkbenchService service;

  /// Gateway base URL, forwarded to [PluginWebView]. E.g.
  /// `"http://127.0.0.1:8640"`.
  final String baseUrl;

  /// Bearer token, forwarded to [PluginWebView] for asset + bridge auth.
  final String bearerToken;

  /// Rendered when no view is open or when the focused view id no
  /// longer matches any contribution (e.g. plugin was uninstalled).
  final Widget fallback;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: service,
      builder: (context, _) {
        final id = service.currentViewID;
        if (id == null) return fallback;

        final view = _findView(service.views, id);
        if (view == null) {
          // Focused plugin was uninstalled mid-session. Close + fall
          // back — schedule on the next frame to avoid mutating state
          // while build runs.
          WidgetsBinding.instance.addPostFrameCallback((_) {
            service.closeView();
          });
          return fallback;
        }

        final body = _buildBody(view);
        return Column(
          children: [
            _ViewTopBar(
              title: view.title.isEmpty ? view.id : view.title,
              onClose: service.closeView,
            ),
            Expanded(child: body),
          ],
        );
      },
    );
  }

  Widget _buildBody(WorkbenchView view) {
    if (view.render == 'webview') {
      final builder = _webViewBuilder;
      if (builder != null) return builder(view);
      return PluginWebView(
        pluginName: view.pluginName,
        viewId: view.id,
        entryPath: view.entry,
        baseUrl: baseUrl,
        bearerToken: bearerToken,
      );
    }
    if (view.render == 'declarative') {
      return const _DeclarativePlaceholder();
    }
    // Unknown render kind — degrade to the declarative hint rather than
    // crashing, so a forward-compat render keyword doesn't bring down
    // the workbench.
    return const _DeclarativePlaceholder();
  }

  WorkbenchView? _findView(List<WorkbenchView> views, String id) {
    for (final v in views) {
      if (v.id == id) return v;
    }
    return null;
  }
}

class _ViewTopBar extends StatelessWidget {
  const _ViewTopBar({required this.title, required this.onClose});

  final String title;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: theme.colorScheme.surfaceContainerLow,
      child: SizedBox(
        height: 40,
        child: Row(
          children: [
            const SizedBox(width: 12),
            Expanded(
              child: Text(
                title,
                style: theme.textTheme.titleSmall,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            IconButton(
              icon: const Icon(Icons.close, size: 18),
              tooltip: 'Close view',
              onPressed: onClose,
            ),
          ],
        ),
      ),
    );
  }
}

class _DeclarativePlaceholder extends StatelessWidget {
  const _DeclarativePlaceholder();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Card(
          elevation: 0,
          color: theme.colorScheme.surfaceContainerHighest,
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.construction,
                    size: 36, color: theme.colorScheme.outline),
                const SizedBox(height: 12),
                Text(
                  'Declarative views arrive in M5',
                  style: theme.textTheme.titleSmall,
                ),
                const SizedBox(height: 6),
                Text(
                  'Use render: "webview" in your contribution for now.',
                  style: theme.textTheme.bodySmall,
                  textAlign: TextAlign.center,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
