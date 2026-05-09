import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:path/path.dart' as p;

// Notes surface inside the session inspector. Mirrors the web admin
// NotesPanel structure: two distinct authoring lanes against the
// same vault.
//
//   "My notes"     → personal/<basename>.md — single human-authored
//                    scratchpad. Inline editor with debounced
//                    auto-save. AI agents do not write here.
//
//   "Project docs" → projects/<basename>/*.md — multiple agent-
//                    authored docs. List view, click to open in a
//                    full-screen editor dialog. "New doc" creates
//                    via /notes/write. ⚙ pins the project mapping
//                    if the operator's vault uses a non-default
//                    layout.
//
// Both sections back into the same vault prefixes the web admin
// uses, and the project mapping override is shared (stored at
// <vault>/.opendray-projects.json), so anything the user pins on
// either surface is reflected on the other.
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
      final api = ref.read(notesApiProvider);
      final info = await api.info();
      final mapping = await api.projectMapping(widget.cwd);
      // Vault-relative project docs prefix derived by stripping the
      // vault root from mapping.path. Falls back to "" if the mapping
      // landed outside the vault (operator misconfig).
      final projectsRel = _relPrefix(info.root, mapping.path);
      final projectsList = projectsRel.isEmpty
          ? <NoteSummary>[]
          : await api.list(prefix: projectsRel);
      // Server already filters to prefix, but be defensive.
      final scoped = projectsRel.isEmpty
          ? projectsList
          : projectsList.where((n) => n.path.startsWith(projectsRel)).toList();
      if (!mounted) return;
      setState(() => _state = AsyncValue.data(
            _NotesView(
              info: info,
              mapping: mapping,
              projectsRel: projectsRel,
              projects: scoped,
            ),
          ));
    } on ApiException catch (e) {
      if (mounted) setState(() => _state = AsyncValue.error(e, StackTrace.current));
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  String _relPrefix(String vaultRoot, String absPath) {
    if (vaultRoot.isEmpty || absPath.isEmpty) return '';
    final root = vaultRoot.endsWith('/') ? vaultRoot : '$vaultRoot/';
    if (!absPath.startsWith(root)) return '';
    final rel = absPath.substring(root.length);
    // The list/read API expects a trailing slash for prefix matches
    // when the prefix is a directory; we'll normalise on the call
    // site.
    return rel;
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    return _state.when(
      data: (view) => _Body(
        sessionId: widget.sessionId,
        cwd: widget.cwd,
        view: view,
        onRefresh: _load,
      ),
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _ErrorView(error: e, onRetry: _load),
    );
  }
}

class _NotesView {
  _NotesView({
    required this.info,
    required this.mapping,
    required this.projectsRel,
    required this.projects,
  });
  final NotesInfo info;
  final ProjectMapping mapping;
  final String projectsRel;
  final List<NoteSummary> projects;
}

class _Body extends StatelessWidget {
  const _Body({
    required this.sessionId,
    required this.cwd,
    required this.view,
    required this.onRefresh,
  });

  final String sessionId;
  final String cwd;
  final _NotesView view;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    final personalPath = _personalNotePath(cwd);
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          _PersonalSection(
            sessionId: sessionId,
            personalPath: personalPath,
            cwdBase: _cwdBasename(cwd),
          ),
          const SizedBox(height: 16),
          _ProjectDocsSection(
            sessionId: sessionId,
            cwd: cwd,
            mapping: view.mapping,
            projectsRel: view.projectsRel,
            docs: view.projects,
            onRefresh: onRefresh,
          ),
        ],
      ),
    );
  }
}

// ─── Personal scratchpad ───────────────────────────────────────────

class _PersonalSection extends ConsumerStatefulWidget {
  const _PersonalSection({
    required this.sessionId,
    required this.personalPath,
    required this.cwdBase,
  });

  final String sessionId;
  final String personalPath;
  final String cwdBase;

