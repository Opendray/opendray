// Custom tasks — user-defined slash commands that get listed in
// each session's task picker. Single screen with a list + a
// full-screen editor for create/edit. Mirrors the web's
// CustomTasksSection + CustomTaskDialog from Plugins.tsx, dropped
// to a phone-friendly shape.
//
// Empty `cwd` = global task (any session). Non-empty cwd = scoped
// to a specific project; only visible to sessions whose cwd
// matches. The picker on the editor exposes the distinction
// explicitly so operators don't accidentally create a global task
// when they wanted project-scoped, or vice-versa.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/custom_tasks_api.dart';
import 'package:opendray/core/i18n/strings.g.dart' as i18n;

class CustomTasksScreen extends ConsumerStatefulWidget {
  const CustomTasksScreen({super.key});

  @override
  ConsumerState<CustomTasksScreen> createState() =>
      _CustomTasksScreenState();
}

class _CustomTasksScreenState extends ConsumerState<CustomTasksScreen> {
  AsyncValue<List<CustomTask>> _state = const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final list = await ref.read(customTasksApiProvider).list();
      if (!mounted) return;
      list.sort((a, b) => a.name.toLowerCase().compareTo(b.name.toLowerCase()));
      setState(() => _state = AsyncValue.data(list));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _openEditor({CustomTask? existing}) async {
    final saved = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => _CustomTaskEditorScreen(existing: existing),
      ),
    );
    if ((saved ?? false) && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(existing == null
              ? i18n.t.customTasks.snackCreated
              : i18n.t.customTasks.snackUpdated),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    }
  }

  Future<void> _confirmDelete(CustomTask t) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(i18n.t.customTasks.deleteTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              t.name,
              style: const TextStyle(fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 6),
            Text(
              i18n.t.customTasks.deleteBody,
              style: Theme.of(ctx).textTheme.bodySmall,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(i18n.t.common.cancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(i18n.t.common.delete),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(customTasksApiProvider).delete(t.id);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(i18n.t.customTasks.deletedSnack(name: t.name)),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(i18n.t.customTasks.deleteFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(i18n.t.customTasks.deleteFailedGeneric(error: e.toString()))));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(i18n.t.customTasks.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: i18n.t.sessions.inspector.shared.refresh,
            onPressed: _state is AsyncLoading ? null : _load,
          ),
        ],
      ),
      body: _state.when(
        data: _buildList,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
      floatingActionButton: FloatingActionButton.extended(
        heroTag: 'custom_tasks_fab',
        onPressed: _openEditor,
        icon: const Icon(Icons.add),
        label: Text(i18n.t.customTasks.newTask),
      ),
    );
  }

  Widget _buildList(List<CustomTask> list) {
    if (list.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            i18n.t.customTasks.introBanner,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        itemCount: list.length,
        separatorBuilder: (_, __) => Divider(
          height: 1,
          color: Theme.of(context).dividerColor,
        ),
        itemBuilder: (_, i) {
          final t = list[i];
          final isGlobal = t.cwd.isEmpty;
          return ListTile(
            leading: Icon(
              isGlobal ? Icons.public : Icons.folder_outlined,
              size: 20,
            ),
            title: Row(
              children: [
                Flexible(
                  child: Text(
                    t.name,
                    style: const TextStyle(fontWeight: FontWeight.w600),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (!isGlobal) ...[
                  const SizedBox(width: 6),
                  Text(
                    '· scoped',
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          fontSize: 11,
                        ),
                  ),
                ],
              ],
            ),
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (t.description.isNotEmpty)
                  Text(
                    t.description,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                Text(
                  t.command,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        fontFamily: 'monospace',
                        fontSize: 11,
                      ),
                ),
              ],
            ),
            trailing: PopupMenuButton<String>(
              icon: const Icon(Icons.more_vert, size: 20),
              itemBuilder: (_) => [
                PopupMenuItem(value: 'edit', child: Text(i18n.t.customTasks.popupEdit)),
                PopupMenuItem(value: 'delete', child: Text(i18n.t.customTasks.popupDelete)),
              ],
              onSelected: (action) {
                if (action == 'edit') {
                  _openEditor(existing: t);
                } else if (action == 'delete') {
                  _confirmDelete(t);
                }
              },
            ),
            onTap: () => _openEditor(existing: t),
            isThreeLine: t.description.isNotEmpty,
          );
        },
      ),
    );
  }
}

enum _Scope { global, project }

class _CustomTaskEditorScreen extends ConsumerStatefulWidget {
  const _CustomTaskEditorScreen({this.existing});
  final CustomTask? existing;

  @override
  ConsumerState<_CustomTaskEditorScreen> createState() =>
      _CustomTaskEditorScreenState();
}

