import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../../../shared/theme/app_theme.dart';

typedef SendKeyCallback = void Function(String data);

class QuickKey {
  final String label;
  final String data; // escape sequence or text to send
  final Color? color;

  const QuickKey({required this.label, required this.data, this.color});

  Map<String, dynamic> toJson() => {'label': label, 'data': data};
  factory QuickKey.fromJson(Map<String, dynamic> j) => QuickKey(
        label: j['label'] as String? ?? '',
        data: j['data'] as String? ?? '',
      );
}

class QuickKeysBar extends StatefulWidget {
  final SendKeyCallback onSendKey;

  const QuickKeysBar({super.key, required this.onSendKey});

  @override
  State<QuickKeysBar> createState() => _QuickKeysBarState();
}

class _QuickKeysBarState extends State<QuickKeysBar> {
  int _activeTab = 0;
  List<QuickKey> _userCommands = [];

  static const _tabs = ['Keys', 'Ctrl', 'Commands'];
  static const _prefsKey = 'user_quick_commands';

  static const _keyRow = <QuickKey>[
    QuickKey(label: 'Tab', data: '\t'),
    QuickKey(label: 'Esc', data: '\x1b'),
    QuickKey(label: '↑', data: '\x1b[A'),
    QuickKey(label: '↓', data: '\x1b[B'),
    QuickKey(label: '←', data: '\x1b[D'),
    QuickKey(label: '→', data: '\x1b[C'),
    QuickKey(label: 'Home', data: '\x1b[H'),
    QuickKey(label: 'End', data: '\x1b[F'),
  ];

  /// Slimmed down to the most commonly needed ctrl shortcuts.
  static const _ctrlRow = <QuickKey>[
    QuickKey(label: '^C', data: '\x03', color: AppColors.error),
    QuickKey(label: '^D', data: '\x04', color: AppColors.warning),
    QuickKey(label: '^L', data: '\x0c'),
    QuickKey(label: '^R', data: '\x12'),
  ];

  static const _defaultCommands = <QuickKey>[
    QuickKey(label: '/clear', data: '/clear\r'),
    QuickKey(label: '/compact', data: '/compact\r'),
    QuickKey(label: 'yes', data: 'yes\r'),
    QuickKey(label: 'continue', data: 'continue\r'),
    QuickKey(label: 'Paste', data: '__paste__'),
  ];

  @override
  void initState() {
    super.initState();
    _loadCommands();
  }

  Future<void> _loadCommands() async {
    final prefs = await SharedPreferences.getInstance();
    final stored = prefs.getString(_prefsKey);
    if (stored != null) {
      try {
        final list = (jsonDecode(stored) as List).cast<Map<String, dynamic>>();
        setState(() => _userCommands = list.map(QuickKey.fromJson).toList());
        return;
      } catch (_) {}
    }
    setState(() => _userCommands = List.from(_defaultCommands));
  }

