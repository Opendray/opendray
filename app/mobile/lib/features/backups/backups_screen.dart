import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/backups_api.dart';
import 'package:opendray/features/backups/backup_schedules_screen.dart';
import 'package:opendray/features/backups/backup_targets_screen.dart';

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

// Combined page data loaded in parallel. Keeping a single bundle
// instead of four separate AsyncValues means the banner / summary /
// list paint together — no progressive-load flicker on a phone
// where the whole screen fits above the fold.
class _PageData {
  _PageData({
    required this.status,
    required this.rows,
    required this.targets,
    required this.schedules,
  });

  final BackupStatusReport status;
  final List<BackupRow> rows;
  final List<BackupTarget> targets;
  final List<BackupSchedule> schedules;
}

class _BackupsScreenState extends ConsumerState<BackupsScreen> {
  AsyncValue<_PageData> _state = const AsyncValue.loading();
  bool _running = false;
  // Active poll for a single-row run-now → succeeded/failed
  // transition. Cancelled when the screen disposes or the row
  // settles, so we never leak a pending Timer past pop.
  Timer? _runPoll;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _runPoll?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final api = ref.read(backupsApiProvider);
      // Fan out — four cheap GETs, ~200ms total on a LAN. If the
      // status endpoint dies the whole screen errors; if a sub-list
      // fails we still want a usable header, so non-status calls
      // degrade to empty.
      final results = await Future.wait<Object>([
        api.status(),
        api.list(limit: 50).catchError((_) => <BackupRow>[]),
        api.listTargets().catchError((_) => <BackupTarget>[]),
        api.listSchedules().catchError((_) => <BackupSchedule>[]),
      ]);
      if (!mounted) return;
      final rows = (results[1] as List<BackupRow>)
        ..sort((a, b) => b.startedAt.compareTo(a.startedAt));
      setState(() => _state = AsyncValue.data(_PageData(
            status: results[0] as BackupStatusReport,
            rows: rows,
            targets: results[2] as List<BackupTarget>,
            schedules: results[3] as List<BackupSchedule>,
          )));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  // Lightweight in-place refresh: re-fetches rows + status without
  // flashing the spinner. Used by the run-now poll and pull-to-
  // refresh paths. Targets/schedules don't change during a single
  // dump, so they're skipped.
  Future<void> _softRefresh() async {
    final current = _state.valueOrNull;
    if (current == null) {
      await _load();
      return;
    }
    try {
      final api = ref.read(backupsApiProvider);
      final results = await Future.wait<Object>([
        api.status(),
        api.list(limit: 50),
      ]);
      if (!mounted) return;
      final rows = (results[1] as List<BackupRow>)
        ..sort((a, b) => b.startedAt.compareTo(a.startedAt));
      setState(() => _state = AsyncValue.data(_PageData(
            status: results[0] as BackupStatusReport,
            rows: rows,
            targets: current.targets,
            schedules: current.schedules,
          )));
    } on Object {
      // Swallow soft-refresh errors — the page is already
      // rendered, no point flashing a full-screen error.
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
          content: Text('Backup queued (${row.id}). Watching for progress…'),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
      if (mounted) _startRunPoll(row.id, messenger);
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

  // After a successful runNow() the row is `pending`. The server
  // transitions it through `running` → `succeeded`/`failed` async.
  // Poll every 3s for up to 60s (a typical pg_dump on a dev DB
  // finishes in well under that; long-running ones get the snackbar
  // hint and the operator can pull-to-refresh themselves).
  void _startRunPoll(String rowId, ScaffoldMessengerState messenger) {
    _runPoll?.cancel();
    var ticks = 0;
    const maxTicks = 20; // 20 * 3s = 60s budget.
    _runPoll = Timer.periodic(const Duration(seconds: 3), (t) async {
      ticks++;
      if (!mounted) {
        t.cancel();
        return;
      }
      await _softRefresh();
      if (!mounted) {
        t.cancel();
        return;
      }
      final row = _state.valueOrNull?.rows.firstWhere(
        (r) => r.id == rowId,
        orElse: () => BackupRow(
          id: '',
          targetId: '',
          status: '',
          triggeredBy: '',
          startedAt: DateTime.now().toUtc(),
          bytes: 0,
          encrypted: false,
        ),
      );
      final settled =
          row != null && (row.status == 'succeeded' || row.status == 'failed');
      if (settled) {
        t.cancel();
        _runPoll = null;
        final ok = row.status == 'succeeded';
        messenger.showSnackBar(
          SnackBar(
            content: Text(ok
                ? 'Backup succeeded (${_formatBytes(row.bytes)}).'
                : 'Backup failed: ${row.error ?? "unknown error"}'),
            duration: const Duration(seconds: 3),
            behavior: SnackBarBehavior.floating,
            backgroundColor: ok ? null : Theme.of(context).colorScheme.error,
          ),
        );
      } else if (ticks >= maxTicks) {
        t.cancel();
        _runPoll = null;
        // Don't fire a "still running" snackbar — too noisy.
        // The row is on screen with the running chip; operator can
        // pull-to-refresh.
      }
    });
  }

  Future<void> _showDetail(BackupRow b) async {
    final action = await showDialog<_DetailAction>(
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
          if (b.status != 'deleted' && b.status != 'pending')
            TextButton(
              style: TextButton.styleFrom(
                foregroundColor: Theme.of(ctx).colorScheme.error,
              ),
              onPressed: () => Navigator.of(ctx).pop(_DetailAction.delete),
              child: const Text('Delete'),
            ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(_DetailAction.close),
            child: const Text('Close'),
          ),
        ],
      ),
    );
    if (action == _DetailAction.delete && mounted) {
      await _confirmAndDelete(b);
    }
  }