  @override
  ConsumerState<_PersonalSection> createState() => _PersonalSectionState();
}

class _PersonalSectionState extends ConsumerState<_PersonalSection> {
  final _ctrl = TextEditingController();
  Timer? _saveDebounce;
  bool _loading = true;
  bool _saving = false;
  String? _saveError;
  DateTime? _lastSaved;
  String _initial = '';

  @override
  void initState() {
    super.initState();
    _bootstrap();
  }

  @override
  void dispose() {
    _saveDebounce?.cancel();
    _ctrl.dispose();
    super.dispose();
  }

  Future<void> _bootstrap() async {
    try {
      final note = await ref.read(notesApiProvider).read(widget.personalPath);
      if (!mounted) return;
      _initial = note.body;
      _ctrl.text = note.body;
      setState(() => _loading = false);
    } on ApiException catch (e) {
      // 404 just means the file doesn't exist yet — fine, start blank.
      if (e.statusCode == 404) {
        if (mounted) setState(() => _loading = false);
        return;
      }
      if (mounted) {
        setState(() {
        _loading = false;
        _saveError = 'Load failed: ${e.message}';
      });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
        _loading = false;
        _saveError = 'Load failed: $e';
      });
      }
    }
  }

  void _onChanged(String value) {
    _saveDebounce?.cancel();
    _saveDebounce = Timer(const Duration(milliseconds: 800), _save);
  }

  Future<void> _save() async {
    final body = _ctrl.text;
    if (body == _initial) return;
    setState(() {
      _saving = true;
      _saveError = null;
    });
    try {
      await ref.read(notesApiProvider).write(
            path: widget.personalPath,
            body: body,
          );
      if (!mounted) return;
      _initial = body;
      setState(() {
        _saving = false;
        _lastSaved = DateTime.now();
      });
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
        _saving = false;
        _saveError = 'Save failed: ${e.message}';
      });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
        _saving = false;
        _saveError = 'Save failed: $e';
      });
      }
    }
  }

  Future<void> _insertReference() async {
    try {
      await ref
          .read(sessionsApiProvider)
          .input(widget.sessionId, '@${widget.personalPath}');
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Inserted: @${widget.personalPath}'),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Insert failed: ${e.message}')),
      );
    } on Object catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Insert failed: $e')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return _SectionCard(
      icon: Icons.edit_note_outlined,
      title: 'My notes',
      subtitle: widget.personalPath,
      hint: 'Personal scratchpad — auto-saves as you type. AI agents do '
          'not write here.',
      action: IconButton(
        icon: const Icon(Icons.alternate_email, size: 18),
        tooltip: 'Insert as @reference',
        onPressed: _insertReference,
      ),
      child: _loading
          ? const Padding(
              padding: EdgeInsets.symmetric(vertical: 24),
              child: Center(child: CircularProgressIndicator()),
            )
          : Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                TextField(
                  controller: _ctrl,
                  onChanged: _onChanged,
                  maxLines: null,
                  minLines: 6,
                  textInputAction: TextInputAction.newline,
                  keyboardType: TextInputType.multiline,
                  style: const TextStyle(fontSize: 13, height: 1.5),
                  decoration: InputDecoration(
                    hintText: '# ${widget.cwdBase}\n\nThoughts, todos, '
                        'context for the agent…',
                    border: OutlineInputBorder(
                      borderSide: BorderSide(
                        color: Theme.of(context).dividerColor,
                      ),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    contentPadding: const EdgeInsets.all(12),
                  ),
                ),
                const SizedBox(height: 6),
                _SaveStatus(
                  saving: _saving,
                  lastSaved: _lastSaved,
                  error: _saveError,
                ),
              ],
            ),
    );
  }
}

