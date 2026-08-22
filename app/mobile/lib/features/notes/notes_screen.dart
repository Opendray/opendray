import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/notes/flatten_notice.dart';
import 'package:opendray/features/notes/note_actions.dart';
import 'package:opendray/features/notes/note_editor_dialog.dart';
import 'package:opendray/features/notes/vault_sync_screen.dart';
import 'package:opendray/features/notes/vault_text.dart';
import 'package:opendray/features/project/project_screen.dart';
import 'package:path/path.dart' as p;

// NotesScreen (Notes tab) is the project's official doc — goal/plan/tech/
// activity/journal/inbox. Deconflated per the Experience Flywheel: memory
// hygiene lives under Memory, the freeform markdown vault under More.
// Memory = facts; Knowledge = cross-project; Notes = where this project is.
class NotesScreen extends ConsumerWidget {
  const NotesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return const ProjectScreen();
  }
}

// NotesVaultScreen — the markdown doc library, demoted out of the
// core triad (reachable from the More menu). It syncs through a plain
// git remote; Obsidian is merely one editor that can be pointed at the
// same repo, not something opendray integrates with.
//
// A flat list mixes personal scratchpads with every project's docs in one
// stream — usable at 10 notes, unreadable at 200. Mirrors the web vault: the
// user starts at the root, sees top-level folders + root-level .md files, and
// drills down level by level. Search collapses the tree into a flat result
// list; quick chips jump to common roots (`personal/`, `projects/`).
class NotesVaultScreen extends ConsumerStatefulWidget {
  const NotesVaultScreen({super.key});

  @override
  ConsumerState<NotesVaultScreen> createState() => _NotesScreenState();
}

/// What the left-hand listing is showing. Mirrors the web vault's
/// tree/tags switch — the two ways of finding a document that isn't
/// where you remembered filing it.
enum _BrowseMode { folders, tags }

class _NotesScreenState extends ConsumerState<NotesVaultScreen> {
  AsyncValue<List<NoteSummary>> _state = const AsyncValue.loading();
  // Vault-relative directory the user is currently viewing. '' means
  // the vault root. Never has a trailing slash.
  String _currentPath = '';
  String _query = '';
  bool _flattenable = false;
  final _searchCtrl = TextEditingController();

  _BrowseMode _mode = _BrowseMode.folders;
  // Loaded the first time the tags view is opened. The gateway walks
  // the whole vault to build it, so it is not fetched alongside the
  // listing that every visit needs.
  AsyncValue<List<TagCount>>? _tagState;
  String? _activeTag;

  List<String> get _allPaths =>
      _state.valueOrNull?.map((n) => n.path).toList() ?? const [];

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
      final api = ref.read(notesApiProvider);
      // Fetched alongside the listing so the layout offer can appear
      // with the tree rather than popping in a moment later. A failure
      // here must not cost the listing — the notice is optional, the
      // documents are not.
      unawaited(
        api.info().then((i) {
          if (mounted) setState(() => _flattenable = i.flattenable);
        }).catchError((Object _) {}),
      );
      final notes = await api.list();
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

  // Tags are counted by a full walk of the vault on the gateway, so
  // they are fetched when the view is opened rather than on every
  // listing refresh.
  Future<void> _loadTags() async {
    setState(() => _tagState = const AsyncValue.loading());
    try {
      final tags = await ref.read(notesApiProvider).tags();
      if (!mounted) return;
      tags.sort((a, b) {
        // Most-used first; alphabetical inside a tie so the order is
        // stable between refreshes.
        final byCount = b.count.compareTo(a.count);
        return byCount != 0 ? byCount : a.tag.compareTo(b.tag);
      });
      setState(() => _tagState = AsyncValue.data(tags));
    } on Object catch (e, st) {
      if (mounted) setState(() => _tagState = AsyncValue.error(e, st));
    }
  }

  Future<void> _openNote(NoteSummary note) async {
    await NoteEditorDialog.show(
      context: context,
      path: note.path,
      // The editor resolves and completes [[wiki links]] against this;
      // handing over the listing already in hand saves it a fetch.
      allPaths: _allPaths,
    );
    if (!mounted) return;
    await _load();
  }

