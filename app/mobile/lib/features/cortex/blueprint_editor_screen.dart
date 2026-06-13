import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/cortex_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// BlueprintEditorScreen — edit a project's doc blueprint (the section set
// behind its Notes/Project docs): rename sections, switch maintainer mode,
// toggle injection, reorder, add/remove, or ask the AI to propose a set
// tailored to the project. Mirrors the web BlueprintEditor; reuses the
// existing cortex_api listSections/proposeBlueprint/applyBlueprint.
class BlueprintEditorScreen extends ConsumerStatefulWidget {
  const BlueprintEditorScreen({required this.cwd, super.key});

  final String cwd;

  @override
  ConsumerState<BlueprintEditorScreen> createState() =>
      _BlueprintEditorScreenState();
}

// A draft row with a stable id (for keys/controllers across reorders) and
// its own title controller, so editing + moving rows don't fight.
class _Row {
  _Row(this.id, this.section) : titleCtrl = TextEditingController(text: section.title);
  final int id;
  BlueprintSection section;
  final TextEditingController titleCtrl;
}

class _BlueprintEditorScreenState extends ConsumerState<BlueprintEditorScreen> {
  AsyncValue<void> _init = const AsyncValue.loading();
  final List<_Row> _rows = [];
  int _nextId = 0;
  bool _busy = false;
  String? _proposalNote;

