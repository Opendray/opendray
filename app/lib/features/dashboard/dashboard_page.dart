import 'dart:async';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../core/api/api_client.dart';
import '../../core/models/session.dart';
import '../../core/services/auth_service.dart';
import '../../shared/session_launcher.dart';
import '../../shared/theme/app_theme.dart';
import '../browser/preview_page.dart';
import '../docs/docs_page.dart';
import '../files/files_page.dart';
import '../pg/pg_page.dart';
import '../source_control/source_control_page.dart';
import '../logs/logs_page.dart';
import '../mcp/mcp_page.dart';
import '../messaging/telegram_page.dart';
import '../tasks/tasks_page.dart';
import '../workbench/menu_slot.dart';
import '../workbench/status_bar_strip.dart';
import '../workbench/view_host.dart';
import '../workbench/workbench_service.dart';
import '../workbench/workbench_sources.dart';
import 'widgets/session_card.dart';

/// Legacy panel bridge: maps the viewId compat.Synthesize emits for each
/// pre-M2 panel plugin to the existing bespoke Flutter page. When a user
/// taps a panel icon in the activity bar, ViewHost calls the matching
/// builder so the page renders inline inside the dashboard instead of
/// showing the "Declarative views arrive in M5" placeholder.
///
/// Keys must match the plugin name in `/opt/opendray/plugins/panels/*/
/// manifest.json` one-to-one — compat uses the manifest name as both the
/// view id AND the activityBar item id.
Map<String, WidgetBuilder> get _legacyPanelBuilders => <String, WidgetBuilder>{
      // 'database' retired — replaced by pg-browser v1 plugin installed
      // from the Hub (plugin/marketplace/packages/pg-browser/).
      'file-browser': (_) => const FilesPage(),
      'source-control': (_) => const SourceControlPage(),
      'pg-browser': (_) => const PGPage(),
      'log-viewer': (_) => const LogsPage(),
      'mcp': (_) => const MCPPage(),
      'obsidian-reader': (_) => const DocsPage(),
      'simulator-preview': (_) =>
          const PreviewPage(categoryFilter: 'simulator'),
      'task-runner': (_) => const TasksPage(),
      'telegram': (_) => const TelegramPage(),
      'web-browser': (_) => const PreviewPage(categoryFilter: 'preview'),
    };

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key});
  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  List<Session> _sessions = [];
  bool _loading = true;
  String? _error;
  Timer? _pollTimer;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
    _pollTimer = Timer.periodic(const Duration(seconds: 5), (_) => _load());
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final sessions = await _api.listSessions();
      if (mounted) setState(() { _sessions = sessions; _loading = false; _error = null; });
    } catch (e) {
      if (mounted) setState(() { _loading = false; _error = e.toString(); });
    }
  }

  Future<void> _showCreateDialog() => launchNewSession(context);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Row(
          children: [
            Container(
              width: 28, height: 28,
              decoration: BoxDecoration(color: AppColors.accent, borderRadius: BorderRadius.circular(7)),
              child: const Icon(Icons.terminal_rounded, color: Colors.white, size: 18),
            ),
            const SizedBox(width: 10),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('OpenDray', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                Text('${_sessions.length} sessions', style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
              ],
            ),
          ],
        ),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 4),
            child: TextButton.icon(
              onPressed: () => context.go('/'),
              icon: const Icon(Icons.arrow_back, size: 14),
              label: const Text('Back to new Hub', style: TextStyle(fontSize: 12)),
            ),
          ),
          Padding(
            padding: const EdgeInsets.only(right: 4),
            child: FilledButton.icon(
              onPressed: _showCreateDialog,
              icon: const Icon(Icons.add, size: 16),
              label: const Text('New', style: TextStyle(fontSize: 13)),
              style: FilledButton.styleFrom(
                backgroundColor: AppColors.accent,
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
              ),
            ),
          ),
          // T22 — plugin-contributed menu entries (e.g. time-ninja's
          // `time.start`). Collapses to SizedBox.shrink when no plugin
          // has contributed to appBar/right, so this adds zero chrome
          // until a plugin registers a menu contribution.
          const _AppBarMenuSlot(),
          const SizedBox(width: 8),
        ],
      ),
      // Plugin views (opened via workbench.openView from a plugin's
      // command or menu entry) replace the session list when focused.
      // With no view focused, the dashboard shows its usual body. The
      // activity bar rail was removed — plugin discovery now lives under
      // /plugins (install/manage) and the launcher cards on /browser.
      body: _DashboardViewHost(fallback: _buildMainBody()),
      // T20 footer — renders nothing until a plugin contributes a status-bar
      // item. Backed by the real WorkbenchService via an adapter held in
      // _DashboardStatusBar so the adapter's listener lifecycle is bound
      // to a stable State (no leak on every Scaffold rebuild).
      bottomNavigationBar: const _DashboardStatusBar(),
    );
  }

  /// The dashboard's normal body — extracted so [ViewHost] can use it as
  /// a fallback while still letting plugin views replace it when focused.
  Widget _buildMainBody() {
    if (_loading && _sessions.isEmpty) {
      return const Center(
        child: CircularProgressIndicator(color: AppColors.accent),
      );
    }
    if (_error != null && _sessions.isEmpty) return _buildOfflineState();
    if (_sessions.isEmpty) return _buildEmpty();
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: _sessions.length,
        separatorBuilder: (_, _) => const SizedBox(height: 10),
        itemBuilder: (_, i) => SessionCard(
          session: _sessions[i],
          onTap: () => context.push('/session/${_sessions[i].id}'),
          onStart: () async { await _api.startSession(_sessions[i].id); _load(); },
          onStop: () async { await _api.stopSession(_sessions[i].id); _load(); },
          onDelete: () async { await _api.deleteSession(_sessions[i].id); _load(); },
        ),
      ),
    );
  }

  /// Offline / "can't reach backend" state. Beyond the "Retry" button we
  /// expose explicit escape hatches so the user is never trapped: go to
  /// Settings (fix server URL), or sign out (drop a dead token and return
  /// to /login under a different account).
  Widget _buildOfflineState() {
    final auth = context.watch<AuthService>();
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.cloud_off, color: AppColors.error, size: 40),
          const SizedBox(height: 12),
          const Text('Cannot connect to server',
              style: TextStyle(fontWeight: FontWeight.w500)),
          const SizedBox(height: 4),
          const Text('Check Settings → Server URL, or sign in again',
              style: TextStyle(color: AppColors.textMuted, fontSize: 12),
              textAlign: TextAlign.center),
          const SizedBox(height: 18),
          FilledButton.icon(
            onPressed: _load,
            style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
            icon: const Icon(Icons.refresh, size: 16),
            label: const Text('Retry'),
          ),
          const SizedBox(height: 10),
          Wrap(
            spacing: 8,
            alignment: WrapAlignment.center,
            children: [
              OutlinedButton.icon(
                onPressed: () => context.go('/settings'),
                icon: const Icon(Icons.settings, size: 16),
                label: const Text('Settings'),
              ),
              if (auth.hasStoredToken)
                OutlinedButton.icon(
                  onPressed: () async {
                    await context.read<AuthService>().logout();
                    // Router redirect picks up the state change and sends
                    // us to /login automatically.
                  },
                  style: OutlinedButton.styleFrom(
                      foregroundColor: AppColors.error),
                  icon: const Icon(Icons.logout, size: 16),
                  label: const Text('Sign out'),
                ),
            ],
          ),
        ]),
      ),
    );
  }

  Widget _buildEmpty() {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 64, height: 64,
            decoration: BoxDecoration(color: AppColors.surfaceAlt, borderRadius: BorderRadius.circular(16)),
            child: const Icon(Icons.terminal, size: 32, color: AppColors.textMuted),
          ),
          const SizedBox(height: 16),
          const Text('No sessions', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w500)),
          const SizedBox(height: 4),
          const Text('Create a session to start', style: TextStyle(color: AppColors.textMuted, fontSize: 13)),
          const SizedBox(height: 20),
          FilledButton(
            onPressed: _showCreateDialog,
            style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
            child: const Text('Create Session'),
          ),
        ],
      ),
    );
  }
}

