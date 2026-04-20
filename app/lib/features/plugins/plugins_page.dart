import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart' as provider_model;
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';
import '../settings/plugin_consents_page.dart';
import 'plugin_configure_page.dart';
import 'plugin_run_page.dart';

/// Installed-plugins management at `/plugins`.
///
/// One screen, one job: CRUD over whatever `/api/providers` says is
/// currently installed — enable/disable, configure, view permissions,
/// uninstall (unless the manifest marks the plugin required).
///
/// Discovery/install of new plugins lives at `/hub`. Keeping the two
/// surfaces apart avoids the redundancy the previous two-section
/// layout created.
class PluginsPage extends StatefulWidget {
  const PluginsPage({super.key});

  @override
  State<PluginsPage> createState() => _PluginsPageState();
}

class _PluginsPageState extends State<PluginsPage> {
  List<provider_model.ProviderInfo> _providers = [];
  /// Map of "publisher/name" → latest version from the marketplace.
  /// Populated by _load; used by _updateAvailable to decide whether
  /// to render the "Update → vX" chip on each card.
  Map<String, String> _latestVersions = const {};
  bool _loading = true;
  String? _error;
  StreamSubscription<void>? _providersSub;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
    // Any toggle/install/uninstall across the app fires ProvidersBus —
    // we refetch so rows stay in sync without manual refresh.
    _providersSub = ProvidersBus.instance.changes.listen((_) => _load());
  }

  @override
  void dispose() {
    _providersSub?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      // Fetch installed + marketplace in parallel. Marketplace
      // failures are non-fatal — we drop the update-available
      // chip if the registry's unreachable, but the installed
      // list still renders.
      final results = await Future.wait([
        _api.listProviders(),
        _api.listMarketplace().catchError((_) => <MarketplaceEntry>[]),
      ]);
      final providers = results[0] as List<provider_model.ProviderInfo>;
      final market = results[1] as List<MarketplaceEntry>;
      final latest = <String, String>{
        for (final e in market) '${e.publisher}/${e.name}': e.version,
      };
      if (!mounted) return;
      setState(() {
        _providers = providers;
        _latestVersions = latest;
        _loading = false;
        _error = null;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = e.toString();
      });
    }
  }

  /// Returns the newer version string if the marketplace has one
  /// above what's installed; empty string means up-to-date (or no
  /// marketplace record). v1-only plugins get checked; legacy
  /// plugins never appear in the marketplace so they stay null.
  String _updateAvailable(provider_model.ProviderInfo p) {
    if (!p.provider.isV1) return '';
    final key = '${p.provider.publisher}/${p.provider.name}';
    final latest = _latestVersions[key];
    if (latest == null || latest.isEmpty) return '';
    if (_compareSemver(p.provider.version, latest) < 0) return latest;
    return '';
  }

  /// Very small semver comparator. Handles dotted-int triples
  /// (1.0.0, 2.10.3); falls back to lexical compare for anything
  /// weirder. Sufficient for the "update available" heuristic —
  /// pub_semver would be better but isn't worth a new dep.
  int _compareSemver(String a, String b) {
    final pa = a.split('.').map((x) => int.tryParse(x) ?? 0).toList();
    final pb = b.split('.').map((x) => int.tryParse(x) ?? 0).toList();
    final len = pa.length > pb.length ? pa.length : pb.length;
    for (int i = 0; i < len; i++) {
      final ai = i < pa.length ? pa[i] : 0;
      final bi = i < pb.length ? pb[i] : 0;
      if (ai != bi) return ai - bi;
    }
    return a.compareTo(b);
  }

  void _notify(String msg, {bool isError = false}) {
    final messenger = ScaffoldMessenger.maybeOf(context);
    messenger?.showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: isError ? AppColors.error : null,
    ));
  }

  Future<void> _toggleProvider(
      provider_model.ProviderInfo p, bool enabled) async {
    try {
      await _api.toggleProvider(p.provider.name, enabled);
      ProvidersBus.instance.notify();
    } catch (e) {
      _notify('Failed: $e', isError: true);
    }
  }

  Future<void> _uninstall(provider_model.ProviderInfo p) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Uninstall ${p.provider.displayName}?'),
        content: const Text(
          'Removes the plugin, its stored data, and all granted '
          'permissions. This cannot be undone.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(backgroundColor: AppColors.error),
            child: const Text('Uninstall'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    try {
      await _api.deleteProvider(p.provider.name);
      _notify('Uninstalled ${p.provider.displayName}');
      ProvidersBus.instance.notify();
    } catch (e) {
      _notify('Failed: $e', isError: true);
    }
  }

  void _openConsents(provider_model.ProviderInfo p) {
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => PluginConsentsPage(
        pluginName: p.provider.name,
        api: _api,
        onMessage: _notify,
      ),
    ));
  }

  void _openConfig(provider_model.ProviderInfo p) {
    // Every installed plugin is v1 after the Phase 5/6 migration —
    // the legacy PluginConfigPage and the PUT /api/providers/{name}/
    // config route it backed are gone together with the plugins.config
    // JSONB column.
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => PluginConfigurePage(
        pluginName: p.provider.name,
        displayName: p.provider.displayName,
      ),
    ));
  }

  /// Hand-routes for plugins whose Open action has a bespoke Flutter
  /// destination instead of the generic workbench host. Covers both
  /// pre-v1 panels (compat rewrote most names — log-viewer → logs,
  /// log-viewer → logs, etc) AND v1-declarative plugins whose
  /// runtime surface lives outside the workbench (claude → accounts
  /// manager). Consulted before the v1-webview branch so a v1 migration
  /// doesn't accidentally shadow an existing destination.
  static const Map<String, String> _handOpenRoute = {
    'claude': '/settings/claude-accounts',
    'file-browser': '/browser/files',
    'git-viewer': '/browser/git',
    'git-forge': '/browser/forge',
    'pg-browser': '/browser/database',
    'log-viewer': '/browser/logs',
    'mcp': '/browser/mcp',
    'obsidian-reader': '/browser/docs',
    'simulator-preview': '/browser/simulator',
    'task-runner': '/browser/tasks',
    'telegram': '/browser/messaging',
    'web-browser': '/browser/preview',
  };

  /// Routes the Open action per plugin kind:
  ///   - name in [_handOpenRoute] → bespoke page (legacy panel or the
  ///     claude accounts manager)
  ///   - v1 host plugin → PluginRunPage (command list + result viewer)
  ///   - v1 webview plugin → generic /browser/plugin/:name
  ///   - v1 declarative without a hand-route → friendly notice; the
  ///     generic declarative renderer is later M-work, so jumping to
  ///     the stub page would be misleading
  void _openPlugin(provider_model.ProviderInfo p) {
    final name = p.provider.name;

    // Hand-routed plugin — legacy panel page or claude accounts.
    // Checked first so a v1 manifest migration can't shadow an
    // existing bespoke destination.
    final hand = _handOpenRoute[name];
    if (hand != null) {
      GoRouter.of(context).go(hand);
      return;
    }

    // v1 host-form plugins — command runner.
    if (p.provider.isV1 && p.provider.form == 'host') {
      Navigator.of(context).push(MaterialPageRoute(
        builder: (_) => PluginRunPage(
          pluginName: name,
          displayName: p.provider.displayName,
        ),
      ));
      return;
    }

    // v1 webview — generic workbench route resolves the first view.
    if (p.provider.isV1 && p.provider.form == 'webview') {
      GoRouter.of(context).go('/browser/plugin/$name');
      return;
    }

    _notify('No open target for $name', isError: true);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Installed plugins')),
      body: _loading
          ? const Center(
              child: CircularProgressIndicator(color: AppColors.accent))
          : RefreshIndicator(
              onRefresh: _load,
              child: ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  if (_error != null) _errorBanner(),
                  _sectionHeader('Installed', '${_providers.length}'),
                  const SizedBox(height: 8),
                  for (final p in _sortProviders(_providers)) _pluginCard(p),
                  if (_providers.isEmpty && _error == null) _emptyState(),
                ],
              ),
            ),
    );
  }

  Widget _emptyState() {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 40, horizontal: 8),
      child: Column(
        children: const [
          Icon(Icons.extension_off, size: 40, color: AppColors.textMuted),
          SizedBox(height: 10),
          Text('No plugins installed',
              style: TextStyle(
                  fontWeight: FontWeight.w500,
                  fontSize: 14,
                  color: AppColors.text)),
          SizedBox(height: 4),
          Text('Browse the Hub tab to install one.',
              style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
        ],
      ),
    );
  }

  /// Sort: required first, then by type group (panel → webview → cli →
  /// shell), then by name. Gives a stable, predictable layout.
  List<provider_model.ProviderInfo> _sortProviders(
      List<provider_model.ProviderInfo> input) {
    int rank(provider_model.Provider p) {
      if (p.required) return 0;
      if (p.form == 'webview') return 1;
      if (p.type == 'panel') return 2;
      if (p.type == 'cli') return 3;
      if (p.type == 'shell') return 4;
      return 5;
    }

    final copy = [...input];
    copy.sort((a, b) {
      final r = rank(a.provider).compareTo(rank(b.provider));
      if (r != 0) return r;
      return a.provider.displayName.compareTo(b.provider.displayName);
    });
    return copy;
  }

  Widget _errorBanner() {
    return Container(
      margin: const EdgeInsets.only(bottom: 16),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.errorSoft,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(children: [
        const Icon(Icons.error_outline,
            color: AppColors.error, size: 18),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            _error ?? '',
            style: const TextStyle(color: AppColors.error, fontSize: 12),
          ),
        ),
      ]),
    );
  }

  Widget _sectionHeader(String title, String badge) {
    return Row(
      children: [
        Text(title,
            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15)),
        const SizedBox(width: 8),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          decoration: BoxDecoration(
            color: AppColors.surfaceAlt,
            borderRadius: BorderRadius.circular(10),
          ),
          child: Text(badge,
              style: const TextStyle(color: AppColors.textMuted, fontSize: 11)),
        ),
      ],
    );
  }

  /// How the user can enter a plugin from the Plugin page.
  ///   • [inApp]      — tapping the card opens a dedicated page
  ///                    (legacy panel, PluginRunPage, or webview).
  ///   • [launchFromSession] — cli/shell agents; launched via the
  ///     dashboard's New Session dialog, not a page in this app.
  ///   • [none]       — config-only plugin, no runtime surface. The
  ///     card is rendered disabled (grayed, non-tappable).
  _EntryKind _entryKind(provider_model.Provider prov) {
    if (_handOpenRoute.containsKey(prov.name)) return _EntryKind.inApp;
    if (prov.isV1 && (prov.form == 'host' || prov.form == 'webview')) {
      return _EntryKind.inApp;
    }
    if (prov.type == 'cli' || prov.type == 'shell') {
      return _EntryKind.launchFromSession;
    }
    return _EntryKind.none;
  }

  Widget _pluginCard(provider_model.ProviderInfo p) {
    final prov = p.provider;
    final required = prov.required;
    final kindLabel = _kindLabel(prov);
    final entry = _entryKind(prov);
    // Body is tappable only when the plugin is enabled AND has a
    // runtime surface reachable from this page. Config-only plugins
    // render dimmed with no tap — the `…` menu (Configure / Perms /
    // Uninstall) and the Enable switch on the trailing side still
    // work because they're separate widgets outside the InkWell.
    final bodyTappable = p.enabled && entry != _EntryKind.none;
    final VoidCallback? onBodyTap = !bodyTappable
        ? null
        : entry == _EntryKind.inApp
            ? () => _openPlugin(p)
            : () => _notify(context.tr(
                'Launch this agent from the New Session dialog on the dashboard.'));
    final bodyOpacity = bodyTappable ? 1.0 : 0.5;
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          Expanded(
            child: Opacity(
              opacity: bodyOpacity,
              child: InkWell(
                borderRadius: const BorderRadius.horizontal(
                    left: Radius.circular(8)),
                onTap: onBodyTap,
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(14, 10, 8, 10),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.center,
                    children: [
                      _iconBadge(prov.icon),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Row(
                              children: [
                                Flexible(
                                  child: Text(
                                    prov.displayName,
                                    style: const TextStyle(
                                        fontWeight: FontWeight.w600,
                                        fontSize: 14),
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                ),
                                const SizedBox(width: 6),
                                if (required)
                                  _chip('required', AppColors.accent)
                                else
                                  _chip(kindLabel, AppColors.textMuted),
                                if (entry == _EntryKind.none) ...[
                                  const SizedBox(width: 4),
                                  _chip(context.tr('config only'),
                                      AppColors.textMuted),
                                ],
                                if (_updateAvailable(p) != '') ...[
                                  const SizedBox(width: 4),
                                  _chip('update → v${_updateAvailable(p)}',
                                      AppColors.accent),
                                ],
                              ],
                            ),
                            const SizedBox(height: 2),
                            Text(
                              prov.description.isEmpty
                                  ? 'v${prov.version}'
                                  : '${prov.description} · v${prov.version}',
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(
                                  color: AppColors.textMuted,
                                  fontSize: 11,
                                  height: 1.3),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
          _actionsMenu(p),
          Switch(
            value: p.enabled,
            activeTrackColor: AppColors.accent,
            onChanged: required ? null : (v) => _toggleProvider(p, v),
          ),
          const SizedBox(width: 4),
        ],
      ),
    );
  }

  String _kindLabel(provider_model.Provider p) {
    if (p.form == 'webview') return 'webview';
    if (p.type == 'panel') return 'panel';
    if (p.type == 'cli') return 'agent';
    if (p.type == 'shell') return 'shell';
    return p.type.isEmpty ? 'plugin' : p.type;
  }

  Widget _chip(String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        text,
        style: TextStyle(color: color, fontSize: 10, fontWeight: FontWeight.w500),
      ),
    );
  }

  Widget _iconBadge(String icon) {
    final isEmoji = icon.length <= 4 && !icon.contains('/');
    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: AppColors.accent.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(8),
      ),
      alignment: Alignment.center,
      child: isEmoji
          ? Text(icon, style: const TextStyle(fontSize: 18))
          : const Icon(Icons.extension, color: AppColors.accent, size: 20),
    );
  }

  Widget _actionsMenu(provider_model.ProviderInfo p) {
    return PopupMenuButton<String>(
      icon: const Icon(Icons.more_vert, color: AppColors.textMuted, size: 20),
      onSelected: (value) {
        switch (value) {
          case 'open':
            _openPlugin(p);
            break;
          case 'configure':
            _openConfig(p);
            break;
          case 'accounts':
            GoRouter.of(context).go('/settings/claude-accounts');
            break;
          case 'consents':
            _openConsents(p);
            break;
          case 'uninstall':
            _uninstall(p);
            break;
        }
      },
      itemBuilder: (ctx) => [
        // Open is only useful when the plugin has a real runtime
        // surface in the Flutter app — panels, webviews, host
        // sidecars. CLI/shell agents are launched from the dashboard;
        // config-only plugins have nothing to open. Both drop the
        // menu entry instead of offering a dead button.
        if (p.enabled && _entryKind(p.provider) == _EntryKind.inApp)
          const PopupMenuItem(
            value: 'open',
            child: ListTile(
              dense: true,
              contentPadding: EdgeInsets.zero,
              leading: Icon(Icons.open_in_new, size: 18),
              title: Text('Open'),
            ),
          ),
        // Dedicated surface for Claude's multi-account manager. Also
        // reachable via Open (hand-route map), but the explicit menu
        // entry keeps discovery cheap when the user is scanning for
        // "where do I add another token".
        if (p.provider.name == 'claude')
          const PopupMenuItem(
            value: 'accounts',
            child: ListTile(
              dense: true,
              contentPadding: EdgeInsets.zero,
              leading: Icon(Icons.people_outline, size: 18),
              title: Text('Accounts'),
            ),
          ),
        if (p.provider.configSchema.isNotEmpty)
          const PopupMenuItem(
            value: 'configure',
            child: ListTile(
              dense: true,
              contentPadding: EdgeInsets.zero,
              leading: Icon(Icons.tune, size: 18),
              title: Text('Configure'),
            ),
          ),
        const PopupMenuItem(
          value: 'consents',
          child: ListTile(
            dense: true,
            contentPadding: EdgeInsets.zero,
            leading: Icon(Icons.shield, size: 18),
            title: Text('Permissions'),
          ),
        ),
        if (!p.provider.required)
          const PopupMenuItem(
            value: 'uninstall',
            child: ListTile(
              dense: true,
              contentPadding: EdgeInsets.zero,
              leading:
                  Icon(Icons.delete_outline, size: 18, color: AppColors.error),
              title: Text('Uninstall',
                  style: TextStyle(color: AppColors.error)),
            ),
          ),
      ],
    );
  }
}

/// How the Plugin page lets the user enter a plugin. Drives both the
/// card-body tap handler and the Open menu item visibility — keep the
/// two in sync via [_PluginsPageState._entryKind].
enum _EntryKind { inApp, launchFromSession, none }
