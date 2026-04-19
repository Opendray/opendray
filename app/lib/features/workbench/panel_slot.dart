/// The Workbench panel slot (T19 — M2 plugin platform).
///
/// Hosts plugin-contributed panels (`contributes.panels[]`) at the bottom
/// of the session layout. Layout follows `docs/plugin-platform/08-
/// workbench-slots.md`:
///
///   - **Phone** (`width < 600`): collapsed by default — only a thin bar
///     showing a chevron + the first panel's title. Tap the chevron to
///     expand the panel to `expandedHeight`. Tap again to collapse.
///   - **Tablet** (`width ≥ 600`): always visible. Horizontal tab row +
///     the active panel's content pinned below.
///
/// Rendering rules per panel (mirrors `ViewHost`):
///   - `render == "webview"`    → [PluginWebView] with `viewId = panel.id`
///   - `render == "declarative"` → M5 placeholder card
///   - unknown render            → M5 placeholder (forward-compatible)
///
/// Empty `service.panels` collapses to `SizedBox.shrink()` — zero chrome
/// until a plugin contributes.
///
/// **Per-panel isolation** (M2 design decision): tapping another tab
/// disposes the current panel's WebView and builds a fresh one for the
/// new tab. This is intentional — at the cost of a rebuild, it keeps the
/// bridge-channel lifecycle trivially scoped to "one panel at a time",
/// avoids leaking state across panels from different plugins, and matches
/// the per-view isolation contract `PluginWebView` already enforces.
/// Multi-panel persistence is a post-M2 polish item.
library;

import 'package:flutter/material.dart';

import 'webview_host.dart';
import 'workbench_models.dart';
import 'workbench_service.dart';

/// Test seam + public builder signature. Widget tests inject a stand-in
/// that returns a simple [Text] so they don't have to pump a real
/// platform WebView. The same signature documents exactly which args the
/// host forwards into [PluginWebView].
typedef PluginWebViewBuilder = Widget Function(
  BuildContext context, {
  required String pluginName,
  required String viewId,
  required String entryPath,
  required String baseUrl,
  required String bearerToken,
});

/// Breakpoint separating phone (collapsed drawer) from tablet (always-on
/// tab bar). Matches the 600-dp breakpoint used throughout the shell.
const double _phoneBreakpoint = 600;

class PanelSlot extends StatefulWidget {
  const PanelSlot({
    required this.service,
    required this.baseUrl,
    required this.bearerToken,
    this.collapsedHeight = 32,
    this.expandedHeight = 280,
    @visibleForTesting this.webViewBuilder,
    super.key,
  });

  /// Workbench service — provides `panels` and fires `notifyListeners`
  /// on contribution changes. Listened to via [ListenableBuilder].
  final WorkbenchService service;

  /// Gateway base URL forwarded to [PluginWebView] (e.g.
  /// `"http://127.0.0.1:8640"`).
  final String baseUrl;

  /// Bearer token forwarded to [PluginWebView] for asset + bridge auth.
  final String bearerToken;

  /// Collapsed (phone, closed) height of the slot — the thin tab bar
  /// that surfaces the chevron + first panel title. Default 32.
  final double collapsedHeight;

  /// Expanded height of the slot when the panel body is visible.
  /// Default 280.
  final double expandedHeight;

  /// Test seam — when non-null, used in place of constructing a real
  /// [PluginWebView]. Production callers never set this.
  @visibleForTesting
  final PluginWebViewBuilder? webViewBuilder;

  @override
  State<PanelSlot> createState() => _PanelSlotState();
}

class _PanelSlotState extends State<PanelSlot> {
  /// Id of the currently-active panel. Null when no panels are
  /// contributed. Set to the first panel's id on first render.
  String? _activePanelID;

  /// Phone-only. Tablet always renders expanded.
  bool _expanded = false;

  WorkbenchPanel? _resolveActive(List<WorkbenchPanel> panels) {
    if (panels.isEmpty) return null;
    final id = _activePanelID;
    if (id != null) {
      for (final p in panels) {
        if (p.id == id) return p;
      }
    }
    // Activated id missing (first frame or contribution removed) — fall
    // back to the first panel.
    return panels.first;
  }

  void _selectPanel(WorkbenchPanel panel) {
    if (_activePanelID == panel.id) return;
    setState(() => _activePanelID = panel.id);
  }

