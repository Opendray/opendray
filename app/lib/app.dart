import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'core/api/api_client.dart';
import 'core/services/l10n.dart';
import 'core/services/server_config.dart';
import 'shared/theme/app_theme.dart';
import 'features/dashboard/dashboard_page.dart';
import 'features/session/session_page.dart';
import 'features/browser/browser_page.dart';
import 'features/browser/preview_page.dart';
import 'features/claude_accounts/claude_accounts_page.dart';
import 'features/database/database_page.dart';
import 'features/docs/docs_page.dart';
import 'features/endpoints/endpoints_page.dart';
import 'features/files/files_page.dart';
import 'features/git/git_page.dart';
import 'features/logs/logs_page.dart';
import 'features/mcp/mcp_page.dart';
import 'features/messaging/telegram_page.dart';
import 'features/settings/settings_page.dart';
import 'features/settings/setup_page.dart';
import 'features/tasks/tasks_page.dart';

class NtcApp extends StatelessWidget {
  final ServerConfig serverConfig;
  final L10n l10n;

  const NtcApp({super.key, required this.serverConfig, required this.l10n});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider<ServerConfig>.value(value: serverConfig),
        ChangeNotifierProvider<L10n>.value(value: l10n),
      ],
      child: Consumer<ServerConfig>(
        builder: (context, config, _) {
          final apiClient = ApiClient(
            baseUrl: config.effectiveUrl,
            extraHeaders: config.cfAccessHeaders,
          );
          return Provider<ApiClient>.value(
            value: apiClient,
            child: MaterialApp.router(
              title: 'NTC',
              theme: buildAppTheme(),
              debugShowCheckedModeBanner: false,
              routerConfig: config.isConfigured ? _mainRouter : _setupRouter,
            ),
          );
        },
      ),
    );
  }
}

// Setup flow — shown when server URL is not configured (mobile only)
final _setupRouter = GoRouter(
  routes: [GoRoute(path: '/', builder: (_, _) => const SetupPage())],
);

// Panels opened from the launcher. Titles look up via L10n, so the same
// key ("Docs", "Files", …) already used on the launcher card is reused.
Widget _panelShell(BuildContext ctx, String titleKey, Widget child) {
  return Scaffold(
    appBar: AppBar(title: Text(ctx.tr(titleKey))),
    body: child,
  );
}

// Main app
final _mainRouter = GoRouter(
  initialLocation: '/',
  routes: [
    ShellRoute(
      builder: (context, state, child) => _Shell(child: child),
      routes: [
        GoRoute(path: '/',        builder: (_, _) => const DashboardPage()),
        GoRoute(path: '/browser', builder: (_, _) => const BrowserPage()),
        GoRoute(
          path: '/browser/docs',
          builder: (ctx, _) => _panelShell(ctx, 'Docs', const DocsPage()),
        ),
        GoRoute(
          path: '/browser/files',
          builder: (ctx, _) => _panelShell(ctx, 'Files', const FilesPage()),
        ),
        GoRoute(
          path: '/browser/database',
          builder: (ctx, _) => _panelShell(ctx, 'Database', const DatabasePage()),
        ),
        GoRoute(
          path: '/browser/tasks',
          builder: (ctx, _) => _panelShell(ctx, 'Tasks', const TasksPage()),
        ),
        GoRoute(
          path: '/browser/git',
          builder: (ctx, _) => _panelShell(ctx, 'Git', const GitPage()),
        ),
        GoRoute(
          path: '/browser/logs',
          builder: (ctx, _) => _panelShell(ctx, 'Logs', const LogsPage()),
        ),
        GoRoute(
          path: '/browser/messaging',
          builder: (ctx, _) => _panelShell(ctx, 'Messaging', const TelegramPage()),
        ),
        GoRoute(
          path: '/browser/mcp',
          builder: (ctx, _) => _panelShell(ctx, 'MCP Servers', const MCPPage()),
        ),
        GoRoute(
          path: '/browser/preview',
          builder: (ctx, _) => _panelShell(ctx, 'Preview',
              const PreviewPage(categoryFilter: 'preview')),
        ),
        GoRoute(
          path: '/browser/simulator',
          builder: (ctx, _) => _panelShell(ctx, 'Simulator',
              const PreviewPage(categoryFilter: 'simulator')),
        ),
        GoRoute(
          path: '/browser/endpoints',
          builder: (ctx, _) => _panelShell(ctx, 'LLM Providers', const EndpointsPage()),
        ),
        GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
        GoRoute(
          path: '/settings/claude-accounts',
          builder: (ctx, _) => _panelShell(ctx, 'Claude Accounts', const ClaudeAccountsPage()),
        ),
      ],
    ),
    GoRoute(
      path: '/session/:id',
      builder: (_, state) => SessionPage(sessionId: state.pathParameters['id']!),
    ),
  ],
);

class _Shell extends StatelessWidget {
  final Widget child;
  const _Shell({required this.child});

  @override
  Widget build(BuildContext context) {
    final location = GoRouterState.of(context).uri.path;
    // Any /browser* path (launcher and every sub-panel) lights up tab 1.
    final int index;
    if (location == '/settings') {
      index = 2;
    } else if (location.startsWith('/browser')) {
      index = 1;
    } else {
      index = 0;
    }

    return Scaffold(
      body: child,
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: index,
        onTap: (i) {
          final path = switch (i) { 1 => '/browser', 2 => '/settings', _ => '/' };
          context.go(path);
        },
        items: [
          BottomNavigationBarItem(
              icon: const Icon(Icons.terminal), label: context.tr('Sessions')),
          BottomNavigationBarItem(
              icon: const Icon(Icons.folder_copy), label: context.tr('Browser')),
          BottomNavigationBarItem(
              icon: const Icon(Icons.settings), label: context.tr('Settings')),
        ],
      ),
    );
  }
}