class _SaveStatus extends StatelessWidget {
  const _SaveStatus({required this.saving, required this.lastSaved, this.error});
  final bool saving;
  final DateTime? lastSaved;
  final String? error;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    if (error != null) {
      return Text(
        error!,
        style: TextStyle(color: scheme.error, fontSize: 11),
      );
    }
    final muted = Theme.of(context).textTheme.bodySmall;
    if (saving) {
      return Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          SizedBox(
            width: 12,
            height: 12,
            child: CircularProgressIndicator(
              strokeWidth: 1.5,
              color: muted?.color,
            ),
          ),
          const SizedBox(width: 6),
          Text('Saving…', style: muted),
        ],
      );
    }
    if (lastSaved != null) {
      return Text(
        'Saved ${DateFormat.Hm().format(lastSaved!.toLocal())}',
        style: muted,
      );
    }
    return Text('Auto-saves as you type', style: muted);
  }
}

// ─── Project docs ──────────────────────────────────────────────────

class _ProjectDocsSection extends ConsumerStatefulWidget {
  const _ProjectDocsSection({
    required this.sessionId,
    required this.cwd,
    required this.mapping,
    required this.projectsRel,
    required this.docs,
    required this.onRefresh,
  });

  final String sessionId;
  final String cwd;
  final ProjectMapping mapping;
  final String projectsRel;
  final List<NoteSummary> docs;
  final Future<void> Function() onRefresh;

  @override
  ConsumerState<_ProjectDocsSection> createState() =>
      _ProjectDocsSectionState();
}