  Future<void> _saveCommands() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_prefsKey,
        jsonEncode(_userCommands.map((k) => k.toJson()).toList()));
  }

  Future<void> _handleTap(QuickKey key) async {
    HapticFeedback.selectionClick();
    if (key.data == '__paste__') {
      final clip = await Clipboard.getData('text/plain');
      if (clip?.text?.isNotEmpty == true) widget.onSendKey(clip!.text!);
      return;
    }
    widget.onSendKey(key.data);
  }

  Future<void> _editCommands() async {
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: AppColors.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _CommandEditor(
        commands: List.from(_userCommands),
        onSave: (list) async {
          setState(() => _userCommands = list);
          await _saveCommands();
        },
      ),
    );
  }

  List<QuickKey> get _currentRow => switch (_activeTab) {
        1 => _ctrlRow,
        2 => _userCommands,
        _ => _keyRow,
      };

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(top: BorderSide(color: AppColors.border)),
      ),
      child: SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Tab selector + edit button
            SizedBox(
              height: 32,
              child: Row(
                children: [
                  ...List.generate(_tabs.length, (i) {
                    final active = i == _activeTab;
                    return Expanded(
                      child: GestureDetector(
                        onTap: () => setState(() => _activeTab = i),
                        child: Container(
                          decoration: BoxDecoration(
                            border: Border(
                              bottom: BorderSide(
                                color: active ? AppColors.accent : Colors.transparent,
                                width: 2,
                              ),
                            ),
                          ),
                          child: Center(
                            child: Text(
                              _tabs[i],
                              style: TextStyle(
                                color: active ? AppColors.accent : AppColors.textMuted,
                                fontSize: 11,
                                fontWeight: active ? FontWeight.w600 : FontWeight.normal,
                              ),
                            ),
                          ),
                        ),
                      ),
                    );
                  }),
                  // Edit button (only on Commands tab)
                  if (_activeTab == 2)
                    IconButton(
                      icon: const Icon(Icons.edit, size: 16, color: AppColors.textMuted),
                      onPressed: _editCommands,
                      padding: EdgeInsets.zero,
                      constraints: const BoxConstraints(minWidth: 40, minHeight: 32),
                      tooltip: 'Edit commands',
                    ),
                ],
              ),
            ),
            // Keys
            SizedBox(
              height: 40,
              child: _currentRow.isEmpty
                  ? Center(
                      child: TextButton.icon(
                        onPressed: _editCommands,
                        icon: const Icon(Icons.add, size: 16),
                        label: const Text('Add command', style: TextStyle(fontSize: 12)),
                        style: TextButton.styleFrom(foregroundColor: AppColors.accent),
                      ),
                    )
                  : ListView.separated(
                      scrollDirection: Axis.horizontal,
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      itemCount: _currentRow.length,
                      separatorBuilder: (_, _) => const SizedBox(width: 6),
                      itemBuilder: (_, i) => _buildKey(_currentRow[i]),
                    ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildKey(QuickKey key) {
    final color = key.color ?? AppColors.text;
    return GestureDetector(
      onTap: () => _handleTap(key),
      onLongPress: _activeTab == 2 ? _editCommands : null,
      child: Container(
        constraints: const BoxConstraints(minWidth: 44),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
        decoration: BoxDecoration(
          color: AppColors.surfaceAlt,
          borderRadius: BorderRadius.circular(6),
          border: Border.all(color: AppColors.border),
        ),
        child: Center(
          child: Text(
            key.label,
            style: TextStyle(
              color: color,
              fontSize: 12,
              fontWeight: FontWeight.w500,
              fontFamily: 'monospace',
            ),
          ),
        ),
      ),
    );
  }
}

/// Bottom sheet editor for user custom commands.
class _CommandEditor extends StatefulWidget {
  final List<QuickKey> commands;
  final Future<void> Function(List<QuickKey>) onSave;

  const _CommandEditor({required this.commands, required this.onSave});

  @override
  State<_CommandEditor> createState() => _CommandEditorState();
}

class _CommandEditorState extends State<_CommandEditor> {
  late List<QuickKey> _items;

  @override
  void initState() {
    super.initState();
    _items = List.from(widget.commands);
  }

  Future<void> _addCommand() async {
    final result = await _showEditDialog();
    if (result != null) setState(() => _items.add(result));
  }

  Future<void> _editItem(int i) async {
    final result = await _showEditDialog(initial: _items[i]);
    if (result != null) setState(() => _items[i] = result);
  }

  Future<QuickKey?> _showEditDialog({QuickKey? initial}) async {
    final labelCtrl = TextEditingController(text: initial?.label ?? '');
    final dataCtrl = TextEditingController(text: _unescapeForDisplay(initial?.data ?? ''));

    return showDialog<QuickKey>(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: AppColors.surface,
        title: Text(initial == null ? 'New Command' : 'Edit Command',
            style: const TextStyle(fontSize: 15)),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: labelCtrl,
              decoration: const InputDecoration(
                labelText: 'Label',
                hintText: '/mycommand',
              ),
              style: const TextStyle(fontSize: 13),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: dataCtrl,
              decoration: const InputDecoration(
                labelText: 'Send',
                hintText: '/mycommand\\r  (use \\r for Enter, \\n for newline, \\t for Tab)',
              ),
              style: const TextStyle(fontSize: 13, fontFamily: 'monospace'),
              maxLines: 2,
            ),
            const SizedBox(height: 8),
            const Align(
              alignment: Alignment.centerLeft,
              child: Text(
                'Tip: end with \\r to auto-send (Enter)',
                style: TextStyle(color: AppColors.textMuted, fontSize: 10),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () {
              final label = labelCtrl.text.trim();
              final data = _escapeFromInput(dataCtrl.text);
              if (label.isEmpty || data.isEmpty) return;
              Navigator.pop(ctx, QuickKey(label: label, data: data));
            },
            style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
            child: const Text('Save'),
          ),
        ],
      ),
    );
  }

  /// Convert user-typed escapes (\r \n \t) to real control chars.
  String _escapeFromInput(String s) => s
      .replaceAll(r'\r', '\r')
      .replaceAll(r'\n', '\n')
      .replaceAll(r'\t', '\t')
      .replaceAll(r'\e', '\x1b')
      .replaceAll(r'\\', '\\');

  /// Convert real control chars back to display form for editing.
  String _unescapeForDisplay(String s) => s
      .replaceAll('\\', r'\\')
      .replaceAll('\r', r'\r')
      .replaceAll('\n', r'\n')
      .replaceAll('\t', r'\t')
      .replaceAll('\x1b', r'\e');

  @override
  Widget build(BuildContext context) {
    final bottomInset = MediaQuery.of(context).viewInsets.bottom;
    return Padding(
      padding: EdgeInsets.only(bottom: bottomInset),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          // Grab handle
          Padding(
            padding: const EdgeInsets.only(top: 8, bottom: 4),
            child: Container(
              width: 36, height: 4,
              decoration: BoxDecoration(color: AppColors.border, borderRadius: BorderRadius.circular(2)),
            ),
          ),
          // Header
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 8, 8),
            child: Row(
              children: [
                const Expanded(
                  child: Text('Custom Commands',
                      style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
                ),
                IconButton(
                  icon: const Icon(Icons.add, color: AppColors.accent),
                  onPressed: _addCommand,
                  tooltip: 'Add',
                ),
              ],
            ),
          ),
          const Divider(height: 1, color: AppColors.border),
          // List
          Flexible(
            child: ReorderableListView.builder(
              shrinkWrap: true,
              padding: const EdgeInsets.symmetric(vertical: 4),
              itemCount: _items.length,
              onReorder: (oldIdx, newIdx) {
                setState(() {
                  if (newIdx > oldIdx) newIdx--;
                  final item = _items.removeAt(oldIdx);
                  _items.insert(newIdx, item);
                });
              },
              itemBuilder: (_, i) {
                final k = _items[i];
                return ListTile(
                  key: ValueKey('$i-${k.label}'),
                  dense: true,
                  leading: const Icon(Icons.drag_handle, size: 18, color: AppColors.textMuted),
                  title: Text(k.label, style: const TextStyle(fontSize: 13, fontFamily: 'monospace')),
                  subtitle: Text(_unescapeForDisplay(k.data),
                      style: const TextStyle(fontSize: 10, color: AppColors.textMuted),
                      maxLines: 1, overflow: TextOverflow.ellipsis),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      IconButton(
                        icon: const Icon(Icons.edit, size: 16, color: AppColors.textMuted),
                        onPressed: () => _editItem(i),
                        padding: EdgeInsets.zero,
                        constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
                      ),
                      IconButton(
                        icon: const Icon(Icons.delete_outline, size: 16, color: AppColors.error),
                        onPressed: () => setState(() => _items.removeAt(i)),
                        padding: EdgeInsets.zero,
                        constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
                      ),
                    ],
                  ),
                );
              },
            ),
          ),
          const Divider(height: 1, color: AppColors.border),
          Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              children: [
                Expanded(
                  child: TextButton(
                    onPressed: () => Navigator.pop(context),
                    child: const Text('Cancel'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed: () async {
                      await widget.onSave(_items);
                      if (context.mounted) Navigator.pop(context);
                    },
                    style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
                    child: const Text('Save'),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
