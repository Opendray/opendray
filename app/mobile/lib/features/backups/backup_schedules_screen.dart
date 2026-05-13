import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/backups_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Backup schedules — recurring spec (target + interval + retention +
// enabled). Mobile-friendly because the shape is plain seconds, not
// cron, so a few preset chips cover the realistic operator
// configurations without typing arcane syntax on a phone keyboard.
class BackupSchedulesScreen extends ConsumerStatefulWidget {
  const BackupSchedulesScreen({super.key});

  @override
  ConsumerState<BackupSchedulesScreen> createState() =>
      _BackupSchedulesScreenState();
}

class _BackupSchedulesScreenState
    extends ConsumerState<BackupSchedulesScreen> {
  AsyncValue<_Data> _state = const AsyncValue.loading();
  final Set<String> _busy = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final api = ref.read(backupsApiProvider);
      final results = await Future.wait([api.listSchedules(), api.listTargets()]);
      if (!mounted) return;
      final schedules = results[0] as List<BackupSchedule>;
      final targets = results[1] as List<BackupTarget>;
      schedules.sort((a, b) => a.nextRunAt.compareTo(b.nextRunAt));
      setState(() => _state = AsyncValue.data(
            _Data(schedules: schedules, targets: targets),
          ));
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
        content: Text(t.backupSchedules
            .errorWithMessage(prefix: failPrefix, error: e.message)),
      ));
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(
        content: Text(t.backupSchedules
            .errorWithMessage(prefix: failPrefix, error: e.toString())),
      ));
    } finally {
      if (mounted) setState(() => _busy.remove(key));
    }
  }

  Future<void> _onCreate(List<BackupTarget> targets) async {
    if (targets.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text(
            'No backup targets configured. Add one from the web '
            'admin first.',
          ),
        ),
      );
      return;
    }
    final form = await Navigator.of(context).push<_ScheduleFormResult>(
      MaterialPageRoute<_ScheduleFormResult>(
        builder: (_) => _ScheduleFormScreen(targets: targets),
        fullscreenDialog: true,
      ),
    );
    if (form == null || !mounted) return;
    await _runOp(
      key: 'new',
      okMsg: 'Schedule created.',
      failPrefix: 'Create failed',
      op: () => ref
          .read(backupsApiProvider)
          .createSchedule(
            targetId: form.targetId,
            intervalSec: form.intervalSec,
            retention: form.retention,
            enabled: form.enabled,
          )
          .then((_) {}),
    );
  }

  Future<void> _onEdit(BackupSchedule sc, List<BackupTarget> targets) async {
    final form = await Navigator.of(context).push<_ScheduleFormResult>(
      MaterialPageRoute<_ScheduleFormResult>(
        builder: (_) => _ScheduleFormScreen(
          targets: targets,
          initial: sc,
        ),
        fullscreenDialog: true,
      ),
    );
    if (form == null || !mounted) return;
    await _runOp(
      key: 's:${sc.id}',
      okMsg: 'Schedule updated.',
      failPrefix: 'Update failed',
      op: () => ref
          .read(backupsApiProvider)
          .updateSchedule(
            sc.id,
            intervalSec:
                form.intervalSec != sc.intervalSec ? form.intervalSec : null,
            retention: form.retention != sc.retention ? form.retention : null,
            enabled: form.enabled != sc.enabled ? form.enabled : null,
          )
          .then((_) {}),
    );
  }

  Future<void> _onDelete(BackupSchedule sc) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.backupSchedules.deleteTitle),
        content: Text(
          'Removes the recurring spec for target ${sc.targetId}. '
          'Existing backup blobs are not touched.',
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
            child: Text(t.common.delete),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    await _runOp(
      key: 's:${sc.id}',
      okMsg: 'Schedule deleted.',
      failPrefix: 'Delete failed',
      op: () => ref.read(backupsApiProvider).deleteSchedule(sc.id),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.backupSchedules.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.common.refresh,
            onPressed: _state is AsyncLoading ? null : _load,
          ),
        ],
      ),
      body: _state.when(
        data: _buildList,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
      floatingActionButton: _state.maybeWhen(
        data: (d) => FloatingActionButton.extended(
          heroTag: 'backup_schedules_fab',
          onPressed: () => _onCreate(d.targets),
          icon: const Icon(Icons.add),
          label: const Text('New'),
        ),
        orElse: () => null,
      ),
    );
  }

  Widget _buildList(_Data d) {
    if (d.schedules.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'No schedules yet.\nTap "New" to create one.',
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        itemCount: d.schedules.length,
        separatorBuilder: (_, __) => Divider(
          height: 1,
          color: Theme.of(context).dividerColor,
        ),
        itemBuilder: (_, i) {
          final sc = d.schedules[i];
          return _ScheduleTile(
            schedule: sc,
            busy: _busy.contains('s:${sc.id}'),
            onEdit: () => _onEdit(sc, d.targets),
            onDelete: () => _onDelete(sc),
          );
        },
      ),
    );
  }
}

