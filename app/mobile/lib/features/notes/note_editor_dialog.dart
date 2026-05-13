import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:path/path.dart' as p;

// Full-screen markdown note editor with debounced auto-save.
// Used by both the session inspector's Notes tab and the global
// Notes screen — same auto-save semantics, same error / save-status
// surface, so kept in one place.
class NoteEditorDialog extends ConsumerStatefulWidget {
  const NoteEditorDialog({required this.path, super.key});

  // Vault-relative path (e.g. "personal/foo.md", "projects/bar/spec.md").
  final String path;

  static Future<void> show({
    required BuildContext context,
    required String path,
  }) {
    return showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (_) => NoteEditorDialog(path: path),
    );
  }

  @override
  ConsumerState<NoteEditorDialog> createState() => _NoteEditorDialogState();
}

class _NoteEditorDialogState extends ConsumerState<NoteEditorDialog> {
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
                        // Flush pending save before dismissing — drops
                        // the last few keystrokes otherwise if the
                        // 800ms timer hasn't fired.
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
                        decoration: InputDecoration(
                          border: InputBorder.none,
                          contentPadding: EdgeInsets.zero,
                          hintText: t.notesPage.editor.markdownHint,
                        ),
                      ),
                    ),
            ),
            const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 6, 12, 8),
              child: NoteSaveStatus(
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

// NoteSaveStatus is the compact "Saving… / Saved 14:08 / error" line
// shown beneath the editor. Public because the inspector's personal
// scratchpad reuses it without going through the full dialog.
class NoteSaveStatus extends StatelessWidget {
  const NoteSaveStatus({
    required this.saving,
    required this.lastSaved,
    this.error,
    super.key,
  });

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
          Text(t.notesPage.editor.saving, style: muted),
        ],
      );
    }
    if (lastSaved != null) {
      return Text(
        'Saved ${DateFormat.Hm().format(lastSaved!.toLocal())}',
        style: muted,
      );
    }
    return Text(t.notesPage.editor.autosave, style: muted);
  }
}
