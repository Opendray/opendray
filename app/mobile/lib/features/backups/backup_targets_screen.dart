import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/backups_api.dart';
import 'package:opendray/core/i18n/strings.g.dart' as i18n;
import 'package:opendray/features/backups/backup_target_editor_screen.dart';

// Backup targets — destinations a backup blob can be written to.
// Mobile exposes list / test / toggle / delete; per-kind create+edit
// (S3 / SMB / SFTP / WebDAV / rclone — each with 5+ fields and long
// secret pastes) is intentionally web-only.
class BackupTargetsScreen extends ConsumerStatefulWidget {
  const BackupTargetsScreen({super.key});

  @override
  ConsumerState<BackupTargetsScreen> createState() =>
      _BackupTargetsScreenState();
}

class _BackupTargetsScreenState
    extends ConsumerState<BackupTargetsScreen> {
  AsyncValue<List<BackupTarget>> _state = const AsyncValue.loading();
  final Set<String> _busy = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final list = await ref.read(backupsApiProvider).listTargets();
      if (!mounted) return;
      list.sort((a, b) {
        if (a.enabled != b.enabled) return a.enabled ? -1 : 1;
        return a.id.compareTo(b.id);
      });
      setState(() => _state = AsyncValue.data(list));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _runOp({
    required String key,
    required String okMsg,
    required String failPrefix,
    required Future<void> Function() op,
  }) async {
    setState(() => _busy.add(key));
    final messenger = ScaffoldMessenger.of(context);
    try {
      await op();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(okMsg),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(
        content: Text(i18n.t.backupTargets
            .errorWithMessage(prefix: failPrefix, error: e.message)),
      ));
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(
        content: Text(i18n.t.backupTargets
            .errorWithMessage(prefix: failPrefix, error: e.toString())),
      ));
    } finally {
      if (mounted) setState(() => _busy.remove(key));
    }
  }

  Future<void> _onTap(BackupTarget t) async {
    final isBusy = _busy.contains(t.id);
    final action = await showModalBottomSheet<_TargetAction>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 14, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(t.id,
                      style: Theme.of(sheetCtx)
                          .textTheme
                          .titleSmall
                          ?.copyWith(fontFamily: 'monospace')),
                  Text(t.kind, style: Theme.of(sheetCtx).textTheme.bodySmall),
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              enabled: !isBusy,
              leading: const Icon(Icons.network_check_outlined),
              title: Text(i18n.t.backupTargets.testConnection),
              onTap: () => Navigator.of(sheetCtx).pop(_TargetAction.test),
            ),
            ListTile(
              enabled: !isBusy,
              leading: Icon(
                t.enabled
                    ? Icons.pause_circle_outline
                    : Icons.play_circle_outline,
              ),
              title: Text(t.enabled ? 'Disable' : 'Enable'),
              onTap: () =>
                  Navigator.of(sheetCtx).pop(_TargetAction.toggleEnabled),
            ),
            ListTile(
              enabled: !isBusy,
              leading: const Icon(Icons.edit_outlined),
              title: Text(i18n.t.backupTargets.editConfig),
              subtitle: Text(
                _isBuiltinLocal(t)
                    ? 'Built-in local target — root path only'
                    : 'Per-kind fields. Secret fields are server-redacted '
                        'on read; leave blank to keep current.',
                style: const TextStyle(fontSize: 11),
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_TargetAction.edit),
            ),
            ListTile(
              leading: const Icon(Icons.code),
              title: Text(i18n.t.backupTargets.viewRawConfig),
              subtitle: const Text(
                'Sensitive fields are server-redacted',
                style: TextStyle(fontSize: 11),
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_TargetAction.viewConfig),
            ),
            const Divider(height: 1),
            ListTile(
              enabled: !isBusy,
              leading: Icon(
                Icons.delete_outline,
                color: Theme.of(sheetCtx).colorScheme.error,
              ),
              title: Text(
                'Delete',
                style: TextStyle(color: Theme.of(sheetCtx).colorScheme.error),
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_TargetAction.delete),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    switch (action) {
      case _TargetAction.test:
        await _runOp(
          key: t.id,
          okMsg: 'Target reachable.',
          failPrefix: 'Test failed',
          op: () => ref.read(backupsApiProvider).testTarget(t.id),
        );
      case _TargetAction.toggleEnabled:
        final next = !t.enabled;
        await _runOp(
          key: t.id,
          okMsg: next ? 'Target enabled.' : 'Target disabled.',
          failPrefix: 'Toggle failed',
          op: () => ref
              .read(backupsApiProvider)
              .setTargetEnabled(t.id, enabled: next)
              .then((_) {}),
        );
      case _TargetAction.viewConfig:
        await _showConfig(t);
      case _TargetAction.edit:
        await _openEditor(existing: t);
      case _TargetAction.delete:
        await _confirmAndDelete(t);
    }
  }

  // Built-in local target — when the operator hasn't configured
  // anything, opendray exposes a `local` target derived from
  // cfg.backup.local_dir. It has no DB row, so editing it would
  // need a create. We let the user edit it (which actually
  // creates a real row that overrides the default).
  bool _isBuiltinLocal(BackupTarget t) =>
      t.id == 'local' && t.kind == 'local';

  Future<void> _openEditor({BackupTarget? existing}) async {
    final saved = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => BackupTargetEditorScreen(existing: existing),
      ),
    );
    if ((saved ?? false) && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(existing == null
              ? 'Target created.'
              : 'Target updated.'),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    }
  }

  Future<void> _showConfig(BackupTarget t) async {
    const enc = JsonEncoder.withIndent('  ');
    final pretty = enc.convert(t.config);
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(i18n.t.backupTargets.configDialogTitle(kind: t.kind)),
        content: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.of(ctx).size.height * 0.6,
            maxWidth: 480,
          ),
          child: SingleChildScrollView(
            child: SelectableText(
              pretty,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
            ),
          ),
        ),
        actions: [
          TextButton.icon(
            icon: const Icon(Icons.copy_outlined, size: 18),
            label: Text(i18n.t.common.copy),
            onPressed: () async {
              await Clipboard.setData(ClipboardData(text: pretty));
              if (!ctx.mounted) return;
              Navigator.of(ctx).pop();
            },
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(i18n.t.common.close),
          ),
        ],
      ),
    );
  }

  Future<void> _confirmAndDelete(BackupTarget t) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(i18n.t.backupTargets.deleteTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(t.id,
                style: Theme.of(ctx)
                    .textTheme
                    .bodySmall
                    ?.copyWith(fontFamily: 'monospace')),
            const SizedBox(height: 8),
            Text(
              'Removes the target spec. Schedules pointing at it will '
              'fail their next run; existing backup blobs at the '
              'destination are not deleted.',
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
    await _runOp(
      key: t.id,
      okMsg: 'Target deleted.',
      failPrefix: 'Delete failed',
      op: () => ref.read(backupsApiProvider).deleteTarget(t.id),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(i18n.t.backupTargets.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: i18n.t.common.refresh,
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
        heroTag: 'backup_targets_fab',
        onPressed: _openEditor,
        icon: const Icon(Icons.add),
        label: Text(i18n.t.backupTargets.newTarget),
      ),
    );
  }

  Widget _buildList(List<BackupTarget> list) {
    if (list.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'No backup targets yet.\n\n'
            'Tap "New target" to add a destination '
            '(local / SMB / S3 / WebDAV / SFTP / rclone).',
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
          return _TargetTile(
            target: t,
            busy: _busy.contains(t.id),
            onTap: () => _onTap(t),
          );
        },
      ),
    );
  }
}