class _Data {
  _Data({required this.schedules, required this.targets});
  final List<BackupSchedule> schedules;
  final List<BackupTarget> targets;
}

class _ScheduleTile extends StatelessWidget {
  const _ScheduleTile({
    required this.schedule,
    required this.busy,
    required this.onEdit,
    required this.onDelete,
  });

  final BackupSchedule schedule;
  final bool busy;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return ListTile(
      onTap: busy ? null : onEdit,
      leading: Icon(
        schedule.enabled
            ? Icons.schedule_outlined
            : Icons.schedule_outlined,
        color: schedule.enabled
            ? Theme.of(context).colorScheme.primary
            : Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4),
      ),
      title: Row(
        children: [
          Text(
            schedule.targetId,
            style: const TextStyle(
              fontFamily: 'monospace',
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(width: 8),
          if (!schedule.enabled)
            _Badge(
              label: 'paused',
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
            Text('every ${_formatInterval(schedule.intervalSec)}'),
            Text('· keep ${schedule.retention}'),
            Text(
              '· next ${_relTime(schedule.nextRunAt, future: true)}',
            ),
            if (schedule.lastRunAt != null)
              Text('· last ${_relTime(schedule.lastRunAt!)}'),
          ],
        ),
      ),
      trailing: busy
          ? const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : IconButton(
              icon: Icon(
                Icons.delete_outline,
                color: Theme.of(context).colorScheme.error,
              ),
              tooltip: t.common.delete,
              onPressed: onDelete,
            ),
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

class _ScheduleFormResult {
  _ScheduleFormResult({
    required this.targetId,
    required this.intervalSec,
    required this.retention,
    required this.enabled,
  });
  final String targetId;
  final int intervalSec;
  final int retention;
  final bool enabled;
}

class _ScheduleFormScreen extends StatefulWidget {
  const _ScheduleFormScreen({required this.targets, this.initial});
  final List<BackupTarget> targets;
  final BackupSchedule? initial;

  @override
  State<_ScheduleFormScreen> createState() => _ScheduleFormScreenState();
}

class _ScheduleFormScreenState extends State<_ScheduleFormScreen> {
  static const _intervals = [
    (3600, '1 hour'),
    (6 * 3600, '6 hours'),
    (12 * 3600, '12 hours'),
    (86400, '1 day'),
    (3 * 86400, '3 days'),
    (7 * 86400, '1 week'),
  ];
  static const _retentions = [3, 7, 14, 30];

  late String _targetId;
  late int _intervalSec;
  late int _retention;
  late bool _enabled;
  String? _error;

  @override
  void initState() {
    super.initState();
    final init = widget.initial;
    _targetId = init?.targetId ?? widget.targets.first.id;
    _intervalSec = init?.intervalSec ?? 86400;
    _retention = init?.retention ?? 7;
    _enabled = init?.enabled ?? true;
  }

  void _submit() {
    if (_targetId.isEmpty) {
      setState(() => _error = 'Pick a target.');
      return;
    }
    if (_intervalSec <= 0) {
      setState(() => _error = 'Interval must be > 0.');
      return;
    }
    Navigator.of(context).pop(_ScheduleFormResult(
      targetId: _targetId,
      intervalSec: _intervalSec,
      retention: _retention,
      enabled: _enabled,
    ));
  }

  @override
  Widget build(BuildContext context) {
    final isEdit = widget.initial != null;
    final muted = Theme.of(context).textTheme.bodySmall;
    return Scaffold(
      appBar: AppBar(
        title: Text(isEdit ? 'Edit schedule' : 'New schedule'),
        actions: [
          TextButton(
            onPressed: _submit,
            child: Text(isEdit ? 'Save' : 'Create'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          Text(t.backupSchedules.targetLabel, style: muted),
          const SizedBox(height: 6),
          DropdownButtonFormField<String>(
            initialValue: _targetId,
            items: [
              for (final t in widget.targets)
                DropdownMenuItem(
                  value: t.id,
                  child: Text('${t.id} (${t.kind})'),
                ),
            ],
            onChanged: isEdit
                ? null // Server doesn't allow changing target on existing schedule
                : (v) => setState(() => _targetId = v ?? _targetId),
            decoration: const InputDecoration(
              border: OutlineInputBorder(),
            ),
          ),
          if (isEdit)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Text(
                'Target is fixed once created.',
                style: muted,
              ),
            ),
          const SizedBox(height: 24),
          Text(t.backupSchedules.intervalLabel, style: muted),
          const SizedBox(height: 6),
          Wrap(
            spacing: 6,
            runSpacing: 4,
            children: [
              for (final (sec, label) in _intervals)
                ChoiceChip(
                  label: Text(label),
                  selected: _intervalSec == sec,
                  onSelected: (_) => setState(() => _intervalSec = sec),
                ),
            ],
          ),
          const SizedBox(height: 24),
          Text(t.backupSchedules.retentionLabel, style: muted),
          const SizedBox(height: 6),
          Wrap(
            spacing: 6,
            runSpacing: 4,
            children: [
              for (final n in _retentions)
                ChoiceChip(
                  label: Text('$n'),
                  selected: _retention == n,
                  onSelected: (_) => setState(() => _retention = n),
                ),
            ],
          ),
          const SizedBox(height: 24),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: Text(t.common.enabled),
            subtitle: Text(
              _enabled
                  ? 'Scheduler will run this on cadence.'
                  : 'Paused — no automatic runs until re-enabled.',
              style: muted,
            ),
            value: _enabled,
            onChanged: (v) => setState(() => _enabled = v),
          ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Text(
                _error!,
                style: TextStyle(
                  color: Theme.of(context).colorScheme.error,
                  fontSize: 12,
                ),
              ),
            ),
        ],
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
              'Failed to load schedules',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
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

String _formatInterval(int sec) {
  if (sec < 60) return '${sec}s';
  if (sec < 3600) return '${(sec / 60).toStringAsFixed(0)}m';
  if (sec < 86400) return '${(sec / 3600).toStringAsFixed(0)}h';
  return '${(sec / 86400).toStringAsFixed(0)}d';
}

String _relTime(DateTime ts, {bool future = false}) {
  final diff = future
      ? ts.toUtc().difference(DateTime.now().toUtc())
      : DateTime.now().toUtc().difference(ts.toUtc());
  if (diff.inSeconds.abs() < 60) {
    return future ? 'in ${diff.inSeconds.abs()}s' : '${diff.inSeconds}s ago';
  }
  if (diff.inMinutes.abs() < 60) {
    return future
        ? 'in ${diff.inMinutes.abs()}m'
        : '${diff.inMinutes}m ago';
  }
  if (diff.inHours.abs() < 24) {
    return future ? 'in ${diff.inHours.abs()}h' : '${diff.inHours}h ago';
  }
  if (diff.inDays.abs() < 7) {
    return future ? 'in ${diff.inDays.abs()}d' : '${diff.inDays}d ago';
  }
  return DateFormat.yMMMd().format(ts.toLocal());
}
