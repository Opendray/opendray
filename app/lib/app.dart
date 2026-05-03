import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'core/api/api_client.dart';
import 'core/services/auth_service.dart';
import 'core/services/l10n.dart';
import 'core/services/server_config.dart';
import 'shared/theme/app_theme.dart';
import 'shared/theme/responsive.dart';
import 'features/auth/login_page.dart';
import 'features/dashboard/dashboard_page.dart';
import 'features/session/session_page.dart';
import 'features/claude_accounts/claude_accounts_page.dart';
import 'features/endpoints/endpoints_page.dart';
import 'features/hub/hub_page.dart';
import 'features/hub_v1/hub_v1_page.dart';
import 'shared/v1_app_shell.dart';
import 'features/plugins/plugins_page.dart';
import 'features/plugins/plugins_v1_page.dart';
import 'features/settings/builtin_restore_page.dart';
import 'features/settings/settings_page.dart';
import 'features/settings/setup_page.dart';
import 'features/workbench/command_palette.dart';
import 'features/workbench/keybindings.dart';
import 'features/workbench/running/plugin_registry.dart';
import 'features/workbench/running/plugin_thumbnail_capture.dart';
import 'features/workbench/running/plugin_thumbnail_js_fallback.dart';
import 'features/workbench/running/running_plugins_host.dart';
import 'features/workbench/running/running_plugins_models.dart';
import 'features/workbench/running/running_plugins_service.dart';
import 'features/workbench/running/running_plugins_switcher_page.dart';
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
    // Wire the webview JS snapshot path so the running-plugins host
    // can fall back to an in-page capture when RepaintBoundary produces
    // a blank frame over an iOS WKWebView. One-time install; the
    // capture module holds it as a static field.
    PluginThumbnailCapture.webviewJsFallback = webviewJsThumbnailFallback;
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
                // unknown/unauthed states the workbench endpoints 401,
                // which triggers auth.logout → Consumer2 rebuild →
                // new ApiClient → new WorkbenchService → retry,
                // producing an infinite loop at startup.
                if (auth.state == AuthState.authed ||
                    auth.state == AuthState.disabled) {
                  svc
                    ..refresh()
                    ..startListening();
                }
                return svc;
              },
              child: ChangeNotifierProvider<RunningPluginsService>(
                create: (_) => RunningPluginsService(),
                child: MaterialApp.router(
                  title: 'OpenDray',
                  theme: buildAppTheme(),
                  debugShowCheckedModeBanner: false,
                  scaffoldMessengerKey: _scaffoldMessengerKey,
                  routerConfig: _router,
                  builder: (ctx, child) => _WebDesktopThemeScope(
                    child: _WorkbenchRoot(
                      service: ctx.read<WorkbenchService>(),
                      child: child ?? const SizedBox.shrink(),
                    ),
                  ),
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

      // 3. Auth required but not logged in → /login.
      if (s == AuthState.unauthed) {
        return loc == '/login' ? null : '/login';
      }

      // 4. Authed (or auth disabled on server): bounce away from any
      //    gate route if we somehow land there. Setup is a terminal-
      //    only flow now (`opendray setup`), so the Flutter app never
      //    routes to /setup.
      if (loc == '/login' || loc == '/connect') return '/';
      return null;
    },
    routes: [
      GoRoute(path: '/connect', builder: (_, _) => const SetupPage()),
      GoRoute(path: '/login',   builder: (_, _) => const LoginPage()),
      ShellRoute(
        builder: (context, state, child) => _Shell(child: child),
        routes: [
          GoRoute(path: '/',                  builder: (_, _) => const HubV1Page()),
          GoRoute(path: '/dashboard-classic', builder: (_, _) => const DashboardPage()),
          // All `/browser/*` routes return a sentinel — the real
          // rendering happens in `_Shell` via [RunningPluginsHost]'s
          // IndexedStack, which keeps every opened plugin mounted
          // across navigation. Route builders exist only so GoRouter
          // matches the path; [_Shell] picks up `location` and drives
          // mount / activation from there.
          GoRoute(path: '/browser/docs', builder: _sentinel),
          GoRoute(path: '/browser/files', builder: _sentinel),
          GoRoute(path: '/browser/tasks', builder: _sentinel),
          GoRoute(path: '/browser/source-control', builder: _sentinel),
          GoRoute(path: '/browser/database', builder: _sentinel),
          GoRoute(path: '/browser/logs', builder: _sentinel),
          GoRoute(path: '/browser/messaging', builder: _sentinel),
          GoRoute(path: '/browser/mcp', builder: _sentinel),
          GoRoute(path: '/browser/preview', builder: _sentinel),
          GoRoute(path: '/browser/simulator', builder: _sentinel),
          // Generic v1 webview plugin route — same sentinel pattern.
          GoRoute(path: '/browser/plugin/:name', builder: _sentinel),
          GoRoute(path: '/plugins',         builder: (_, _) => const PluginsV1Page()),
          GoRoute(path: '/plugins-classic', builder: (_, _) => const PluginsPage()),
          GoRoute(path: '/hub', builder: (_, _) => const HubPage()),
          GoRoute(path: '/hub-v1', builder: (_, _) => const HubV1Page()),
          GoRoute(
            path: '/running',
            builder: (_, _) => const RunningPluginsSwitcherPage(),
          ),
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

/// Sentinel builder for plugin routes — the actual page renders from
/// `_Shell.body` via RunningPluginsHost. The SizedBox is never
/// visible: `_Shell` detects the plugin route, shows the IndexedStack
/// directly, and ignores this child. Keeping the route registered
/// is what lets GoRouter navigate to the path; the widget returned is
/// just placeholder tissue.
Widget _sentinel(BuildContext context, GoRouterState state) =>
    const SizedBox.shrink();

// Panels opened from Settings — plain wrapper. Source-of-truth for
// plugin-owned panels is PluginRegistry.
Widget _panelShell(BuildContext ctx, String titleKey, Widget child) {
  return Scaffold(
    appBar: AppBar(title: Text(ctx.tr(titleKey))),
    body: child,
  );
}

/// Shell wrapper around every routed page. On narrow viewports (phones,
/// skinny browser windows) it keeps the bottom-nav layout the mobile app
/// has always used. On wide viewports (desktop web) it swaps in a left
/// [NavigationRail] so the UI stops feeling like a blown-up phone screen.
///
/// No top AppBar is added here — every routed page owns its own AppBar
/// and would clash with a shell-level one.
class _Shell extends StatelessWidget {
  final Widget child;
  const _Shell({required this.child});

  /// Below this width we render BottomNavigationBar (phone layout).
  /// Above it we render NavigationRail (desktop layout).
  static const double _railBreakpoint = 900;

  @override
  Widget build(BuildContext context) {
    final location = GoRouterState.of(context).uri.path;
    // Tab layout (Running slot lives after Plugin):
    //   Hub hidden   → Sessions | Plugin | Running | Settings           (0..3)
    //   Hub enabled  → Sessions | Plugin | Hub | Running | Settings     (0..4)
    final int runningIndex = kHubEnabled ? 3 : 2;
    final int settingsIndex = kHubEnabled ? 4 : 3;
    final int index;
    if (location == '/settings' || location.startsWith('/settings/')) {
      index = settingsIndex;
    } else if (location == '/running') {
      index = runningIndex;
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

    String pathForIndex(int i) {
      return kHubEnabled
          ? switch (i) {
              1 => '/plugins',
              2 => '/hub',
              3 => '/running',
              4 => '/settings',
              _ => '/',
            }
          : switch (i) {
              1 => '/plugins',
              2 => '/running',
              3 => '/settings',
              _ => '/',
            };
    }

    final runningCount =
        context.watch<RunningPluginsService>().entries.length;
    final runningLabel = runningCount > 0
        ? '${context.tr('Running')} ($runningCount)'
        : context.tr('Running');

    final destinations = <_NavDest>[
      _NavDest(icon: Icons.terminal, label: context.tr('Sessions')),
      _NavDest(icon: Icons.extension_outlined, label: context.tr('Plugin')),
      if (kHubEnabled)
        _NavDest(icon: Icons.storefront, label: context.tr('Hub')),
      _NavDest(icon: Icons.layers_outlined, label: runningLabel),
      _NavDest(icon: Icons.settings, label: context.tr('Settings')),
    ];

    // Compute what the running host needs: whether we're on a plugin
    // route, and — if so — which plugin id to focus. Mount and active
    // updates are scheduled post-frame so we don't call
    // notifyListeners during build.
    final bool isPluginRoute = location.startsWith('/browser/');
    String? targetPluginId;
    RunningPluginEntry? seed;
    if (isPluginRoute) {
      // Built-in panel: resolve directly from route → id.
      final builtinId = PluginRegistry.builtinIdForRoute(location);
      if (builtinId != null) {
        targetPluginId = builtinId;
        seed = PluginRegistry.builtinSeed(builtinId);
      } else {
        // Webview plugin: /browser/plugin/<name>. Resolve against the
        // live WorkbenchService view catalog so the seed carries the
        // entry path for PluginWebView.
        const prefix = '/browser/plugin/';
        if (location.startsWith(prefix)) {
          final name = location.substring(prefix.length);
          if (name.isNotEmpty) {
            targetPluginId = 'webview:$name';
            final workbench = context.read<WorkbenchService>();
            final api = context.read<ApiClient>();
            for (final v in workbench.views) {
              if (v.pluginName == name && v.render == 'webview') {
                seed = PluginRegistry.webviewSeed(view: v, api: api);
                break;
              }
            }
          }
        }
      }
    }

    final runningService = context.read<RunningPluginsService>();
    if (isPluginRoute && seed != null && targetPluginId != null) {
      final sid = targetPluginId;
      final sseed = seed;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        runningService.ensureOpened(sseed);
        runningService.setActive(sid);
      });
    } else if (!isPluginRoute) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        runningService.clearActive();
      });
    }

    final Widget bodyWithHost = RunningPluginsHost(
      isPluginRoute: isPluginRoute,
      targetPluginId: targetPluginId,
      nonPluginChild: child,
    );

    return LayoutBuilder(
      builder: (context, constraints) {
        final isWide = constraints.maxWidth >= _railBreakpoint;
        if (!isWide) {
          return Scaffold(
            body: bodyWithHost,
            bottomNavigationBar: BottomNavigationBar(
              type: BottomNavigationBarType.fixed,
              currentIndex: index,
              onTap: (i) => context.go(pathForIndex(i)),
              items: [
                for (final d in destinations)
                  BottomNavigationBarItem(
                    icon: Icon(d.icon),
                    label: d.label,
                  ),
              ],
            ),
          );
        }

        // V1 app shell — sidebar + top bar from the prototype design.
        // Wraps every routed page so the whole app feels cohesive even
        // before each inner page is rebuilt against the prototype layout.
        final sections = _v1NavSections(runningCount: runningCount);
        final crumb = _crumbForRoute(location);
        return V1AppShell(
          sections: sections,
          currentRoute: location,
          breadcrumbTitle: crumb,
          workspaceName: 'Workspace',
          userInitials: 'OD',
          userName: 'Signed in',
          userEmail: '',
          child: bodyWithHost,
        );
      },
    );
  }
}