enum _TargetAction { test, toggleEnabled, viewConfig, edit, delete }

class _TargetTile extends StatelessWidget {
  const _TargetTile({
    required this.target,
    required this.busy,
    required this.onTap,
  });

  final BackupTarget target;
  final bool busy;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return ListTile(
      onTap: busy ? null : onTap,
      leading: Container(
        width: 36,
        height: 36,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: Theme.of(context)
              .colorScheme
              .primary
              .withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          target.kind.isNotEmpty ? target.kind[0].toUpperCase() : '?',
          style: TextStyle(
            color: Theme.of(context).colorScheme.primary,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      title: Row(
        children: [
          Flexible(
            child: Text(
              target.id,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                fontFamily: 'monospace',
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
          const SizedBox(width: 6),
          if (!target.enabled)
            _Badge(
              label: 'disabled',
              color: Theme.of(context).colorScheme.error,
            ),
        ],
      ),
      subtitle: DefaultTextStyle.merge(
        style: muted ?? const TextStyle(),
        child: Wrap(
          spacing: 6,
          runSpacing: 2,
          children: [
            Text(target.kind),
            Text(
              '· updated ${DateFormat.yMMMd().format(target.updatedAt.toLocal())}',
            ),
          ],
        ),
      ),
      trailing: busy
          ? const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.more_vert),
    );
  }
}

class _Badge extends StatelessWidget {
  const _Badge({required this.label, required this.color});
  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withValues(alpha: 0.6), width: 0.5),
      ),
      child: Text(
        label,
        style: TextStyle(color: color, fontSize: 10),
      ),
    );
  }
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
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              'Failed to load targets',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(i18n.t.common.retry)),
          ],
        ),
      ),
    );
  }
}
