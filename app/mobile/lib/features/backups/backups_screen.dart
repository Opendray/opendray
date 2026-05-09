import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/backups_api.dart';

// Backups — observability surface. List recent backup rows with
// status/target/size/duration; FAB kicks off a fresh dump against
// the local target. Restore/download/schedule editing live on the
// web admin (multi-GB uploads from a phone are neither practical
// nor safe).
class BackupsScreen extends ConsumerStatefulWidget {
  const BackupsScreen({super.key});

  @override
  ConsumerState<BackupsScreen> createState() => _BackupsScreenState();
}

class _BackupsScreenState extends ConsumerState<BackupsScreen> {
  AsyncValue<List<BackupRow>> _state = const AsyncValue.loading();
  bool _running = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final list = await ref.read(backupsApiProvider).list(limit: 50);
      if (!mounted) return;
      list.sort((a, b) => b.startedAt.compareTo(a.startedAt));
      setState(() => _state = AsyncValue.data(list));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _runNow() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Run backup now?'),
        content: const Text(
          'Triggers a fresh dump against the local target. The job '
          'runs server-side; this list will refresh as it progresses.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Run'),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    setState(() => _running = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final row = await ref.read(backupsApiProvider).runNow();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Backup queued (${row.id}).'),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text('Run failed: ${e.message}')),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text('Run failed: $e')));
    } finally {
      if (mounted) setState(() => _running = false);
    }
  }

  Future<void> _showDetail(BackupRow b) async {
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Backup detail'),
        content: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.of(ctx).size.height * 0.6,
            maxWidth: 480,
          ),
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _kv('ID', b.id, mono: true),
                _kv('Status', b.status),
                _kv('Target', b.targetId),
                _kv('Triggered by', b.triggeredBy),
                _kv(
                  'Started',
                  DateFormat.yMMMd().add_Hms().format(b.startedAt.toLocal()),
                ),
                if (b.finishedAt != null)
                  _kv(
                    'Finished',
                    DateFormat.yMMMd()
                        .add_Hms()
                        .format(b.finishedAt!.toLocal()),
                  ),
                _kv('Size', _formatBytes(b.bytes)),
                _kv('Encrypted', b.encrypted ? 'yes' : 'no'),
                if ((b.targetPath ?? '').isNotEmpty)
                  _kv('Target path', b.targetPath!, mono: true),
                if ((b.error ?? '').isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text(
                    'Error',
                    style: TextStyle(
                      color: Theme.of(ctx).colorScheme.error,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  SelectableText(
                    b.error!,
                    style: TextStyle(
                      color: Theme.of(ctx).colorScheme.error,
                      fontFamily: 'monospace',
                      fontSize: 11,
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
        actions: [
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Close'),
          ),
        ],
      ),
    );
  }

  Widget _kv(String label, String value, {bool mono = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: const TextStyle(fontSize: 11, color: Colors.grey)),
          SelectableText(
            value,
            style: TextStyle(
              fontSize: 13,
              fontFamily: mono ? 'monospace' : null,
            ),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Backups'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
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
        onPressed: _running ? null : _runNow,
        icon: _running
            ? const SizedBox(
                width: 16,
                height: 16,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : const Icon(Icons.cloud_upload_outlined),
        label: Text(_running ? 'Queueing…' : 'Run now'),
      ),
    );
  }

  Widget _buildList(List<BackupRow> list) {
    if (list.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'No backups yet.\n\nTap "Run now" to take a fresh snapshot.',
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
        itemBuilder: (_, i) =>
            _BackupTile(row: list[i], onTap: () => _showDetail(list[i])),
      ),
    );
  }
}

class _BackupTile extends StatelessWidget {
  const _BackupTile({required this.row, required this.onTap});
  final BackupRow row;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final duration = row.finishedAt?.difference(row.startedAt);
    return ListTile(
      onTap: onTap,
      leading: _StatusChip(status: row.status),
      title: Row(
        children: [
          Text(
            row.targetId,
            style: const TextStyle(
              fontFamily: 'monospace',
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(width: 8),
          if (row.encrypted)
            Icon(
              Icons.lock_outline,
              size: 13,
              color: muted?.color,
            ),
          const Spacer(),
          if (duration != null) Text(_formatDuration(duration), style: muted),
        ],
      ),
      subtitle: DefaultTextStyle.merge(
        style: muted ?? const TextStyle(),
        child: Wrap(
          spacing: 6,
          runSpacing: 2,
          children: [
            Text(row.triggeredBy),
            Text('· ${_formatBytes(row.bytes)}'),
            Text('· ${_relTime(row.startedAt)}'),
          ],
        ),
      ),
      trailing: const Icon(Icons.chevron_right),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.status});
  final String status;

  @override
  Widget build(BuildContext context) {
    final color = switch (status) {
      'succeeded' => Colors.greenAccent,
      'running' => Colors.lightBlueAccent,
      'pending' => Colors.amberAccent,
      'failed' => Colors.redAccent,
      'deleted' => Colors.grey,
      _ => Colors.grey,
    };
    return Container(
      width: 84,
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: color.withValues(alpha: 0.6)),
      ),
      alignment: Alignment.center,
      child: Text(
        status,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w600,
        ),
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
              'Failed to load backups',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
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

String _formatBytes(int n) {
  if (n <= 0) return '—';
  if (n < 1024) return '$n B';
  if (n < 1024 * 1024) return '${(n / 1024).toStringAsFixed(1)} KiB';
  if (n < 1024 * 1024 * 1024) {
    return '${(n / (1024 * 1024)).toStringAsFixed(1)} MiB';
  }
  return '${(n / (1024 * 1024 * 1024)).toStringAsFixed(2)} GiB';
}

String _formatDuration(Duration d) {
  if (d.inSeconds < 60) return '${d.inSeconds}s';
  if (d.inMinutes < 60) {
    final s = d.inSeconds % 60;
    return '${d.inMinutes}m ${s}s';
  }
  final m = d.inMinutes % 60;
  return '${d.inHours}h ${m}m';
}

String _relTime(DateTime ts) {
  final diff = DateTime.now().toUtc().difference(ts.toUtc());
  if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
  if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
  if (diff.inHours < 24) return '${diff.inHours}h ago';
  if (diff.inDays < 7) return '${diff.inDays}d ago';
  return DateFormat.yMMMd().format(ts.toLocal());
}
