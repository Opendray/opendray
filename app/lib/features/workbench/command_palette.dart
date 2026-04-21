import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'workbench_models.dart';
import 'workbench_service.dart';

/// A searchable list of every plugin-contributed command.
///
/// Opens as a centered dialog (Material's built-in modal-barrier +
/// focus scope gives us autofocus, Esc-to-dismiss, and accessibility
/// announcement for free, which an `Overlay.insert` wouldn't).
class CommandPalette extends StatefulWidget {
  const CommandPalette({required this.service, super.key});

  final WorkbenchService service;

  /// Convenience opener. Pops itself on selection, Esc, or barrier tap.
  static Future<void> show(
    BuildContext context,
    WorkbenchService service,
  ) {
    return showDialog<void>(
      context: context,
      barrierLabel: 'Command Palette',
      builder: (_) => CommandPalette(service: service),
    );
  }

  @override
  State<CommandPalette> createState() => _CommandPaletteState();
}

class _CommandPaletteState extends State<CommandPalette> {
  final _queryCtrl = TextEditingController();
  final _queryFocus = FocusNode();
  final _listFocus = FocusNode(skipTraversal: true);
  int _selected = 0;

  @override
  void initState() {
    super.initState();
    _queryCtrl.addListener(() {
      // Query changed → reset selection to the top match.
      setState(() => _selected = 0);
    });
    widget.service.addListener(_onServiceChanged);
  }

  @override
  void dispose() {
    widget.service.removeListener(_onServiceChanged);
    _queryCtrl.dispose();
    _queryFocus.dispose();
    _listFocus.dispose();
    super.dispose();
  }

  void _onServiceChanged() {
    if (mounted) setState(() {});
  }

  List<WorkbenchCommand> get _filtered {
    final q = _queryCtrl.text.trim().toLowerCase();
    final all = widget.service.commands;
    if (q.isEmpty) return all;
    // Split on whitespace so "pom start" matches commands that contain
    // both substrings anywhere in title or plugin name.
    final terms = q.split(RegExp(r'\s+'));
    return all.where((c) {
      final hay = '${c.title.toLowerCase()} ${c.pluginName.toLowerCase()}';
      for (final t in terms) {
        if (!hay.contains(t)) return false;
      }
      return true;
    }).toList(growable: false);
  }

  KeyEventResult _handleKey(FocusNode _, KeyEvent event) {
    if (event is! KeyDownEvent && event is! KeyRepeatEvent) {
      return KeyEventResult.ignored;
    }
    final filtered = _filtered;
    if (event.logicalKey == LogicalKeyboardKey.arrowDown) {
      if (filtered.isEmpty) return KeyEventResult.handled;
      setState(() => _selected = (_selected + 1) % filtered.length);
      return KeyEventResult.handled;
    }
    if (event.logicalKey == LogicalKeyboardKey.arrowUp) {
      if (filtered.isEmpty) return KeyEventResult.handled;
      setState(() =>
          _selected = (_selected - 1 + filtered.length) % filtered.length);
      return KeyEventResult.handled;
    }
    if (event.logicalKey == LogicalKeyboardKey.enter ||
        event.logicalKey == LogicalKeyboardKey.numpadEnter) {
      if (filtered.isEmpty) return KeyEventResult.handled;
      _invoke(filtered[_selected.clamp(0, filtered.length - 1)]);
      return KeyEventResult.handled;
    }
    if (event.logicalKey == LogicalKeyboardKey.escape) {
      Navigator.of(context).maybePop();
      return KeyEventResult.handled;
    }
    return KeyEventResult.ignored;
  }

  void _invoke(WorkbenchCommand cmd) {
    // Fire-and-forget: service surfaces its own toasts.
    widget.service.invoke(cmd.pluginName, cmd.id);
    Navigator.of(context).maybePop();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final filtered = _filtered;
    final hasAnyCommands = widget.service.commands.isNotEmpty;

    return Dialog(
      alignment: Alignment.topCenter,
      insetPadding: const EdgeInsets.only(top: 80, left: 16, right: 16),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 640, maxHeight: 520),
        child: Focus(
          focusNode: _listFocus,
          onKeyEvent: _handleKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.all(12),
                child: TextField(
                  controller: _queryCtrl,
                  focusNode: _queryFocus,
                  autofocus: true,
                  decoration: const InputDecoration(
                    prefixIcon: Icon(Icons.search),
                    hintText: 'Type a command…',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
              ),
              const Divider(height: 1),
              Flexible(
                child: !hasAnyCommands
                    ? _EmptyState(
                        text: 'No plugins installed with commands',
                        theme: theme,
                      )
                    : filtered.isEmpty
                        ? _EmptyState(text: 'No commands match', theme: theme)
                        : ListView.builder(
                            shrinkWrap: true,
                            itemCount: filtered.length,
                            itemBuilder: (ctx, i) {
                              final cmd = filtered[i];
                              final selected = i == _selected.clamp(
                                  0, filtered.length - 1);
                              return _CommandTile(
                                command: cmd,
                                selected: selected,
                                onTap: () => _invoke(cmd),
                              );
                            },
                          ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _CommandTile extends StatelessWidget {
  const _CommandTile({
    required this.command,
    required this.selected,
    required this.onTap,
  });

  final WorkbenchCommand command;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: selected
          ? theme.colorScheme.primary.withValues(alpha: 0.12)
          : Colors.transparent,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      command.title.isEmpty ? command.id : command.title,
                      style: theme.textTheme.bodyLarge,
                    ),
                    if (command.category.isNotEmpty || command.pluginName.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 2),
                        child: Text(
                          [
                            if (command.category.isNotEmpty) command.category,
                            command.pluginName,
                          ].join(' · '),
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurface
                                .withValues(alpha: 0.6),
                          ),
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.text, required this.theme});
  final String text;
  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 48, horizontal: 16),
      child: Center(
        child: Text(
          text,
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
          ),
        ),
      ),
    );
  }
}
