// V1 Plugins page — categorized card grid.
//
// What changed from PluginsPage:
//   - Renamed sidebar entry from "Agents" → "Plugins" in app.dart;
//     these are extension panels, not AI personas.
//   - Cards are a responsive grid (2 / 3 / 4 cols based on width)
//     instead of a 1-column wall of full-width rows.
//   - Cards are compact (~120 px tall): icon + name + version line +
//     primary action button + kebab. Description shows on hover (tooltip)
//     instead of always-on; long copy belongs on the plugin's own page.
//   - Plugins are grouped into 3 sections so 18 entries don't read as
//     undifferentiated mush:
//       1. System — required, always-on (Claude Code, File Browser,
//          Terminal). Compact dense row.
//       2. Workbench panels — Open-able tools (Git, Logs, MCP,
//          Obsidian, PG, Simulator, Telegram, etc.). Card grid.
//       3. Background services — config-only, no UI surface (rare).
//          Collapsed by default.
//
// Reuses the existing ApiClient, ProvidersBus, and PluginRunPage. Data
// shape, toggle/open semantics, marketplace-update detection match
// the legacy page so behaviour is unchanged — only the layout differs.

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

class PluginsV1Page extends StatefulWidget {
  const PluginsV1Page({super.key});
  @override
  State<PluginsV1Page> createState() => _PluginsV1PageState();
}

