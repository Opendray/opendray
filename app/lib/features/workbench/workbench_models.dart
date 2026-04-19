/// Data transfer objects for the workbench contribution registry
/// (server package `plugin/contributions`). Fields mirror the Go
/// `FlatContributions` + `Owned*` types in
/// `plugin/contributions/registry.go` — wire format must stay in sync.
///
/// Typed exceptions for plugin-command failures live here too so that
/// callers of `ApiClient.invokePluginCommand` can `catch` on the
/// specific failure (`catch (PluginPermissionDeniedException)` vs the
/// generic `ApiException`) without sniffing message strings.
library;

/// A single command a plugin contributed to the workbench.
class WorkbenchCommand {
  final String pluginName;
  final String id;
  final String title;
  final String icon;
  final String category;
  final String when;

  const WorkbenchCommand({
    required this.pluginName,
    required this.id,
    required this.title,
    this.icon = '',
    this.category = '',
    this.when = '',
  });

  factory WorkbenchCommand.fromJson(Map<String, dynamic> json) =>
      WorkbenchCommand(
        pluginName: json['pluginName'] as String? ?? '',
        id: json['id'] as String? ?? '',
        title: json['title'] as String? ?? '',
        icon: json['icon'] as String? ?? '',
        category: json['category'] as String? ?? '',
        when: json['when'] as String? ?? '',
      );
}

/// A status-bar chip a plugin contributed.
class WorkbenchStatusBarItem {
  final String pluginName;
  final String id;
  final String text;
  final String tooltip;
  final String command;
  final String alignment; // "left" | "right"
  final int priority;

  const WorkbenchStatusBarItem({
    required this.pluginName,
    required this.id,
    required this.text,
    this.tooltip = '',
    this.command = '',
    this.alignment = 'right',
    this.priority = 0,
  });

  factory WorkbenchStatusBarItem.fromJson(Map<String, dynamic> json) =>
      WorkbenchStatusBarItem(
        pluginName: json['pluginName'] as String? ?? '',
        id: json['id'] as String? ?? '',
        text: json['text'] as String? ?? '',
        tooltip: json['tooltip'] as String? ?? '',
        command: json['command'] as String? ?? '',
        alignment: (json['alignment'] as String?)?.isNotEmpty == true
            ? json['alignment'] as String
            : 'right',
        priority: (json['priority'] as num?)?.toInt() ?? 0,
      );
}

/// A keybinding binding a key combo to a command id.
class WorkbenchKeybinding {
  final String pluginName;
  final String command;
  final String key;
  final String mac;
  final String when;

  const WorkbenchKeybinding({
    required this.pluginName,
    required this.command,
    required this.key,
    this.mac = '',
    this.when = '',
  });

  factory WorkbenchKeybinding.fromJson(Map<String, dynamic> json) =>
      WorkbenchKeybinding(
        pluginName: json['pluginName'] as String? ?? '',
        command: json['command'] as String? ?? '',
        key: json['key'] as String? ?? '',
        mac: json['mac'] as String? ?? '',
        when: json['when'] as String? ?? '',
      );
}

/// An activity-bar icon a plugin contributed.
///
/// The `viewId` field (often matches the linked `contributes.views[].id`)
/// is the handle tapping the icon uses to ask the workbench to focus that
/// view. `icon` is either an emoji-ish Unicode string or a plugin-relative
/// asset path — M2 renders emoji as-is and defers asset-backed icons to
/// M6 polish (see [ActivityBar] for the render contract).
class WorkbenchActivityBarItem {
  final String pluginName;
  final String id;
  final String icon;
  final String title;
  final String viewId;

  const WorkbenchActivityBarItem({
    required this.pluginName,
    required this.id,
    required this.icon,
    required this.title,
    this.viewId = '',
  });

  factory WorkbenchActivityBarItem.fromJson(Map<String, dynamic> json) =>
      WorkbenchActivityBarItem(
        pluginName: json['pluginName'] as String? ?? '',
        id: json['id'] as String? ?? '',
        icon: json['icon'] as String? ?? '',
        title: json['title'] as String? ?? '',
        viewId: json['viewId'] as String? ?? '',
      );
}

/// A view a plugin contributed — rendered inside a `ViewHost`.
///
/// `container` ∈ {"activityBar", "panel", "sidebar"} mirrors the Go-side
/// slot name. `render` ∈ {"webview", "declarative"} picks the host:
///   - "webview"     → `PluginWebView` loads `entry` from the plugin bundle.
///   - "declarative" → placeholder in M2; full JSON-driven widget tree in M5.
class WorkbenchView {
  final String pluginName;
  final String id;
  final String title;
  final String container;
  final String icon;
  final String when;
  final String render;
  final String entry;

  const WorkbenchView({
    required this.pluginName,
    required this.id,
    required this.title,
    this.container = '',
    this.icon = '',
    this.when = '',
    this.render = '',
    this.entry = '',
  });

  factory WorkbenchView.fromJson(Map<String, dynamic> json) => WorkbenchView(
        pluginName: json['pluginName'] as String? ?? '',
        id: json['id'] as String? ?? '',
        title: json['title'] as String? ?? '',
        container: json['container'] as String? ?? '',
        icon: json['icon'] as String? ?? '',
        when: json['when'] as String? ?? '',
        render: json['render'] as String? ?? '',
        entry: json['entry'] as String? ?? '',
      );
}

/// A panel a plugin contributed — rendered inside the bottom/side panel
/// host (T19, not wired here). Parsed in M2 so the service carries the
/// data even before the panel slot widget lands.
class WorkbenchPanel {
  final String pluginName;
  final String id;
  final String title;
  final String icon;
  final String position;
  final String render;
  final String entry;