  Future<void> _confirmAndDelete(BackupRow b) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete backup?'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Removes the blob from ${b.targetId} and marks the row '
              'deleted. The audit entry is retained but the data '
              'cannot be recovered.',
              style: Theme.of(ctx).textTheme.bodySmall,
            ),
            const SizedBox(height: 8),
            Text(
              b.id,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(backupsApiProvider).delete(b.id);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Deleted ${b.id}.'),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text('Delete failed: ${e.message}')),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text('Delete failed: $e')));
    }
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
          PopupMenuButton<_AppBarAction>(
            tooltip: 'More',
            onSelected: (a) {
              switch (a) {
                case _AppBarAction.schedules:
                  Navigator.of(context).push(
                    MaterialPageRoute<void>(
                      builder: (_) => const BackupSchedulesScreen(),
                    ),
                  );
                case _AppBarAction.targets:
                  Navigator.of(context).push(
                    MaterialPageRoute<void>(
                      builder: (_) => const BackupTargetsScreen(),
                    ),
                  );
              }
            },
            itemBuilder: (_) => const [
              PopupMenuItem(
                value: _AppBarAction.schedules,
                child: ListTile(
                  leading: Icon(Icons.schedule_outlined),
                  title: Text('Schedules'),
                ),
              ),
              PopupMenuItem(
                value: _AppBarAction.targets,
                child: ListTile(
                  leading: Icon(Icons.cloud_outlined),
                  title: Text('Targets'),
                ),
              ),
            ],
          ),
        ],
      ),
      body: _state.when(
        data: _buildBody,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
      floatingActionButton: FloatingActionButton.extended(
        // Disable Run now when pg_dump is broken or no targets are
        // configured — clicking it would just produce a failed row.
        onPressed: _canRunNow() ? _runNow : null,
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

  bool _canRunNow() {
    if (_running) return false;
    final data = _state.valueOrNull;
    if (data == null) return false;
    if (!data.status.ok) return false;
    final hasEnabledTarget = data.targets.any((t) => t.enabled);
    return hasEnabledTarget;
  }

  Widget _buildBody(_PageData data) {
    final list = data.rows;
    // Always render a scrollable surface so pull-to-refresh works
    // even when the list is empty.
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          _StatusBanner(status: data.status),
          _SummaryCard(
            status: data.status,
            targets: data.targets,
            schedules: data.schedules,
            rows: data.rows,
          ),
          if (list.isEmpty)
            _emptyState(data)
          else
            ...list.map(
              (r) => Column(
                children: [
                  _BackupTile(row: r, onTap: () => _showDetail(r)),
                  Divider(height: 1, color: Theme.of(context).dividerColor),
                ],
              ),
            ),
          // Footer padding so the FAB doesn't cover the last row.
          const SizedBox(height: 96),
        ],
      ),
    );
  }

  Widget _emptyState(_PageData data) {
    final hasTargets = data.targets.any((t) => t.enabled);
    final pgOk = data.status.ok;
    final theme = Theme.of(context);
    String headline;
    String body;
    IconData icon;
    if (!pgOk) {
      icon = Icons.error_outline;
      headline = "Backups can't run yet";
      body = data.status.pgDumpError ??
          'pg_dump is not available on the server. '
              'Install postgresql-client and restart opendray.';
    } else if (!hasTargets) {
      icon = Icons.cloud_off_outlined;
      headline = 'No backup targets configured';
      body =
          'Open the More menu → Targets to add a destination (local / S3 / SMB / SFTP / WebDAV / rclone). '
          'Then come back and tap "Run now".';
    } else {
      icon = Icons.archive_outlined;
      headline = 'No backups yet';
      body = 'Tap "Run now" to take a fresh snapshot, or open '
          'Schedules to set up recurring runs.';
    }
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 40),
      child: Column(
        children: [
          Icon(icon, size: 48, color: theme.colorScheme.outline),
          const SizedBox(height: 12),
          Text(headline,
              style: theme.textTheme.titleMedium,
              textAlign: TextAlign.center),
          const SizedBox(height: 6),
          Text(body,
              style: theme.textTheme.bodySmall,
              textAlign: TextAlign.center),
        ],
      ),
    );
  }
}