class _CustomTaskEditorScreenState
    extends ConsumerState<_CustomTaskEditorScreen> {
  final _nameCtrl = TextEditingController();
  final _commandCtrl = TextEditingController();
  final _descriptionCtrl = TextEditingController();
  final _cwdCtrl = TextEditingController();
  _Scope _scope = _Scope.global;
  bool _submitting = false;
  String? _error;

  bool get _isEdit => widget.existing != null;

  @override
  void initState() {
    super.initState();
    final ex = widget.existing;
    if (ex != null) {
      _nameCtrl.text = ex.name;
      _commandCtrl.text = ex.command;
      _descriptionCtrl.text = ex.description;
      _cwdCtrl.text = ex.cwd;
      _scope = ex.cwd.isEmpty ? _Scope.global : _Scope.project;
    }
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    _commandCtrl.dispose();
    _descriptionCtrl.dispose();
    _cwdCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final name = _nameCtrl.text.trim();
    final command = _commandCtrl.text.trim();
    if (name.isEmpty) {
      setState(() => _error = i18n.t.customTasks.validateNameRequired);
      return;
    }
    if (command.isEmpty) {
      setState(() => _error = i18n.t.customTasks.validateCommandRequired);
      return;
    }
    final cwd = _scope == _Scope.global ? '' : _cwdCtrl.text.trim();
    if (_scope == _Scope.project && cwd.isEmpty) {
      setState(() => _error = i18n.t.customTasks.validateProjectCwd);
      return;
    }
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final api = ref.read(customTasksApiProvider);
      if (_isEdit) {
        await api.update(
          widget.existing!.id,
          name: name,
          command: command,
          description: _descriptionCtrl.text.trim(),
          cwd: cwd,
        );
      } else {
        await api.create(
          name: name,
          command: command,
          description: _descriptionCtrl.text.trim(),
          cwd: cwd,
        );
      }
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _error = e.message;
          _submitting = false;
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _submitting = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(_isEdit
            ? i18n.t.customTasks.appBarEdit
            : i18n.t.customTasks.appBarNew),
      ),
      body: SafeArea(
        bottom: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
          children: [
            _label(i18n.t.customTasks.fieldName),
            const SizedBox(height: 4),
            TextField(
              controller: _nameCtrl,
              enabled: !_submitting,
              decoration: InputDecoration(
                hintText: i18n.t.customTasks.nameHint,
                helperText: i18n.t.customTasks.nameHelper,
              ),
            ),
            const SizedBox(height: 14),
            _label(i18n.t.customTasks.fieldCommand),
            const SizedBox(height: 4),
            TextField(
              controller: _commandCtrl,
              enabled: !_submitting,
              autocorrect: false,
              maxLines: 3,
              minLines: 1,
              decoration: InputDecoration(
                hintText: i18n.t.customTasks.commandHint,
                helperText: i18n.t.customTasks.commandHelper,
              ),
              style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
            ),
            const SizedBox(height: 14),
            _label(i18n.t.customTasks.fieldDescription),
            const SizedBox(height: 4),
            TextField(
              controller: _descriptionCtrl,
              enabled: !_submitting,
              maxLines: 2,
              minLines: 1,
              decoration: InputDecoration(
                hintText: i18n.t.customTasks.descriptionHint,
              ),
            ),
            const SizedBox(height: 14),
            _label(i18n.t.customTasks.fieldScope),
            const SizedBox(height: 4),
            SegmentedButton<_Scope>(
              segments: [
                ButtonSegment(
                  value: _Scope.global,
                  icon: const Icon(Icons.public, size: 18),
                  label: Text(i18n.t.customTasks.scopeGlobal),
                ),
                ButtonSegment(
                  value: _Scope.project,
                  icon: const Icon(Icons.folder_outlined, size: 18),
                  label: Text(i18n.t.customTasks.scopeProject),
                ),
              ],
              selected: {_scope},
              onSelectionChanged: _submitting
                  ? null
                  : (s) => setState(() => _scope = s.first),
            ),
            const SizedBox(height: 8),
            Text(
              _scope == _Scope.global
                  ? i18n.t.customTasks.globalScopeHint
                  : i18n.t.customTasks.projectScopeHint,
              style: theme.textTheme.bodySmall,
            ),
            if (_scope == _Scope.project) ...[
              const SizedBox(height: 12),
              _label(i18n.t.customTasks.fieldProjectCwd),
              const SizedBox(height: 4),
              TextField(
                controller: _cwdCtrl,
                enabled: !_submitting,
                autocorrect: false,
                keyboardType: TextInputType.url,
                decoration: InputDecoration(
                  hintText: i18n.t.customTasks.cwdHint,
                  helperText: i18n.t.customTasks.cwdHelper,
                ),
                style: const TextStyle(fontFamily: 'monospace'),
              ),
            ],
            if (_error != null) ...[
              const SizedBox(height: 14),
              Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: theme.colorScheme.error.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(6),
                  border: Border.all(
                    color: theme.colorScheme.error.withValues(alpha: 0.4),
                  ),
                ),
                child: Text(
                  _error!,
                  style: TextStyle(
                    color: theme.colorScheme.error,
                    fontSize: 12,
                  ),
                ),
              ),
            ],
            const SizedBox(height: 20),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: _submitting
                        ? null
                        : () => Navigator.of(context).pop(false),
                    child: Text(i18n.t.common.cancel),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton.icon(
                    onPressed: _submitting ? null : _submit,
                    icon: _submitting
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.check, size: 18),
                    label: Text(_submitting
                        ? i18n.t.customTasks.saving
                        : _isEdit
                            ? i18n.t.customTasks.save
                            : i18n.t.customTasks.create),
                  ),
                ),
              ],
            ),
            if (_isEdit) ...[
              const SizedBox(height: 14),
              Text(
                'Created: ${DateFormat.yMMMd().add_Hms().format(widget.existing!.createdAt.toLocal())}',
                style: theme.textTheme.bodySmall,
              ),
              Text(
                'Updated: ${DateFormat.yMMMd().add_Hms().format(widget.existing!.updatedAt.toLocal())}',
                style: theme.textTheme.bodySmall,
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _label(String text) => Text(
        text,
        style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600),
      );
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final String error;
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
              size: 40,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 8),
            Text(
              i18n.t.customTasks.failedToLoad,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 4),
            Text(
              error,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            FilledButton(onPressed: onRetry, child: Text(i18n.t.common.retry)),
          ],
        ),
      ),
    );
  }
}
