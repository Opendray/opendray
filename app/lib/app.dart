import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
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
import 'features/browser/preview_page.dart';
import 'features/claude_accounts/claude_accounts_page.dart';
import 'features/docs/docs_page.dart';
import 'features/endpoints/endpoints_page.dart';
import 'features/forge/forge_page.dart';
import 'features/pg/pg_page.dart';
import 'features/files/files_page.dart';
import 'features/git/git_page.dart';
import 'features/logs/logs_page.dart';
import 'features/mcp/mcp_page.dart';
import 'features/messaging/telegram_page.dart';
import 'features/hub/hub_page.dart';
import 'features/plugins/plugins_page.dart';
import 'features/settings/builtin_restore_page.dart';
import 'features/settings/settings_page.dart';
import 'features/settings/setup_page.dart';
import 'features/setup/setup_wizard.dart';
import 'features/tasks/tasks_page.dart';
import 'features/workbench/command_palette.dart';
import 'features/workbench/keybindings.dart';
import 'features/workbench/webview_host.dart';
import 'features/workbench/workbench_models.dart';
import 'features/workbench/workbench_service.dart';

/// Feature flag: show the Hub (third-party marketplace) tab.
///
/// Kept `false` through v1 — the catalog is intentionally empty at
/// launch (see `docs/plugin-platform/M5-RELEASE.md`), so exposing the
/// tab would train users to check a page that never has anything.
/// Flip to `true` once:
///   1. marketplace.opendray.dev DNS is live (or the syz mock has at
///      least one genuinely third-party bundle),
///   2. the publisher CLI (M4.2) is unparked so the ecosystem can
///      actually accept submissions.
///
/// The `/hub` route itself stays registered so devs can still reach
/// the page via a typed URL — this flag only controls the bottom-nav
/// entry + tab-index math.
const bool kHubEnabled = false;

class NtcApp extends StatefulWidget {
  final ServerConfig serverConfig;
  final L10n l10n;
  final AuthService authService;

  const NtcApp({
    super.key,
    required this.serverConfig,
    required this.l10n,
    required this.authService,
  });

  @override
  State<NtcApp> createState() => _NtcAppState();
}

class _NtcAppState extends State<NtcApp> {
  late final GoRouter _router;
  // Shared messenger key so WorkbenchService can surface SnackBars
  // from anywhere (command palette invocations, plugin install errors)
  // without needing a BuildContext tied to a specific subtree.
  final GlobalKey<ScaffoldMessengerState> _scaffoldMessengerKey =
      GlobalKey<ScaffoldMessengerState>();

  @override
  void initState() {
    super.initState();
    _router = _buildRouter(widget.serverConfig, widget.authService);
  }

