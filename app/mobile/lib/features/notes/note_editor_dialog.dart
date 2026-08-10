import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/notes/doc_preview.dart';
import 'package:opendray/features/notes/markdown_highlight_controller.dart';
import 'package:opendray/features/notes/note_actions.dart';
import 'package:opendray/features/notes/vault_text.dart';
import 'package:path/path.dart' as p;

// Full-screen markdown note editor. Used by both the session
// inspector's Notes tab and the vault browser — same save semantics,
// same error surface, so kept in one place.
//
// Beyond editing it carries the vault's navigation layer, which the
// phone previously had none of: the document's tags, its outline, the
// notes linking to it, and [[wiki links]] that actually go somewhere.
class NoteEditorDialog extends ConsumerStatefulWidget {
  const NoteEditorDialog({required this.path, this.allPaths, super.key});

  // Vault-relative path (e.g. "personal/foo.md", "projects/bar/spec.md").
  final String path;

  /// Every path in the vault, for resolving and completing wiki-links.
  /// Callers that already hold the listing pass it to save a round-trip;
  /// when null the editor fetches it once for itself.
  final List<String>? allPaths;

  static Future<void> show({
    required BuildContext context,
    required String path,
    List<String>? allPaths,
  }) {
    return showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (_) => NoteEditorDialog(path: path, allPaths: allPaths),
    );
  }

  @override
  ConsumerState<NoteEditorDialog> createState() => _NoteEditorDialogState();
}

/// What the editor's overflow menu offers beyond editing.
enum _DocAction { rename, delete }

class _NoteEditorDialogState extends ConsumerState<NoteEditorDialog> {
  /// The document being edited. Deliberately not `widget.path`: a
  /// rename moves the open document, and everything below — reads,
  /// writes, backlinks, the preview — has to follow it rather than keep
  /// addressing the path it was opened under.
  late String _path = widget.path;

  /// Set once the document has been deleted from under us, so the
  /// flush-on-close cannot write it straight back.
  bool _deleted = false;

  final _ctrl = MarkdownHighlightController();
  final _focus = FocusNode();
  bool _dirty = false;
  bool _loading = true;
  bool _saving = false;
  String? _error;
  String _initial = '';
  DateTime? _lastSaved;
  // The phone had no rendered view of a document at all — only source.
  // HTML made that untenable (a page of tags), so preview now covers
  // BOTH kinds rather than leaving markdown as the poor relation.
  bool _preview = false;

  List<String> _allPaths = const [];
  // Active `[[…` span under the caret, or null. Drives the suggestion
  // list; kept in state rather than recomputed in build so a rebuild
  // triggered by anything else doesn't resurrect a dismissed popup.
  WikiLinkContext? _linkCtx;

  List<Backlink>? _backlinks;
  bool _backlinksLoading = false;
  String? _backlinksError;

  String get _currentDir {
    final dir = p.dirname(_path);
    return (dir == '.' || dir == '/') ? '' : dir;
  }

  @override
  void initState() {
    super.initState();
    _allPaths = widget.allPaths ?? const [];
    // One listener rather than TextField.onChanged: the caret moving
    // without the text changing is exactly when a wiki-link suggestion
    // has to appear or go away.
    _ctrl.addListener(_onEdit);
    _bootstrap();
  }

  @override
  void dispose() {
    _ctrl
      ..removeListener(_onEdit)
      ..dispose();
    _focus.dispose();
    super.dispose();
  }

  Future<void> _bootstrap() async {
    final api = ref.read(notesApiProvider);
    if (widget.allPaths == null) {
      // Wiki-links degrade to plain paths without this, so a failure
      // costs resolution quality, not the editor.
      unawaited(
        api.list().then((notes) {
          if (mounted) {
            setState(() => _allPaths = notes.map((n) => n.path).toList());
          }
        }).catchError((Object _) {}),
      );
    }
    try {
      final note = await api.read(_path);
      if (!mounted) return;
      _initial = note.body;
      _ctrl.text = note.body;
      setState(() => _loading = false);
    } on ApiException catch (e) {
      // 404 just means the file doesn't exist yet — fine, start blank
      // (this is how a freshly-created note opens before its first
      // save round-trip).
      if (e.statusCode == 404) {
        if (mounted) setState(() => _loading = false);
        return;
      }
      if (mounted) {
        setState(() {
          _loading = false;
          _error = t.notesPage.editor.loadFailedApi(error: e.message);
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _error = t.notesPage.editor.loadFailedGeneric(error: e.toString());
        });
      }
    }
  }