class _NavDest {
  final IconData icon;
  final String label;
  const _NavDest({required this.icon, required this.label});
}

/// V1 sidebar nav sections. Maps the prototype's nav items to existing
/// routes; surfaces we haven't built yet are flagged comingSoon=true so
/// they show a "coming soon" toast instead of routing into a 404.
List<V1NavSection> _v1NavSections({required int runningCount}) {
  return [
    V1NavSection(label: 'Workspace', items: [
      const V1NavItem(icon: Icons.dashboard_outlined, label: 'Hub', route: '/'),
      V1NavItem(
        icon: Icons.terminal,
        label: 'Workbench',
        route: '/running',
        badgeCount: runningCount > 0 ? runningCount : null,
      ),
      const V1NavItem(
        icon: Icons.extension_outlined,
        label: 'Plugins',
        route: '/plugins',
      ),
      const V1NavItem(
        icon: Icons.link,
        label: 'Connections',
        route: '/settings/llm-endpoints',
      ),
      const V1NavItem(
        icon: Icons.folder_outlined,
        label: 'Files',
        route: '/browser/files',
      ),
      const V1NavItem(
        icon: Icons.notes_outlined,
        label: 'Logs',
        route: '/browser/logs',
      ),
    ]),
    const V1NavSection(label: 'Admin', items: [
      V1NavItem(
        icon: Icons.group_outlined,
        label: 'Team & roles',
        route: '/settings',
        comingSoon: true,
      ),
      V1NavItem(
        icon: Icons.fact_check_outlined,
        label: 'Audit log',
        route: '/settings',
        comingSoon: true,
      ),
      V1NavItem(
        icon: Icons.settings_outlined,
        label: 'Settings',
        route: '/settings',
      ),
    ]),
  ];
}