enum _DetailAction { close, delete }

enum _AppBarAction { schedules, targets }

// Feature-health banner. Renders red when pg_dump is unavailable
// (with the underlying error so the operator can fix it from the
// server), green otherwise with the pg_dump version + cipher key
// fingerprint so they can confirm "backups can run AND will be
// encrypted with the key I think they're encrypted with."
class _StatusBanner extends StatelessWidget {
  const _StatusBanner({required this.status});
  final BackupStatusReport status;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final ok = status.ok;
    final color = ok ? Colors.green : theme.colorScheme.error;
    final bg = color.withValues(alpha: 0.10);
    final border = color.withValues(alpha: 0.45);
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 0),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(ok ? Icons.check_circle_outline : Icons.error_outline,
                  size: 18, color: color),
              const SizedBox(width: 8),
              Text(
                ok ? 'Backups ready' : 'Backups cannot run',
                style: TextStyle(
                  color: color,
                  fontWeight: FontWeight.w600,
                  fontSize: 13,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          if (ok) ...[
            _kvRow(context, 'pg_dump', status.pgDumpVersion),
            const SizedBox(height: 4),
            _kvRow(
              context,
              'key fingerprint',
              status.keyFingerprint.isEmpty ? '—' : status.keyFingerprint,
              mono: true,
            ),
          ] else
            Text(
              status.pgDumpError ??
                  'pg_dump is not on PATH. Install postgresql-client '
                      'on the server and restart opendray.',
              style: theme.textTheme.bodySmall?.copyWith(color: color),
            ),
        ],
      ),
    );
  }

  Widget _kvRow(BuildContext context, String label, String value,
      {bool mono = false}) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 110,
          child: Text(label, style: muted),
        ),
        Expanded(
          child: SelectableText(
            value,
            style: TextStyle(
              fontSize: 12,
              fontFamily: mono ? 'monospace' : null,
            ),
          ),
        ),
      ],
    );
  }
}

// "Where do I stand" overview: targets, schedules, total runs,
// disk usage. Each tile tappable into the corresponding sub-page
// where it makes sense.
class _SummaryCard extends StatelessWidget {
  const _SummaryCard({
    required this.status,
    required this.targets,
    required this.schedules,
    required this.rows,
  });
  final BackupStatusReport status;
  final List<BackupTarget> targets;
  final List<BackupSchedule> schedules;
  final List<BackupRow> rows;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final targetsEnabled = targets.where((t) => t.enabled).length;
    final schedulesEnabled = schedules.where((s) => s.enabled).length;
    final liveRows = rows.where((r) => r.status != 'deleted').toList();
    final totalBytes =
        liveRows.fold<int>(0, (acc, r) => acc + (r.bytes > 0 ? r.bytes : 0));
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Row(
        children: [
          _SummaryTile(
            icon: Icons.cloud_outlined,
            label: 'Targets',
            value: '$targetsEnabled / ${targets.length}',
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => const BackupTargetsScreen(),
              ),
            ),
          ),
          _Divider(),
          _SummaryTile(
            icon: Icons.schedule_outlined,
            label: 'Schedules',
            value: '$schedulesEnabled / ${schedules.length}',
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => const BackupSchedulesScreen(),
              ),
            ),
          ),
          _Divider(),
          _SummaryTile(
            icon: Icons.archive_outlined,
            label: 'Backups',
            value: '${liveRows.length}',
            sub: _formatBytes(totalBytes),
          ),
        ],
      ),
    );
  }
}

class _Divider extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      width: 1,
      height: 44,
      color: Theme.of(context).dividerColor,
    );
  }
}

class _SummaryTile extends StatelessWidget {
  const _SummaryTile({
    required this.icon,
    required this.label,
    required this.value,
    this.sub,
    this.onTap,
  });
  final IconData icon;
  final String label;
  final String value;
  final String? sub;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Expanded(
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 8),
          child: Column(
            children: [
              Icon(icon, size: 18, color: theme.colorScheme.outline),
              const SizedBox(height: 4),
              Text(label,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.outline)),
              const SizedBox(height: 2),
              Text(value,
                  style: const TextStyle(
                      fontSize: 14, fontWeight: FontWeight.w600)),
              if (sub != null)
                Text(sub!,
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.outline)),
            ],
          ),
        ),
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