  // Saving is explicit — a timer that wrote after every pause in typing
  // kept rewriting a long document while it was being worked on. The
  // dirty flag drives the Save action's enabled state; the flushes on
  // preview-toggle and close below are the safety net.
  void _onEdit() {
    if (_loading) return;
    final nowDirty = _ctrl.text != _initial;
    final sel = _ctrl.selection;
    final ctx = (sel.isCollapsed && sel.start >= 0 && sel.start <= _ctrl.text.length)
        ? detectWikiLinkContext(_ctrl.text, sel.start)
        : null;
    final ctxChanged =
        (ctx == null) != (_linkCtx == null) || ctx?.query != _linkCtx?.query;
    if (nowDirty == _dirty && !ctxChanged) return;
    setState(() {
      _dirty = nowDirty;
      _linkCtx = ctx;
    });
  }

  Future<void> _save() async {
    // Writing after a delete would recreate the document — the close
    // button flushes, and closing is exactly what follows a delete.
    if (_deleted) return;
    final body = _ctrl.text;
    if (body == _initial) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await ref.read(notesApiProvider).write(path: _path, body: body);
      if (!mounted) return;
      _initial = body;
      setState(() {
        _dirty = false;
        _saving = false;
        _lastSaved = DateTime.now();
      });
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _saving = false;
          _error = t.notesPage.editor.saveFailedApi(error: e.message);
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _saving = false;
          _error = t.notesPage.editor.saveFailedGeneric(error: e.toString());
        });
      }
    }
  }

  // ------------------------------------------------------------ outline

  Future<void> _showOutline() async {
    final headings = extractOutline(_ctrl.text);
    final picked = await showModalBottomSheet<OutlineHeading>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => SafeArea(
        top: false,
        child: headings.isEmpty
            ? Padding(
                padding: const EdgeInsets.all(24),
                child: Text(
                  t.notesPage.outline.empty,
                  style: Theme.of(sheetCtx).textTheme.bodyMedium,
                ),
              )
            : Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 14, 16, 6),
                    child: Align(
                      alignment: Alignment.centerLeft,
                      child: Text(
                        t.notesPage.outline.title,
                        style: Theme.of(sheetCtx).textTheme.titleSmall,
                      ),
                    ),
                  ),
                  const Divider(height: 1),
                  Flexible(
                    child: ListView.builder(
                      shrinkWrap: true,
                      itemCount: headings.length,
                      itemBuilder: (_, i) {
                        final h = headings[i];
                        return ListTile(
                          dense: true,
                          contentPadding: EdgeInsets.only(
                            // Indent by depth so the document's shape is
                            // visible, but cap it — a level-6 heading
                            // shouldn't end up off the right edge.
                            left: 16.0 + (math.min(h.level, 4) - 1) * 14.0,
                            right: 16,
                          ),
                          title: Text(
                            h.text,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              fontSize: h.level <= 2 ? 14 : 13,
                              fontWeight: h.level <= 2
                                  ? FontWeight.w600
                                  : FontWeight.normal,
                            ),
                          ),
                          onTap: () => Navigator.of(sheetCtx).pop(h),
                        );
                      },
                    ),
                  ),
                  const SizedBox(height: 4),
                ],
              ),
      ),
    );
    if (picked == null || !mounted) return;

    // The jump lands in SOURCE view, and that is a constraint rather
    // than a preference: the preview runs with JavaScript off because a
    // vault document can arrive by `git pull`, and scrolling a webview
    // to an anchor needs a script. Moving the caret is what the phone
    // can do honestly — Flutter scrolls a focused field to its caret.
    final wasPreview = _preview;
    if (wasPreview) setState(() => _preview = false);
    _ctrl.selection = TextSelection.collapsed(
      offset: math.min(picked.charOffset, _ctrl.text.length),
    );
    _focus.requestFocus();
    if (wasPreview) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(t.notesPage.outline.jumpedToSource),
          behavior: SnackBarBehavior.floating,
          duration: const Duration(seconds: 3),
        ),
      );
    }
  }

  // ---------------------------------------------------------- backlinks

  Future<void> _loadBacklinks() async {
    if (_backlinksLoading) return;
    setState(() {
      _backlinksLoading = true;
      _backlinksError = null;
    });
    try {
      final links = await ref.read(notesApiProvider).backlinks(_path);
      if (!mounted) return;
      setState(() {
        _backlinks = links;
        _backlinksLoading = false;
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _backlinksLoading = false;
        // A note that hasn't been written yet has no links to it. That
        // is an empty list, not a failure to report.
        _backlinks = e.statusCode == 404 ? const [] : null;
        _backlinksError = e.statusCode == 404
            ? null
            : t.notesPage.backlinks.failed(error: e.message);
      });
    } on Object catch (e) {
      if (!mounted) return;
      setState(() {
        _backlinksLoading = false;
        _backlinksError = t.notesPage.backlinks.failed(error: e.toString());
      });
    }
  }

  // ---------------------------------------------------- rename / delete

  // These live here as well as on the browser's row menu because the
  // moment you decide a document is finished with is the moment you are
  // looking at it. Having to close the editor, find the row again and
  // press-and-hold it is how "the vault has no delete" happens.

  Future<void> _rename() async {
    // Flush first: the move copies what is on disk, so unsaved edits
    // would be left behind at the old path.
    await _save();
    if (!mounted) return;
    final to = await renameNoteFlow(context: context, ref: ref, path: _path);
    if (to == null || !mounted) return;
    setState(() {
      _path = to;
      // Backlinks were resolved against the old path; the answer for
      // the new one has to be asked for again.
      _backlinks = null;
      _backlinksError = null;
    });
  }

  Future<void> _delete() async {
    // The confirmation names the document by its first heading, the
    // same thing the listing shows, so the dialog is recognisably about
    // the note on screen and not just a path.
    final headings = extractOutline(_ctrl.text);
    final gone = await deleteNoteFlow(
      context: context,
      ref: ref,
      path: _path,
      title: headings.isEmpty ? '' : headings.first.text,
    );
    if (!gone || !mounted) return;
    // _deleted before popping: closing flushes, and a flush here would
    // put the document straight back.
    setState(() => _deleted = true);
    Navigator.of(context).pop();
  }

  // --------------------------------------------------------- wiki links

  String _resolve(String target) => resolveWikiLink(
        target,
        allPaths: _allPaths,
        currentDir: _currentDir,
      );

  Future<void> _openLinked(String path) async {
    // Flush first: the linked note may link back, and reading a stale
    // copy of the document just left would be confusing.
    await _save();
    if (!mounted) return;
    await NoteEditorDialog.show(
      context: context,
      path: path,
      allPaths: _allPaths,
    );
    if (!mounted) return;
    // A note opened through a link may have been created; refresh the
    // backlinks that are already on screen.
    if (_backlinks != null) unawaited(_loadBacklinks());
  }

  /// Candidate notes for the active `[[…` span, basename-first.
  List<String> _suggestions(String query) {
    final q = query.trim().toLowerCase();
    final out = <String>[];
    for (final path in _allPaths) {
      if (path == _path) continue;
      final base = p.basenameWithoutExtension(path).toLowerCase();
      if (q.isEmpty || base.contains(q) || path.toLowerCase().contains(q)) {
        out.add(path);
      }
      if (out.length >= 8) break;
    }
    return out;
  }

  void _completeWikiLink(String display, {String? createAt}) {
    final ctx = _linkCtx;
    if (ctx == null) return;
    final text = _ctrl.text;
    final end = math.min(_ctrl.selection.start, text.length);
    if (ctx.openIdx < 0 || ctx.openIdx > end) return;
    final inserted = '[[$display]]';
    final next = text.substring(0, ctx.openIdx) + inserted + text.substring(end);
    _ctrl.value = TextEditingValue(
      text: next,
      selection: TextSelection.collapsed(offset: ctx.openIdx + inserted.length),
    );
    setState(() => _linkCtx = null);
    if (createAt != null) {
      // Lazy-create so the note exists in the index and the next
      // completion finds it. A failure here is non-fatal: the document
      // gets created the first time it is saved.
      unawaited(
        ref
            .read(notesApiProvider)
            .write(path: createAt, body: '# $display\n\n')
            .then((_) {
          if (mounted && !_allPaths.contains(createAt)) {
            setState(() => _allPaths = [..._allPaths, createAt]);
          }
        }).catchError((Object _) {}),
      );
    }
  }

  // -------------------------------------------------------------- build

  @override
  Widget build(BuildContext context) {
    final tags = _loading ? const <String>[] : extractTags(_ctrl.text);
    return Dialog(
      insetPadding: const EdgeInsets.all(8),
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(context).size.height * 0.92,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            _header(),
            const Divider(height: 1),
            if (tags.isNotEmpty) _tagChips(tags),
            Expanded(
              child: _loading
                  ? const Center(child: CircularProgressIndicator())
                  : _preview
                      ? DocPreview(
                          path: _path,
                          body: _ctrl.text,
                          onOpenWikiLink: _openLinked,
                          resolveWikiTarget: _resolve,
                        )
                      : Padding(
                          padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
                          child: TextField(
                            controller: _ctrl,
                            focusNode: _focus,
                            maxLines: null,
                            expands: true,
                            textInputAction: TextInputAction.newline,
                            keyboardType: TextInputType.multiline,
                            style: const TextStyle(
                              fontSize: 13,
                              height: 1.5,
                              fontFamily: 'monospace',
                            ),
                            decoration: InputDecoration(
                              border: InputBorder.none,
                              contentPadding: EdgeInsets.zero,
                              hintText: t.notesPage.editor.markdownHint,
                            ),
                          ),
                        ),
            ),
            if (!_preview && _linkCtx != null) _wikiSuggestions(_linkCtx!),
            const Divider(height: 1),
            if (!_loading) _backlinksTile(),
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 6, 12, 8),
              child: NoteSaveStatus(
                saving: _saving,
                lastSaved: _lastSaved,
                dirty: _dirty,
                error: _error,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _header() {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 4, 8),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  p.basename(_path),
                  style: Theme.of(context).textTheme.titleSmall,
                  overflow: TextOverflow.ellipsis,
                ),
                Text(
                  _path,
                  style: Theme.of(context).textTheme.bodySmall,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ),
          IconButton(
            tooltip: t.notesPage.outline.action,
            icon: const Icon(Icons.toc),
            visualDensity: VisualDensity.compact,
            onPressed: _loading ? null : () => unawaited(_showOutline()),
          ),
          IconButton(
            tooltip: t.notesPage.editor.save,
            icon: const Icon(Icons.save_outlined),
            visualDensity: VisualDensity.compact,
            // Disabled with nothing to save, so the control doubles as
            // the answer to "did that go through?".
            onPressed: _dirty && !_saving ? () => unawaited(_save()) : null,
          ),
          IconButton(
            tooltip: _preview
                ? t.notesPage.editor.showSource
                : t.notesPage.editor.showPreview,
            icon: Icon(_preview ? Icons.code : Icons.visibility_outlined),
            visualDensity: VisualDensity.compact,
            onPressed: () {
              // Flush before switching: preview renders the SAVED body,
              // so an unsaved keystroke would otherwise render stale.
              unawaited(_save());
              setState(() {
                _preview = !_preview;
                _linkCtx = null;
              });
            },
          ),
          PopupMenuButton<_DocAction>(
            icon: const Icon(Icons.more_vert),
            tooltip: t.common.more,
            enabled: !_loading,
            // Five controls plus the path share this row on a phone;
            // the default padding is what pushes the title into an
            // ellipsis two characters in.
            padding: EdgeInsets.zero,
            onSelected: (action) {
              switch (action) {
                case _DocAction.rename:
                  unawaited(_rename());
                case _DocAction.delete:
                  unawaited(_delete());
              }
            },
            itemBuilder: (menuCtx) => [
              PopupMenuItem(
                value: _DocAction.rename,
                child: ListTile(
                  dense: true,
                  contentPadding: EdgeInsets.zero,
                  leading: const Icon(Icons.drive_file_rename_outline),
                  title: Text(t.notesPage.rename.action),
                ),
              ),
              PopupMenuItem(
                value: _DocAction.delete,
                child: ListTile(
                  dense: true,
                  contentPadding: EdgeInsets.zero,
                  leading: Icon(
                    Icons.delete_outline,
                    color: Theme.of(menuCtx).colorScheme.error,
                  ),
                  title: Text(
                    t.notesPage.popupDelete,
                    style:
                        TextStyle(color: Theme.of(menuCtx).colorScheme.error),
                  ),
                ),
              ),
            ],
          ),
          Builder(
            builder: (innerCtx) => IconButton(
              icon: const Icon(Icons.close),
              visualDensity: VisualDensity.compact,
              onPressed: () async {
                // Flush pending edits before dismissing rather than
                // dropping the last keystrokes on the floor.
                await _save();
                if (!innerCtx.mounted) return;
                Navigator.of(innerCtx).pop();
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _tagChips(List<String> tags) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      // Same reasoning as the backlinks list: a non-flex child that can
      // grow without limit steals height from the editor. A heavily
      // tagged document scrolls its chips instead.
      constraints: const BoxConstraints(maxHeight: 62),
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
      child: SingleChildScrollView(
        child: Wrap(
          spacing: 6,
          runSpacing: 4,
          children: [
            for (final tag in tags)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
                decoration: BoxDecoration(
                  color: scheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(
                  '#$tag',
                  style: TextStyle(
                    fontSize: 11,
                    fontFamily: 'monospace',
                    color: scheme.onSurfaceVariant,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _wikiSuggestions(WikiLinkContext ctx) {
    final scheme = Theme.of(context).colorScheme;
    final matches = _suggestions(ctx.query);
    final query = ctx.query.trim();
    final exact = matches.any(
      (path) =>
          p.basenameWithoutExtension(path).toLowerCase() == query.toLowerCase(),
    );
    // Anchored above the status bar rather than floated at the caret:
    // a caret-tracking popup on a phone fights the keyboard for the
    // same strip of screen and loses.
    return Container(
      constraints: const BoxConstraints(maxHeight: 168),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHighest,
        border: Border(top: BorderSide(color: Theme.of(context).dividerColor)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 6, 12, 2),
            child: Text(
              t.notesPage.wikiLink.suggestions,
              style: TextStyle(fontSize: 11, color: scheme.onSurfaceVariant),
            ),
          ),
          Flexible(
            child: ListView(
              shrinkWrap: true,
              padding: EdgeInsets.zero,
              children: [
                for (final path in matches)
                  ListTile(
                    dense: true,
                    visualDensity: VisualDensity.compact,
                    title: Text(
                      p.basenameWithoutExtension(path),
                      style: const TextStyle(fontSize: 13),
                      overflow: TextOverflow.ellipsis,
                    ),
                    subtitle: Text(
                      path,
                      style: const TextStyle(
                          fontSize: 10, fontFamily: 'monospace'),
                      overflow: TextOverflow.ellipsis,
                    ),
                    onTap: () => _completeWikiLink(
                      p.basenameWithoutExtension(path),
                    ),
                  ),
                if (query.isNotEmpty && !exact)
                  ListTile(
                    dense: true,
                    visualDensity: VisualDensity.compact,
                    leading: const Icon(Icons.add, size: 18),
                    title: Text(
                      t.notesPage.wikiLink.createNew(name: query),
                      style: const TextStyle(fontSize: 13),
                      overflow: TextOverflow.ellipsis,
                    ),
                    onTap: () => _completeWikiLink(
                      query,
                      createAt: _resolve(query),
                    ),
                  ),
                if (matches.isEmpty && query.isEmpty)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(12, 4, 12, 10),
                    child: Text(
                      t.notesPage.wikiLink.noMatches,
                      style: TextStyle(
                        fontSize: 12,
                        color: scheme.onSurfaceVariant,
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _backlinksTile() {
    final links = _backlinks;
    final scheme = Theme.of(context).colorScheme;
    return Theme(
      // The default expansion tile draws its own dividers, which double
      // up with the ones this dialog already has.
      data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
      child: ExpansionTile(
        tilePadding: const EdgeInsets.symmetric(horizontal: 12),
        childrenPadding: const EdgeInsets.only(bottom: 4),
        dense: true,
        visualDensity: VisualDensity.compact,
        leading: const Icon(Icons.link, size: 18),
        title: Text(
          links == null
              ? t.notesPage.backlinks.title
              : '${t.notesPage.backlinks.title} · ${links.length}',
          style: const TextStyle(fontSize: 13),
        ),
        onExpansionChanged: (open) {
          // Deferred until asked for: it is a full scan of the vault on
          // the gateway, and most of the time nobody opens it.
          if (open && _backlinks == null) unawaited(_loadBacklinks());
        },
        children: [
          if (_backlinksLoading)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 10),
              child: Row(
                children: [
                  const SizedBox(
                    width: 12,
                    height: 12,
                    child: CircularProgressIndicator(strokeWidth: 1.5),
                  ),
                  const SizedBox(width: 8),
                  Text(
                    t.notesPage.backlinks.loading,
                    style: TextStyle(
                        fontSize: 12, color: scheme.onSurfaceVariant),
                  ),
                ],
              ),
            )
          else if (_backlinksError != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 10),
              child: Text(
                _backlinksError!,
                style: TextStyle(fontSize: 12, color: scheme.error),
              ),
            )
          else if (links != null && links.isEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 10),
              child: Text(
                t.notesPage.backlinks.empty,
                style:
                    TextStyle(fontSize: 12, color: scheme.onSurfaceVariant),
              ),
            )
          else if (links != null)
            // Capped and internally scrollable. The tile's children are
            // a NON-FLEX child of the dialog's column, so an unbounded
            // list would squeeze the editor above it to nothing and
            // overflow — a document with twenty inbound links is not
            // unusual in a wiki.
            ConstrainedBox(
              constraints: BoxConstraints(
                maxHeight: MediaQuery.of(context).size.height * 0.28,
              ),
              child: ListView(
                shrinkWrap: true,
                padding: EdgeInsets.zero,
                children: [
                  for (final b in links)
                    ListTile(
                      dense: true,
                      visualDensity: VisualDensity.compact,
                      title: Text(
                        b.title.isNotEmpty ? b.title : p.basename(b.path),
                        style: const TextStyle(fontSize: 13),
                        overflow: TextOverflow.ellipsis,
                      ),
                      subtitle: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            b.path,
                            style: const TextStyle(
                                fontSize: 10, fontFamily: 'monospace'),
                            overflow: TextOverflow.ellipsis,
                          ),
                          // Two snippets: enough to recognise the
                          // reference, not so many that one note buries
                          // the rest.
                          for (final line in b.lines.take(2))
                            Text(
                              line.trim(),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 10.5,
                                fontFamily: 'monospace',
                                color: scheme.onSurfaceVariant,
                              ),
                            ),
                        ],
                      ),
                      onTap: () => unawaited(_openLinked(b.path)),
                    ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

// NoteSaveStatus is the compact "Saving… / Saved 14:08 / error" line
// shown beneath the editor. Public because the inspector's personal
// scratchpad reuses it without going through the full dialog.
class NoteSaveStatus extends StatelessWidget {
  const NoteSaveStatus({
    required this.saving,
    required this.lastSaved,
    this.dirty = false,
    this.error,
    super.key,
  });

  final bool saving;
  final DateTime? lastSaved;
  /// Unsaved edits are pending. Saving is explicit, so this is the only
  /// thing telling someone their last keystrokes are not on disk yet.
  final bool dirty;
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
          Text(t.notesPage.editor.saving, style: muted),
        ],
      );
    }
    // Unsaved wins over "saved at 14:03": the older fact is true and
    // the newer one is what matters.
    if (dirty) {
      return Text(t.notesPage.editor.unsaved, style: muted);
    }
    if (lastSaved != null) {
      return Text(
        t.notesPage.editor.savedAt(time: DateFormat.Hm().format(lastSaved!.toLocal())),
        style: muted,
      );
    }
    // This label used to read "auto-saves as you type", which stopped
    // being true when saving became explicit.
    return Text(t.notesPage.editor.autosave, style: muted);
  }
}