class _ProjectDocsSectionState extends ConsumerState<_ProjectDocsSection> {
  final _searchCtrl = TextEditingController();
  String _query = '';
  bool _creating = false;
  final _newNameCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    _newNameCtrl.dispose();
    super.dispose();
  }

  Future<void> _onDocTap(NoteSummary note) async {
    await _NoteEditorDialog.show(context: context, ref: ref, path: note.path);
    await widget.onRefresh();
  }

  Future<void> _create() async {
    final raw = _newNameCtrl.text.trim();
    if (raw.isEmpty) return;
    final name = _sanitiseFilename(raw);
    final prefix = widget.projectsRel.endsWith('/')
        ? widget.projectsRel
        : '${widget.projectsRel}/';
    final path = '$prefix$name';
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(notesApiProvider).write(
            path: path,
            body: '# ${_stripExt(name)}\n\n',
          );
      _newNameCtrl.clear();
      setState(() => _creating = false);
      await widget.onRefresh();
      if (!mounted) return;
      await _NoteEditorDialog.show(context: context, ref: ref, path: path);
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text('Create failed: ${e.message}')),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text('Create failed: $e')));
    }
  }

  Future<void> _editMapping() async {
    final result = await showDialog<String>(
      context: context,
      builder: (dialogCtx) => _MappingDialog(
        cwd: widget.cwd,
        currentPath: widget.mapping.path,
        defaultPath: widget.mapping.defaultPath,
      ),
    );
    if (result == null) return; // dialog cancelled
    if (!mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(notesApiProvider).setProjectMapping(
            cwd: widget.cwd,
            path: result,
          );
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            result.isEmpty
                ? 'Mapping cleared — using default'
                : 'Mapped to $result',
          ),
        ),
      );
      await widget.onRefresh();
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text('Save failed: ${e.message}')),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text('Save failed: $e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final filtered = _query.isEmpty
        ? widget.docs
        : widget.docs
            .where((d) =>
                d.path.toLowerCase().contains(_query) ||
                d.title.toLowerCase().contains(_query))
            .toList();
    final hint = widget.mapping.custom
        ? 'Pinned to ${widget.projectsRel}/ (overrides ${widget.mapping.defaultPath}). '
            'AI agents author docs here too.'
        : 'Architecture / spec / decisions / plan / retros — typically '
            'authored by AI agents. Tap ⚙ to point at a different vault folder.';
    return _SectionCard(
      icon: Icons.auto_awesome,
      title: 'Project docs',
      subtitle: widget.projectsRel.isEmpty
          ? '(no project mapping)'
          : '${widget.projectsRel}/',
      hint: hint,
      action: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            icon: const Icon(Icons.tune, size: 18),
            tooltip: 'Change project docs location',
            onPressed: _editMapping,
          ),
          IconButton(
            icon: Icon(_creating ? Icons.close : Icons.add, size: 18),
            tooltip: _creating ? 'Cancel' : 'New doc',
            onPressed: () => setState(() {
              _creating = !_creating;
              if (!_creating) _newNameCtrl.clear();
            }),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (_creating) ...[
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _newNameCtrl,
                    autofocus: true,
                    autocorrect: false,
                    textInputAction: TextInputAction.done,
                    onSubmitted: (_) => _create(),
                    decoration: const InputDecoration(
                      isDense: true,
                      hintText: 'filename (e.g. spec or design.md)',
                      contentPadding: EdgeInsets.symmetric(
                        horizontal: 10,
                        vertical: 10,
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 6),
                FilledButton(
                  onPressed: _create,
                  child: const Text('Create'),
                ),
              ],
            ),
            const SizedBox(height: 8),
          ],
          if (widget.docs.isNotEmpty) ...[
            TextField(
              controller: _searchCtrl,
              onChanged: (v) =>
                  setState(() => _query = v.trim().toLowerCase()),
              decoration: const InputDecoration(
                isDense: true,
                hintText: 'Filter…',
                prefixIcon: Icon(Icons.search, size: 18),
                contentPadding: EdgeInsets.symmetric(
                  horizontal: 8,
                  vertical: 8,
                ),
              ),
            ),
            const SizedBox(height: 6),
          ],
          if (widget.projectsRel.isEmpty)
            _empty(
              context,
              'Vault root not detected — check that /api/v1/notes/info '
              'returns the path your gateway expects.',
            )
          else if (widget.docs.isEmpty)
            _empty(
              context,
              'No project docs yet. Tap + to create one, or let an AI '
              'agent write to ${widget.projectsRel}/<file>.md.',
            )
          else if (filtered.isEmpty)
            _empty(context, 'No matches for "$_query".')
          else
            for (final d in filtered)
              _DocTile(
                doc: d,
                relStripPrefix: widget.projectsRel.endsWith('/')
                    ? widget.projectsRel
                    : '${widget.projectsRel}/',
                onTap: () => _onDocTap(d),
                onInsertRef: () =>
                    _pushInput(widget.sessionId, ref, '@${d.path}'),
              ),
        ],
      ),
    );
  }

  Future<void> _pushInput(String sid, WidgetRef ref, String text) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(sessionsApiProvider).input(sid, text);
      messenger.showSnackBar(
        SnackBar(
          content: Text('Inserted: $text'),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text('Insert failed: ${e.message}')),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text('Insert failed: $e')));
    }
  }

  Widget _empty(BuildContext context, String text) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 4),
      child: Text(
        text,
        style: Theme.of(context).textTheme.bodySmall,
      ),
    );
  }
}

class _DocTile extends StatelessWidget {
  const _DocTile({
    required this.doc,
    required this.relStripPrefix,
    required this.onTap,
    required this.onInsertRef,
  });

  final NoteSummary doc;
  final String relStripPrefix;
  final VoidCallback onTap;
  final VoidCallback onInsertRef;

