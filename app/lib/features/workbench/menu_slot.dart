/// A menu slot (T22) that renders plugin-contributed menu entries for a
/// named slot (e.g. `"appBar/right"`) as a [PopupMenuButton].
///
/// Behavior:
///   - Reads entries from a [MenuSource] via [ListenableBuilder], so
///     updates to the underlying `WorkbenchService` (install / uninstall
///     a plugin) re-render the button automatically.
///   - Empty entry list → collapses to [SizedBox.shrink()] (no icon,
///     no space). Pages can drop a slot anywhere without worrying about
///     reserving layout room for plugins that haven't been installed.
///   - Non-empty → renders a [PopupMenuButton] with a configurable
///     [icon]. Each entry becomes a [PopupMenuItem] labelled with the
///     entry's raw `command` id. (M1 decision: `WorkbenchMenuEntry` has
///     no `title` field — deferred to M6 alongside localization.)
///   - Entries are sorted stably by `group` ascending, with the empty
///     group sorted *last*. This matches the usual "ungrouped goes at
///     the bottom" UX pattern.
///   - Selecting an entry calls `source.invoke(pluginName, command)`
///     fire-and-forget (the `onSelected` callback returns sync).
library;

import 'package:flutter/material.dart';

import 'workbench_models.dart';
import 'workbench_sources.dart';

class MenuSlot extends StatelessWidget {
  const MenuSlot({
    required this.id,
    required this.source,
    this.icon = const Icon(Icons.more_vert),
    super.key,
  });

  /// Slot id — contributions arrive keyed by this string from
  /// `/api/workbench/contributions`.
  final String id;

  /// Decoupled source. Typically a `WorkbenchMenuSource` wrapping a
  /// `WorkbenchService`, but tests drive this directly with a fake.
  final MenuSource source;

  /// Icon shown on the [PopupMenuButton] when at least one entry is
  /// contributed. Defaults to [Icons.more_vert].
  final Icon icon;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: source,
      builder: (context, _) {
        final entries = source.entriesFor(id);
        if (entries.isEmpty) {
          return const SizedBox.shrink();
        }

        final sorted = _sortEntries(entries);

        return PopupMenuButton<WorkbenchMenuEntry>(
          icon: icon,
          onSelected: (entry) {
            // Fire-and-forget — UI stays responsive. The service
            // surfaces any error via its showMessage channel.
            source.invoke(entry.pluginName, entry.command);
          },
          itemBuilder: (context) => [
            for (final entry in sorted)
              PopupMenuItem<WorkbenchMenuEntry>(
                value: entry,
                child: Text(_labelFor(entry)),
              ),
          ],
        );
      },
    );
  }

  /// Stable sort by `group` asc, with empty group bucketed *after* all
  /// grouped entries. Dart's `List.sort` is stable since Dart 2.x, so
  /// two entries with identical group keys keep their original order.
  List<WorkbenchMenuEntry> _sortEntries(List<WorkbenchMenuEntry> input) {
    final copy = [...input];
    copy.sort((a, b) {
      final aEmpty = a.group.isEmpty;
      final bEmpty = b.group.isEmpty;
      if (aEmpty && bEmpty) return 0;
      if (aEmpty) return 1; // empty group → later
      if (bEmpty) return -1;
      return a.group.compareTo(b.group);
    });
    return copy;
  }

  /// Renders the menu item label. M1: raw `command` id (unambiguous,
  /// matches what the command palette shows). If the entry is a submenu
  /// declaration (command empty, submenu set), fall back to the submenu
  /// id — M1 doesn't render nested submenus but we still want a
  /// readable label rather than a blank row.
  String _labelFor(WorkbenchMenuEntry entry) {
    if (entry.command.isNotEmpty) return entry.command;
    if (entry.submenu.isNotEmpty) return entry.submenu;
    return entry.pluginName;
  }
}