/// Holds a [WorkbenchMenuSource] tied to this State's lifecycle so the
/// adapter's forwarding listener is registered exactly once — not per
/// Scaffold rebuild — and released on dispose. Keeps app.dart untouched
/// (that file is frozen for this task) while avoiding a listener leak.
class _AppBarMenuSlot extends StatefulWidget {
  const _AppBarMenuSlot();

  @override
  State<_AppBarMenuSlot> createState() => _AppBarMenuSlotState();
}

class _AppBarMenuSlotState extends State<_AppBarMenuSlot> {
  late final WorkbenchMenuSource _source =
      WorkbenchMenuSource(context.read<WorkbenchService>());

  @override
  void dispose() {
    _source.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) =>
      MenuSlot(id: 'appBar/right', source: _source);
}

/// Holds a [WorkbenchStatusBarSource] for the dashboard footer. Same
/// pattern as [_AppBarMenuSlot] — State owns the adapter so disposal
/// removes the listener from the underlying service cleanly.
class _DashboardStatusBar extends StatefulWidget {
  const _DashboardStatusBar();

  @override
  State<_DashboardStatusBar> createState() => _DashboardStatusBarState();
}

class _DashboardStatusBarState extends State<_DashboardStatusBar> {
  late final WorkbenchStatusBarSource _source =
      WorkbenchStatusBarSource(context.read<WorkbenchService>());

  @override
  void dispose() {
    _source.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => StatusBarStrip(source: _source);
}

/// T18 — wraps [ViewHost] and pulls `baseUrl` + `bearerToken` from the
/// ambient [ApiClient]. A thin adapter so dashboard_page keeps its
/// surgical footprint (one Widget, not a chunk of inlined setup).
class _DashboardViewHost extends StatelessWidget {
  const _DashboardViewHost({required this.fallback});

  final Widget fallback;

  @override
  Widget build(BuildContext context) {
    final api = context.read<ApiClient>();
    final service = context.read<WorkbenchService>();
    return ViewHost(
      service: service,
      baseUrl: api.baseUrl,
      bearerToken: api.token ?? '',
      fallback: fallback,
      legacyPanelBuilders: _legacyPanelBuilders,
    );
  }
}