  @override
  Widget build(BuildContext context) {
    final shown = doc.path.startsWith(relStripPrefix)
        ? doc.path.substring(relStripPrefix.length)
        : doc.path;
    final muted = Theme.of(context).textTheme.bodySmall;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Icon(
                Icons.description_outlined,
                size: 16,
                color: Theme.of(context).colorScheme.primary,
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    shown.isNotEmpty ? shown : (doc.title.isEmpty ? doc.path : doc.title),
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                  Text(
                    '${_formatBytes(doc.size)} · ${_relTime(doc.modified)}',
                    style: muted,
                  ),
                ],
              ),
            ),
            IconButton(
              icon: const Icon(Icons.alternate_email, size: 16),
              tooltip: 'Insert @reference',
              visualDensity: VisualDensity.compact,
              onPressed: onInsertRef,
            ),
          ],
        ),
      ),
    );
  }

  static String _formatBytes(int n) {
    if (n < 1024) return '$n B';
    if (n < 1024 * 1024) return '${(n / 1024).toStringAsFixed(1)} KiB';
    return '${(n / (1024 * 1024)).toStringAsFixed(2)} MiB';
  }

  static String _relTime(DateTime ts) {
    final diff = DateTime.now().toUtc().difference(ts.toUtc());
    if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays < 7) return '${diff.inDays}d ago';
    return DateFormat.yMMMd().format(ts.toLocal());
  }
}

// ─── Note editor dialog ────────────────────────────────────────────

class _NoteEditorDialog extends ConsumerStatefulWidget {
  const _NoteEditorDialog({required this.path});

  final String path;

  static Future<void> show({
    required BuildContext context,
    required WidgetRef ref,
    required String path,
  }) {
    return showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (_) => _NoteEditorDialog(path: path),
    );
  }

  @override
  ConsumerState<_NoteEditorDialog> createState() => _NoteEditorDialogState();
}

class _NoteEditorDialogState extends ConsumerState<_NoteEditorDialog> {
  final _ctrl = TextEditingController();
  Timer? _saveDebounce;
  bool _loading = true;
  bool _saving = false;
  String? _error;
  String _initial = '';
  DateTime? _lastSaved;

  @override
  void initState() {
    super.initState();
    _bootstrap();
  }

  @override
  void dispose() {
    _saveDebounce?.cancel();
    _ctrl.dispose();
    super.dispose();
  }