  void _toast(String text, {bool isError = false}) {
    final m = _scaffoldMessengerKey.currentState;
    if (m == null) return;
    m.hideCurrentSnackBar();
    m.showSnackBar(SnackBar(
      content: Text(text),
      backgroundColor: isError ? Colors.red.shade700 : null,
    ));
  }

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider<ServerConfig>.value(value: widget.serverConfig),
        ChangeNotifierProvider<L10n>.value(value: widget.l10n),
        ChangeNotifierProvider<AuthService>.value(value: widget.authService),
      ],
      // Rebuild ApiClient whenever the server URL or bearer token change,
      // so every screen picks up the right identity.
      child: Consumer2<ServerConfig, AuthService>(
        builder: (context, config, auth, _) {
          final apiClient = ApiClient(
            baseUrl: config.effectiveUrl,
            tokenProvider: () => auth.token ?? '',
            onUnauthorized: () {
              // Server says our token is dead — log out, router redirect
              // pushes us to /login.
              auth.logout();
            },
          );
          return Provider<ApiClient>.value(
            value: apiClient,
            // WorkbenchService lives above MaterialApp so pages can
            // context.read it; rebuilt whenever ApiClient identity changes.
            child: ChangeNotifierProvider<WorkbenchService>(
              key: ValueKey(apiClient),
              create: (_) {
                final svc =
                    WorkbenchService(api: apiClient, showMessage: _toast);
                // Only reach out to the server when auth is usable. In
                // unknown/unauthed/setupRequired states the workbench
                // endpoints 401, which triggers auth.logout → Consumer2
                // rebuild → new ApiClient → new WorkbenchService → retry,
                // producing an infinite loop at startup.
                if (auth.state == AuthState.authed ||
                    auth.state == AuthState.disabled) {
                  svc
                    ..refresh()
                    ..startListening();
                }
                return svc;
              },
              child: MaterialApp.router(
                title: 'OpenDray',
                theme: buildAppTheme(),
                debugShowCheckedModeBanner: false,
                scaffoldMessengerKey: _scaffoldMessengerKey,
                routerConfig: _router,
                builder: (ctx, child) => _WorkbenchRoot(
                  service: ctx.read<WorkbenchService>(),
                  child: child ?? const SizedBox.shrink(),
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

/// Intent fired by Cmd/Ctrl+Shift+P to open the command palette.
class _OpenPaletteIntent extends Intent {
  const _OpenPaletteIntent();
}

/// Mounts plugin keybindings (T21) + the Cmd/Ctrl+Shift+P palette
/// shortcut (T19) above every routed page.
class _WorkbenchRoot extends StatelessWidget {
  final Widget child;
  final WorkbenchService service;
  const _WorkbenchRoot({required this.child, required this.service});

  @override
  Widget build(BuildContext context) {
    return WorkbenchKeybindings(
      service: service,
      child: Shortcuts(
        shortcuts: <LogicalKeySet, Intent>{
          LogicalKeySet(LogicalKeyboardKey.control, LogicalKeyboardKey.shift,
              LogicalKeyboardKey.keyP): const _OpenPaletteIntent(),
          LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.shift,
              LogicalKeyboardKey.keyP): const _OpenPaletteIntent(),
        },
        child: Actions(
          actions: <Type, Action<Intent>>{
            _OpenPaletteIntent: CallbackAction<_OpenPaletteIntent>(
              onInvoke: (_) {
                CommandPalette.show(context, service);
                return null;
              },
            ),
          },
          child: child,
        ),
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
          // /browser parent grid is gone — per-plugin surfaces are now
          // opened from /plugins. The /browser/<panel> children below
          // stay because plugins_page._handOpenRoute pushes to them.
          GoRoute(
            path: '/browser/docs',
            builder: (ctx, _) => _panelShell(ctx, 'Docs', const DocsPage()),
          ),
          GoRoute(
            path: '/browser/files',
            builder: (ctx, _) => _panelShell(ctx, 'Files', const FilesPage()),
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
            path: '/browser/forge',
            builder: (ctx, _) =>
                _panelShell(ctx, 'Pull Requests', const ForgePage()),
          ),
          GoRoute(
            path: '/browser/database',
            builder: (ctx, _) =>
                _panelShell(ctx, 'PostgreSQL', const PGPage()),
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
          // Generic v1 webview plugin route. Resolves the first
          // activityBar view owned by the plugin and hosts it in a
          // PluginWebView. Legacy panels still have their dedicated
          // routes above and never hit this path.
          GoRoute(
            path: '/browser/plugin/:name',
            builder: (ctx, state) => _PluginWebRoute(
              pluginName: state.pathParameters['name']!,
            ),
          ),
          GoRoute(path: '/plugins', builder: (_, _) => const PluginsPage()),
          GoRoute(path: '/hub', builder: (_, _) => const HubPage()),
          GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
          GoRoute(
            path: '/settings/claude-accounts',
            builder: (ctx, _) => _panelShell(ctx, 'Claude Accounts', const ClaudeAccountsPage()),
          ),
          GoRoute(
            path: '/settings/llm-endpoints',
            builder: (ctx, _) => _panelShell(
                ctx, 'LLM Endpoints', const EndpointsPage()),
          ),
          GoRoute(
            path: '/settings/builtin-plugins',
            builder: (_, _) => const BuiltinRestorePage(),
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

/// Route shell for `/browser/plugin/:name`. Finds the plugin's first
/// contributed view in the workbench registry and renders it via the
/// right backend — [PluginWebView] for `render:"webview"`. Works for
/// every webview plugin the server exposes without per-plugin routing
/// glue.
class _PluginWebRoute extends StatelessWidget {
  const _PluginWebRoute({required this.pluginName});
  final String pluginName;

  @override
  Widget build(BuildContext context) {
    final service = context.watch<WorkbenchService>();
    final api = context.read<ApiClient>();

    WorkbenchView? view;
    for (final v in service.views) {
      if (v.pluginName == pluginName) {
        view = v;
        break;
      }
    }
    if (view == null) {
      return Scaffold(
        appBar: AppBar(title: Text(pluginName)),
        body: const Center(
          child: Padding(
            padding: EdgeInsets.all(24),
            child: Text(
              'Plugin view not found. The plugin may have been '
              'uninstalled or disabled.',
              textAlign: TextAlign.center,
            ),
          ),
        ),
      );
    }
    final title = view.title.isEmpty ? pluginName : view.title;
    if (view.render == 'webview') {
      return Scaffold(
        appBar: AppBar(title: Text(title)),
        body: PluginWebView(
          pluginName: view.pluginName,
          viewId: view.id,
          entryPath: view.entry,
          baseUrl: api.baseUrl,
          bearerToken: api.token ?? '',
        ),
      );
    }
    // Declarative renderer is M5 work. Show a pointer until then.
    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: const Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Text(
            'Declarative views arrive in M5.',
            textAlign: TextAlign.center,
          ),
        ),
      ),
    );
  }
}

class _Shell extends StatelessWidget {
  final Widget child;
  const _Shell({required this.child});

  @override
  Widget build(BuildContext context) {
    final location = GoRouterState.of(context).uri.path;
    // Tab layout:
    //   Hub enabled  → Sessions | Plugin | Hub | Settings  (indices 0..3)
    //   Hub hidden   → Sessions | Plugin | Settings        (indices 0..2)
    //
    // `/browser/*` highlights the Plugin tab because those pages are
    // plugin-owned surfaces opened from /plugins. The /hub route is
    // still registered even when kHubEnabled is false — a dev can
    // reach it by typing the URL; regular users just don't see the
    // entry point.
    final int settingsIndex = kHubEnabled ? 3 : 2;
    final int index;
    if (location == '/settings' || location.startsWith('/settings/')) {
      index = settingsIndex;
    } else if (kHubEnabled &&
        (location == '/hub' || location.startsWith('/hub/'))) {
      index = 2;
    } else if (location == '/plugins' ||
        location.startsWith('/plugins/') ||
        location.startsWith('/browser')) {
      index = 1;
    } else {
      index = 0;
    }

    return Scaffold(
      body: child,
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        currentIndex: index,
        onTap: (i) {
          final path = kHubEnabled
              ? switch (i) {
                  1 => '/plugins',
                  2 => '/hub',
                  3 => '/settings',
                  _ => '/',
                }
              : switch (i) {
                  1 => '/plugins',
                  2 => '/settings',
                  _ => '/',
                };
          context.go(path);
        },
        items: [
          BottomNavigationBarItem(
              icon: const Icon(Icons.terminal), label: context.tr('Sessions')),
          BottomNavigationBarItem(
              icon: const Icon(Icons.extension_outlined),
              label: context.tr('Plugin')),
          if (kHubEnabled)
            BottomNavigationBarItem(
                icon: const Icon(Icons.storefront), label: context.tr('Hub')),
          BottomNavigationBarItem(
              icon: const Icon(Icons.settings), label: context.tr('Settings')),
        ],
      ),
    );
  }
}
