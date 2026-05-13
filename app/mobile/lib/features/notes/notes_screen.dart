import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/notes/note_editor_dialog.dart';
import 'package:path/path.dart' as p;

// Global Notes tab — vault drill-down browser.
//
// A flat list mixes personal scratchpads with every project's docs
// in one stream — usable when the vault has 10 notes, unreadable
// at 200. Mirrors the web admin's NotesTreeView pattern: the user
// starts at the vault root, sees top-level folders + any root-level
// .md files, and drills down level by level. Search collapses the
// tree into a flat result list across the whole vault. Quick chips
// jump to common roots (`personal/`, `projects/`).
class NotesScreen extends ConsumerStatefulWidget {
  const NotesScreen({super.key});

  @override
  ConsumerState<NotesScreen> createState() => _NotesScreenState();
}

class _NotesScreenState extends ConsumerState<NotesScreen> {
  AsyncValue<List<NoteSummary>> _state = const AsyncValue.loading();
  // Vault-relative directory the user is currently viewing. '' means
  // the vault root. Never has a trailing slash.
  String _currentPath = '';
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
    await _load();
  }

  void _enterFolder(String path) {
    setState(() => _currentPath = path);
  }

  void _goUp() {
    if (_currentPath.isEmpty) return;
    final parent = p.dirname(_currentPath);
    setState(() => _currentPath = (parent == '.' || parent == '/') ? '' : parent);
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
              title: Text(t.notesPage.open),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.open),
            ),
            ListTile(
              leading: const Icon(Icons.copy),
              title: Text(t.notesPage.copyPath),
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
            content: Text(t.notesPage.copiedSnack(path: note.path)),
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
        title: Text(t.notesPage.deleteTitle),
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
            child: Text(t.common.cancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(dialogCtx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(dialogCtx).pop(true),
            child: Text(t.common.delete),
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
          content: Text(t.notesPage.deletedSnack(path: note.path)),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(t.notesPage.deleteFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(t.notesPage.deleteFailedGeneric(error: e.toString()))));
    }
  }

  Future<void> _newNote() async {
    // Default the new note's path under the directory the user is
    // currently viewing — saves them typing the prefix.
    final prefix = _currentPath.isEmpty ? '' : '$_currentPath/';
    final path = await showDialog<String>(
      context: context,
      builder: (_) => _NewNoteDialog(initialPathPrefix: prefix),
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
        SnackBar(content: Text(t.notesPage.createFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(t.notesPage.createFailedGeneric(error: e.toString()))));
    }
  }

  // _LevelView reduces the flat list of all vault notes into the
  // immediate children of [_currentPath] — direct subfolders + the
  // .md files that live exactly at this depth. Subfolder counts
  // include every descendant recursively so the operator can see
  // "projects/foo/ has 12 notes" without drilling in.
  _LevelView _buildLevel(List<NoteSummary> all) {
    final prefix = _currentPath.isEmpty ? '' : '$_currentPath/';
    final notesHere = <NoteSummary>[];
    final folderCounts = <String, int>{};
    final folderLatest = <String, DateTime>{};
    for (final n in all) {
      if (prefix.isNotEmpty && !n.path.startsWith(prefix)) continue;
      final relative = n.path.substring(prefix.length);
      final slash = relative.indexOf('/');
      if (slash < 0) {
        // .md sitting directly in the current directory.
        notesHere.add(n);
      } else {
        final folder = relative.substring(0, slash);
        folderCounts[folder] = (folderCounts[folder] ?? 0) + 1;
        final cur = folderLatest[folder];
        if (cur == null || n.modified.isAfter(cur)) {
          folderLatest[folder] = n.modified;
        }
      }
    }
    final folders = folderCounts.entries
        .map((e) => _FolderRow(
              name: e.key,
              fullPath:
                  _currentPath.isEmpty ? e.key : '$_currentPath/${e.key}',
              count: e.value,
              latestModified: folderLatest[e.key]!,
            ))
        .toList()
      ..sort((a, b) {
        // Latest-modified first within folders too, matches notes.
        return b.latestModified.compareTo(a.latestModified);
      });
    return _LevelView(folders: folders, notes: notesHere);
  }

  // Search is a flat scan across every note in the vault; we don't
  // restrict to _currentPath because the typical "I forgot which
  // project this lives under" use case demands it.
  List<NoteSummary> _searchAll(List<NoteSummary> all) {
    final q = _query.toLowerCase();
    return all
        .where(
          (n) =>
              n.path.toLowerCase().contains(q) ||
              n.title.toLowerCase().contains(q),
        )
        .toList();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.notesPage.title),
        leading: _currentPath.isEmpty
            ? null
            : IconButton(
                icon: const Icon(Icons.arrow_back),
                tooltip: t.notesPage.up,
                onPressed: _goUp,
              ),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.sessions.inspector.shared.refresh,
            onPressed: _load,
          ),
        ],
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
                hintText: t.notesPage.searchHint,
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
          if (_query.isEmpty) _Breadcrumb(
            path: _currentPath,
            onGoRoot: () => setState(() => _currentPath = ''),
            onJumpTo: (segments) => setState(() => _currentPath = segments),
          ),
          Expanded(child: _body()),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _newNote,
        icon: const Icon(Icons.add),
        label: Text(t.notesPage.newButton),
      ),
    );
  }

  Widget _body() {
    return _state.when(
      data: (notes) {
        if (_query.isNotEmpty) {
          final results = _searchAll(notes);
          if (results.isEmpty) {
            return _Empty(text: 'No notes match "$_query".');
          }
          return RefreshIndicator(
            onRefresh: _load,
            child: ListView.separated(
              itemCount: results.length,
              separatorBuilder: (_, __) => Divider(
                height: 1,
                color: Theme.of(context).dividerColor,
              ),
              itemBuilder: (_, i) => _NoteRow(
                note: results[i],
                showFullPath: true,
                onTap: () => _openNote(results[i]),
                onLongPress: () => _onLongPress(results[i]),
              ),
            ),
          );
        }
        final level = _buildLevel(notes);
        if (level.folders.isEmpty && level.notes.isEmpty) {
          return _Empty(
            text: _currentPath.isEmpty
                ? 'Vault is empty. Tap + to create your first note.'
                : 'Folder "$_currentPath" is empty.',
          );
        }
        return RefreshIndicator(
          onRefresh: _load,
          child: ListView(
            children: [
              for (final f in level.folders)
                _FolderTile(
                  folder: f,
                  onTap: () => _enterFolder(f.fullPath),
                ),
              if (level.folders.isNotEmpty && level.notes.isNotEmpty)
                Divider(
                  height: 1,
                  color: Theme.of(context).dividerColor,
                ),
              for (final n in level.notes)
                _NoteRow(
                  note: n,
                  showFullPath: false,
                  onTap: () => _openNote(n),
                  onLongPress: () => _onLongPress(n),
                ),
            ],
          ),
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _Error(error: e, onRetry: _load),
    );
  }
}

enum _RowAction { open, copyPath, delete }

class _LevelView {
  _LevelView({required this.folders, required this.notes});
  final List<_FolderRow> folders;
  final List<NoteSummary> notes;
}

class _FolderRow {
  _FolderRow({
    required this.name,
    required this.fullPath,
    required this.count,
    required this.latestModified,
  });
  final String name;
  final String fullPath;
  final int count;
  final DateTime latestModified;
}

class _Breadcrumb extends StatelessWidget {
  const _Breadcrumb({
    required this.path,
    required this.onGoRoot,
    required this.onJumpTo,
  });

  final String path;
  final VoidCallback onGoRoot;
  // Jumps to the prefix that ends at the given vault-relative path.
  final ValueChanged<String> onJumpTo;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final segments = path.split('/').where((s) => s.isNotEmpty).toList();
    if (segments.isEmpty) {
      return Padding(
        padding: const EdgeInsets.fromLTRB(14, 4, 14, 6),
        child: Text(
          'vault root',
          style: muted?.copyWith(fontFamily: 'monospace'),
        ),
      );
    }
    final children = <Widget>[
      _BreadcrumbLink(label: 'vault', onTap: onGoRoot),
    ];
    var acc = '';
    for (var i = 0; i < segments.length; i++) {
      children.add(Text(' / ', style: muted));
      acc = acc.isEmpty ? segments[i] : '$acc/${segments[i]}';
      if (i == segments.length - 1) {
        children.add(Text(
          segments[i],
          style: muted?.copyWith(fontFamily: 'monospace'),
        ));
      } else {
        final target = acc;
        children.add(_BreadcrumbLink(
          label: segments[i],
          onTap: () => onJumpTo(target),
        ));
      }
    }
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 4, 14, 6),
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(children: children),
      ),
    );
  }
}