  /// Open (creating first if needed) today's daily note. Mirrors the
  /// web vault's "Today" button, template included — the same date on
  /// two clients has to produce one document, not two shapes of it.
  Future<void> _openToday() async {
    final now = DateTime.now();
    final path = dailyNotePath(now);
    final exists =
        _state.valueOrNull?.any((n) => n.path == path) ?? false;
    final messenger = ScaffoldMessenger.of(context);
    if (!exists) {
      final body = dailyNoteBody(
        now,
        // en_US explicitly, matching the web's date-fns default, so the
        // heading reads the same on both clients.
        longDate: DateFormat('EEEE, MMMM d, y', 'en_US').format(now),
      );
      try {
        await ref.read(notesApiProvider).write(path: path, body: body);
      } on ApiException catch (e) {
        messenger.showSnackBar(
          SnackBar(content: Text(t.notesPage.createFailedApi(error: e.message))),
        );
        return;
      } on Object catch (e) {
        messenger.showSnackBar(
          SnackBar(
            content: Text(t.notesPage.createFailedGeneric(error: e.toString())),
          ),
        );
        return;
      }
    }
    if (!mounted) return;
    await NoteEditorDialog.show(
      context: context,
      path: path,
      allPaths: _allPaths,
    );
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

  /// The row's action sheet. Reached from the trailing button and
  /// from a long press on the row.
  Future<void> _showRowActions(NoteSummary note) async {
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
              leading: const Icon(Icons.drive_file_rename_outline),
              title: Text(t.notesPage.rename.action),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.rename),
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
                t.notesPage.popupDelete,
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
      case _RowAction.rename:
        await _promptRename(note);
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

  Future<void> _promptRename(NoteSummary note) async {
    final to = await renameNoteFlow(context: context, ref: ref, path: note.path);
    if (to == null || !mounted) return;
    await _load();
  }

  Future<void> _confirmAndDelete(NoteSummary note) async {
    final gone = await deleteNoteFlow(
      context: context,
      ref: ref,
      path: note.path,
      title: note.title,
    );
    if (!gone || !mounted) return;
    await _load();
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
      await NoteEditorDialog.show(
        context: context,
        path: path,
        allPaths: _allPaths,
      );
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
        // "Up" belongs to the folder listing. Showing it while a tag
        // filter or the tag index is on screen offers to walk a
        // hierarchy that isn't what's being displayed.
        leading: (_currentPath.isEmpty ||
                _activeTag != null ||
                _mode == _BrowseMode.tags)
            ? null
            : IconButton(
                icon: const Icon(Icons.arrow_back),
                tooltip: t.notesPage.up,
                onPressed: _goUp,
              ),
        actions: [
          const VaultSyncBadge(),
          IconButton(
            icon: const Icon(Icons.today_outlined),
            tooltip: t.notesPage.today.tooltip,
            onPressed: () => unawaited(_openToday()),
          ),
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.sessions.inspector.shared.refresh,
            onPressed: _load,
          ),
        ],
      ),
      body: Column(
        children: [
          if (_flattenable) FlattenNotice(onConverted: _load),
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
            child: SegmentedButton<_BrowseMode>(
              showSelectedIcon: false,
              style: const ButtonStyle(
                visualDensity: VisualDensity.compact,
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
              segments: [
                ButtonSegment(
                  value: _BrowseMode.folders,
                  icon: const Icon(Icons.folder_outlined, size: 16),
                  label: Text(t.notesPage.browse.tree),
                ),
                ButtonSegment(
                  value: _BrowseMode.tags,
                  icon: const Icon(Icons.tag, size: 16),
                  label: Text(t.notesPage.browse.tags),
                ),
              ],
              selected: {_mode},
              onSelectionChanged: (sel) {
                final next = sel.first;
                setState(() {
                  _mode = next;
                  // The filter box means different things in the two
                  // views, so carrying a query across is misleading —
                  // and so is a tag filter still narrowing the folder
                  // listing after the user has left the tag view.
                  _query = '';
                  _activeTag = null;
                  _searchCtrl.clear();
                });
                if (next == _BrowseMode.tags && _tagState == null) {
                  unawaited(_loadTags());
                }
              },
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 4, 12, 4),
            child: TextField(
              controller: _searchCtrl,
              onChanged: (v) =>
                  setState(() => _query = v.trim().toLowerCase()),
              decoration: InputDecoration(
                hintText: _showingTags
                    ? t.notesPage.browse.tags
                    : t.notesPage.searchHint,
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
          if (_activeTag != null) _activeTagChip(),
          if (_mode == _BrowseMode.folders &&
              _activeTag == null &&
              _query.isEmpty)
            _Breadcrumb(
              path: _currentPath,
              onGoRoot: () => setState(() => _currentPath = ''),
              onJumpTo: (segments) => setState(() => _currentPath = segments),
            ),
          Expanded(child: _body()),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        heroTag: 'notes_fab',
        onPressed: _newNote,
        icon: const Icon(Icons.add),
        label: Text(t.notesPage.newButton),
      ),
    );
  }

  /// True while the tag INDEX is on screen. Picking a tag swaps the
  /// listing back to notes, so the two are not the same condition.
  bool get _showingTags => _mode == _BrowseMode.tags && _activeTag == null;

  Widget _activeTagChip() {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 0, 12, 4),
      child: Align(
        alignment: Alignment.centerLeft,
        child: InputChip(
          avatar: const Icon(Icons.tag, size: 15),
          label: Text('${t.notesPage.tags.filteredBy}: ${_activeTag!}'),
          onDeleted: () => setState(() => _activeTag = null),
          deleteButtonTooltipMessage: t.notesPage.tags.clear,
        ),
      ),
    );
  }

