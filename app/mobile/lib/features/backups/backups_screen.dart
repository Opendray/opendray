import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/backups_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
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
//
// `status.enabled` decides which sub-tree to render: false → either
// SetupWizard (configured=false) or RestartPrompt (configured=true,
// requires_restart=true), true → normal Backups dashboard.
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

  // True when the backup feature isn't running this process — i.e.
  // the operator hasn't set it up yet, or set it up but hasn't
  // restarted.
  bool get featureOff => !status.enabled;
  // True when setup has happened (key file or env var present) but
  // the feature isn't yet running — the operator needs to bounce
  // the gateway.
  bool get awaitingRestart => status.requiresRestart;
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
      // Status first — its `enabled` field decides whether we even
      // bother fetching the rest. When the feature isn't running
      // the data endpoints aren't mounted either, and fanning out
      // three more 404s just to throw them away is wasteful (and
      // noisy in the server logs).
      final status = await api.status();
      if (!mounted) return;
      if (!status.enabled) {
        setState(() => _state = AsyncValue.data(_PageData(
              status: status,
              rows: const [],
              targets: const [],
              schedules: const [],
            )));
        return;
      }
      final results = await Future.wait<Object>([
        api.list(limit: 50).catchError((_) => <BackupRow>[]),
        api.listTargets().catchError((_) => <BackupTarget>[]),
        api.listSchedules().catchError((_) => <BackupSchedule>[]),
      ]);
      if (!mounted) return;
      final rows = (results[0] as List<BackupRow>)
        ..sort((a, b) => b.startedAt.compareTo(a.startedAt));
      setState(() => _state = AsyncValue.data(_PageData(
            status: status,
            rows: rows,
            targets: results[1] as List<BackupTarget>,
            schedules: results[2] as List<BackupSchedule>,
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
      final status = await api.status();
      if (!mounted) return;
      if (!status.enabled) {
        // Feature was just disabled on the server between loads
        // (or this is the post-setup pre-restart window). Drop
        // back to the setup/restart view rather than keep stale
        // rows.
        setState(() => _state = AsyncValue.data(_PageData(
              status: status,
              rows: const [],
              targets: const [],
              schedules: const [],
            )));
        return;
      }
      final list = await api.list(limit: 50);
      if (!mounted) return;
      list.sort((a, b) => b.startedAt.compareTo(a.startedAt));
      setState(() => _state = AsyncValue.data(_PageData(
            status: status,
            rows: list,
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
        title: Text(t.backups.runConfirmTitle),
        content: const Text(
          'Triggers a fresh dump against the local target. The job '
          'runs server-side; this list will refresh as it progresses.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.backups.run),
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
          content: Text(t.backups.queuedSnack(id: row.id)),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
      if (mounted) _startRunPoll(row.id, messenger);
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(t.backups.runFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(
        content: Text(t.backups.runFailedGeneric(error: e.toString())),
      ));
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
        title: Text(t.backups.detailTitle),
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
              child: Text(t.common.delete),
            ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(_DetailAction.close),
            child: Text(t.common.close),
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
        title: Text(t.backups.deleteTitle),
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
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(backupsApiProvider).delete(b.id);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.backups.deletedSnack(id: b.id)),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(t.backups.deleteFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(
        content: Text(t.backups.deleteFailedGeneric(error: e.toString())),
      ));
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
        title: Text(t.backups.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.common.refresh,
            onPressed: _state is AsyncLoading ? null : _load,
          ),
          PopupMenuButton<_AppBarAction>(
            tooltip: t.more.title,
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
            itemBuilder: (_) => [
              PopupMenuItem(
                value: _AppBarAction.schedules,
                child: ListTile(
                  leading: const Icon(Icons.schedule_outlined),
                  title: Text(t.backups.menuSchedules),
                ),
              ),
              PopupMenuItem(
                value: _AppBarAction.targets,
                child: ListTile(
                  leading: const Icon(Icons.cloud_outlined),
                  title: Text(t.backups.menuTargets),
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
      // Hide the FAB entirely when the feature is off — it can't do
      // anything useful and the setup/restart view already gives the
      // operator the next step.
      floatingActionButton: _state.valueOrNull?.featureOff ?? true
          ? null
          : FloatingActionButton.extended(
              heroTag: 'backups_fab',
              // Greyed-out when pg_dump is broken or no targets are
              // configured — clicking would just produce a failed row.
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
    if (!data.status.enabled || !data.status.ok) return false;
    return data.targets.any((t) => t.enabled);
  }

  Widget _buildBody(_PageData data) {
    if (data.featureOff) {
      if (data.awaitingRestart) {
        return _RestartRequiredView(
          status: data.status,
          onRecheck: _load,
        );
      }
      return _SetupWizardView(
        status: data.status,
        onComplete: _load,
      );
    }
    final status = data.status;
    final list = data.rows;
    // Always render a scrollable surface so pull-to-refresh works
    // even when the list is empty.
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          _StatusBanner(status: status),
          _SummaryCard(
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

// Rendered when the operator has already set up a passphrase (env
// var present OR key file on disk) but the feature isn't running
// in *this* process — i.e. they POSTed /backup-setup, we wrote the
// file, and now they need to bounce the gateway to pick it up. The
// "Check again" button is the same _load callback as the rest of
// the screen; after a real restart, the next refresh transitions
// the page to the live dashboard.
class _RestartRequiredView extends StatelessWidget {
  const _RestartRequiredView({required this.status, required this.onRecheck});
  final BackupStatusReport status;
  final Future<void> Function() onRecheck;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final muted = theme.colorScheme.outline;
    return RefreshIndicator(
      onRefresh: onRecheck,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(24),
        children: [
          const SizedBox(height: 32),
          Icon(Icons.restart_alt, size: 56, color: theme.colorScheme.primary),
          const SizedBox(height: 16),
          Text(
            'Restart opendray to activate backups',
            style: theme.textTheme.titleMedium,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          Text(
            'Your passphrase is saved. The gateway only loads it at '
            'startup, so backups stay off until you bounce the process.',
            style: theme.textTheme.bodySmall?.copyWith(color: muted),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 20),
          if (status.configuredVia == 'file' && status.keyFilePath.isNotEmpty)
            _kvBox(context, label: 'Key file', value: status.keyFilePath),
          if (status.configuredVia == 'env')
            _kvBox(
              context,
              label: 'Configured via',
              value: 'OPENDRAY_BACKUP_KEY env var',
            ),
          const SizedBox(height: 24),
          Center(
            child: FilledButton.icon(
              onPressed: onRecheck,
              icon: const Icon(Icons.refresh, size: 18),
              label: Text(t.backups.encryption.checkAgain),
            ),
          ),
        ],
      ),
    );
  }

  Widget _kvBox(BuildContext context,
      {required String label, required String value}) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label,
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.outline)),
          const SizedBox(height: 4),
          SelectableText(
            value,
            style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
          ),
        ],
      ),
    );
  }
}

// First-time setup wizard. Two modes:
//   Generate — server picks a base64 32-byte key and returns it
//   once; the operator must save it before continuing.
//   Paste — operator types/pastes their own (min 20 chars).
//
// Both write to ~/.opendray/secrets/backup.key (0600) and require
// a gateway restart to activate. The _RestartRequiredView sibling
// takes over once the file is on disk; the operator may have to
// physically restart opendray before the page transitions to the
// live dashboard.
class _SetupWizardView extends ConsumerStatefulWidget {
  const _SetupWizardView({required this.status, required this.onComplete});
  final BackupStatusReport status;
  final Future<void> Function() onComplete;

  @override
  ConsumerState<_SetupWizardView> createState() => _SetupWizardViewState();
}

enum _SetupMode { generate, paste }

class _SetupWizardViewState extends ConsumerState<_SetupWizardView> {
  _SetupMode _mode = _SetupMode.generate;
  final _pasteCtrl = TextEditingController();
  bool _submitting = false;
  // Result of a successful generate call — must be displayed once
  // and acknowledged by the operator before we transition to the
  // restart screen. Null otherwise.
  BackupSetupResult? _generated;
  bool _ackSaved = false;
  String? _error;

  @override
  void dispose() {
    _pasteCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final api = ref.read(backupsApiProvider);
      final result = _mode == _SetupMode.generate
          ? await api.setup(mode: 'generate')
          : await api.setup(
              mode: 'paste',
              passphrase: _pasteCtrl.text.trim(),
            );
      if (!mounted) return;
      if (result.passphrase != null) {
        // Generate path — keep on this screen, show the passphrase
        // for save confirmation. Continue button finalises the
        // flow by triggering a parent reload.
        setState(() {
          _generated = result;
          _submitting = false;
        });
      } else {
        // Paste path — caller already knows their passphrase,
        // no save-confirm step needed.
        await widget.onComplete();
      }
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
    final muted = theme.colorScheme.outline;

    if (_generated != null) {
      return _generatedView(context, _generated!);
    }

    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(20),
      children: [
        const SizedBox(height: 16),
        Icon(Icons.lock_outlined, size: 48, color: theme.colorScheme.primary),
        const SizedBox(height: 12),
        Text(
          'Set up backups',
          style: theme.textTheme.titleMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 6),
        Text(
          'Choose a master passphrase. opendray uses it to encrypt every '
          'backup blob. Lose it and your backups become unrecoverable, '
          'so save it in a password manager (Vaultwarden, 1Password) '
          'before continuing.',
          style: theme.textTheme.bodySmall?.copyWith(color: muted),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 20),
        SegmentedButton<_SetupMode>(
          segments: [
            ButtonSegment(
              value: _SetupMode.generate,
              icon: const Icon(Icons.casino_outlined, size: 18),
              label: Text(t.backups.encryption.generate),
            ),
            ButtonSegment(
              value: _SetupMode.paste,
              icon: const Icon(Icons.edit_outlined, size: 18),
              label: Text(t.backups.encryption.paste),
            ),
          ],
          selected: {_mode},
          onSelectionChanged: (s) => setState(() => _mode = s.first),
        ),
        const SizedBox(height: 20),
        if (_mode == _SetupMode.generate)
          _generateExplainer(context)
        else
          _pasteForm(context),
        if (_error != null) ...[
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: theme.colorScheme.error.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(6),
              border: Border.all(
                  color: theme.colorScheme.error.withValues(alpha: 0.4)),
            ),
            child: Text(
              _error!,
              style: TextStyle(color: theme.colorScheme.error, fontSize: 12),
            ),
          ),
        ],
        const SizedBox(height: 20),
        FilledButton.icon(
          onPressed: _submitting ? null : _submit,
          icon: _submitting
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.check, size: 18),
          label: Text(_submitting
              ? 'Saving…'
              : _mode == _SetupMode.generate
                  ? 'Generate and save'
                  : 'Save passphrase'),
        ),
        const SizedBox(height: 20),
        if (widget.status.keyFilePath.isNotEmpty)
          Text(
            'Key file will be written to:\n${widget.status.keyFilePath}',
            style: theme.textTheme.bodySmall?.copyWith(color: muted),
            textAlign: TextAlign.center,
          ),
      ],
    );
  }

  Widget _generateExplainer(BuildContext context) {
    final theme = Theme.of(context);
    final muted = theme.colorScheme.outline;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.shield_outlined, size: 18, color: muted),
              const SizedBox(width: 8),
              Text(t.backups.encryption.random256bit,
                  style: const TextStyle(fontWeight: FontWeight.w600)),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            'Server generates a cryptographically random passphrase, '
            'shows it to you once. You must copy it to a password manager '
            'before continuing — there is no recovery path.',
            style: theme.textTheme.bodySmall?.copyWith(color: muted),
          ),
        ],
      ),
    );
  }

  Widget _pasteForm(BuildContext context) {
    final theme = Theme.of(context);
    return TextField(
      controller: _pasteCtrl,
      obscureText: false,
      maxLines: 2,
      minLines: 1,
      decoration: InputDecoration(
        labelText: t.backups.encryption.passphraseLabel,
        hintText: t.backups.encryption.passphraseHint,
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
        helperText: 'Recommended: 40+ chars from a password manager',
        helperStyle: TextStyle(color: theme.colorScheme.outline),
      ),
      style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
    );
  }

  Widget _generatedView(BuildContext context, BackupSetupResult result) {
    final theme = Theme.of(context);
    final muted = theme.colorScheme.outline;
    final pass = result.passphrase ?? '';
    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(20),
      children: [
        const SizedBox(height: 16),
        Icon(Icons.warning_amber_rounded,
            size: 48, color: Colors.amber.shade700),
        const SizedBox(height: 12),
        Text(
          'Save this passphrase NOW',
          style: theme.textTheme.titleMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 6),
        Text(
          'This is shown ONCE. It will not be retrievable from opendray '
          'or anywhere else. Copy it into a password manager before '
          'tapping Continue.',
          style: theme.textTheme.bodySmall?.copyWith(color: muted),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 20),
        Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest
                .withValues(alpha: 0.4),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
                color: theme.colorScheme.primary.withValues(alpha: 0.5)),
          ),
          child: SelectableText(
            pass,
            style: const TextStyle(
              fontFamily: 'monospace',
              fontSize: 14,
              height: 1.3,
            ),
          ),
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () async {
                  // Use the system clipboard. Selection is mostly
                  // redundant since the SelectableText supports tap-
                  // and-hold, but operators on phones with awkward
                  // selection (especially with the on-screen
                  // keyboard up) appreciate the explicit button.
                  await _copyToClipboard(context, pass);
                },
                icon: const Icon(Icons.copy, size: 16),
                label: Text(t.common.copy),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        if (result.keyFilePath.isNotEmpty)
          Text(
            'Saved to: ${result.keyFilePath}',
            style: theme.textTheme.bodySmall?.copyWith(color: muted),
            textAlign: TextAlign.center,
          ),
        const SizedBox(height: 20),
        CheckboxListTile(
          value: _ackSaved,
          onChanged: (v) => setState(() => _ackSaved = v ?? false),
          dense: true,
          title: const Text(
            'I have saved this passphrase to my password manager',
            style: TextStyle(fontSize: 13),
          ),
        ),
        const SizedBox(height: 12),
        FilledButton.icon(
          onPressed: _ackSaved ? () => widget.onComplete() : null,
          icon: const Icon(Icons.arrow_forward, size: 18),
          label: Text(t.onboarding.kContinue),
        ),
      ],
    );
  }

  Future<void> _copyToClipboard(BuildContext context, String text) async {
    await Clipboard.setData(ClipboardData(text: text));
    if (!context.mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(t.backups.encryption.passphraseCopied),
        duration: const Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }
}

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
    required this.targets,
    required this.schedules,
    required this.rows,
  });
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
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
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
