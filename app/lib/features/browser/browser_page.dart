import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// Browser launcher — a scalable grid of panel entries.
///
/// Each category with ≥ 1 enabled panel plugin gets one card. Tapping a
/// card pushes the dedicated panel route; back returns here. This avoids
/// the tab-overflow problem we hit when the app had 8+ panel plugins
/// crammed into a scrolling TabBar.
class BrowserPage extends StatefulWidget {
  const BrowserPage({super.key});
  @override
  State<BrowserPage> createState() => _BrowserPageState();
}

class _BrowserPageState extends State<BrowserPage> {
  List<_PanelEntry> _panels = [];
  bool _loading = true;
  StreamSubscription<void>? _providersSub;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
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
      final panels = providers
          .where((p) => p.provider.type == 'panel' && p.enabled)
          .toList();
      bool has(bool Function(ProviderInfo) pred) => panels.any(pred);

      final entries = <_PanelEntry>[
        if (has((p) => p.provider.category == 'docs'))
          const _PanelEntry(
            route: 'docs', titleKey: 'Docs', icon: Icons.description,
            descKey: 'Read markdown and Git-forge sources',
          ),
        if (has((p) => p.provider.category == 'files'))
          const _PanelEntry(
            route: 'files', titleKey: 'Files', icon: Icons.folder,
            descKey: 'Browse & edit server files',
          ),
        if (has((p) => p.provider.category == 'database'))
          const _PanelEntry(
            route: 'database', titleKey: 'Database', icon: Icons.storage,
            descKey: 'Read-only Postgres browsing & SQL',
          ),
        if (has((p) => p.provider.category == 'tasks' ||
                       p.provider.name     == 'task-runner'))
          const _PanelEntry(
            route: 'tasks', titleKey: 'Tasks', icon: Icons.play_circle_outline,
            descKey: 'Run Makefile / npm / shell tasks',
          ),
        if (has((p) => p.provider.name == 'git'))
          const _PanelEntry(
            route: 'git', titleKey: 'Git', icon: Icons.park_outlined,
            descKey: 'Track and commit per-session changes',
          ),
        if (has((p) => p.provider.category == 'logs'))
          const _PanelEntry(
            route: 'logs', titleKey: 'Logs', icon: Icons.article_outlined,
            descKey: 'Tail and grep log files live',
          ),
        if (has((p) => p.provider.category == 'messaging'))
          const _PanelEntry(
            route: 'messaging', titleKey: 'Messaging', icon: Icons.send,
            descKey: 'Telegram bridge & session links',
          ),
        if (has((p) => p.provider.category == 'mcp'))
          const _PanelEntry(
            route: 'mcp', titleKey: 'MCP Servers', icon: Icons.electrical_services,
            descKey: 'Manage MCP servers injected into agents',
          ),
        if (has((p) => p.provider.category == 'preview'))
          const _PanelEntry(
            route: 'preview', titleKey: 'Preview', icon: Icons.web,
            descKey: 'In-app browser with multi-tab URL preview',
          ),
        if (has((p) => p.provider.category == 'simulator'))
          const _PanelEntry(
            route: 'simulator', titleKey: 'Simulator', icon: Icons.phone_iphone,
            descKey: 'Live iOS / Android device screen with touch & key input',
          ),
        if (has((p) => p.provider.category == 'endpoints'))
          const _PanelEntry(
            route: 'endpoints', titleKey: 'LLM Providers', icon: Icons.satellite_alt_outlined,
            descKey: 'Local & free model endpoints routed at spawn time',
          ),
      ];
      if (!mounted) return;
      setState(() { _panels = entries; _loading = false; });
    } catch (_) {
      if (!mounted) return;
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(context.tr('Browser'))),
      body: _loading
          ? const Center(child: CircularProgressIndicator(color: AppColors.accent))
          : _panels.isEmpty
              ? _emptyState(context)
              : RefreshIndicator(
                  onRefresh: _load,
                  child: GridView.builder(
                    padding: const EdgeInsets.all(14),
                    itemCount: _panels.length,
                    gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                      crossAxisCount: 2,
                      mainAxisSpacing: 12,
                      crossAxisSpacing: 12,
                      childAspectRatio: 1.05,
                    ),
                    itemBuilder: (_, i) => _card(context, _panels[i]),
                  ),
                ),
    );
  }

  Widget _emptyState(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.extension_off, size: 48, color: AppColors.textMuted),
          const SizedBox(height: 16),
          Text(
            context.tr('No browser panels enabled'),
            style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 15),
          ),
          const SizedBox(height: 8),
          Text(
            context.tr('Enable a File Browser, Database, Tasks, Preview or other panel plugin in Settings → Plugins.'),
            style: const TextStyle(color: AppColors.textMuted, fontSize: 12),
            textAlign: TextAlign.center,
          ),
        ]),
      ),
    );
  }

  Widget _card(BuildContext context, _PanelEntry e) {
    return InkWell(
      borderRadius: BorderRadius.circular(14),
      onTap: () => context.push('/browser/${e.route}'),
      child: Ink(
        decoration: BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: AppColors.border),
        ),
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: AppColors.accent.withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Icon(e.icon, color: AppColors.accent, size: 22),
            ),
            const Spacer(),
            Text(
              context.tr(e.titleKey),
              style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15),
            ),
            const SizedBox(height: 4),
            Expanded(
              child: Text(
                context.tr(e.descKey),
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(color: AppColors.textMuted, fontSize: 12, height: 1.3),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _PanelEntry {
  final String route;
  final String titleKey;
  final String descKey;
  final IconData icon;
  const _PanelEntry({
    required this.route,
    required this.titleKey,
    required this.descKey,
    required this.icon,
  });
}