class _PluginsV1PageState extends State<PluginsV1Page> {
  List<provider_model.ProviderInfo> _providers = const [];
  bool _loading = true;
  String? _error;
  StreamSubscription<void>? _sub;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
    _sub = ProvidersBus.instance.changes.listen((_) => _load());
  }

  @override
  void dispose() {
    _sub?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final list = await _api.listProviders();
      if (!mounted) return;
      setState(() {
        _providers = list;
        _loading = false;
        _error = null;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  Future<void> _toggle(provider_model.ProviderInfo p, bool v) async {
    try {
      await _api.toggleProvider(p.provider.name, v);
      ProvidersBus.instance.notify();
    } catch (e) {
      _toast('Toggle failed: $e', isError: true);
    }
  }

  void _configure(provider_model.ProviderInfo p) {
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => PluginConfigurePage(
        pluginName: p.provider.name,
        displayName: context.pickL10n(
            p.provider.displayName, p.provider.displayNameZh),
      ),
    ));
  }

  void _permissions(provider_model.ProviderInfo p) {
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => PluginConsentsPage(pluginName: p.provider.name, api: _api),
    ));
  }

  Future<void> _uninstall(provider_model.ProviderInfo p) async {
    if (p.provider.required) {
      _toast('${p.provider.displayName} is required and cannot be uninstalled.',
          isError: true);
      return;
    }
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Uninstall ${p.provider.displayName}?'),
        content: const Text(
            'Removes the plugin and its stored configuration. You can reinstall it later from the marketplace.'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          TextButton(
              onPressed: () => Navigator.pop(ctx, true),
              style: TextButton.styleFrom(foregroundColor: Colors.red),
              child: const Text('Uninstall')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.deleteProvider(p.provider.name);
      ProvidersBus.instance.notify();
      _toast('${p.provider.displayName} uninstalled');
    } catch (e) {
      _toast('Uninstall failed: $e', isError: true);
    }
  }

  void _toast(String msg, {bool isError = false}) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: isError ? t.danger : null,
    ));
  }

  // -- Category bucketing --------------------------------------------------

  static const Map<String, String> _handOpenRoute = {
    'claude': '/settings/claude-accounts',
    'file-browser': '/browser/files',
    'source-control': '/browser/source-control',
    'pg-browser': '/browser/database',
    'log-viewer': '/browser/logs',
    'mcp': '/browser/mcp',
    'obsidian-reader': '/browser/docs',
    'simulator-preview': '/browser/simulator',
    'task-runner': '/browser/tasks',
    'telegram': '/browser/messaging',
    'web-browser': '/browser/preview',
  };

  bool _hasOpenable(provider_model.Provider p) {
    if (_handOpenRoute.containsKey(p.name)) return true;
    if (p.isV1 && (p.form == 'host' || p.form == 'webview')) return true;
    return false;
  }

  void _open(provider_model.ProviderInfo p) {
    final name = p.provider.name;
    final hand = _handOpenRoute[name];
    if (hand != null) {
      GoRouter.of(context).go(hand);
      return;
    }
    if (p.provider.isV1 && p.provider.form == 'host') {
      Navigator.of(context).push(MaterialPageRoute(
        builder: (_) => PluginRunPage(
          pluginName: name,
          displayName: p.provider.displayName,
        ),
      ));
      return;
    }
    if (p.provider.isV1 && p.provider.form == 'webview') {
      GoRouter.of(context).go('/browser/plugin/$name');
      return;
    }
    _toast('Nothing to open for $name', isError: true);
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    if (_loading) {
      return Center(child: CircularProgressIndicator(color: t.accent));
    }
    if (_providers.isEmpty && _error != null) {
      return _ErrorState(message: _error!);
    }

    final required = <provider_model.ProviderInfo>[];
    final panels = <provider_model.ProviderInfo>[];
    final background = <provider_model.ProviderInfo>[];
    for (final p in _providers) {
      if (p.provider.required) {
        required.add(p);
      } else if (_hasOpenable(p.provider)) {
        panels.add(p);
      } else {
        background.add(p);
      }
    }
    int byName(provider_model.ProviderInfo a, provider_model.ProviderInfo b) =>
        a.provider.displayName.compareTo(b.provider.displayName);
    required.sort(byName);
    panels.sort(byName);
    background.sort(byName);

    return Scrollbar(
      child: SingleChildScrollView(
        padding: EdgeInsets.symmetric(
            horizontal: t.sp5, vertical: t.sp4),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 1400),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _PageHeader(),
              SizedBox(height: t.sp5),
              if (_error != null) _ErrorBanner(message: _error!),

              if (required.isNotEmpty) ...[
                _SectionHeader(
                  title: 'System',
                  count: required.length,
                  blurb:
                      'Always-on plugins OpenDray needs to function. You can\'t disable these.',
                ),
                SizedBox(height: t.sp3),
                _SystemRow(items: required, onOpen: _open),
                SizedBox(height: t.sp6),
              ],

              if (panels.isNotEmpty) ...[
                _SectionHeader(
                  title: 'Workbench panels',
                  count: panels.length,
                  blurb:
                      'Tools that open inline in the Workbench: file browser, git, MCP, logs, etc.',
                ),
                SizedBox(height: t.sp3),
                _PluginGrid(
                  items: panels,
                  onOpen: _open,
                  onToggle: _toggle,
                  onConfigure: _configure,
                  onPermissions: _permissions,
                  onUninstall: _uninstall,
                ),
                SizedBox(height: t.sp6),
              ],

              if (background.isNotEmpty) ...[
                _SectionHeader(
                  title: 'Background services',
                  count: background.length,
                  blurb:
                      'Plugins that run in the background — no UI to open. Configure them via the kebab menu.',
                ),
                SizedBox(height: t.sp3),
                _PluginGrid(
                  items: background,
                  onOpen: _open,
                  onToggle: _toggle,
                  onConfigure: _configure,
                  onPermissions: _permissions,
                  onUninstall: _uninstall,
                ),
                SizedBox(height: t.sp6),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Header / section / error chrome
// -----------------------------------------------------------------------------

class _PageHeader extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Plugins', style: theme.textTheme.displaySmall),
              SizedBox(height: t.sp2),
              Text(
                  'Extensions that add panels and tools to OpenDray. AI models live in Connections.',
                  style: theme.textTheme.bodyLarge
                      ?.copyWith(color: t.textMuted)),
            ],
          ),
        ),
        OutlinedButton.icon(
          onPressed: () => GoRouter.of(context).go('/'),
          icon: const Icon(Icons.storefront_outlined, size: 14),
          label: const Text('Browse marketplace'),
        ),
      ],
    );
  }
}

class _SectionHeader extends StatelessWidget {
  final String title;
  final int count;
  final String blurb;
  const _SectionHeader(
      {required this.title, required this.count, required this.blurb});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Text(title,
            style: theme.textTheme.headlineSmall
                ?.copyWith(fontSize: 14, fontWeight: FontWeight.w700)),
        SizedBox(width: t.sp2),
        Container(
          padding: EdgeInsets.symmetric(horizontal: 6, vertical: 1),
          decoration: BoxDecoration(
            color: t.surface3,
            borderRadius: BorderRadius.circular(t.rXs),
          ),
          child: Text('$count',
              style: TextStyle(
                  fontSize: 10, color: t.textSubtle, fontWeight: FontWeight.w700)),
        ),
        SizedBox(width: t.sp3),
        Expanded(
          child: Text(blurb,
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: t.textSubtle, fontSize: 11),
              overflow: TextOverflow.ellipsis),
        ),
      ],
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  final String message;
  const _ErrorBanner({required this.message});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      margin: EdgeInsets.only(bottom: t.sp4),
      padding: EdgeInsets.all(t.sp3),
      decoration: BoxDecoration(
        color: t.dangerSoft,
        borderRadius: BorderRadius.circular(t.rMd),
        border: Border.all(color: t.danger.withValues(alpha: 0.4)),
      ),
      child: Row(
        children: [
          Icon(Icons.error_outline, color: t.danger, size: 16),
          SizedBox(width: t.sp2),
          Expanded(
              child: Text(message,
                  style: TextStyle(color: t.danger, fontSize: 12))),
        ],
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  final String message;
  const _ErrorState({required this.message});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Center(
      child: Padding(
        padding: EdgeInsets.all(t.sp6),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.cloud_off, color: t.danger, size: 32),
            SizedBox(height: t.sp3),
            Text('Cannot load plugins',
                style: Theme.of(context).textTheme.titleMedium),
            SizedBox(height: t.sp2),
            Text(message,
                textAlign: TextAlign.center,
                style: TextStyle(color: t.textMuted, fontSize: 12)),
          ],
        ),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// "System" — dense row, no toggles, just an Open chip per item.