String _crumbForRoute(String loc) {
  if (loc == '/') return 'Hub';
  if (loc == '/dashboard-classic') return 'Dashboard (classic)';
  if (loc == '/hub-v1') return 'Hub';
  if (loc == '/hub') return 'Hub (legacy)';
  if (loc == '/plugins' || loc.startsWith('/plugins/')) return 'Plugins';
  if (loc == '/plugins-classic') return 'Plugins (classic)';
  if (loc == '/running') return 'Workbench';
  if (loc == '/settings/claude-accounts') return 'Connections · Accounts';
  if (loc == '/settings/llm-endpoints') return 'Connections · Endpoints';
  if (loc == '/settings/builtin-plugins') return 'Settings · Built-in plugins';
  if (loc.startsWith('/settings')) return 'Settings';
  if (loc == '/browser/files') return 'Files';
  if (loc == '/browser/logs') return 'Logs';
  if (loc == '/browser/messaging') return 'Connections · Telegram';
  if (loc == '/browser/mcp') return 'Connections · MCP';
  if (loc.startsWith('/browser/')) {
    final tail = loc.substring('/browser/'.length);
    return tail.isEmpty ? 'Browser' : tail;
  }
  return loc;
}

/// Bumps [MediaQueryData.textScaler] on desktop web so every `Text`
/// widget — including the ones that hard-code `fontSize: 11/12/13`
/// constants and don't inherit from [TextTheme] — reads at a size
/// that's comfortable from a normal desk distance.
///
/// This is the **correct** knob for "make text readable on a 27"
/// monitor": `Theme.textTheme` only reaches widgets that read it
/// (which misses ~60 hard-coded Text sites in this codebase), whereas
/// the textScaler is applied by every `RichText` / `Text` layout pass
/// regardless of whether the style came from the theme or a literal.
///
/// On mobile (iOS/Android) this widget is a pass-through — phone
/// builds are unchanged.
class _WebDesktopThemeScope extends StatelessWidget {
  final Widget child;
  const _WebDesktopThemeScope({required this.child});

  @override
  Widget build(BuildContext context) {
    if (!Responsive.isDesktopWeb(context)) return child;

    final scale = Responsive.fontScale(context);
    final current = MediaQuery.of(context);
    return MediaQuery(
      data: current.copyWith(textScaler: TextScaler.linear(scale)),
      child: child,
    );
  }
}
