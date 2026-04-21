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
    final displayName =
        context.pickL10nOnce(p.provider.displayName, p.provider.displayNameZh);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Uninstall $displayName?'),
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
      _notify('Uninstalled $displayName');
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
      appBar: AppBar(title: const Text('Plugins')),
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
                  const SizedBox(height: 10),
                  for (final p in _sortProviders(_providers)) _pluginCard(p),
                  if (_providers.isEmpty && _error == null) _emptyState(),
                ],
              ),
            ),
    );
  }

  Widget _emptyState() {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 40, horizontal: 8),
      child: Column(
        children: [
          Icon(Icons.extension_off, size: 40, color: AppColors.textMuted),
          SizedBox(height: 10),
          Text('No plugins registered',
              style: TextStyle(
                  fontWeight: FontWeight.w500,
                  fontSize: 14,
                  color: AppColors.text)),
          SizedBox(height: 4),
          Text('The kernel should seed built-in panels on boot. '
              'Pull-to-refresh to retry.',
              textAlign: TextAlign.center,
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

  /// Hub-style card: stacked header → chip row → description → action
  /// bar. Same logic as the old compact row (_entryKind, _openPlugin,
  /// toggle, uninstall) — just laid out with the same breathing room
  /// the Hub marketplace cards use so built-in panels get the same
  /// visual weight as third-party installs will in v1.1.
  Widget _pluginCard(provider_model.ProviderInfo p) {
    final prov = p.provider;
    final required = prov.required;
    final kindLabel = _kindLabel(prov);
    final entry = _entryKind(prov);
    final displayName = context.pickL10n(prov.displayName, prov.displayNameZh);
    final description = context.pickL10n(prov.description, prov.descriptionZh);
    final updateVer = _updateAvailable(p);
    final publisher = prov.publisher.isEmpty ? '' : prov.publisher;
    // Whole body dimmed when the plugin is disabled OR has no
    // reachable surface — the action row still renders at full opacity
    // so Enable / overflow stay obvious.
    final bodyOpacity =
        p.enabled && entry != _EntryKind.none ? 1.0 : 0.55;

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 14, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Opacity(
                    opacity: bodyOpacity, child: _iconBadge(prov.icon)),
                const SizedBox(width: 14),
                Expanded(
                  child: Opacity(
                    opacity: bodyOpacity,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          displayName,
                          style: const TextStyle(
                            fontWeight: FontWeight.w600,
                            fontSize: 15,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          publisher.isEmpty
                              ? 'v${prov.version}'
                              : 'v${prov.version} · $publisher',
                          style: const TextStyle(
                            color: AppColors.textMuted,
                            fontSize: 11,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
                Switch(
                  value: p.enabled,
                  activeTrackColor: AppColors.accent,
                  onChanged: required ? null : (v) => _toggleProvider(p, v),
                ),
              ],
            ),
            const SizedBox(height: 10),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                if (required)
                  _chip('required', AppColors.accent)
                else
                  _chip(kindLabel, AppColors.textMuted),
                if (_isBuiltin(prov))
                  Tooltip(
                    message: 'Ships with OpenDray. Upgrades arrive with '
                        'OpenDray releases, not through the Hub.',
                    child:
                        _chip('built-in', const Color(0xFF7C3AED)),
                  ),
                if (entry == _EntryKind.none)
                  _chip(context.tr('config only'), AppColors.textMuted),
                if (updateVer.isNotEmpty)
                  _chip('update → v$updateVer', AppColors.accent),
              ],
            ),
            if (description.isNotEmpty) ...[
              const SizedBox(height: 10),
              Opacity(
                opacity: bodyOpacity,
                child: Text(
                  description,
                  style: const TextStyle(
                    fontSize: 12,
                    color: AppColors.text,
                    height: 1.4,
                  ),
                ),
              ),
            ],
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(child: _primaryAction(p, entry)),
                const SizedBox(width: 8),
                _actionsMenu(p),
              ],
            ),
          ],
        ),
      ),
    );
  }

  /// The main call-to-action button on each card. Its label and handler
  /// depend on whether the plugin has a reachable runtime surface:
  ///   - enabled + inApp         → "Open"        (navigates to the panel/webview)
  ///   - enabled + launchFromSession → "Launch from session" (toast explainer)
  ///   - enabled + none          → "Configure"   (only if configSchema exists) else disabled
  ///   - disabled                → "Disabled"    (greyed out)
  /// Keeps action intent visible without making the user hunt through
  /// the `…` menu.
  Widget _primaryAction(provider_model.ProviderInfo p, _EntryKind entry) {
    if (!p.enabled) {
      return OutlinedButton.icon(
        onPressed: null,
        icon: const Icon(Icons.power_settings_new, size: 16),
        label: Text(context.tr('Disabled')),
      );
    }
    if (entry == _EntryKind.inApp) {
      return FilledButton.icon(
        onPressed: () => _openPlugin(p),
        icon: const Icon(Icons.open_in_new, size: 16),
        label: Text(context.tr('Open')),
        style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
      );
    }
    if (entry == _EntryKind.launchFromSession) {
      return OutlinedButton.icon(
        onPressed: () => _notify(context.tr(
            'Launch this agent from the New Session dialog on the dashboard.')),
        icon: const Icon(Icons.play_arrow, size: 16),
        label: Text(context.tr('Launch from session')),
      );
    }
    // _EntryKind.none — config-only plugin. Open Configure directly
    // if the plugin declares a schema; otherwise render the button
    // disabled so users see the card is intentional, not broken.
    return OutlinedButton.icon(
      onPressed: p.provider.configSchema.isNotEmpty ? () => _openConfig(p) : null,
      icon: const Icon(Icons.tune, size: 16),
      label: Text(context.tr('Configure')),
    );
  }

  String _kindLabel(provider_model.Provider p) {
    if (p.form == 'webview') return 'webview';
    if (p.type == 'panel') return 'panel';
    if (p.type == 'cli') return 'agent';
    if (p.type == 'shell') return 'shell';
    return p.type.isEmpty ? 'plugin' : p.type;
  }

  /// A plugin is "built-in" when it's published by `opendray-builtin`
  /// with a declarative form — its code isn't in the bundle (manifest
  /// only); the feature itself lives in the OpenDray gateway + Flutter
  /// binaries. Surfacing this in a chip makes the Hub's marketplace
  /// semantics honest: users understand these aren't third-party
  /// extensions they can replace — upgrades ship with OpenDray itself.
  /// See docs/plugin-platform/06-plugin-formats.md §"Declarative = a
  /// registration form, not a third-party form".
  bool _isBuiltin(provider_model.Provider p) {
    return p.publisher == 'opendray-builtin' && p.form == 'declarative';
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
      width: 44,
      height: 44,
      decoration: BoxDecoration(
        color: AppColors.accent.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(10),
      ),
      alignment: Alignment.center,
      child: isEmoji
          ? Text(icon, style: const TextStyle(fontSize: 22))
          : const Icon(Icons.extension, color: AppColors.accent, size: 24),
    );
  }

  Widget _actionsMenu(provider_model.ProviderInfo p) {
    final entry = _entryKind(p.provider);
    // Configure is already the primary button for config-only plugins
    // (entry == none); duplicating it in the overflow adds noise.
    final showConfigureInMenu =
        p.provider.configSchema.isNotEmpty && entry != _EntryKind.none;
    return PopupMenuButton<String>(
      icon: const Icon(Icons.more_vert, color: AppColors.textMuted, size: 20),
      onSelected: (value) {
        switch (value) {
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
        // Open/Launch/Configure already live on the primary action
        // button in the card footer — the overflow menu stays focused
        // on secondary actions (Accounts, Configure-when-not-primary,
        // Permissions, Uninstall) so it doesn't duplicate CTAs.
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
        if (showConfigureInMenu)
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

/// How the Plugin page lets the user enter a plugin. Drives the
/// card's primary-action button label + handler (see
/// [_PluginsPageState._primaryAction]) and whether Configure appears
/// in the overflow menu (it's promoted to the primary button for
/// `none`-kind entries).
enum _EntryKind { inApp, launchFromSession, none }