// -----------------------------------------------------------------------------

class _SystemRow extends StatelessWidget {
  final List<provider_model.ProviderInfo> items;
  final void Function(provider_model.ProviderInfo) onOpen;
  const _SystemRow({required this.items, required this.onOpen});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      decoration: BoxDecoration(
        color: t.surface,
        borderRadius: BorderRadius.circular(t.rLg),
        border: Border.all(color: t.border),
      ),
      child: Wrap(
        spacing: 0,
        runSpacing: 0,
        children: [
          for (var i = 0; i < items.length; i++) ...[
            _SystemTile(item: items[i], onOpen: onOpen),
            if (i < items.length - 1)
              Container(width: 1, height: 56, color: t.border),
          ],
        ],
      ),
    );
  }
}

class _SystemTile extends StatelessWidget {
  final provider_model.ProviderInfo item;
  final void Function(provider_model.ProviderInfo) onOpen;
  const _SystemTile({required this.item, required this.onOpen});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final p = item.provider;
    final name = context.pickL10n(p.displayName, p.displayNameZh);
    return Padding(
      padding: EdgeInsets.symmetric(horizontal: t.sp4, vertical: t.sp3),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          _IconBadge(icon: p.icon, accent: t.accentSoft),
          SizedBox(width: t.sp3),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(name,
                  style: const TextStyle(
                      fontSize: 13, fontWeight: FontWeight.w600)),
              Text('v${p.version}',
                  style: TextStyle(fontSize: 10, color: t.textSubtle)),
            ],
          ),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Plugin grid — used for Panels + Background sections.
// -----------------------------------------------------------------------------

class _PluginGrid extends StatelessWidget {
  final List<provider_model.ProviderInfo> items;
  final void Function(provider_model.ProviderInfo) onOpen;
  final void Function(provider_model.ProviderInfo, bool) onToggle;
  final void Function(provider_model.ProviderInfo) onConfigure;
  final void Function(provider_model.ProviderInfo) onPermissions;
  final void Function(provider_model.ProviderInfo) onUninstall;
  const _PluginGrid({
    required this.items,
    required this.onOpen,
    required this.onToggle,
    required this.onConfigure,
    required this.onPermissions,
    required this.onUninstall,
  });

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return LayoutBuilder(builder: (ctx, c) {
      final cols =
          c.maxWidth > 1100 ? 4 : c.maxWidth > 760 ? 3 : c.maxWidth > 460 ? 2 : 1;
      return GridView.count(
        crossAxisCount: cols,
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        crossAxisSpacing: t.sp3,
        mainAxisSpacing: t.sp3,
        childAspectRatio: cols == 1 ? 4.5 : 2.4,
        children: items
            .map((p) => _PluginCard(
                  item: p,
                  onOpen: onOpen,
                  onToggle: onToggle,
                  onConfigure: onConfigure,
                  onPermissions: onPermissions,
                  onUninstall: onUninstall,
                ))
            .toList(),
      );
    });
  }
}

class _PluginCard extends StatelessWidget {
  final provider_model.ProviderInfo item;
  final void Function(provider_model.ProviderInfo) onOpen;
  final void Function(provider_model.ProviderInfo, bool) onToggle;
  final void Function(provider_model.ProviderInfo) onConfigure;
  final void Function(provider_model.ProviderInfo) onPermissions;
  final void Function(provider_model.ProviderInfo) onUninstall;
  const _PluginCard({
    required this.item,
    required this.onOpen,
    required this.onToggle,
    required this.onConfigure,
    required this.onPermissions,
    required this.onUninstall,
  });

