import 'package:flutter/material.dart';

import '../../../core/api/api_client.dart';
import '../../browser/preview_page.dart';
import '../../docs/docs_page.dart';
import '../../files/files_page.dart';
import '../../logs/logs_page.dart';
import '../../mcp/mcp_page.dart';
import '../../messaging/telegram_page.dart';
import '../../pg/pg_page.dart';
import '../../source_control/source_control_page.dart';
import '../../tasks/tasks_page.dart';
import '../webview_host.dart';
import '../workbench_models.dart';
import 'running_plugins_models.dart';

/// Catalog of every plugin surface that can enter the running set.
///
/// Two categories:
///   • **Built-ins** — the ten `/browser/*` panels. Each has a fixed id,
///     title key, icon, route, and page widget factory.
///   • **Webview plugins** — generic v1 plugins. Seeded on demand from
///     `WorkbenchService.views`; the factory [webviewSeed] below takes
///     the manifest-derived view metadata plus the runtime pieces
///     ([ApiClient] for base URL + token) needed to build a
///     [PluginWebView].
class PluginRegistry {
  PluginRegistry._();

  /// Stable id for each built-in panel. Keeping these as symbolic
  /// constants (rather than strings scattered across app.dart) makes
  /// the route → plugin wiring typo-proof.
  static const String builtinDocs = 'builtin:docs';
  static const String builtinFiles = 'builtin:files';
  static const String builtinTasks = 'builtin:tasks';
  static const String builtinSourceControl = 'builtin:source-control';
  static const String builtinDatabase = 'builtin:database';
  static const String builtinLogs = 'builtin:logs';
  static const String builtinMessaging = 'builtin:messaging';
  static const String builtinMcp = 'builtin:mcp';
  static const String builtinPreview = 'builtin:preview';
  static const String builtinSimulator = 'builtin:simulator';

  /// Built-in route → plugin id. `_Shell` uses this to resolve the
  /// current GoRouter location to a seed without per-route boilerplate.
  static const Map<String, String> _builtinRoutes = {
    '/browser/docs': builtinDocs,
    '/browser/files': builtinFiles,
    '/browser/tasks': builtinTasks,
    '/browser/source-control': builtinSourceControl,
    '/browser/database': builtinDatabase,
    '/browser/logs': builtinLogs,
    '/browser/messaging': builtinMessaging,
    '/browser/mcp': builtinMcp,
    '/browser/preview': builtinPreview,
    '/browser/simulator': builtinSimulator,
  };

  /// Returns the builtin plugin id for [route] (exact match), or null
  /// when [route] is not a built-in panel path. Webview plugin routes
  /// (`/browser/plugin/<name>`) are handled separately by the caller
  /// since they need live WorkbenchService / ApiClient context.
  static String? builtinIdForRoute(String route) => _builtinRoutes[route];

  /// Convenience: seed for [route], or null when the route is not a
  /// known built-in panel.
  static RunningPluginEntry? seedForBuiltinRoute(String route) {
    final id = builtinIdForRoute(route);
    if (id == null) return null;
    return builtinSeed(id);
  }

  /// Returns a fresh seed for the built-in panel with [id]. `openedAt`
  /// / `lastActiveAt` are set to "now" so the first entry lands at the
  /// top of the recency sort.
  static RunningPluginEntry? builtinSeed(String id) {
    final meta = _builtins[id];
    if (meta == null) return null;
    final now = DateTime.now();
    return RunningPluginEntry(
      id: id,
      titleKey: meta.titleKey,
      icon: meta.icon,
      route: meta.route,
      kind: RunningPluginKind.builtin,
      builder: meta.builder,
      openedAt: now,
      lastActiveAt: now,
    );
  }

  /// Builds a seed for a generic v1 webview plugin. Caller supplies
  /// the [WorkbenchView] (from `WorkbenchService.views`) plus the live
  /// API identity so the produced [PluginWebView] can authenticate its
  /// asset and bridge requests.
  static RunningPluginEntry webviewSeed({
    required WorkbenchView view,
    required ApiClient api,
  }) {
    final now = DateTime.now();
    final pluginName = view.pluginName;
    final title = view.title.isEmpty ? pluginName : view.title;
    return RunningPluginEntry(
      id: 'webview:$pluginName',
      titleKey: title,
      icon: Icons.extension_outlined,
      route: '/browser/plugin/$pluginName',
      kind: RunningPluginKind.webview,
      builder: (_) => PluginWebView(
        pluginName: pluginName,
        viewId: view.id,
        entryPath: view.entry,
        baseUrl: api.baseUrl,
        bearerToken: api.token ?? '',
      ),
      openedAt: now,
      lastActiveAt: now,
    );
  }

  static final Map<String, _BuiltinMeta> _builtins = {
    builtinDocs: _BuiltinMeta(
      titleKey: 'Docs',
      icon: Icons.menu_book_outlined,
      route: '/browser/docs',
      builder: (_) => const DocsPage(),
    ),
    builtinFiles: _BuiltinMeta(
      titleKey: 'Files',
      icon: Icons.folder_outlined,
      route: '/browser/files',
      builder: (_) => const FilesPage(),
    ),
    builtinTasks: _BuiltinMeta(
      titleKey: 'Tasks',
      icon: Icons.check_box_outlined,
      route: '/browser/tasks',
      builder: (_) => const TasksPage(),
    ),
    builtinSourceControl: _BuiltinMeta(
      titleKey: 'Source Control',
      icon: Icons.account_tree_outlined,
      route: '/browser/source-control',
      builder: (_) => const SourceControlPage(),
    ),
    builtinDatabase: _BuiltinMeta(
      titleKey: 'PostgreSQL',
      icon: Icons.dns_outlined,
      route: '/browser/database',
      builder: (_) => const PGPage(),
    ),
    builtinLogs: _BuiltinMeta(
      titleKey: 'Logs',
      icon: Icons.article_outlined,
      route: '/browser/logs',
      builder: (_) => const LogsPage(),
    ),
    builtinMessaging: _BuiltinMeta(
      titleKey: 'Messaging',
      icon: Icons.forum_outlined,
      route: '/browser/messaging',
      builder: (_) => const TelegramPage(),
    ),
    builtinMcp: _BuiltinMeta(
      titleKey: 'MCP Servers',
      icon: Icons.hub_outlined,
      route: '/browser/mcp',
      builder: (_) => const MCPPage(),
    ),
    builtinPreview: _BuiltinMeta(
      titleKey: 'Preview',
      icon: Icons.public_outlined,
      route: '/browser/preview',
      builder: (_) => const PreviewPage(categoryFilter: 'preview'),
    ),
    builtinSimulator: _BuiltinMeta(
      titleKey: 'Simulator',
      icon: Icons.smartphone_outlined,
      route: '/browser/simulator',
      builder: (_) => const PreviewPage(categoryFilter: 'simulator'),
    ),
  };
}

class _BuiltinMeta {
  final String titleKey;
  final IconData icon;
  final String route;
  final Widget Function(BuildContext) builder;
  const _BuiltinMeta({
    required this.titleKey,
    required this.icon,
    required this.route,
    required this.builder,
  });
}