  Widget _tagsBody() {
    final state = _tagState;
    if (state == null) return const SizedBox.shrink();
    return state.when(
      data: (tags) {
        final visible = _query.isEmpty
            ? tags
            : tags.where((x) => x.tag.toLowerCase().contains(_query)).toList();
        if (visible.isEmpty) {
          return _Empty(
            text: tags.isEmpty
                ? t.notesPage.tags.empty
                : t.notesPage.tags.noMatches(query: _query),
          );
        }
        return RefreshIndicator(
          onRefresh: _loadTags,
          child: ListView.separated(
            itemCount: visible.length,
            separatorBuilder: (_, __) =>
                Divider(height: 1, color: Theme.of(context).dividerColor),
            itemBuilder: (_, i) {
              final tag = visible[i];
              return ListTile(
                leading: Icon(
                  Icons.tag,
                  color: Theme.of(context).colorScheme.primary,
                ),
                title: Text(tag.tag),
                trailing: Text(
                  '${tag.count}',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                onTap: () => setState(() {
                  _activeTag = tag.tag;
                  _query = '';
                  _searchCtrl.clear();
                }),
              );
            },
          ),
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _Error(error: e, onRetry: _loadTags),
    );
  }

  /// Paths carrying [_activeTag], from the tag index already fetched.
  Set<String> _pathsForActiveTag() {
    final tags = _tagState?.valueOrNull;
    if (tags == null) return const {};
    for (final tag in tags) {
      if (tag.tag == _activeTag) return tag.notes.toSet();
    }
    return const {};
  }

  Widget _body() {
    if (_showingTags) return _tagsBody();
    return _state.when(
      data: (notes) {
        if (_activeTag != null) {
          final wanted = _pathsForActiveTag();
          final results =
              notes.where((n) => wanted.contains(n.path)).toList();
          if (results.isEmpty) {
            return _Empty(
              text: t.notesPage.tags.noNotes(tag: _activeTag!),
            );
          }
          return RefreshIndicator(
            onRefresh: _load,
            child: ListView.separated(
              itemCount: results.length,
              separatorBuilder: (_, __) =>
                  Divider(height: 1, color: Theme.of(context).dividerColor),
              itemBuilder: (_, i) => _NoteRow(
                note: results[i],
                showFullPath: true,
                onTap: () => _openNote(results[i]),
                onMenu: () => _showRowActions(results[i]),
              ),
            ),
          );
        }
        if (_query.isNotEmpty) {
          final results = _searchAll(notes);
          if (results.isEmpty) {
            return _Empty(text: t.notesPage.emptyFilterMatch(query: _query));
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
                onMenu: () => _showRowActions(results[i]),
              ),
            ),
          );
        }
        final level = _buildLevel(notes);
        if (level.folders.isEmpty && level.notes.isEmpty) {
          return _Empty(
            text: _currentPath.isEmpty
                ? t.notesPage.emptyVault
                : t.notesPage.emptyFolder(path: _currentPath),
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
                  onMenu: () => _showRowActions(n),
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

enum _RowAction { open, rename, copyPath, delete }

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
    required this.onMenu,
  });

  final NoteSummary note;
  final bool showFullPath;
  final VoidCallback onTap;

  /// Opens the row's actions. Wired to both the trailing button and a
  /// long press: the button is the one anybody finds, the gesture is
  /// the shortcut for whoever already knows it.
  final VoidCallback onMenu;

  @override
  Widget build(BuildContext context) {
    final subtitle = showFullPath
        ? '${note.path}  ·  ${_formatBytes(note.size)} · ${_relTime(note.modified)}'
        : '${_formatBytes(note.size)} · ${_relTime(note.modified)}';
    return ListTile(
      onTap: onTap,
      onLongPress: onMenu,
      leading: Icon(
        // The operator's own note carries a different icon from the
        // agent-written docs. Both layouts have to be recognised: the
        // nested one files it under personal/, the flat one puts
        // personal.md inside the project's own directory.
        _isPersonalNote(note.path)
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
      // This slot used to hold a chevron. Tapping the row already opens
      // the document, so the arrow said nothing the row didn't — while
      // rename and delete sat behind a long press with no hint that
      // they existed at all. Same pixels, an affordance instead of a
      // decoration.
      trailing: IconButton(
        icon: const Icon(Icons.more_vert),
        tooltip: t.common.more,
        onPressed: onMenu,
      ),
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
      setState(() => _error = t.notesPage.validatePath);
      return;
    }
    if (raw.contains('..')) {
      setState(() => _error = t.notesPage.validatePathDots);
      return;
    }
    // sanitizeNotePath rather than a blanket `.md`: appending it
    // unconditionally turned `guide.html` into `guide.html.md`, so an
    // HTML document could not be created from the phone at all.
    Navigator.of(context).pop(sanitizeNotePath(raw));
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
              helperText: t.notesPage.pathHelper,
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

// _isPersonalNote recognises the operator's own scratchpad in either
// vault layout: `personal/<project>.md` when projects are nested, and
// `<project>/personal.md` when they are flat.
bool _isPersonalNote(String path) =>
    path.startsWith('personal/') ||
    path == 'personal.md' ||
    path.endsWith('/personal.md');
