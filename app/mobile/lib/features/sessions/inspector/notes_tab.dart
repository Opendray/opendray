import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/notes/note_editor_dialog.dart';

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
//                    via /notes/write. The settings icon in the
//                    section header pins the project mapping if
//                    the operator's vault uses a non-default layout.
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
      final projectsRel = _projectsRel(info, mapping, widget.cwd);
      final projectsList = projectsRel.isEmpty
          ? <NoteSummary>[]
          : await api.list(prefix: projectsRel);
      // Server already filters to prefix, but be defensive — and match folder
      // CHILDREN only, so a project bound to projects/app doesn't pull in a
      // sibling like projects/app-old/.
      final scoped = projectsRel.isEmpty
          ? projectsList
          : projectsList
              .where((n) =>
                  n.path.startsWith('$projectsRel/') || n.path == projectsRel)
              .toList();
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

  // Vault-relative project docs prefix. The backend resolves the mapping to a
  // VAULT-RELATIVE path — the override, or the auto-derived
  // <projects_prefix>/<basename> — so use mapping.path AS-IS.
  //
  // The previous version tried to strip the absolute vault root (info.root)
  // off mapping.path, but mapping.path is already relative, so the strip
  // always failed and silently dropped CUSTOM overrides to the convention
  // fallback (the project bind never took effect). Fall back to the
  // convention only when the server returns nothing (notes misconfigured /
  // empty root).
  String _projectsRel(NotesInfo info, ProjectMapping mapping, String cwd) {
    final p = mapping.path.trim().replaceAll(RegExp(r'^/+|/+$'), '');
    if (p.isNotEmpty) return p;
    final prefix = info.projectsPrefix.isNotEmpty
        ? info.projectsPrefix
        : 'projects';
    return '$prefix/${_cwdSlug(cwd)}';
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
        _saveError = t.sessions.inspector.notes.loadFailedApi(error: e.message);
      });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
        _loading = false;
        _saveError = t.sessions.inspector.notes.loadFailedGeneric(error: e.toString());
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
        _saveError = t.sessions.inspector.notes.saveFailedApi(error: e.message);
      });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
        _saving = false;
        _saveError = t.sessions.inspector.notes.saveFailedGeneric(error: e.toString());
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
          content: Text(t.sessions.inspector.notes.insertedAt(path: widget.personalPath)),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(t.sessions.inspector.notes.insertFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(t.sessions.inspector.notes.insertFailedGeneric(error: e.toString()))),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return _SectionCard(
      icon: Icons.edit_note_outlined,
      title: t.sessions.inspector.notes.myNotes,
      subtitle: widget.personalPath,
      hint: t.sessions.inspector.notes.personalHint,
      action: IconButton(
        icon: const Icon(Icons.alternate_email, size: 18),
        tooltip: t.sessions.inspector.notes.insertAtRefTooltip,
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
                    hintText: t.sessions.inspector.notes.draftHint(project: widget.cwdBase),
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
                NoteSaveStatus(
                  saving: _saving,
                  lastSaved: _lastSaved,
                  error: _saveError,
                ),
              ],
            ),
    );
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
    await NoteEditorDialog.show(context: context, path: note.path);
    await widget.onRefresh();
  }

  /// Selected template for the next new doc. Server-rendered, so this
  /// only has to carry the id.
  String _template = 'blank';
  List<NoteTemplate> _templates = const [];

  Future<void> _loadTemplates() async {
    try {
      final list = await ref.read(notesApiProvider).templates();
      if (mounted) setState(() => _templates = list);
    } on Object {
      // A missing template list must not block creating a doc — the
      // server defaults to blank when none is named.
    }
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
      await ref.read(notesApiProvider).newFromTemplate(
            path: path,
            template: _template,
          );
      _newNameCtrl.clear();
      setState(() => _creating = false);
      await widget.onRefresh();
      if (!mounted) return;
      await NoteEditorDialog.show(context: context, path: path);
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(t.sessions.inspector.notes.createFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(t.sessions.inspector.notes.createFailedGeneric(error: e.toString()))));
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
                ? t.sessions.inspector.notes.mappingCleared
                : t.sessions.inspector.notes.mappedTo(path: result),
          ),
        ),
      );
      await widget.onRefresh();
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(t.sessions.inspector.notes.saveFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(t.sessions.inspector.notes.saveFailedGeneric(error: e.toString()))));
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
        ? t.sessions.inspector.notes.pinnedHint(
            path: widget.projectsRel,
            defaultPath: widget.mapping.defaultPath,
          )
        : t.sessions.inspector.notes.projectDocsHint;
    return _SectionCard(
      icon: Icons.auto_awesome,
      title: t.sessions.inspector.notes.projectDocs,
      subtitle: widget.projectsRel.isEmpty
          ? t.sessions.inspector.notes.noProjectMapping2
          : '${widget.projectsRel}/',
      hint: hint,
      action: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            icon: const Icon(Icons.tune, size: 18),
            tooltip: t.sessions.inspector.notes.changeLocationTooltip,
            onPressed: _editMapping,
          ),
          IconButton(
            icon: Icon(_creating ? Icons.close : Icons.add, size: 18),
            tooltip: _creating ? t.sessions.inspector.notes.cancelTooltip : t.sessions.inspector.notes.newDocTooltip,
            onPressed: () {
              setState(() {
                _creating = !_creating;
                if (!_creating) _newNameCtrl.clear();
              });
              if (_creating && _templates.isEmpty) _loadTemplates();
            },
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (_creating) ...[
            if (_templates.isNotEmpty) ...[
              Wrap(
                spacing: 6,
                runSpacing: 4,
                children: [
                  for (final tpl in _templates)
                    ChoiceChip(
                      label: Text(
                        tpl.source == 'vault' ? '${tpl.name} *' : tpl.name,
                        style: const TextStyle(fontSize: 11),
                      ),
                      selected: _template == tpl.id,
                      visualDensity: VisualDensity.compact,
                      onSelected: (_) => setState(() => _template = tpl.id),
                    ),
                ],
              ),
              const SizedBox(height: 6),
            ],
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _newNameCtrl,
                    autofocus: true,
                    autocorrect: false,
                    textInputAction: TextInputAction.done,
                    onSubmitted: (_) => _create(),
                    decoration: InputDecoration(
                      isDense: true,
                      hintText: t.sessions.inspector.notes.filenameHint,
                      contentPadding: const EdgeInsets.symmetric(
                        horizontal: 10,
                        vertical: 10,
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 6),
                FilledButton(
                  onPressed: _create,
                  child: Text(t.sessions.inspector.notes.create),
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
              decoration: InputDecoration(
                isDense: true,
                hintText: t.sessions.inspector.notes.filterHint,
                prefixIcon: const Icon(Icons.search, size: 18),
                contentPadding: const EdgeInsets.symmetric(
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
              t.sessions.inspector.notes.noProjectMapping,
            )
          else if (widget.docs.isEmpty)
            _empty(
              context,
              t.sessions.inspector.notes.emptyProjectDocs,
            )
          else if (filtered.isEmpty)
            _empty(context, t.sessions.inspector.notes.emptyFilterMatch(query: _query))
          // While filtering, a flat list of hits is the right answer —
          // the question is "which doc", not "where does it live". The
          // folder tree is for browsing.
          else if (_query.isNotEmpty)
            for (final d in filtered)
              _DocTile(
                doc: d,
                relStripPrefix: _stripPrefix,
                onTap: () => _onDocTap(d),
                onInsertRef: () =>
                    _pushInput(widget.sessionId, ref, '@${d.path}'),
                onRename: () => _renameDoc(d),
              )
          else
            _DocTree(
              docs: filtered,
              stripPrefix: _stripPrefix,
              onTap: _onDocTap,
              onInsertRef: (d) =>
                  _pushInput(widget.sessionId, ref, '@${d.path}'),
              onRename: _renameDoc,
              onOpenIndex: _onDocTap,
            ),
        ],
      ),
    );
  }

  String get _stripPrefix => widget.projectsRel.endsWith('/')
      ? widget.projectsRel
      : '${widget.projectsRel}/';

  /// Rename / re-file a doc. The path is shown relative to the project
  /// folder so typing `features/canvas.md` files it in a folder — the
  /// same gesture that creates one.
  Future<void> _renameDoc(NoteSummary doc) async {
    final current = doc.path.startsWith(_stripPrefix)
        ? doc.path.substring(_stripPrefix.length)
        : doc.path;
    final ctrl = TextEditingController(text: current);
    final next = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.sessions.inspector.notes.renameTitle),
        content: TextField(
          controller: ctrl,
          autofocus: true,
          autocorrect: false,
          decoration: InputDecoration(
            isDense: true,
            helperText: t.sessions.inspector.notes.renameHelp,
            helperMaxLines: 3,
          ),
          onSubmitted: (v) => Navigator.of(ctx).pop(v.trim()),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(ctrl.text.trim()),
            child: Text(t.sessions.inspector.notes.move),
          ),
        ],
      ),
    );
    if (next == null || next.isEmpty || next == current) return;
    if (!mounted) return;

    final messenger = ScaffoldMessenger.of(context);
    try {
      final res = await ref.read(notesApiProvider).move(
            from: doc.path,
            to: '$_stripPrefix${_sanitiseFilename(next)}',
          );
      await widget.onRefresh();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            res.warning != null
                ? t.sessions.inspector.notes
                    .renamedWithWarning(warning: res.warning!)
                : res.linksRewritten > 0
                    ? t.sessions.inspector.notes.renamedWithLinks(
                        count: res.linksRewritten,
                        notes: res.notesRewritten,
                      )
                    : t.sessions.inspector.notes.renamed,
          ),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(t.sessions.inspector.notes.renameFailed(error: e.message))),
      );
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(t.sessions.inspector.notes.renameFailed(error: e.toString()))),
      );
    }
  }

  Future<void> _pushInput(String sid, WidgetRef ref, String text) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(sessionsApiProvider).input(sid, text);
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.sessions.inspector.shared.inserted(text: text)),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(t.sessions.inspector.notes.insertFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(t.sessions.inspector.notes.insertFailedGeneric(error: e.toString()))));
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
    required this.onRename,
    this.indent = 0,
    this.showName,
  });

  final NoteSummary doc;
  final String relStripPrefix;
  final VoidCallback onTap;
  final VoidCallback onInsertRef;
  final VoidCallback onRename;
  /// Extra left padding when rendered inside the folder tree.
  final double indent;
  /// Label override — inside a tree the folder is already on screen, so
  /// the row shows the bare filename instead of the whole path.
  final String? showName;

  @override
  Widget build(BuildContext context) {
    final shown = showName ??
        (doc.path.startsWith(relStripPrefix)
            ? doc.path.substring(relStripPrefix.length)
            : doc.path);
    final muted = Theme.of(context).textTheme.bodySmall;
    return InkWell(
      onTap: onTap,
      onLongPress: onRename,
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: EdgeInsets.only(
          left: 8 + indent,
          right: 8,
          top: 8,
          bottom: 8,
        ),
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
              icon: const Icon(Icons.drive_file_rename_outline, size: 16),
              tooltip: t.sessions.inspector.notes.renameTitle,
              visualDensity: VisualDensity.compact,
              onPressed: onRename,
            ),
            IconButton(
              icon: const Icon(Icons.alternate_email, size: 16),
              tooltip: t.sessions.inspector.notes.insertAtRefShort,
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
      title: Text(t.sessions.inspector.notes.locationDialogTitle),
      content: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              t.sessions.inspector.notes.locationDialogHelp,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            Text(
              t.sessions.inspector.notes.sessionCwd,
              style: Theme.of(context).textTheme.labelSmall,
            ),
            const SizedBox(height: 4),
            SelectableText(
              widget.cwd,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
            ),
            const SizedBox(height: 12),
            Text(
              t.sessions.inspector.notes.projectDocsPath,
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
              t.sessions.inspector.notes.locationStoredHint,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(null),
          child: Text(t.common.cancel),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(_ctrl.text.trim()),
          child: Text(_ctrl.text.trim().isEmpty
              ? t.sessions.inspector.notes.clearOverride
              : t.sessions.inspector.notes.save),
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
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
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


// ─── Project docs folder tree ──────────────────────────────────────

/// _DocTree renders the project's docs as folders, mirroring the web
/// NotesTreeView. The paths were always hierarchical — both surfaces
/// just flattened them, which made a vault of any depth unreadable and
/// made the folders themselves invisible.
///
/// Folders start expanded: a project's doc set is small enough that
/// hiding it behind chevrons costs more than it saves, and the whole
/// point of the view is seeing the structure.
class _DocTree extends StatefulWidget {
  const _DocTree({
    required this.docs,
    required this.stripPrefix,
    required this.onTap,
    required this.onInsertRef,
    required this.onRename,
    required this.onOpenIndex,
  });

  final List<NoteSummary> docs;
  final String stripPrefix;
  final void Function(NoteSummary) onTap;
  final void Function(NoteSummary) onInsertRef;
  final void Function(NoteSummary) onRename;
  /// Opens a folder's README/index note. A folder that carries one can
  /// explain what lives in it instead of being a bare row of chevrons.
  final void Function(NoteSummary) onOpenIndex;

  @override
  State<_DocTree> createState() => _DocTreeState();
}

class _DocTreeState extends State<_DocTree> {
  final _collapsed = <String>{};

  @override
  Widget build(BuildContext context) {
    final root = _TreeNode(name: '', path: '');
    for (final d in widget.docs) {
      final rel = d.path.startsWith(widget.stripPrefix)
          ? d.path.substring(widget.stripPrefix.length)
          : d.path;
      final parts = rel.split('/').where((p) => p.isNotEmpty).toList();
      var node = root;
      for (var i = 0; i < parts.length - 1; i++) {
        final path = node.path.isEmpty ? parts[i] : '${node.path}/${parts[i]}';
        node = node.children.putIfAbsent(
          parts[i],
          () => _TreeNode(name: parts[i], path: path),
        );
      }
      if (parts.isNotEmpty) {
        node.files.add(_TreeFile(parts.last, d));
        if (kIndexNames.contains(parts.last) && node.index == null) {
          node.index = d;
        }
      }
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: _render(context, root, 0),
    );
  }

  List<Widget> _render(BuildContext context, _TreeNode node, int depth) {
    final out = <Widget>[];
    // Folders before files, each alphabetical — a stable shape beats
    // recency here, because the tree is how you learn the layout.
    final dirs = node.children.values.toList()
      ..sort((a, b) => a.name.compareTo(b.name));
    for (final dir in dirs) {
      final isCollapsed = _collapsed.contains(dir.path);
      out.add(
        InkWell(
          onTap: () => setState(() {
            if (isCollapsed) {
              _collapsed.remove(dir.path);
            } else {
              _collapsed.add(dir.path);
            }
          }),
          borderRadius: BorderRadius.circular(8),
          child: Padding(
            padding: EdgeInsets.only(
              left: 8 + depth * 14,
              right: 8,
              top: 8,
              bottom: 8,
            ),
            child: Row(
              children: [
                Icon(
                  isCollapsed ? Icons.chevron_right : Icons.expand_more,
                  size: 16,
                  color: Theme.of(context).colorScheme.outline,
                ),
                const SizedBox(width: 2),
                Icon(
                  Icons.folder_outlined,
                  size: 16,
                  color: Theme.of(context).colorScheme.outline,
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    dir.name,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                Text(
                  '${dir.count}',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                if (dir.index != null)
                  IconButton(
                    icon: const Icon(Icons.menu_book_outlined, size: 16),
                    tooltip: t.sessions.inspector.notes.openFolderIndex,
                    visualDensity: VisualDensity.compact,
                    onPressed: () => widget.onOpenIndex(dir.index!),
                  ),
              ],
            ),
          ),
        ),
      );
      if (!isCollapsed) out.addAll(_render(context, dir, depth + 1));
    }
    final files = node.files.toList()
      ..sort((a, b) => a.name.compareTo(b.name));
    for (final f in files) {
      out.add(
        _DocTile(
          doc: f.doc,
          relStripPrefix: widget.stripPrefix,
          showName: f.name,
          indent: depth * 14 + 18,
          onTap: () => widget.onTap(f.doc),
          onInsertRef: () => widget.onInsertRef(f.doc),
          onRename: () => widget.onRename(f.doc),
        ),
      );
    }
    return out;
  }
}

class _TreeNode {
  _TreeNode({required this.name, required this.path});
  final String name;
  final String path;
  final Map<String, _TreeNode> children = {};
  final List<_TreeFile> files = [];
  /// This folder's index note, if it has one.
  NoteSummary? index;

  /// Total docs beneath this folder, so a collapsed folder still says
  /// how much it hides.
  int get count =>
      files.length + children.values.fold(0, (sum, c) => sum + c.count);
}

class _TreeFile {
  _TreeFile(this.name, this.doc);
  final String name;
  final NoteSummary doc;
}
