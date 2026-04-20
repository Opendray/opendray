import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart' as provider_model;
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';
import '../settings/plugin_consents_page.dart';
import 'plugin_config_page.dart';
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
      final providers = await _api.listProviders();
      if (!mounted) return;
      setState(() {
        _providers = providers;
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
    // v1 plugins use the new configSchema pipeline (plugin_kv +
    // plugin_secret + sidecar restart). Legacy providers still edit
    // the old `providers.config` column via PluginConfigPage.
    final builder = p.provider.isV1
        ? (BuildContext _) => PluginConfigurePage(
              pluginName: p.provider.name,
              displayName: p.provider.displayName,
            )
        : (BuildContext _) => PluginConfigPage(info: p);
    Navigator.of(context).push(MaterialPageRoute(builder: builder));
  }

  /// Legacy panel plugins don't map 1:1 to their /browser/* route —
  /// compat rewrote most names (log-viewer → logs, llm-providers →
  /// endpoints, simulator-preview → simulator, etc). Keep a single
  /// source-of-truth mapping here so the Open action lines up with
  /// the Browser tab's existing panel routes.
  static const Map<String, String> _legacyPanelRoute = {
    'file-browser': '/browser/files',
    'git': '/browser/git',
    'log-viewer': '/browser/logs',
    'mcp': '/browser/mcp',
    'obsidian-reader': '/browser/docs',
    'simulator-preview': '/browser/simulator',
    'task-runner': '/browser/tasks',
    'telegram': '/browser/messaging',
    'web-preview': '/browser/preview',
    'llm-providers': '/browser/endpoints',
  };

  /// Routes the Open action per plugin kind:
  ///   - legacy panel plugin → its existing /browser/* route
  ///   - v1 host plugin → PluginRunPage (command list + result viewer)
  ///   - v1 webview / declarative → the generic /browser/plugin/:name
  ///   - otherwise a friendly message
  void _openPlugin(provider_model.ProviderInfo p) {
    final name = p.provider.name;

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

    // v1 webview / declarative — generic workbench route already
    // resolves the first contributed view.
    if (p.provider.isV1 &&
        (p.provider.form == 'webview' || p.provider.form == 'declarative')) {
      GoRouter.of(context).go('/browser/plugin/$name');
      return;
    }

    // Legacy panel plugin — hand off to the existing bespoke page.
    final legacy = _legacyPanelRoute[name];
    if (legacy != null) {
      GoRouter.of(context).go(legacy);
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

  Widget _pluginCard(provider_model.ProviderInfo p) {
    final prov = p.provider;
    final required = prov.required;
    final kindLabel = _kindLabel(prov);
    // Tapping the card body opens the plugin's runtime surface (legacy
    // panel page, command runner, or webview). The switch + menu on
    // the right trail have their own tap handlers and don't bubble.
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: p.enabled ? () => _openPlugin(p) : null,
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
                                fontWeight: FontWeight.w600, fontSize: 14),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const SizedBox(width: 6),
                        if (required)
                          _chip('required', AppColors.accent)
                        else
                          _chip(kindLabel, AppColors.textMuted),
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
                          color: AppColors.textMuted, fontSize: 11, height: 1.3),
                    ),
                  ],
                ),
              ),
              _actionsMenu(p),
              Switch(
                value: p.enabled,
                activeTrackColor: AppColors.accent,
                onChanged: required ? null : (v) => _toggleProvider(p, v),
              ),
            ],
          ),
        ),
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
          case 'consents':
            _openConsents(p);
            break;
          case 'uninstall':
            _uninstall(p);
            break;
        }
      },
      itemBuilder: (ctx) => [
        if (p.enabled)
          const PopupMenuItem(
            value: 'open',
            child: ListTile(
              dense: true,
              contentPadding: EdgeInsets.zero,
              leading: Icon(Icons.open_in_new, size: 18),
              title: Text('Open'),
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
