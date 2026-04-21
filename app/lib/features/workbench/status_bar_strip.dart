/// A thin footer strip that surfaces plugin-contributed status-bar items
/// (the VS-Code-style row along the bottom of the workbench). T20 of the
/// M1 plugin-platform plan.
///
/// The widget listens to a [StatusBarSource] — a tiny surface that
/// decouples the strip from the still-under-construction `WorkbenchService`
/// (T19). Any `ChangeNotifier` that exposes `statusBarItems` + `invoke`
/// satisfies it, so the real service can be wired in later without
/// touching this file.
///
/// Rendering:
///   left-group ── Spacer ── right-group
///
/// Within each group items are sorted by `priority` descending (mirrors
/// the server's own sort in `plugin/contributions/registry.go`; sorting
/// again here makes the widget robust to future server changes and keeps
/// tests self-contained). Items with alignment other than `"left"` or
/// `"right"` fall into the right group — matches the T23 default.
///
/// An empty item list collapses to `SizedBox.shrink` — no divider, no
/// padding, no chrome. The parent page layout should therefore not break
/// if no plugin contributed a chip.
library;

import 'package:flutter/material.dart';

import 'workbench_models.dart';

/// Minimum surface the [StatusBarStrip] needs. T19's `WorkbenchService`
/// (extends [ChangeNotifier] and exposes `statusBarItems` + `invoke`)
/// will satisfy this automatically, which lets T20 ship in parallel with
/// T19 without importing it.
abstract class StatusBarSource implements Listenable {
  List<WorkbenchStatusBarItem> get statusBarItems;

  /// Fire-and-forget command invocation. Callers don't await — this is
  /// a UI tap path and any failure is surfaced by the service itself
  /// (e.g. via a SnackBar driven by the notify InvokeResult).
  Future<void> invoke(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  });
}

/// A [StatusBarSource] that reports zero items and no-ops on invoke.
/// Handy for pages that want to reserve the strip slot before the real
/// [WorkbenchService] is wired up (M1 scope ends before main.dart
/// integration).
class NullStatusBarSource extends ChangeNotifier implements StatusBarSource {
  @override
  List<WorkbenchStatusBarItem> get statusBarItems => const [];

  @override
  Future<void> invoke(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  }) async {
    // Intentional no-op — no plugin source is wired yet.
  }
}

/// A footer strip rendering plugin-contributed status-bar chips. See the
/// library docstring for layout rules.
class StatusBarStrip extends StatelessWidget {
  final StatusBarSource source;
  final bool showLabels;
  final EdgeInsetsGeometry padding;
  final double height;

  const StatusBarStrip({
    required this.source,
    this.showLabels = true,
    this.padding = const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
    this.height = 28,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: source,
      builder: (context, _) {
        final items = source.statusBarItems;
        if (items.isEmpty) {
          // Zero-height: occupy no space at all when there's nothing
          // to show, so pages don't end up with an empty stripe.
          return const SizedBox.shrink();
        }

        final left = <WorkbenchStatusBarItem>[];
        final right = <WorkbenchStatusBarItem>[];
        for (final it in items) {
          if (it.alignment == 'left') {
            left.add(it);
          } else {
            // "right" or anything else → right group (T23 default).
            right.add(it);
          }
        }

        // Priority DESC, stable for equal priorities. Matches the server-
        // side sort in plugin/contributions/registry.go; sorting locally
        // keeps the widget defensive + tests self-contained.
        int cmp(WorkbenchStatusBarItem a, WorkbenchStatusBarItem b) =>
            b.priority.compareTo(a.priority);
        left.sort(cmp);
        right.sort(cmp);

        final theme = Theme.of(context);
        final bg = theme.colorScheme.surfaceContainerLow;
        final divider = theme.dividerColor;

        return DecoratedBox(
          decoration: BoxDecoration(
            color: bg,
            border: Border(top: BorderSide(color: divider, width: 0.5)),
          ),
          child: SizedBox(
            height: height,
            child: Padding(
              padding: padding,
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  for (final it in left) _StatusChip(
                    item: it,
                    showLabel: showLabels,
                    onTap: () => source.invoke(it.pluginName, it.command),
                  ),
                  const Spacer(),
                  for (final it in right) _StatusChip(
                    item: it,
                    showLabel: showLabels,
                    onTap: () => source.invoke(it.pluginName, it.command),
                  ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}

class _StatusChip extends StatelessWidget {
  final WorkbenchStatusBarItem item;
  final bool showLabel;
  final VoidCallback onTap;

  const _StatusChip({
    required this.item,
    required this.showLabel,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final label = Text(
      showLabel ? item.text : '',
      style: theme.textTheme.bodySmall?.copyWith(
        fontSize: 11,
        color: theme.colorScheme.onSurface,
      ),
      maxLines: 1,
      overflow: TextOverflow.ellipsis,
    );

    final child = Tooltip(
      message: item.tooltip,
      child: label,
    );

    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(4),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
        child: child,
      ),
    );
  }
}
