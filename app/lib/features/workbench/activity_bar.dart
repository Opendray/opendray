/// The Workbench activity-bar rail (T17 — M2 plugin platform).
///
/// Renders plugin-contributed icons as tap targets that focus the
/// associated view via [WorkbenchService.openView]. The widget is layout-
/// agnostic — pass [Axis.vertical] for tablet (left rail) or
/// [Axis.horizontal] for phone (bottom nav-style).
///
/// Phone collapse (per `docs/plugin-platform/08-workbench-slots.md`):
///   - ≤4 items  → render inline
///   - >4 items  → first 3 inline + a trailing "More" button that opens
///                 a modal bottom sheet listing the rest
///
/// Icon rendering (M2):
///   - emoji / short unicode strings → rendered as `Text`
///   - plugin asset paths (e.g. "icons/foo.svg") → fallback to the first
///     letter of the title as a `Text`; full asset-backed icons ship in
///     M6 polish (tracked alongside the rest of the theming work).
///
/// Empty list → `SizedBox.shrink()`; the rail adds zero chrome until a
/// plugin contributes.
library;

import 'package:flutter/material.dart';

import 'workbench_models.dart';
import 'workbench_service.dart';

class ActivityBar extends StatelessWidget {
  const ActivityBar({
    required this.service,
    this.axis = Axis.vertical,
    this.onClose,
    super.key,
  });

  /// Workbench service — provides the item list, current view id, and
  /// the `openView` sink. Listened to via [ListenableBuilder].
  final WorkbenchService service;

  /// Rail orientation. Phone calls use `Axis.horizontal` (bottom nav-
  /// style), tablet uses `Axis.vertical` (left rail).
  final Axis axis;

  /// Optional close handler for the host — used only by the phone-sheet
  /// "close" affordance if the rail itself is presented inside a sheet
  /// (no-op by default; the dashboard keeps the rail docked).
  final VoidCallback? onClose;

  /// Visible-inline item budget before overflow into the "More" sheet.
  /// Matches the 08-workbench-slots.md phone rule (4 nav slots: 3 visible
  /// + "More"). We apply the same budget regardless of axis so tablet
  /// behavior matches the phone — tablet rails rarely overflow, and when
  /// they do a consistent menu is preferable to an unbounded scroll.
  static const int _maxInlineItems = 3;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: service,
      builder: (context, _) {
        final items = service.activityBarItems;
        if (items.isEmpty) {
          return const SizedBox.shrink();
        }

        final needsMore = items.length > _maxInlineItems + 1;
        final inline = needsMore ? items.take(_maxInlineItems).toList() : items;
        final overflow = needsMore ? items.skip(_maxInlineItems).toList() : const <WorkbenchActivityBarItem>[];

        final currentViewID = service.currentViewID;
        final theme = Theme.of(context);

        final buttons = <Widget>[
          for (final item in inline)
            _ActivityBarButton(
              item: item,
              selected: currentViewID != null && item.viewId == currentViewID,
              onTap: () => _openItem(item),
              axis: axis,
            ),
          if (needsMore)
            _MoreButton(
              axis: axis,
              onTap: () => _showMoreSheet(context, overflow),
            ),
        ];

        final bg = theme.colorScheme.surfaceContainerLow;
        final border = theme.dividerColor;

        if (axis == Axis.horizontal) {
          return DecoratedBox(
            decoration: BoxDecoration(
              color: bg,
              border: Border(top: BorderSide(color: border, width: 0.5)),
            ),
            child: SafeArea(
              top: false,
              child: SizedBox(
                height: 56,
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                  children: buttons,
                ),
              ),
            ),
          );
        }

        return DecoratedBox(
          decoration: BoxDecoration(
            color: bg,
            border: Border(right: BorderSide(color: border, width: 0.5)),
          ),
          child: SizedBox(
            width: 56,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                const SizedBox(height: 8),
                for (final b in buttons) Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: b,
                ),
              ],
            ),
          ),
        );
      },
    );
  }

  void _openItem(WorkbenchActivityBarItem item) {
    // An activity bar icon without a linked view is a no-op — it acts as
    // a pure tooltip in v1. This mirrors the VS-Code behaviour where an
    // activity item with no viewContainer is legal but invisible-on-tap.
    if (item.viewId.isEmpty) return;
    service.openView(item.viewId);
  }

  Future<void> _showMoreSheet(
    BuildContext context,
    List<WorkbenchActivityBarItem> overflow,
  ) {
    return showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (sheetContext) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: [
            for (final item in overflow)
              ListTile(
                leading: _renderIcon(item, size: 20),
                title: Text(item.title.isEmpty ? item.id : item.title),
                selected: service.currentViewID != null &&
                    service.currentViewID == item.viewId,
                onTap: () {
                  Navigator.of(sheetContext).maybePop();
                  _openItem(item);
                  onClose?.call();
                },
              ),
          ],
        ),
      ),
    );
  }
}

/// Render a plugin-supplied icon. Emoji / short glyph strings go as
/// [Text]; asset paths fall back to the first letter of the title (M2
/// compromise — see library docstring).
Widget _renderIcon(WorkbenchActivityBarItem item, {double size = 22}) {
  final icon = item.icon;
  final looksLikeAssetPath =
      icon.contains('/') || icon.endsWith('.png') || icon.endsWith('.svg');
  final glyph = looksLikeAssetPath || icon.isEmpty
      ? _firstLetter(item.title.isNotEmpty ? item.title : item.id)
      : icon;
  return Text(
    glyph,
    style: TextStyle(fontSize: size),
    textAlign: TextAlign.center,
  );
}

String _firstLetter(String s) {
  if (s.isEmpty) return '?';
  return s.characters.first.toUpperCase();
}

class _ActivityBarButton extends StatelessWidget {
  const _ActivityBarButton({
    required this.item,
    required this.selected,
    required this.onTap,
    required this.axis,
  });

  final WorkbenchActivityBarItem item;
  final bool selected;
  final VoidCallback onTap;
  final Axis axis;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // Minimum tap target: 48x48 logical px on phone (per
    // 08-workbench-slots.md accessibility rules). The rail gives us 56
    // in both axes which already exceeds this bar.
    final bg = selected
        ? theme.colorScheme.primary.withValues(alpha: 0.14)
        : Colors.transparent;
    final borderColor = selected ? theme.colorScheme.primary : Colors.transparent;

    final label = item.title.isEmpty ? item.id : item.title;

    return Semantics(
      label: label,
      selected: selected,
      button: true,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: Tooltip(
          message: label,
          child: Container(
            width: 44,
            height: 44,
            decoration: BoxDecoration(
              color: bg,
              borderRadius: BorderRadius.circular(8),
              border: Border(
                left: axis == Axis.vertical
                    ? BorderSide(color: borderColor, width: 2)
                    : BorderSide.none,
                bottom: axis == Axis.horizontal
                    ? BorderSide(color: borderColor, width: 2)
                    : BorderSide.none,
              ),
            ),
            alignment: Alignment.center,
            child: _renderIcon(item),
          ),
        ),
      ),
    );
  }
}

class _MoreButton extends StatelessWidget {
  const _MoreButton({
    required this.axis,
    required this.onTap,
  });

  final Axis axis;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: 'More',
      button: true,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: const Tooltip(
          message: 'More',
          child: SizedBox(
            width: 44,
            height: 44,
            child: Icon(Icons.more_horiz, size: 22),
          ),
        ),
      ),
    );
  }
}