class _BreadcrumbLink extends StatelessWidget {
  const _BreadcrumbLink({required this.label, required this.onTap});
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(4),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
        child: Text(
          label,
          style: TextStyle(
            color: Theme.of(context).colorScheme.primary,
            fontSize: 12,
            fontFamily: 'monospace',
            decoration: TextDecoration.underline,
          ),
        ),
      ),
    );
  }
}

class _FolderTile extends StatelessWidget {
  const _FolderTile({required this.folder, required this.onTap});
  final _FolderRow folder;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      onTap: onTap,
      leading: Icon(
        Icons.folder_outlined,
        color: Theme.of(context).colorScheme.primary,
      ),
      title: Text(folder.name),
      subtitle: Text(
        '${folder.count} note${folder.count == 1 ? '' : 's'}'
        '  ·  latest ${_relTime(folder.latestModified)}',
        style: Theme.of(context).textTheme.bodySmall,
      ),
      trailing: const Icon(Icons.chevron_right),
    );
  }
}

class _NoteRow extends StatelessWidget {
  const _NoteRow({
    required this.note,
    required this.showFullPath,
    required this.onTap,
    required this.onLongPress,
  });

  final NoteSummary note;
  final bool showFullPath;
  final VoidCallback onTap;
  final VoidCallback onLongPress;

