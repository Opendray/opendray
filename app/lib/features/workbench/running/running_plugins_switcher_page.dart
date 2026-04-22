import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/services/l10n.dart';
import '../../../shared/theme/app_theme.dart';
import 'running_plugins_models.dart';
import 'running_plugins_service.dart';

/// Full-screen iOS-style app switcher for running plugins.
///
/// Grid of preview cards (each with title, icon, thumbnail, ✕). Tap a
/// card to jump back to the plugin with its state intact. Tap ✕ to
/// actually dispose the plugin's widget — the only path that calls
/// `RunningPluginsService.close`.
class RunningPluginsSwitcherPage extends StatelessWidget {
  const RunningPluginsSwitcherPage({super.key});

  @override
  Widget build(BuildContext context) {
    final service = context.watch<RunningPluginsService>();
    // Recency-sorted copy so the most-recently-used card lands in the
    // top-left slot (standard iOS switcher ordering).
    final entries = [...service.entries]
      ..sort((a, b) => b.lastActiveAt.compareTo(a.lastActiveAt));

    return Scaffold(
      appBar: AppBar(
        title: Text(context.tr('Running')),
        actions: [
          if (entries.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(right: 12),
              child: Center(
                child: Text(
                  '${entries.length}',
                  style: const TextStyle(
                    color: AppColors.textMuted,
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
            ),
        ],
      ),
      body: entries.isEmpty ? _empty(context) : _grid(context, entries),
    );
  }

  Widget _empty(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.layers_outlined,
                size: 48, color: AppColors.textMuted),
            const SizedBox(height: 12),
            Text(
              context.tr('No running plugins'),
              style: const TextStyle(
                fontSize: 14,
                color: AppColors.text,
                fontWeight: FontWeight.w500,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              context.tr('Open a plugin to see it here.'),
              style: const TextStyle(
                fontSize: 12,
                color: AppColors.textMuted,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _grid(BuildContext context, List<RunningPluginEntry> entries) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final w = constraints.maxWidth;
        // Match the responsive breakpoints used by _Shell so a phone
        // gets 2 cols, a tablet 3, a desktop browser 4.
        final int cols = w >= 1280 ? 4 : (w >= 900 ? 3 : 2);
        return GridView.builder(
          padding: const EdgeInsets.all(16),
          itemCount: entries.length,
          gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: cols,
            crossAxisSpacing: 12,
            mainAxisSpacing: 12,
            childAspectRatio: 3 / 4,
          ),
          itemBuilder: (context, index) => _SwitcherCard(entry: entries[index]),
        );
      },
    );
  }
}

class _SwitcherCard extends StatelessWidget {
  final RunningPluginEntry entry;
  const _SwitcherCard({required this.entry});

  @override
  Widget build(BuildContext context) {
    final title = context.tr(entry.titleKey);
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(10),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () {
          final service = context.read<RunningPluginsService>();
          service.setActive(entry.id);
          context.go(entry.route);
        },
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(child: _Preview(entry: entry)),
            _Footer(entry: entry, title: title),
          ],
        ),
      ),
    );
  }
}

class _Preview extends StatelessWidget {
  final RunningPluginEntry entry;
  const _Preview({required this.entry});

  @override
  Widget build(BuildContext context) {
    // Step 1 ships only the icon-placeholder path. ImageThumbnail is
    // handled here so later steps can light up the real preview
    // without touching this widget.
    switch (entry.thumbnail) {
      case ImageThumbnail t:
        return Image.memory(
          t.pngBytes,
          fit: BoxFit.cover,
          gaplessPlayback: true,
        );
      case PendingThumbnail _:
      case IconThumbnail _:
        return Container(
          color: AppColors.surfaceAlt,
          alignment: Alignment.center,
          child: Icon(entry.icon, size: 40, color: AppColors.textMuted),
        );
    }
  }
}

class _Footer extends StatelessWidget {
  final RunningPluginEntry entry;
  final String title;
  const _Footer({required this.entry, required this.title});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(10, 6, 4, 6),
      decoration: const BoxDecoration(
        border: Border(top: BorderSide(color: AppColors.border, width: 1)),
      ),
      child: Row(
        children: [
          Icon(entry.icon, size: 14, color: AppColors.textMuted),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w500,
                color: AppColors.text,
              ),
            ),
          ),
          IconButton(
            visualDensity: VisualDensity.compact,
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 28, minHeight: 28),
            iconSize: 16,
            icon: const Icon(Icons.close, color: AppColors.textMuted),
            tooltip: context.tr('Close'),
            onPressed: () =>
                context.read<RunningPluginsService>().close(entry.id),
          ),
        ],
      ),
    );
  }
}
