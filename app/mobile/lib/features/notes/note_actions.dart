import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/notes/vault_text.dart';
import 'package:path/path.dart' as p;

// Rename and delete, as flows rather than as buttons — prompt, call,
// report — so the vault browser and the open editor can both offer them
// without either one owning the behaviour. They were the browser's
// private methods until the editor needed them too, and the answer to
// "the editor needs this as well" is one implementation, not two.

/// Prompt for a new path and move the note, repointing the
/// [[wiki links]] that referenced the old one. Returns the new path, or
/// null when cancelled or failed — failures are reported to the user
/// here, so callers only have to handle success.
Future<String?> renameNoteFlow({
  required BuildContext context,
  required WidgetRef ref,
  required String path,
}) async {
  final ctrl = TextEditingController(text: path);
  final String? typed;
  try {
    typed = await showDialog<String>(
      context: context,
      builder: (dialogCtx) => AlertDialog(
        title: Text(t.notesPage.rename.title),
        content: TextField(
          controller: ctrl,
          autofocus: true,
          autocorrect: false,
          decoration: InputDecoration(
            helperText: t.notesPage.rename.helper,
            hintText: t.notesPage.pathHint,
          ),
          style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogCtx).pop(),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogCtx).pop(ctrl.text.trim()),
            child: Text(t.notesPage.rename.action),
          ),
        ],
      ),
    );
  } finally {
    ctrl.dispose();
  }
  if (typed == null || typed.isEmpty) return null;
  // Same cleaning as creation, so a rename to `guide.html` stays HTML
  // instead of becoming `guide.html.md`.
  final to = sanitizeNotePath(typed);
  if (to == path || !context.mounted) return null;

  final messenger = ScaffoldMessenger.of(context);
  try {
    final res = await ref.read(notesApiProvider).move(from: path, to: to);
    // A warning means the file moved but the link rewrite did not
    // finish. Reporting only "renamed" would hide a vault full of
    // references to a path that no longer exists.
    messenger.showSnackBar(
      SnackBar(
        content: Text(
          (res.warning?.isNotEmpty ?? false)
              ? '${t.notesPage.rename.doneWithWarning}: ${res.warning}'
              : t.notesPage.rename.doneSnack(count: res.linksRewritten),
        ),
        behavior: SnackBarBehavior.floating,
      ),
    );
    return res.to.isNotEmpty ? res.to : to;
  } on ApiException catch (e) {
    messenger.showSnackBar(
      SnackBar(content: Text(e.message), behavior: SnackBarBehavior.floating),
    );
    return null;
  }
}

/// Confirm, then delete. Returns true only when the note is gone.
Future<bool> deleteNoteFlow({
  required BuildContext context,
  required WidgetRef ref,
  required String path,
  String title = '',
}) async {
  final ok = await showDialog<bool>(
    context: context,
    builder: (dialogCtx) => AlertDialog(
      title: Text(t.notesPage.deleteTitle),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title.isNotEmpty ? title : p.basename(path),
            style: Theme.of(dialogCtx).textTheme.bodyMedium,
          ),
          const SizedBox(height: 4),
          Text(
            path,
            style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
          ),
          const SizedBox(height: 8),
          Text(
            t.notesPage.deleteBody,
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
  if (ok != true || !context.mounted) return false;

  final messenger = ScaffoldMessenger.of(context);
  try {
    await ref.read(notesApiProvider).delete(path);
    messenger.showSnackBar(
      SnackBar(
        content: Text(t.notesPage.deletedSnack(path: path)),
        behavior: SnackBarBehavior.floating,
      ),
    );
    return true;
  } on ApiException catch (e) {
    messenger.showSnackBar(
      SnackBar(content: Text(t.notesPage.deleteFailedApi(error: e.message))),
    );
    return false;
  } on Object catch (e) {
    messenger.showSnackBar(
      SnackBar(
        content: Text(t.notesPage.deleteFailedGeneric(error: e.toString())),
      ),
    );
    return false;
  }
}