  void _toggleExpanded() {
    setState(() => _expanded = !_expanded);
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: widget.service,
      builder: (context, _) {
        final panels = widget.service.panels;
        if (panels.isEmpty) {
          return const SizedBox.shrink();
        }

        final active = _resolveActive(panels);
        // _resolveActive returns null only when panels is empty — which
        // we just handled — so active is always non-null from here on.
        if (active == null) return const SizedBox.shrink();

        final width = MediaQuery.of(context).size.width;
        final isPhone = width < _phoneBreakpoint;

        if (isPhone && !_expanded) {
          return _CollapsedBar(
            title: active.title.isEmpty ? active.id : active.title,
            height: widget.collapsedHeight,
            onTap: _toggleExpanded,
          );
        }

        return _ExpandedSlot(
          panels: panels,
          active: active,
          height: widget.expandedHeight,
          isPhone: isPhone,
          onSelect: _selectPanel,
          onCollapse: isPhone ? _toggleExpanded : null,
          body: _buildBody(active),
        );
      },
    );
  }

  Widget _buildBody(WorkbenchPanel panel) {
    if (panel.render == 'webview') {
      final builder = widget.webViewBuilder;
      if (builder != null) {
        return builder(
          context,
          pluginName: panel.pluginName,
          viewId: panel.id,
          entryPath: panel.entry,
          baseUrl: widget.baseUrl,
          bearerToken: widget.bearerToken,
        );
      }
      // Key the webview by the panel id so Flutter disposes the old
      // state and builds a fresh one when the tab switches — the
      // per-panel isolation contract documented in the library
      // docstring. Without this key, Flutter would reuse the State
      // object and the underlying controller would keep the prior
      // panel's bridge alive.
      return PluginWebView(
        key: ValueKey('panel-${panel.pluginName}-${panel.id}'),
        pluginName: panel.pluginName,
        viewId: panel.id,
        entryPath: panel.entry,
        baseUrl: widget.baseUrl,
        bearerToken: widget.bearerToken,
      );
    }
    if (panel.render == 'declarative') {
      return const _DeclarativePlaceholder();
    }
    return const _DeclarativePlaceholder();
  }
}

/// The thin collapsed bar shown on phones when the panel drawer is
/// closed. Tapping anywhere on the bar (drag handle included) expands
/// the panel.
class _CollapsedBar extends StatelessWidget {
  const _CollapsedBar({
    required this.title,
    required this.height,
    required this.onTap,
  });

  final String title;
  final double height;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: theme.colorScheme.surfaceContainerLow,
      child: InkWell(
        onTap: onTap,
        child: SizedBox(
          height: height,
          child: Row(
            children: [
              const SizedBox(width: 12),
              IconButton(
                icon: const Icon(Icons.keyboard_arrow_up, size: 18),
                tooltip: 'Expand panel',
                onPressed: onTap,
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(
                  minWidth: 32,
                  minHeight: 32,
                ),
              ),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  title,
                  style: theme.textTheme.labelMedium,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// The expanded slot — a horizontal tab row on top of the active
/// panel's body. Used on tablet always, and on phone when the drawer is
/// open.
class _ExpandedSlot extends StatelessWidget {
  const _ExpandedSlot({
    required this.panels,
    required this.active,
    required this.height,
    required this.isPhone,
    required this.onSelect,
    required this.onCollapse,
    required this.body,
  });

  final List<WorkbenchPanel> panels;
  final WorkbenchPanel active;
  final double height;
  final bool isPhone;
  final ValueChanged<WorkbenchPanel> onSelect;
  final VoidCallback? onCollapse;
  final Widget body;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: theme.colorScheme.surface,
      child: SizedBox(
        height: height,
        child: Column(
          children: [
            _TabBar(
              panels: panels,
              activeId: active.id,
              onSelect: onSelect,
              onCollapse: onCollapse,
            ),
            const Divider(height: 1),
            Expanded(child: body),
          ],
        ),
      ),
    );
  }
}

class _TabBar extends StatelessWidget {
  const _TabBar({
    required this.panels,
    required this.activeId,
    required this.onSelect,
    required this.onCollapse,
  });

  final List<WorkbenchPanel> panels;
  final String activeId;
  final ValueChanged<WorkbenchPanel> onSelect;
  final VoidCallback? onCollapse;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      height: 36,
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLow,
      ),
      child: Row(
        children: [
          Expanded(
            child: ListView(
              scrollDirection: Axis.horizontal,
              children: [
                for (final p in panels)
                  _TabButton(
                    panel: p,
                    selected: p.id == activeId,
                    onTap: () => onSelect(p),
                  ),
              ],
            ),
          ),
          if (onCollapse != null)
            IconButton(
              icon: const Icon(Icons.keyboard_arrow_down, size: 18),
              tooltip: 'Collapse panel',
              onPressed: onCollapse,
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(
                minWidth: 32,
                minHeight: 32,
              ),
            ),
        ],
      ),
    );
  }
}

class _TabButton extends StatelessWidget {
  const _TabButton({
    required this.panel,
    required this.selected,
    required this.onTap,
  });

  final WorkbenchPanel panel;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final label = panel.title.isEmpty ? panel.id : panel.title;
    final fg = selected
        ? theme.colorScheme.primary
        : theme.colorScheme.onSurfaceVariant;
    final underline = selected
        ? theme.colorScheme.primary
        : Colors.transparent;
    return Semantics(
      label: label,
      button: true,
      selected: selected,
      child: InkWell(
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12),
          decoration: BoxDecoration(
            border: Border(
              bottom: BorderSide(color: underline, width: 2),
            ),
          ),
          alignment: Alignment.center,
          child: Text(
            label,
            style: theme.textTheme.labelMedium?.copyWith(
              color: fg,
              fontWeight:
                  selected ? FontWeight.w600 : FontWeight.normal,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
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
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.construction,
                size: 28, color: theme.colorScheme.outline),
            const SizedBox(height: 8),
            Text(
              'Declarative panels arrive in M5',
              style: theme.textTheme.labelMedium,
            ),
          ],
        ),
      ),
    );
  }
}