  Future<void> _bootstrap() async {
    try {
      final note = await ref.read(notesApiProvider).read(widget.path);
      if (!mounted) return;
      _initial = note.body;
      _ctrl.text = note.body;
      setState(() => _loading = false);
    } on ApiException catch (e) {
      if (e.statusCode == 404) {
        if (mounted) setState(() => _loading = false);
        return;
      }
      if (mounted) {
        setState(() {
        _loading = false;
        _error = 'Load failed: ${e.message}';
      });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
        _loading = false;
        _error = 'Load failed: $e';
      });
      }
    }
  }

  void _onChanged(String _) {
    _saveDebounce?.cancel();
    _saveDebounce = Timer(const Duration(milliseconds: 800), _save);
  }

  Future<void> _save() async {
    final body = _ctrl.text;
    if (body == _initial) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await ref.read(notesApiProvider).write(path: widget.path, body: body);
      if (!mounted) return;
      _initial = body;
      setState(() {
        _saving = false;
        _lastSaved = DateTime.now();
      });
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
        _saving = false;
        _error = 'Save failed: ${e.message}';
      });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
        _saving = false;
        _error = 'Save failed: $e';
      });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Dialog(
      insetPadding: const EdgeInsets.all(8),
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(context).size.height * 0.92,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 8, 8),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          p.basename(widget.path),
                          style: Theme.of(context).textTheme.titleSmall,
                          overflow: TextOverflow.ellipsis,
                        ),
                        Text(
                          widget.path,
                          style: Theme.of(context).textTheme.bodySmall,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                    ),
                  ),
                  Builder(
                    builder: (innerCtx) => IconButton(
                      icon: const Icon(Icons.close),
                      onPressed: () async {
                        // Flush pending save before dismissing.
                        _saveDebounce?.cancel();
                        await _save();
                        if (!innerCtx.mounted) return;
                        Navigator.of(innerCtx).pop();
                      },
                    ),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            Expanded(
              child: _loading
                  ? const Center(child: CircularProgressIndicator())
                  : Padding(
                      padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
                      child: TextField(
                        controller: _ctrl,
                        onChanged: _onChanged,
                        maxLines: null,
                        expands: true,
                        textInputAction: TextInputAction.newline,
                        keyboardType: TextInputType.multiline,
                        style: const TextStyle(
                          fontSize: 13,
                          height: 1.5,
                          fontFamily: 'monospace',
                        ),
                        decoration: const InputDecoration(
                          border: InputBorder.none,
                          contentPadding: EdgeInsets.zero,
                          hintText: 'Markdown…',
                        ),
                      ),
                    ),
            ),
            const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 6, 12, 8),
              child: _SaveStatus(
                saving: _saving,
                lastSaved: _lastSaved,
                error: _error,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ─── Mapping override dialog ───────────────────────────────────────

class _MappingDialog extends StatefulWidget {
  const _MappingDialog({
    required this.cwd,
    required this.currentPath,
    required this.defaultPath,
  });

  final String cwd;
  final String currentPath;
  final String defaultPath;

  @override
  State<_MappingDialog> createState() => _MappingDialogState();
}

class _MappingDialogState extends State<_MappingDialog> {
  late final TextEditingController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.currentPath);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Project docs location'),
      content: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              "Pin this session's cwd to a specific folder under your "
              'vault. Leave empty to revert to default.',
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            Text(
              'Session cwd',
              style: Theme.of(context).textTheme.labelSmall,
            ),
            const SizedBox(height: 4),
            SelectableText(
              widget.cwd,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
            ),
            const SizedBox(height: 12),
            Text(
              'Vault-relative project docs path',
              style: Theme.of(context).textTheme.labelSmall,
            ),
            const SizedBox(height: 4),
            TextField(
              controller: _ctrl,
              autocorrect: false,
              decoration: InputDecoration(
                hintText: widget.defaultPath,
                isDense: true,
                contentPadding: const EdgeInsets.all(10),
              ),
            ),
            const SizedBox(height: 8),
            Text(
              'Stored in <vault>/.opendray-projects.json — git-syncs with '
              'your notes.',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(null),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(_ctrl.text.trim()),
          child: Text(_ctrl.text.trim().isEmpty ? 'Clear override' : 'Save'),
        ),
      ],
    );
  }
}

// ─── Shared ────────────────────────────────────────────────────────

class _SectionCard extends StatelessWidget {
  const _SectionCard({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.hint,
    required this.action,
    required this.child,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final String hint;
  final Widget action;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border.all(color: Theme.of(context).dividerColor),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 10, 4, 4),
            child: Row(
              children: [
                Icon(
                  icon,
                  size: 14,
                  color: Theme.of(context).colorScheme.primary,
                ),
                const SizedBox(width: 6),
                Text(
                  title.toUpperCase(),
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                        letterSpacing: 1,
                      ),
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    subtitle,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 10,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                action,
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
            child: Text(
              hint,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
          const Divider(height: 1),
          Padding(
            padding: const EdgeInsets.all(12),
            child: child,
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

// ─── Path conventions (mirror app/shared/src/lib/notes.ts) ─────────

String _personalNotePath(String cwd) => 'personal/${_cwdSlug(cwd)}.md';

String _cwdBasename(String cwd) {
  final parts = cwd.split('/').where((s) => s.isNotEmpty).toList();
  return parts.isEmpty ? 'project' : parts.last;
}

String _cwdSlug(String cwd) {
  final base = _cwdBasename(cwd);
  final clean = base.replaceAll(RegExp(r'[^A-Za-z0-9_.\-]'), '-');
  return clean.isEmpty ? 'untitled' : clean;
}

String _sanitiseFilename(String input) {
  var name = input.trim().replaceAll(RegExp('^/+'), '').replaceAll('../', '');
  if (!name.toLowerCase().endsWith('.md')) name = '$name.md';
  return name;
}

String _stripExt(String name) {
  final i = name.lastIndexOf('.');
  return i > 0 ? name.substring(0, i) : name;
}
