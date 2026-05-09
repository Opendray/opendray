import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:path/path.dart' as p;

// Notes surface inside the session inspector. Resolves the vault
// subfolder for the current session.cwd via /notes/project-mapping,
// then lists notes under that prefix. Tap a note to view its body
// in a dialog or push the path / @reference into the live PTY.
class NotesTab extends ConsumerStatefulWidget {
  const NotesTab({required this.sessionId, required this.cwd, super.key});

  final String sessionId;
  final String cwd;

  @override
  ConsumerState<NotesTab> createState() => _NotesTabState();
}

class _NotesTabState extends ConsumerState<NotesTab>
    with AutomaticKeepAliveClientMixin {
  AsyncValue<_NotesView> _state = const AsyncValue.loading();

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final notesApi = ref.read(notesApiProvider);
      final info = await notesApi.info();
      final mapping = await notesApi.projectMapping(widget.cwd);
      final relPrefix = _relPrefix(info.root, mapping.path);
      final notes = await notesApi.list(prefix: relPrefix);
      // Filter to only notes whose path starts with prefix — the
      // server already does this for non-empty prefix, but vaults
      // without a prefix would return everything; we want project-
      // scoped only.
      final scoped = relPrefix.isEmpty
          ? notes
          : notes.where((n) => n.path.startsWith(relPrefix)).toList();
      if (!mounted) return;
      setState(() => _state = AsyncValue.data(
            _NotesView(
              info: info,
              mapping: mapping,
              relPrefix: relPrefix,
              notes: scoped,
            ),
          ));
    } on ApiException catch (e) {
      if (mounted) setState(() => _state = AsyncValue.error(e, StackTrace.current));
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  // Vault-relative prefix is the project's absolute path with the
  // vault root stripped; falls back to "" if the mapping landed
  // outside the vault (operator misconfig — list everything).
  String _relPrefix(String vaultRoot, String projectAbs) {
    if (vaultRoot.isEmpty || projectAbs.isEmpty) return '';
    final root = vaultRoot.endsWith('/') ? vaultRoot : '$vaultRoot/';
    if (!projectAbs.startsWith(root)) return '';
    return projectAbs.substring(root.length);
  }

  Future<void> _onNoteTap(NoteSummary note) async {
    final action = await showModalBottomSheet<_NoteAction>(
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
              leading: const Icon(Icons.text_snippet_outlined),
              title: const Text('View'),
              onTap: () => Navigator.of(sheetCtx).pop(_NoteAction.view),
            ),
            ListTile(
              leading: const Icon(Icons.alternate_email),
              title: const Text('Insert as @reference'),
              subtitle: const Text(
                'Pastes "@<vault-path>" into the running prompt',
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_NoteAction.insertAt),
            ),
            ListTile(
              leading: const Icon(Icons.content_paste_go),
              title: const Text('Insert path'),
              subtitle: const Text('Pastes the vault-relative path'),
              onTap: () => Navigator.of(sheetCtx).pop(_NoteAction.insertPath),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    switch (action) {
      case _NoteAction.view:
        await _viewNote(note);
      case _NoteAction.insertAt:
        await _pushInput('@${note.path}');
      case _NoteAction.insertPath:
        await _pushInput(note.path);
    }
  }

  Future<void> _pushInput(String text) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(sessionsApiProvider).input(widget.sessionId, text);
      messenger.showSnackBar(
        SnackBar(
          content: Text('Inserted: $text'),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text('Insert failed (${e.statusCode}): ${e.message}')),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text('Insert failed: $e')));
    }
  }

  Future<void> _viewNote(NoteSummary note) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      final full = await ref.read(notesApiProvider).read(note.path);
      if (!mounted) return;
      await showDialog<void>(
        context: context,
        builder: (dialogCtx) => Dialog(
          insetPadding: const EdgeInsets.all(16),
          child: ConstrainedBox(
            constraints: BoxConstraints(
              maxHeight: MediaQuery.of(dialogCtx).size.height * 0.85,
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 8, 8),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          full.title.isNotEmpty
                              ? full.title
                              : p.basename(full.path),
                          style: Theme.of(dialogCtx).textTheme.titleSmall,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      IconButton(
                        icon: const Icon(Icons.close),
                        onPressed: () => Navigator.of(dialogCtx).pop(),
                      ),
                    ],
                  ),
                ),
                const Divider(height: 1),
                Flexible(
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(12),
                    child: SelectableText(
                      full.body.isEmpty ? '(empty note)' : full.body,
                      style: const TextStyle(fontSize: 13, height: 1.4),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text('Read failed (${e.statusCode}): ${e.message}')),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text('Read failed: $e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    return _state.when(
      data: (view) => _Body(
        view: view,
        onTap: _onNoteTap,
        onRefresh: _load,
      ),
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _ErrorView(error: e, onRetry: _load),
    );
  }
}

enum _NoteAction { view, insertAt, insertPath }

class _NotesView {
  _NotesView({
    required this.info,
    required this.mapping,
    required this.relPrefix,
    required this.notes,
  });
  final NotesInfo info;
  final ProjectMapping mapping;
  final String relPrefix;
  final List<NoteSummary> notes;
}

class _Body extends StatelessWidget {
  const _Body({
    required this.view,
    required this.onTap,
    required this.onRefresh,
  });

  final _NotesView view;
  final ValueChanged<NoteSummary> onTap;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView(
        children: [
          _Header(view: view, onRefresh: onRefresh),
          const Divider(height: 1),
          if (view.notes.isEmpty)
            Padding(
              padding: const EdgeInsets.all(32),
              child: Center(
                child: Text(
                  view.relPrefix.isEmpty
                      ? 'No notes match this project'
                      : 'No notes under ${view.relPrefix}',
                  style: Theme.of(context).textTheme.bodyMedium,
                  textAlign: TextAlign.center,
                ),
              ),
            )
          else
            for (final n in view.notes)
              Column(
                children: [
                  ListTile(
                    onTap: () => onTap(n),
                    title: Text(
                      n.title.isNotEmpty ? n.title : p.basename(n.path),
                      style: Theme.of(context).textTheme.bodyMedium,
                      overflow: TextOverflow.ellipsis,
                    ),
                    subtitle: Text(
                      '${n.path}  ·  ${_relative(n.modified)}',
                      style: Theme.of(context).textTheme.bodySmall,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    trailing: const Icon(Icons.chevron_right),
                  ),
                  Divider(height: 1, color: Theme.of(context).dividerColor),
                ],
              ),
        ],
      ),
    );
  }

  static String _relative(DateTime ts) {
    final diff = DateTime.now().toUtc().difference(ts.toUtc());
    if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays < 7) return '${diff.inDays}d ago';
    return DateFormat.yMMMd().format(ts.toLocal());
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.view, required this.onRefresh});
  final _NotesView view;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 10, 8, 10),
      child: Row(
        children: [
          Icon(
            Icons.folder_special_outlined,
            size: 16,
            color: Theme.of(context).colorScheme.primary,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  view.relPrefix.isEmpty
                      ? '(no project mapping)'
                      : view.relPrefix,
                  style: const TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
                Text(
                  '${view.notes.length} note${view.notes.length == 1 ? '' : 's'}'
                  '${view.mapping.custom ? '  ·  custom mapping' : ''}',
                  style: muted,
                ),
              ],
            ),
          ),
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: onRefresh,
          ),
        ],
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final Object error;
  final VoidCallback onRetry;

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
