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

  // Vault-relative project docs prefix. Tries three sources in order:
  //   1. mapping.path with vault root stripped — respects custom
  //      overrides AND the default when both paths agree on
  //      normalization.
  //   2. mapping.path with the home tilde prefix swapped against
  //      info.root — handles the macOS /Users vs /private/Users
  //      symlink case where startsWith fails on otherwise-equal
  //      paths.
  //   3. <projects_prefix>/<slug(basename(cwd))> — matches the
  //      web admin convention (app/shared/src/lib/notes.ts
  //      projectNoteDir()) and works even when info.root is empty
  //      because notes is misconfigured server-side.
  String _projectsRel(NotesInfo info, ProjectMapping mapping, String cwd) {
    // 1. literal startsWith strip
    if (info.root.isNotEmpty && mapping.path.isNotEmpty) {
      final r1 = _stripPrefix(mapping.path, info.root);
      if (r1 != null) return r1;

      // 2. /private prefix tolerance for macOS
      const privatePrefix = '/private';
      if (mapping.path.startsWith(privatePrefix) &&
          !info.root.startsWith(privatePrefix)) {
        final r2 = _stripPrefix(
          mapping.path.substring(privatePrefix.length),
          info.root,
        );
        if (r2 != null) return r2;
      }
      if (info.root.startsWith(privatePrefix) &&
          !mapping.path.startsWith(privatePrefix)) {
        final r3 = _stripPrefix(
          mapping.path,
          info.root.substring(privatePrefix.length),
        );
        if (r3 != null) return r3;
      }
    }
    // 3. convention fallback
    final prefix = info.projectsPrefix.isNotEmpty
        ? info.projectsPrefix
        : 'projects';
    return '$prefix/${_cwdSlug(cwd)}';
  }

  // Returns the substring of [absPath] after [root], or null if
  // [absPath] is not under [root]. Tolerates trailing-slash mismatch
  // on either side.
  String? _stripPrefix(String absPath, String root) {
    if (root.isEmpty || absPath.isEmpty) return null;
    final r = root.endsWith('/') ? root : '$root/';
    if (absPath == root || absPath == r.substring(0, r.length - 1)) {
      return '';
    }
    if (absPath.startsWith(r)) {
      return absPath.substring(r.length);
    }
    return null;
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
          content: Text(t.sessions.inspector.notes.insertedAt(path: widget.personalPath)),
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
      await NoteEditorDialog.show(context: context, path: path);
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
            'authored by AI agents. Use the settings button to point at a '
            'different vault folder.';
    return _SectionCard(
      icon: Icons.auto_awesome,
      title: t.sessions.inspector.notes.projectDocs,
      subtitle: widget.projectsRel.isEmpty
          ? '(no project mapping)'
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
              'Could not resolve a project mapping for this session. '
              'Check that the gateway has a notes vault configured '
              'and the session has a non-empty cwd.',
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
          content: Text(t.sessions.inspector.shared.inserted(text: text)),
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
          child: Text(t.common.cancel),
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

String _stripExt(String name) {
  final i = name.lastIndexOf('.');
  return i > 0 ? name.substring(0, i) : name;
}
