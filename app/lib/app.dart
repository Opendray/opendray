import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'core/api/api_client.dart';
import 'core/services/auth_service.dart';
import 'core/services/l10n.dart';
import 'core/services/server_config.dart';
import 'shared/theme/app_theme.dart';
import 'features/auth/login_page.dart';
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
import 'features/setup/setup_wizard.dart';
import 'features/tasks/tasks_page.dart';

class OpendrayApp extends StatefulWidget {
  final ServerConfig serverConfig;
  final L10n l10n;
  final AuthService authService;

  const OpendrayApp({
    super.key,
    required this.serverConfig,
    required this.l10n,
    required this.authService,
  });

  @override
  State<OpendrayApp> createState() => _OpendrayAppState();
}

class _OpendrayAppState extends State<OpendrayApp> {
  late final GoRouter _router;

  @override
  void initState() {
    super.initState();
    _router = _buildRouter(widget.serverConfig, widget.authService);
  }

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider<ServerConfig>.value(value: widget.serverConfig),
        ChangeNotifierProvider<L10n>.value(value: widget.l10n),
        ChangeNotifierProvider<AuthService>.value(value: widget.authService),
      ],
      // Rebuild ApiClient whenever the server URL, CF Access headers, or
      // bearer token change, so every screen picks up the right identity.
      child: Consumer2<ServerConfig, AuthService>(
        builder: (context, config, auth, _) {
          final apiClient = ApiClient(
            baseUrl: config.effectiveUrl,
            extraHeaders: config.cfAccessHeaders,
            tokenProvider: () => auth.token ?? '',
            onUnauthorized: () {
              // Server says our token is dead — log out, router redirect
              // pushes us to /login.
              auth.logout();
            },
          );
          return Provider<ApiClient>.value(
            value: apiClient,
            child: MaterialApp.router(
              title: 'OpenDray',
              theme: buildAppTheme(),
              debugShowCheckedModeBanner: false,
              routerConfig: _router,
            ),
          );
        },
      ),
    );
  }
}

/// Merges multiple Listenable sources into one refreshListenable so
/// GoRouter re-evaluates its redirect whenever any of them fires.
class _RouterListenable extends ChangeNotifier {
  _RouterListenable(List<Listenable> sources) {
    for (final s in sources) {
      s.addListener(notifyListeners);
    }
  }
}

GoRouter _buildRouter(ServerConfig serverConfig, AuthService authService) {
  return GoRouter(
    initialLocation: '/',
    refreshListenable: _RouterListenable([serverConfig, authService]),
    redirect: (context, state) {
      final loc = state.matchedLocation;

      // 1. Phone / web client doesn't know where the server lives yet.
      //    This is the "enter a URL" prompt, only reachable on mobile
      //    where the app doesn't auto-detect the origin.
      if (!serverConfig.isConfigured) {
        return loc == '/connect' ? null : '/connect';
      }

      final s = authService.state;

      // 2. First probe hasn't resolved yet — don't redirect, let the
      //    current route render; AuthService.probe() fires shortly and
      //    the refreshListenable re-enters this function.
      if (s == AuthState.unknown) return null;

      // 3. The server itself hasn't been set up yet → first-run wizard.
      if (s == AuthState.setupRequired) {
        return loc == '/setup' ? null : '/setup';
      }

      // 4. Auth required but not logged in → /login.
      if (s == AuthState.unauthed) {
        return loc == '/login' ? null : '/login';
      }

      // 5. Authed (or auth disabled on server): bounce away from any
      //    gate route if we somehow land there.
      if (loc == '/login' || loc == '/setup' || loc == '/connect') return '/';
      return null;
    },
    routes: [
      GoRoute(path: '/connect', builder: (_, _) => const SetupPage()),
      GoRoute(path: '/setup',   builder: (_, _) => const SetupWizardPage()),
      GoRoute(path: '/login',   builder: (_, _) => const LoginPage()),
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
}

// Panels opened from the launcher. Titles look up via L10n, so the same
// key ("Docs", "Files", …) already used on the launcher card is reused.
Widget _panelShell(BuildContext ctx, String titleKey, Widget child) {
  return Scaffold(
    appBar: AppBar(title: Text(ctx.tr(titleKey))),
    body: child,
  );
}

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
