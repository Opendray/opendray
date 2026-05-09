import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/features/notes/note_editor_dialog.dart';
import 'package:path/path.dart' as p;

// Global Notes tab — full vault browser.
//
// The session inspector's Notes tab handles per-cwd authoring; this
// screen covers the cases where the operator wants to read or edit a
// note that isn't bound to the current session: a personal scratchpad
// from another project, a project doc whose session isn't running,
// anything reached by typing the path or scrolling rather than via a
// cwd lookup.
//
// Three top-level chips narrow the list:
//   • All       — every .md in the vault
//   • Personal  — paths starting with `personal/` (human scratchpads)
//   • Projects  — paths starting with `projects/` (AI agent docs)
//
// Tap a row → opens the shared NoteEditorDialog (debounced auto-save,
// same component the inspector uses). Long-press → action sheet
// (Open / Copy path / Delete). FAB creates a note at an arbitrary
// vault-relative path.
class NotesScreen extends ConsumerStatefulWidget {
  const NotesScreen({super.key});

  @override
  ConsumerState<NotesScreen> createState() => _NotesScreenState();
}

class _NotesScreenState extends ConsumerState<NotesScreen> {
  AsyncValue<List<NoteSummary>> _state = const AsyncValue.loading();
  _Filter _filter = _Filter.all;
  String _query = '';
  final _searchCtrl = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final notes = await ref.read(notesApiProvider).list();
      if (!mounted) return;
      // Sort newest-modified first — matches what the operator usually
      // wants ("the note I just touched").
      notes.sort((a, b) => b.modified.compareTo(a.modified));
      setState(() => _state = AsyncValue.data(notes));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _openNote(NoteSummary note) async {
    await NoteEditorDialog.show(context: context, path: note.path);
    if (!mounted) return;
    // Reload so the row's modified timestamp reflects any save that
    // happened in the dialog.
    await _load();
  }

  Future<void> _onLongPress(NoteSummary note) async {
    final action = await showModalBottomSheet<_RowAction>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    note.title.isNotEmpty ? note.title : p.basename(note.path),
                    style: Theme.of(sheetCtx).textTheme.titleSmall,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 2),
                  Text(
                    note.path,
                    style: Theme.of(sheetCtx).textTheme.bodySmall,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.edit_outlined),
              title: const Text('Open'),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.open),
            ),
            ListTile(
              leading: const Icon(Icons.copy),
              title: const Text('Copy path'),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.copyPath),
            ),
            ListTile(
              leading: Icon(
                Icons.delete_outline,
                color: Theme.of(sheetCtx).colorScheme.error,
              ),
              title: Text(
                'Delete',
                style: TextStyle(color: Theme.of(sheetCtx).colorScheme.error),
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.delete),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    switch (action) {
      case _RowAction.open:
        await _openNote(note);
      case _RowAction.copyPath:
        await Clipboard.setData(ClipboardData(text: note.path));
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Copied ${note.path}'),
            duration: const Duration(seconds: 2),
            behavior: SnackBarBehavior.floating,
          ),
        );
      case _RowAction.delete:
        await _confirmAndDelete(note);
    }
  }

  Future<void> _confirmAndDelete(NoteSummary note) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (dialogCtx) => AlertDialog(
        title: const Text('Delete note?'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              note.title.isNotEmpty ? note.title : p.basename(note.path),
              style: Theme.of(dialogCtx).textTheme.bodyMedium,
            ),
            const SizedBox(height: 4),
            Text(
              note.path,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
            ),
            const SizedBox(height: 8),
            Text(
              'This is irreversible. Vault git sync will remove the file '
              'from the next commit.',
              style: Theme.of(dialogCtx).textTheme.bodySmall,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogCtx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(dialogCtx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(dialogCtx).pop(true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(notesApiProvider).delete(note.path);
      messenger.showSnackBar(
        SnackBar(
          content: Text('Deleted ${note.path}'),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text('Delete failed: ${e.message}')),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text('Delete failed: $e')));
    }
  }

  Future<void> _newNote() async {
    final path = await showDialog<String>(
      context: context,
      builder: (_) => const _NewNoteDialog(),
    );
    if (path == null || path.isEmpty || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(notesApiProvider).write(
            path: path,
            body: '# ${p.basenameWithoutExtension(path)}\n\n',
          );
      if (!mounted) return;
      await NoteEditorDialog.show(context: context, path: path);
      if (!mounted) return;
      await _load();
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text('Create failed: ${e.message}')),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text('Create failed: $e')));
    }
  }

  List<NoteSummary> _filtered(List<NoteSummary> notes) {
    Iterable<NoteSummary> rows = notes;
    switch (_filter) {
      case _Filter.all:
        break;
      case _Filter.personal:
        rows = rows.where((n) => n.path.startsWith('personal/'));
      case _Filter.projects:
        rows = rows.where((n) => n.path.startsWith('projects/'));
    }
    if (_query.isNotEmpty) {
      final q = _query.toLowerCase();
      rows = rows.where(
        (n) =>
            n.path.toLowerCase().contains(q) ||
            n.title.toLowerCase().contains(q),
      );
    }
    return rows.toList(growable: false);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Notes'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: _load,
          ),
        ],
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(48),
          child: _FilterStrip(
            value: _filter,
            onChanged: (f) => setState(() => _filter = f),
          ),
        ),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
            child: TextField(
              controller: _searchCtrl,
              onChanged: (v) =>
                  setState(() => _query = v.trim().toLowerCase()),
              decoration: InputDecoration(
                hintText: 'Search title / path…',
                prefixIcon: const Icon(Icons.search, size: 18),
                isDense: true,
                suffixIcon: _query.isEmpty
                    ? null
                    : IconButton(
                        icon: const Icon(Icons.clear, size: 18),
                        onPressed: () {
                          _searchCtrl.clear();
                          setState(() => _query = '');
                        },
                      ),
              ),
            ),
          ),
          Expanded(child: _body()),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _newNote,
        icon: const Icon(Icons.add),
        label: const Text('New'),
      ),
    );
  }

  Widget _body() {
    return _state.when(
      data: (notes) {
        final filtered = _filtered(notes);
        if (filtered.isEmpty) {
          return _Empty(
            text: notes.isEmpty
                ? 'Vault is empty. Tap + to create your first note.'
                : 'No notes match the filter.',
          );
        }
        return RefreshIndicator(
          onRefresh: _load,
          child: ListView.separated(
            itemCount: filtered.length,
            separatorBuilder: (_, __) => Divider(
              height: 1,
              color: Theme.of(context).dividerColor,
            ),
            itemBuilder: (_, i) {
              final n = filtered[i];
              return ListTile(
                onTap: () => _openNote(n),
                onLongPress: () => _onLongPress(n),
                leading: Icon(
                  n.path.startsWith('personal/')
                      ? Icons.edit_note_outlined
                      : Icons.description_outlined,
                  color: Theme.of(context).colorScheme.primary,
                ),
                title: Text(
                  n.title.isNotEmpty ? n.title : p.basename(n.path),
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Text(
                  '${n.path}  ·  ${_formatBytes(n.size)} · ${_relTime(n.modified)}',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                trailing: const Icon(Icons.chevron_right),
              );
            },
          ),
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _Error(error: e, onRetry: _load),
    );
  }
}

enum _Filter {
  all('All'),
  personal('Personal'),
  projects('Projects');

  const _Filter(this.label);
  final String label;
}

enum _RowAction { open, copyPath, delete }

class _FilterStrip extends StatelessWidget {
  const _FilterStrip({required this.value, required this.onChanged});
  final _Filter value;
  final ValueChanged<_Filter> onChanged;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 48,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        itemCount: _Filter.values.length,
        separatorBuilder: (_, __) => const SizedBox(width: 6),
        itemBuilder: (_, i) {
          final f = _Filter.values[i];
          return ChoiceChip(
            label: Text(f.label),
            selected: f == value,
            onSelected: (_) => onChanged(f),
          );
        },
      ),
    );
  }
}