  @override
  Widget build(BuildContext context) {
    final subtitle = showFullPath
        ? '${note.path}  ·  ${_formatBytes(note.size)} · ${_relTime(note.modified)}'
        : '${_formatBytes(note.size)} · ${_relTime(note.modified)}';
    return ListTile(
      onTap: onTap,
      onLongPress: onLongPress,
      leading: Icon(
        note.path.startsWith('personal/')
            ? Icons.edit_note_outlined
            : Icons.description_outlined,
        color: Theme.of(context).colorScheme.primary,
      ),
      title: Text(
        note.title.isNotEmpty ? note.title : p.basename(note.path),
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text(
        subtitle,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: Theme.of(context).textTheme.bodySmall,
      ),
      trailing: const Icon(Icons.chevron_right),
    );
  }
}

// _NewNoteDialog asks for a vault-relative path. Auto-appends `.md`
// if the user forgets and refuses path-traversal segments. Defaults
// to whatever directory the user is currently viewing so creating
// a note inside `projects/foo/` only requires typing the filename.
class _NewNoteDialog extends StatefulWidget {
  const _NewNoteDialog({required this.initialPathPrefix});
  final String initialPathPrefix;

  @override
  State<_NewNoteDialog> createState() => _NewNoteDialogState();
}

class _NewNoteDialogState extends State<_NewNoteDialog> {
  late final TextEditingController _ctrl;
  String? _error;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.initialPathPrefix);
  }

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
      title: Text(t.notesPage.newNoteDialogTitle),
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
            decoration: InputDecoration(
              labelText: t.notesPage.pathLabel,
              hintText: t.notesPage.pathHint,
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
          child: Text(t.common.cancel),
        ),
        FilledButton(onPressed: _submit, child: Text(t.notesPage.create)),
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
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
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