  const WorkbenchPanel({
    required this.pluginName,
    required this.id,
    required this.title,
    this.icon = '',
    this.position = '',
    this.render = '',
    this.entry = '',
  });

  factory WorkbenchPanel.fromJson(Map<String, dynamic> json) => WorkbenchPanel(
        pluginName: json['pluginName'] as String? ?? '',
        id: json['id'] as String? ?? '',
        title: json['title'] as String? ?? '',
        icon: json['icon'] as String? ?? '',
        position: json['position'] as String? ?? '',
        render: json['render'] as String? ?? '',
        entry: json['entry'] as String? ?? '',
      );
}

/// A menu entry a plugin contributed under a named slot.
class WorkbenchMenuEntry {
  final String pluginName;
  final String command;
  final String submenu;
  final String when;
  final String group;

  const WorkbenchMenuEntry({
    required this.pluginName,
    this.command = '',
    this.submenu = '',
    this.when = '',
    this.group = '',
  });

  factory WorkbenchMenuEntry.fromJson(Map<String, dynamic> json) =>
      WorkbenchMenuEntry(
        pluginName: json['pluginName'] as String? ?? '',
        command: json['command'] as String? ?? '',
        submenu: json['submenu'] as String? ?? '',
        when: json['when'] as String? ?? '',
        group: json['group'] as String? ?? '',
      );
}

/// The flat view returned by `GET /api/workbench/contributions`.
///
/// Slot ordering mirrors the server's stable sort, so two identical
/// fetches yield identical UI.
class FlatContributions {
  final List<WorkbenchCommand> commands;
  final List<WorkbenchStatusBarItem> statusBar;
  final List<WorkbenchKeybinding> keybindings;
  final Map<String, List<WorkbenchMenuEntry>> menus;
  final List<WorkbenchActivityBarItem> activityBar;
  final List<WorkbenchView> views;
  final List<WorkbenchPanel> panels;

  const FlatContributions({
    this.commands = const [],
    this.statusBar = const [],
    this.keybindings = const [],
    this.menus = const {},
    this.activityBar = const [],
    this.views = const [],
    this.panels = const [],
  });

  static const empty = FlatContributions();

  factory FlatContributions.fromJson(Map<String, dynamic> json) {
    List<T> readList<T>(String k, T Function(Map<String, dynamic>) parse) {
      final raw = json[k];
      if (raw is! List) return const [];
      return raw
          .whereType<Map>()
          .map((e) => parse(Map<String, dynamic>.from(e)))
          .toList(growable: false);
    }

    Map<String, List<WorkbenchMenuEntry>> readMenus() {
      final raw = json['menus'];
      if (raw is! Map) return const {};
      final out = <String, List<WorkbenchMenuEntry>>{};
      for (final entry in raw.entries) {
        final key = entry.key?.toString() ?? '';
        final value = entry.value;
        if (value is! List) continue;
        out[key] = value
            .whereType<Map>()
            .map((e) => WorkbenchMenuEntry.fromJson(
                Map<String, dynamic>.from(e)))
            .toList(growable: false);
      }
      return out;
    }

    return FlatContributions(
      commands: readList('commands', WorkbenchCommand.fromJson),
      statusBar: readList('statusBar', WorkbenchStatusBarItem.fromJson),
      keybindings: readList('keybindings', WorkbenchKeybinding.fromJson),
      menus: readMenus(),
      activityBar: readList('activityBar', WorkbenchActivityBarItem.fromJson),
      views: readList('views', WorkbenchView.fromJson),
      panels: readList('panels', WorkbenchPanel.fromJson),
    );
  }
}

/// Response from `POST /api/plugins/{name}/commands/{id}/invoke` when the
/// server returns 200. `kind` mirrors the dispatcher's `Result.kind`.
class InvokeResult {
  final String kind; // "notify" | "openUrl" | "exec" | "runTask"
  final String message;
  final String url;
  final String taskId;
  final String output;
  final int exit;

  const InvokeResult({
    required this.kind,
    this.message = '',
    this.url = '',
    this.taskId = '',
    this.output = '',
    this.exit = 0,
  });

  factory InvokeResult.fromJson(Map<String, dynamic> json) => InvokeResult(
        kind: json['kind'] as String? ?? '',
        message: json['message'] as String? ?? '',
        url: json['url'] as String? ?? '',
        taskId: json['taskId'] as String? ?? '',
        output: json['output'] as String? ?? '',
        exit: (json['exit'] as num?)?.toInt() ?? 0,
      );
}

/// Thrown when the server denies a plugin command call via the
/// capability gate (HTTP 403, `code="EPERM"`).
class PluginPermissionDeniedException implements Exception {
  final String pluginName;
  final String commandId;
  final String reason;
  PluginPermissionDeniedException(this.pluginName, this.commandId, this.reason);
  @override
  String toString() =>
      'Permission denied invoking $pluginName.$commandId: $reason';
}

/// Thrown when the server reports the command can't be satisfied in v1
/// — either the run kind is deferred (`host`/`openView`, HTTP 501) or
/// the command is unknown (HTTP 404).
class PluginCommandUnavailableException implements Exception {
  final String pluginName;
  final String commandId;
  final String reason;
  final bool deferred; // true = 501 (M2/M3), false = 404 (not found)
  PluginCommandUnavailableException(
    this.pluginName,
    this.commandId,
    this.reason, {
    this.deferred = false,
  });
  @override
  String toString() => 'Command $pluginName.$commandId unavailable: $reason';
}
