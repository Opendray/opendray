import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/skills_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// SkillEditorScreen handles three flows from one page:
//
// - **New skill**: existing == null. Asks for an id slug, then a
//   markdown body. POST creates a vault entry.
// - **Edit vault skill**: existing.isVault. Loads body via GET,
//   PUTs on save. Trash icon deletes the vault row (if it shadows
//   a builtin, the action label switches to "Reset to built-in").
// - **Customize built-in**: existing.isBuiltin. Loads the embedded
//   body, lets the operator edit, and PUTs as a vault override on
//   save (server-side this is the same upsert as Edit).
class SkillEditorScreen extends ConsumerStatefulWidget {
  const SkillEditorScreen({this.existing, super.key});
  final SkillSummary? existing;

  @override
  ConsumerState<SkillEditorScreen> createState() => _SkillEditorScreenState();
}

class _SkillEditorScreenState extends ConsumerState<SkillEditorScreen> {
  final _idCtrl = TextEditingController();
  final _bodyCtrl = TextEditingController();
  bool _loading = true;
  bool _saving = false;
  bool _deleting = false;
  String? _error;
  String _initialBody = '';

  bool get _isNew => widget.existing == null;
  // Customize-from-builtin acts like an upsert; UI labels treat it as
  // a "save creates an override" flow.
  bool get _isCustomize =>
      widget.existing != null && widget.existing!.isBuiltin;

  @override
  void initState() {
    super.initState();
    if (_isNew) {
      _loading = false;
      _initialBody = '';
    } else {
      _bootstrap();
    }
  }

  @override
  void dispose() {
    _idCtrl.dispose();
    _bodyCtrl.dispose();
    super.dispose();
  }

  Future<void> _bootstrap() async {
    final id = widget.existing!.id;
    try {
      final s = await ref.read(skillsApiProvider).get(id);
      if (!mounted) return;
      _idCtrl.text = s.summary.id;
      _bodyCtrl.text = s.body;
      _initialBody = s.body;
      setState(() => _loading = false);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _error = t.skills.loadFailedApi(error: e.message);
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _error = t.skills.loadFailedGeneric(error: e.toString());
        });
      }
    }
  }

  Future<void> _save() async {
    if (_saving) return;
    final id = _idCtrl.text.trim();
    if (id.isEmpty) {
      setState(() => _error = t.skills.idRequired);
      return;
    }
    final body = _bodyCtrl.text;
    if (body.trim().isEmpty) {
      setState(() => _error = t.skills.bodyRequired);
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    try {
      if (_isNew) {
        await ref.read(skillsApiProvider).create(id: id, body: body);
      } else {
        await ref.read(skillsApiProvider).update(id: id, body: body);
      }
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            _isNew
                ? t.skills.snackCreated
                : _isCustomize
                    ? t.skills.snackOverride
                    : t.skills.snackUpdated,
          ),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      _initialBody = body;
      navigator.pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _saving = false;
          _error = t.skills.saveFailedApi(error: e.message);
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _saving = false;
          _error = t.skills.saveFailedGeneric(error: e.toString());
        });
      }
    }
  }

  Future<void> _delete() async {
    final ex = widget.existing;
    if (ex == null || _deleting) return;
    final overrides = ex.overridesBuiltin;
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(overrides ? t.skills.resetTitle : t.skills.deleteTitle),
        content: Text(
          overrides
              ? t.skills.resetBody(id: ex.id)
              : t.skills.deleteBody(id: ex.id),
          style: Theme.of(ctx).textTheme.bodySmall,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(overrides ? t.skills.resetButton : t.common.delete),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    setState(() => _deleting = true);
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    try {
      await ref.read(skillsApiProvider).delete(ex.id);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            overrides
                ? t.skills.resetSnack(id: ex.id)
                : t.skills.deletedSnack(id: ex.id),
          ),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      navigator.pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _deleting = false;
          _error = t.skills.deleteFailedApi(error: e.message);
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _deleting = false;
          _error = t.skills.deleteFailedGeneric(error: e.toString());
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final ex = widget.existing;
    final dirty = _bodyCtrl.text != _initialBody;
    return Scaffold(
      appBar: AppBar(
        title: Text(
          _isNew
              ? t.skills.newSkillTitle
              : _isCustomize
                  ? t.skills.customizeTitle(id: ex!.id)
                  : t.skills.editTitle(id: ex!.id),
        ),
        actions: [
          if (!_isNew && ex!.isVault)
            IconButton(
              icon: _deleting
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Icon(
                      ex.overridesBuiltin
                          ? Icons.restart_alt
                          : Icons.delete_outline,
                      color: Theme.of(context).colorScheme.error,
                    ),
              tooltip: ex.overridesBuiltin
                  ? t.skills.resetTooltip
                  : t.skills.deleteTooltip,
              onPressed: _deleting ? null : _delete,
            ),
          TextButton(
            onPressed: _saving || _loading || (!_isNew && !dirty)
                ? null
                : _save,
            child: Text(
              _saving
                  ? t.skills.saving
                  : _isCustomize
                      ? t.skills.saveOverride
                      : t.common.save,
            ),
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (_isCustomize)
                    _InfoBanner(
                      tone: Theme.of(context).colorScheme.tertiary,
                      title: t.skills.customizingBuiltin(id: ex!.id),
                      body: t.skills.overrideBanner,
                    ),
                  TextField(
                    controller: _idCtrl,
                    autocorrect: false,
                    enabled: _isNew,
                    decoration: InputDecoration(
                      labelText: t.skills.idLabel,
                      hintText: t.skills.idHint,
                      helperText: t.skills.idHelper,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                  const SizedBox(height: 12),
                  Expanded(
                    child: TextField(
                      controller: _bodyCtrl,
                      onChanged: (_) => setState(() {}),
                      autofocus: !_isNew,
                      autocorrect: false,
                      maxLines: null,
                      expands: true,
                      textInputAction: TextInputAction.newline,
                      keyboardType: TextInputType.multiline,
                      style: const TextStyle(
                        fontFamily: 'monospace',
                        fontSize: 13,
                        height: 1.4,
                      ),
                      decoration: InputDecoration(
                        labelText: t.skills.bodyLabel,
                        alignLabelWithHint: true,
                        border: const OutlineInputBorder(),
                      ),
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
            ),
    );
  }
}

class _InfoBanner extends StatelessWidget {
  const _InfoBanner({
    required this.tone,
    required this.title,
    required this.body,
  });
  final Color tone;
  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: tone.withValues(alpha: 0.10),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.info_outline, size: 14, color: tone),
              const SizedBox(width: 6),
              Text(
                title,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: tone,
                ),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            body,
            style: TextStyle(
              fontSize: 11,
              height: 1.4,
              color: Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.75),
            ),
          ),
        ],
      ),
    );
  }
}