// _NewNoteDialog asks for a vault-relative path. Auto-appends `.md`
// if the user forgets and refuses path-traversal segments — same
// posture as the inspector's project-doc create dialog.
class _NewNoteDialog extends StatefulWidget {
  const _NewNoteDialog();

  @override
  State<_NewNoteDialog> createState() => _NewNoteDialogState();
}

class _NewNoteDialogState extends State<_NewNoteDialog> {
  final _ctrl = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  void _submit() {
    final raw = _ctrl.text.trim();
    if (raw.isEmpty) {
      setState(() => _error = 'Path is required');
      return;
    }
    if (raw.contains('..')) {
      setState(() => _error = 'Path cannot contain ".."');
      return;
    }
    final cleaned = raw.replaceAll(RegExp('^/+'), '');
    final withExt =
        cleaned.toLowerCase().endsWith('.md') ? cleaned : '$cleaned.md';
    Navigator.of(context).pop(withExt);
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('New note'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextField(
            controller: _ctrl,
            autofocus: true,
            autocorrect: false,
            textInputAction: TextInputAction.go,
            onSubmitted: (_) => _submit(),
            decoration: const InputDecoration(
              labelText: 'Vault-relative path',
              hintText: 'personal/scratch.md',
              helperText: 'Auto-appends .md if missing.',
            ),
          ),
          if (_error != null) ...[
            const SizedBox(height: 8),
            Text(
              _error!,
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
                fontSize: 12,
              ),
            ),
          ],
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Create')),
      ],
    );
  }
}

class _Empty extends StatelessWidget {
  const _Empty({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Text(
          text,
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.bodyMedium,
        ),
      ),
    );
  }
}

class _Error extends StatelessWidget {
  const _Error({required this.error, required this.onRetry});
  final Object error;
  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              error.toString(),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: const Text('Retry')),
          ],
        ),
      ),
    );
  }
}

String _formatBytes(int n) {
  if (n < 1024) return '$n B';
  if (n < 1024 * 1024) return '${(n / 1024).toStringAsFixed(1)} KiB';
  return '${(n / (1024 * 1024)).toStringAsFixed(2)} MiB';
}

String _relTime(DateTime ts) {
  final diff = DateTime.now().toUtc().difference(ts.toUtc());
  if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
  if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
  if (diff.inHours < 24) return '${diff.inHours}h ago';
  if (diff.inDays < 7) return '${diff.inDays}d ago';
  return DateFormat.yMMMd().format(ts.toLocal());
}