  static final _slugRe = RegExp(r'^[a-z0-9][a-z0-9_]{0,44}$');

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    for (final r in _rows) {
      r.titleCtrl.dispose();
    }
    super.dispose();
  }

  void _setRows(List<BlueprintSection> sections) {
    for (final r in _rows) {
      r.titleCtrl.dispose();
    }
    _rows
      ..clear()
      ..addAll([for (final s in sections) _Row(_nextId++, s)]);
  }

  Future<void> _load() async {
    setState(() => _init = const AsyncValue.loading());
    try {
      final secs = await ref.read(cortexApiProvider).listSections(widget.cwd);
      secs.sort((a, b) => a.position.compareTo(b.position));
      if (!mounted) return;
      setState(() {
        _setRows(secs);
        _init = const AsyncValue.data(null);
      });
    } on Object catch (e, st) {
      if (mounted) setState(() => _init = AsyncValue.error(e, st));
    }
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  Future<void> _propose() async {
    setState(() => _busy = true);
    try {
      final p = await ref.read(cortexApiProvider).proposeBlueprint(widget.cwd);
      if (!mounted) return;
      setState(() {
        _setRows([for (final s in p.sections) s.copyWith()]);
        _proposalNote = t.web.cortex.blueprint
            .proposalNote(type: p.projectType, reason: p.reason);
      });
    } on Object catch (e) {
      _snack(t.web.cortex.blueprint.proposeFailed);
      _snack(e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _apply() async {
    // Re-number positions from current row order before sending.
    final sections = [
      for (var i = 0; i < _rows.length; i++)
        _rows[i].section.copyWith(position: i),
    ];
    setState(() => _busy = true);
    try {
      await ref.read(cortexApiProvider).applyBlueprint(widget.cwd, sections);
      if (!mounted) return;
      _snack(t.web.cortex.blueprint.appliedToast);
      Navigator.of(context).pop(true);
    } on Object catch (e) {
      _snack(t.web.cortex.blueprint.applyFailed);
      _snack(e.toString());
      if (mounted) setState(() => _busy = false);
    }
  }

  void _addSection() {
    setState(() => _rows.add(_Row(
          _nextId++,
          BlueprintSection(
            cwd: widget.cwd,
            slug: '',
            title: '',
            description: '',
            position: _rows.length,
            maintainerMode: 'ai',
            promptHint: '',
            pinned: false,
            inject: true,
          ),
        )));
  }

  void _move(int i, int dir) {
    final j = i + dir;
    if (j < 0 || j >= _rows.length) return;
    setState(() {
      final r = _rows.removeAt(i);
      _rows.insert(j, r);
    });
  }

  bool get _valid => _rows.every((r) {
        final s = r.section;
        return _slugRe.hasMatch(s.slug) &&
            !s.slug.startsWith('kb_') &&
            r.titleCtrl.text.trim().isNotEmpty;
      });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.web.cortex.blueprint.title),
        actions: [
          TextButton.icon(
            onPressed: _busy ? null : _propose,
            icon: const Icon(Icons.auto_awesome, size: 16),
            label: Text(t.web.cortex.blueprint.propose),
          ),
        ],
      ),
      body: _init.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('$e', textAlign: TextAlign.center),
                const SizedBox(height: 12),
                FilledButton(onPressed: _load, child: Text(t.common.retry)),
              ],
            ),
          ),
        ),
        data: (_) => Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
              child: Text(
                _proposalNote ?? t.web.cortex.blueprint.description,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Theme.of(context)
                          .colorScheme
                          .onSurface
                          .withValues(alpha: 0.65),
                    ),
              ),
            ),
            Expanded(
              child: ListView.builder(
                padding: const EdgeInsets.fromLTRB(8, 4, 8, 96),
                itemCount: _rows.length,
                itemBuilder: (context, i) => _sectionCard(i),
              ),
            ),
          ],
        ),
      ),
      floatingActionButton: _init.hasValue
          ? FloatingActionButton.extended(
              heroTag: 'blueprint_apply',
              onPressed: (_busy || !_valid) ? null : _apply,
              icon: _busy
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.check),
              label: Text(t.web.cortex.blueprint.apply),
            )
          : null,
      persistentFooterButtons: _init.hasValue
          ? [
              TextButton.icon(
                onPressed: _addSection,
                icon: const Icon(Icons.add, size: 18),
                label: Text(t.web.cortex.blueprint.addSection),
              ),
            ]
          : null,
    );
  }

  Widget _sectionCard(int i) {
    final row = _rows[i];
    final s = row.section;
    return Card(
      key: ValueKey('bp_${row.id}'),
      margin: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 8, 8, 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: row.titleCtrl,
                    decoration: InputDecoration(
                      isDense: true,
                      hintText: t.web.cortex.blueprint.titlePlaceholder,
                    ),
                    onChanged: (v) {
                      row.section = s.copyWith(title: v);
                      setState(() {}); // refresh _valid
                    },
                  ),
                ),
                IconButton(
                  visualDensity: VisualDensity.compact,
                  icon: const Icon(Icons.keyboard_arrow_up, size: 20),
                  onPressed: i == 0 ? null : () => _move(i, -1),
                ),
                IconButton(
                  visualDensity: VisualDensity.compact,
                  icon: const Icon(Icons.keyboard_arrow_down, size: 20),
                  onPressed: i == _rows.length - 1 ? null : () => _move(i, 1),
                ),
                IconButton(
                  visualDensity: VisualDensity.compact,
                  icon: Icon(Icons.delete_outline,
                      size: 20, color: Theme.of(context).colorScheme.error),
                  onPressed: () => setState(() {
                    _rows.removeAt(i).titleCtrl.dispose();
                  }),
                ),
              ],
            ),
            Row(
              children: [
                Expanded(
                  child: TextFormField(
                    initialValue: s.slug,
                    decoration: const InputDecoration(
                      isDense: true,
                      prefixText: 'slug: ',
                    ),
                    style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
                    onChanged: (v) => setState(() =>
                        row.section = s.copyWith(slug: v.trim())),
                  ),
                ),
                const SizedBox(width: 8),
                DropdownButton<String>(
                  value: s.maintainerMode,
                  underline: const SizedBox.shrink(),
                  items: [
                    DropdownMenuItem(
                        value: 'ai', child: Text(t.web.cortex.blueprint.mode.ai)),
                    DropdownMenuItem(
                        value: 'human',
                        child: Text(t.web.cortex.blueprint.mode.human)),
                    DropdownMenuItem(
                        value: 'scanner',
                        child: Text(t.web.cortex.blueprint.mode.scanner)),
                  ],
                  onChanged: (v) => setState(() =>
                      row.section = s.copyWith(maintainerMode: v ?? 'ai')),
                ),
              ],
            ),
            Row(
              children: [
                Switch(
                  value: s.inject,
                  onChanged: (v) =>
                      setState(() => row.section = s.copyWith(inject: v)),
                ),
                Text(t.web.cortex.blueprint.inject,
                    style: Theme.of(context).textTheme.bodySmall),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