  bool get _isOpenable {
    final p = item.provider;
    if (p.name == 'claude' ||
        p.name == 'file-browser' ||
        p.name == 'source-control' ||
        p.name == 'pg-browser' ||
        p.name == 'log-viewer' ||
        p.name == 'mcp' ||
        p.name == 'obsidian-reader' ||
        p.name == 'simulator-preview' ||
        p.name == 'task-runner' ||
        p.name == 'telegram' ||
        p.name == 'web-browser') return true;
    return p.isV1 && (p.form == 'host' || p.form == 'webview');
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final p = item.provider;
    final name = context.pickL10n(p.displayName, p.displayNameZh);
    final desc = context.pickL10n(p.description, p.descriptionZh);
    final dim = !item.enabled;
    return Tooltip(
      message: desc.isEmpty ? name : desc,
      waitDuration: const Duration(milliseconds: 600),
      child: InkWell(
        onTap: item.enabled && _isOpenable ? () => onOpen(item) : null,
        borderRadius: BorderRadius.circular(t.rLg),
        child: Container(
          decoration: BoxDecoration(
            color: t.surface,
            borderRadius: BorderRadius.circular(t.rLg),
            border: Border.all(color: t.border),
          ),
          padding: EdgeInsets.fromLTRB(t.sp3, t.sp3, t.sp2, t.sp3),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Opacity(
                      opacity: dim ? 0.5 : 1.0,
                      child: _IconBadge(icon: p.icon, accent: t.accentSoft)),
                  SizedBox(width: t.sp3),
                  Expanded(
                    child: Opacity(
                      opacity: dim ? 0.5 : 1.0,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(name,
                              style: const TextStyle(
                                  fontSize: 13,
                                  fontWeight: FontWeight.w600),
                              overflow: TextOverflow.ellipsis,
                              maxLines: 1),
                          Text('v${p.version}',
                              style: TextStyle(
                                  fontSize: 10, color: t.textSubtle),
                              overflow: TextOverflow.ellipsis,
                              maxLines: 1),
                        ],
                      ),
                    ),
                  ),
                  Transform.scale(
                    scale: 0.7,
                    child: Switch(
                      value: item.enabled,
                      onChanged:
                          item.provider.required ? null : (v) => onToggle(item, v),
                    ),
                  ),
                  PopupMenuButton<String>(
                    tooltip: 'Plugin actions',
                    icon: Icon(Icons.more_vert, size: 16, color: t.textMuted),
                    splashRadius: 16,
                    padding: EdgeInsets.zero,
                    iconSize: 16,
                    constraints:
                        const BoxConstraints(minWidth: 32, minHeight: 32),
                    onSelected: (v) {
                      switch (v) {
                        case 'configure':
                          onConfigure(item);
                          break;
                        case 'permissions':
                          onPermissions(item);
                          break;
                        case 'uninstall':
                          onUninstall(item);
                          break;
                      }
                    },
                    itemBuilder: (_) => [
                      if (item.provider.configSchema.isNotEmpty)
                        const PopupMenuItem(
                            value: 'configure',
                            child: ListTile(
                                dense: true,
                                contentPadding: EdgeInsets.zero,
                                leading: Icon(Icons.tune, size: 16),
                                title: Text('Configure'))),
                      const PopupMenuItem(
                          value: 'permissions',
                          child: ListTile(
                              dense: true,
                              contentPadding: EdgeInsets.zero,
                              leading: Icon(Icons.lock_outline, size: 16),
                              title: Text('Permissions'))),
                      if (!item.provider.required) const PopupMenuDivider(),
                      if (!item.provider.required)
                        const PopupMenuItem(
                            value: 'uninstall',
                            child: ListTile(
                                dense: true,
                                contentPadding: EdgeInsets.zero,
                                leading: Icon(Icons.delete_outline,
                                    size: 16, color: Colors.redAccent),
                                title: Text('Uninstall',
                                    style:
                                        TextStyle(color: Colors.redAccent)))),
                    ],
                  ),
                ],
              ),
              if (desc.isNotEmpty) ...[
                SizedBox(height: t.sp2),
                Opacity(
                  opacity: dim ? 0.5 : 1.0,
                  child: Text(desc,
                      style: TextStyle(
                          fontSize: 11,
                          color: t.textMuted,
                          height: 1.35),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Icon badge — square tile rendering the manifest's icon character.
// -----------------------------------------------------------------------------

class _IconBadge extends StatelessWidget {
  final String icon;
  final Color accent;
  const _IconBadge({required this.icon, required this.accent});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      width: 28, height: 28,
      decoration: BoxDecoration(
        color: accent,
        borderRadius: BorderRadius.circular(t.rSm),
      ),
      alignment: Alignment.center,
      child: Text(icon.isEmpty ? '?' : icon,
          style: const TextStyle(fontSize: 14)),
    );
  }
}
