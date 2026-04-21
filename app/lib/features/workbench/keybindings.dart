import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

import 'workbench_models.dart';
import 'workbench_service.dart';

/// Parser for VS Code-style keybinding strings (`"ctrl+alt+p"`).
///
/// Returns null if the combo has no non-modifier key, mentions an
/// unknown key name, or is otherwise unparseable. Callers treat a
/// null result as "skip this binding" rather than surfacing an error.
LogicalKeySet? parseKeyCombo(String combo) {
  final trimmed = combo.trim();
  if (trimmed.isEmpty) return null;

  final parts = trimmed
      .split('+')
      .map((s) => s.trim().toLowerCase())
      .where((s) => s.isNotEmpty)
      .toList();
  if (parts.isEmpty) return null;

  final keys = <LogicalKeyboardKey>{};
  var hasNonModifier = false;
  for (final part in parts) {
    final modifier = _modifierFor(part);
    if (modifier != null) {
      keys.add(modifier);
      continue;
    }
    final key = _keyFor(part);
    if (key == null) return null; // unknown name → unparseable
    keys.add(key);
    hasNonModifier = true;
  }
  if (!hasNonModifier) return null;
  return LogicalKeySet.fromSet(keys);
}

LogicalKeyboardKey? _modifierFor(String name) {
  switch (name) {
    case 'ctrl':
    case 'control':
      return LogicalKeyboardKey.control;
    case 'alt':
    case 'option':
      return LogicalKeyboardKey.alt;
    case 'shift':
      return LogicalKeyboardKey.shift;
    case 'meta':
    case 'cmd':
    case 'command':
    case 'win':
    case 'super':
      return LogicalKeyboardKey.meta;
  }
  return null;
}

LogicalKeyboardKey? _keyFor(String name) {
  // Single printable chars: letters + digits.
  if (name.length == 1) {
    final cu = name.codeUnitAt(0);
    if (cu >= 0x61 && cu <= 0x7a) {
      // a-z → keyA..keyZ
      return LogicalKeyboardKey(0x00000000061 + (cu - 0x61));
    }
    if (cu >= 0x30 && cu <= 0x39) {
      return LogicalKeyboardKey(0x00000000030 + (cu - 0x30));
    }
  }
  // Function keys f1..f12
  final fMatch = RegExp(r'^f(\d{1,2})$').firstMatch(name);
  if (fMatch != null) {
    final n = int.parse(fMatch.group(1)!);
    if (n >= 1 && n <= 12) {
      const fKeys = <LogicalKeyboardKey>[
        LogicalKeyboardKey.f1,
        LogicalKeyboardKey.f2,
        LogicalKeyboardKey.f3,
        LogicalKeyboardKey.f4,
        LogicalKeyboardKey.f5,
        LogicalKeyboardKey.f6,
        LogicalKeyboardKey.f7,
        LogicalKeyboardKey.f8,
        LogicalKeyboardKey.f9,
        LogicalKeyboardKey.f10,
        LogicalKeyboardKey.f11,
        LogicalKeyboardKey.f12,
      ];
      return fKeys[n - 1];
    }
  }
  switch (name) {
    case 'escape':
    case 'esc':
      return LogicalKeyboardKey.escape;
    case 'enter':
    case 'return':
      return LogicalKeyboardKey.enter;
    case 'space':
      return LogicalKeyboardKey.space;
    case 'tab':
      return LogicalKeyboardKey.tab;
    case 'backspace':
      return LogicalKeyboardKey.backspace;
    case 'delete':
    case 'del':
      return LogicalKeyboardKey.delete;
    case 'up':
    case 'arrowup':
      return LogicalKeyboardKey.arrowUp;
    case 'down':
    case 'arrowdown':
      return LogicalKeyboardKey.arrowDown;
    case 'left':
    case 'arrowleft':
      return LogicalKeyboardKey.arrowLeft;
    case 'right':
    case 'arrowright':
      return LogicalKeyboardKey.arrowRight;
    case 'home':
      return LogicalKeyboardKey.home;
    case 'end':
      return LogicalKeyboardKey.end;
    case 'pageup':
      return LogicalKeyboardKey.pageUp;
    case 'pagedown':
      return LogicalKeyboardKey.pageDown;
  }
  return null;
}

/// Binds every contributed [WorkbenchKeybinding] to a
/// `CallbackShortcuts` entry that invokes the underlying command via
/// [WorkbenchService.invoke]. Rebuilds whenever the service notifies
/// (installs, uninstalls, refreshes).
class WorkbenchKeybindings extends StatefulWidget {
  const WorkbenchKeybindings({
    required this.service,
    required this.child,
    super.key,
  });

  final WorkbenchService service;
  final Widget child;

  @override
  State<WorkbenchKeybindings> createState() => _WorkbenchKeybindingsState();
}

class _WorkbenchKeybindingsState extends State<WorkbenchKeybindings> {
  @override
  void initState() {
    super.initState();
    widget.service.addListener(_onServiceChanged);
  }

  @override
  void didUpdateWidget(covariant WorkbenchKeybindings oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.service != widget.service) {
      oldWidget.service.removeListener(_onServiceChanged);
      widget.service.addListener(_onServiceChanged);
    }
  }

  @override
  void dispose() {
    widget.service.removeListener(_onServiceChanged);
    super.dispose();
  }

  void _onServiceChanged() {
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final bindings = _buildMap(widget.service.keybindings);
    if (bindings.isEmpty) return widget.child;
    return CallbackShortcuts(bindings: bindings, child: widget.child);
  }

  Map<ShortcutActivator, VoidCallback> _buildMap(
    List<WorkbenchKeybinding> keybindings,
  ) {
    final isMac = defaultTargetPlatform == TargetPlatform.macOS;
    final map = <ShortcutActivator, VoidCallback>{};
    for (final kb in keybindings) {
      final combo = isMac && kb.mac.isNotEmpty ? kb.mac : kb.key;
      final keySet = parseKeyCombo(combo);
      if (keySet == null) continue;
      map[keySet] = () {
        widget.service.invoke(kb.pluginName, kb.command);
      };
    }
    return map;
  }
}
