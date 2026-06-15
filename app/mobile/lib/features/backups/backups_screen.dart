import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
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
    this.health,
  });

  final BackupStatusReport status;
  final List<BackupRow> rows;
  final List<BackupTarget> targets;
  final List<BackupSchedule> schedules;
  // At-a-glance roll-up for the overview strip. Null when the feature
  // is off or the health endpoint transiently failed — the strip just
  // isn't rendered in that case.
  final BackupHealth? health;

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
        setState(
          () => _state = AsyncValue.data(
            _PageData(
              status: status,
              rows: const [],
              targets: const [],
              schedules: const [],
            ),
          ),
        );
        return;
      }
      // Kick health off concurrently with the lists; tolerate its
      // failure (the strip just hides) rather than fail the whole page.
      final healthFut = api.health();
      final results = await Future.wait<Object>([
        api.list(limit: 50).catchError((_) => <BackupRow>[]),
        api.listTargets().catchError((_) => <BackupTarget>[]),
        api.listSchedules().catchError((_) => <BackupSchedule>[]),
      ]);
      BackupHealth? health;
      try {
        health = await healthFut;
      } on Object {
        health = null;
      }
      if (!mounted) return;
      final rows = (results[0] as List<BackupRow>)
        ..sort((a, b) => b.startedAt.compareTo(a.startedAt));
      setState(
        () => _state = AsyncValue.data(
          _PageData(
            status: status,
            rows: rows,
            targets: results[1] as List<BackupTarget>,
            schedules: results[2] as List<BackupSchedule>,
            health: health,
          ),
        ),
      );
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
        setState(
          () => _state = AsyncValue.data(
            _PageData(
              status: status,
              rows: const [],
              targets: const [],
              schedules: const [],
            ),
          ),
        );
        return;
      }
      final healthFut = api.health();
      final list = await api.list(limit: 50);
      BackupHealth? health;
      try {
        health = await healthFut;
      } on Object {
        health = current.health;
      }
      if (!mounted) return;
      list.sort((a, b) => b.startedAt.compareTo(a.startedAt));
      setState(
        () => _state = AsyncValue.data(
          _PageData(
            status: status,
            rows: list,
            targets: current.targets,
            schedules: current.schedules,
            health: health,
          ),
        ),
      );
    } on Object {
      // Swallow soft-refresh errors — the page is already
      // rendered, no point flashing a full-screen error.
    }
  }

  Future<void> _runNow() async {
    var fullInstance = false;
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setLocal) => AlertDialog(
          title: Text(t.backups.runConfirmTitle),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(t.backups.runConfirmBody),
              const SizedBox(height: 8),
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                value: fullInstance,
                onChanged: (v) => setLocal(() => fullInstance = v),
                title: Text(t.backups.runFullInstance),
                subtitle: Text(t.backups.runFullInstanceHint),
              ),
            ],
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
      ),
    );
    if (ok != true || !mounted) return;
    setState(() => _running = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final row = await ref
          .read(backupsApiProvider)
          .runNow(kind: fullInstance ? 'full_instance' : 'db_only');
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
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.backups.runFailedGeneric(error: e.toString())),
        ),
      );
    } finally {
      if (mounted) setState(() => _running = false);
    }
  }

  // Recovery Kit: wrap the backup passphrase under a recovery passphrase
  // the operator stores out-of-band. On mobile we show the kit JSON with
  // a copy button (no filesystem download UX); the operator pastes it
  // into a password manager along with the recovery passphrase.
  Future<void> _openRecoveryKit() async {
    final passCtrl = TextEditingController();
    final confirmCtrl = TextEditingController();
    final messenger = ScaffoldMessenger.of(context);
    final go = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setLocal) {
          final ready =
              passCtrl.text.length >= 8 && passCtrl.text == confirmCtrl.text;
          return AlertDialog(
            title: Text(t.backups.recoveryKit.title),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(t.backups.recoveryKit.warning),
                const SizedBox(height: 12),
                TextField(
                  controller: passCtrl,
                  obscureText: true,
                  onChanged: (_) => setLocal(() {}),
                  decoration: InputDecoration(
                    labelText: t.backups.recoveryKit.passphraseLabel,
                  ),
                ),
                TextField(
                  controller: confirmCtrl,
                  obscureText: true,
                  onChanged: (_) => setLocal(() {}),
                  decoration: InputDecoration(
                    labelText: t.backups.recoveryKit.confirmLabel,
                  ),
                ),
              ],
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(ctx).pop(false),
                child: Text(t.common.cancel),
              ),
              FilledButton(
                onPressed: ready ? () => Navigator.of(ctx).pop(true) : null,
                child: Text(t.backups.recoveryKit.generate),
              ),
            ],
          );
        },
      ),
    );
    if (go != true || !mounted) return;
    try {
      final kit = await ref
          .read(backupsApiProvider)
          .recoveryKit(passCtrl.text);
      final pretty = const JsonEncoder.withIndent('  ').convert(kit);
      if (!mounted) return;
      await showDialog<void>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: Text(t.backups.recoveryKit.title),
          content: SingleChildScrollView(
            child: SelectableText(
              pretty,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(),
              child: Text(t.common.close),
            ),
            FilledButton.icon(
              onPressed: () async {
                await Clipboard.setData(ClipboardData(text: pretty));
                if (ctx.mounted) Navigator.of(ctx).pop();
                messenger.showSnackBar(
                  SnackBar(content: Text(t.backups.recoveryKit.copied)),
                );
              },
              icon: const Icon(Icons.copy, size: 16),
              label: Text(t.backups.recoveryKit.copy),
            ),
          ],
        ),
      );
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(t.backups.recoveryKit.failed(error: e.message))),
      );
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.backups.recoveryKit.failed(error: e.toString())),
        ),
      );
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
    _runPoll = Timer.periodic(const Duration(seconds: 3), (timer) async {
      ticks++;
      if (!mounted) {
        timer.cancel();
        return;
      }
      await _softRefresh();
      if (!mounted) {
        timer.cancel();
        return;
      }
      final row = _state.valueOrNull?.rows.firstWhere(
        (r) => r.id == rowId,
        orElse: () => BackupRow(
          id: '',
          targetId: '',
          status: '',
          triggeredBy: '',
          kind: 'db_only',
          startedAt: DateTime.now().toUtc(),
          bytes: 0,
          encrypted: false,
        ),
      );
      final settled =
          row != null && (row.status == 'succeeded' || row.status == 'failed');
      if (settled) {
        timer.cancel();
        _runPoll = null;
        final ok = row.status == 'succeeded';
        messenger.showSnackBar(
          SnackBar(
            content: Text(
              ok
                  ? t.backups.rowSucceededSnack(bytes: _formatBytes(row.bytes))
                  : t.backups.rowFailedSnack(
                      error: row.error ?? t.backups.unknownError,
                    ),
            ),
            duration: const Duration(seconds: 3),
            behavior: SnackBarBehavior.floating,
            backgroundColor: ok ? null : Theme.of(context).colorScheme.error,
          ),
        );
      } else if (ticks >= maxTicks) {
        timer.cancel();
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
                _kv(t.backups.kv.status, b.status),
                if (b.status == 'succeeded')
                  _kv(
                    t.backups.kv.verified,
                    b.verifyError != null
                        ? t.backups.verifyFailed
                        : b.verifiedAt != null
                        ? t.backups.verifyOk
                        : t.backups.verifyPending,
                  ),
                _kv(
                  t.backups.kv.kind,
                  b.kind == 'full_instance'
                      ? t.backups.kindFullInstance
                      : t.backups.kindDbOnly,
                ),
                _kv(t.backups.kv.target, b.targetId),
                if (b.deduped) _kv(t.backups.kv.dedup, t.backups.dedupValue),
                if (b.groupId != null && b.groupId!.isNotEmpty)
                  _kv(t.backups.kv.fanout, b.groupId!, mono: true),
                _kv(t.backups.kv.triggeredBy, b.triggeredBy),
                _kv(
                  t.backups.kv.started,
                  DateFormat.yMMMd().add_Hms().format(b.startedAt.toLocal()),
                ),
                if (b.finishedAt != null)
                  _kv(
                    t.backups.kv.finished,
                    DateFormat.yMMMd().add_Hms().format(
                      b.finishedAt!.toLocal(),
                    ),
                  ),
                _kv(t.backups.kv.size, _formatBytes(b.bytes)),
                _kv(
                  t.backups.kv.encrypted,
                  b.encrypted ? t.backups.kv.yes : t.backups.kv.no,
                ),
                if ((b.targetPath ?? '').isNotEmpty)
                  _kv(t.backups.kv.targetPath, b.targetPath!, mono: true),
                if ((b.error ?? '').isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text(
                    t.backups.kv.error,
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
              t.backups.deleteBody(target: b.targetId),
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
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.backups.deleteFailedGeneric(error: e.toString())),
        ),
      );
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
                case _AppBarAction.restore:
                  unawaited(_openRestoreSheet());
                case _AppBarAction.recoveryKit:
                  unawaited(_openRecoveryKit());
              }
            },
            itemBuilder: (_) => [
              PopupMenuItem(
                value: _AppBarAction.restore,
                child: ListTile(
                  leading: const Icon(Icons.restore_outlined),
                  title: Text(t.backups.restoreFromFile),
                ),
              ),
              PopupMenuItem(
                value: _AppBarAction.recoveryKit,
                child: ListTile(
                  leading: const Icon(Icons.vpn_key_outlined),
                  title: Text(t.backups.recoveryKit.menuLabel),
                ),
              ),
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
              label: Text(_running ? t.backups.queueing : t.backups.runNow),
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
        return _RestartRequiredView(status: data.status, onRecheck: _load);
      }
      return _SetupWizardView(status: data.status, onComplete: _load);
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
          if (data.health != null) _HealthStrip(health: data.health!),
          _StatusBanner(status: status),
          const _InventoryCard(),
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

  // ── restore (from uploaded bundle) ─────────────────────────────

  // Opens the multi-step Restore bottom sheet. Pulled out of the
  // PopupMenuButton's onSelected so the sheet itself owns its own
  // state (file pick, target DSN, clean, audit note, confirm
  // sentinel) without the parent rebuilding on every keystroke.
  Future<void> _openRestoreSheet() async {
    final res = await showModalBottomSheet<RestoreResult?>(
      context: context,
      isScrollControlled: true,
      builder: (_) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: const _RestoreSheet(),
      ),
    );
    if (res != null && mounted) {
      await _showRestoreResult(res);
      await _softRefresh();
    }
  }

  Future<void> _showRestoreResult(RestoreResult res) async {
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.backups.restore.succeededTitle),
        content: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.of(ctx).size.height * 0.7,
            maxWidth: 480,
          ),
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  t.backups.restore.succeededBody(
                    bytes: _formatBytes(res.bytesRead),
                    id: res.manifest.backupId,
                  ),
                ),
                const SizedBox(height: 12),
                Text(
                  t.backups.restore.manifestTitle.toUpperCase(),
                  style: Theme.of(ctx).textTheme.labelSmall?.copyWith(
                    letterSpacing: 0.8,
                    color: Theme.of(ctx).colorScheme.outline,
                  ),
                ),
                const SizedBox(height: 6),
                _kv(
                  t.backups.restore.manifestBackupId,
                  res.manifest.backupId,
                  mono: true,
                ),
                _kv(t.backups.restore.manifestVersion, res.manifest.version),
                _kv(
                  t.backups.restore.manifestCreatedAt,
                  DateFormat.yMMMd().add_Hms().format(
                    res.manifest.createdAt.toLocal(),
                  ),
                ),
                if ((res.manifest.pgVersion ?? '').isNotEmpty)
                  _kv(
                    t.backups.restore.manifestPgVersion,
                    res.manifest.pgVersion!,
                  ),
                if ((res.manifest.opendrayVersion ?? '').isNotEmpty)
                  _kv(
                    t.backups.restore.manifestOpendrayVersion,
                    res.manifest.opendrayVersion!,
                  ),
                _kv(
                  t.backups.restore.fingerprint,
                  res.fingerprintOk
                      ? '${res.manifest.encryptionFingerprint} '
                            '· ${t.backups.restore.fingerprintOk}'
                      : '${res.manifest.encryptionFingerprint} '
                            '· ${t.backups.restore.fingerprintMismatch}',
                  mono: true,
                ),
                _kv(
                  t.backups.restore.encryptionAlgo,
                  res.manifest.encryptionAlgo,
                ),
                _kv(t.backups.restore.bytesRead, _formatBytes(res.bytesRead)),
                _kv(
                  t.backups.restore.targetDsnUsed,
                  res.targetDsnUsed.isEmpty
                      ? t.backups.restore.targetDsnSelfLabel
                      : res.targetDsnUsed,
                  mono: res.targetDsnUsed.isNotEmpty,
                ),
                const SizedBox(height: 12),
                Text(
                  t.backups.restore.outputTitle.toUpperCase(),
                  style: Theme.of(ctx).textTheme.labelSmall?.copyWith(
                    letterSpacing: 0.8,
                    color: Theme.of(ctx).colorScheme.outline,
                  ),
                ),
                const SizedBox(height: 6),
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: Theme.of(ctx).colorScheme.surfaceContainerHighest,
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: SelectableText(
                    res.pgRestoreOutput.isEmpty
                        ? t.backups.restore.noPgRestoreOutput
                        : res.pgRestoreOutput,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 11,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
        actions: [
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(t.backups.restore.done),
          ),
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
      headline = t.backups.emptyMissingDeps.headline;
      body = data.status.pgDumpError ?? t.backups.emptyMissingDeps.body;
    } else if (!hasTargets) {
      icon = Icons.cloud_off_outlined;
      headline = t.backups.emptyNoTargets.headline;
      body = t.backups.emptyNoTargets.body;
    } else {
      icon = Icons.archive_outlined;
      headline = t.backups.emptyNoBackups.headline;
      body = t.backups.emptyNoBackups.body;
    }
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 40),
      child: Column(
        children: [
          Icon(icon, size: 48, color: theme.colorScheme.outline),
          const SizedBox(height: 12),
          Text(
            headline,
            style: theme.textTheme.titleMedium,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 6),
          Text(
            body,
            style: theme.textTheme.bodySmall,
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }
}

enum _DetailAction { close, delete }

enum _AppBarAction { schedules, targets, restore, recoveryKit }

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
            t.backups.restartToActivate,
            style: theme.textTheme.titleMedium,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          Text(
            t.backups.passphraseSaved,
            style: theme.textTheme.bodySmall?.copyWith(color: muted),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 20),
          if (status.configuredVia == 'file' && status.keyFilePath.isNotEmpty)
            _kvBox(
              context,
              label: t.backups.keyFileLabel,
              value: status.keyFilePath,
            ),
          if (status.configuredVia == 'env')
            _kvBox(
              context,
              label: t.backups.configuredViaLabel,
              value: t.backups.envVarConfigured,
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

  Widget _kvBox(
    BuildContext context, {
    required String label,
    required String value,
  }) {
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
          Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.outline,
            ),
          ),
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
          : await api.setup(mode: 'paste', passphrase: _pasteCtrl.text.trim());
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
          t.backups.wizard.title,
          style: theme.textTheme.titleMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 6),
        Text(
          t.backups.wizard.intro,
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
                color: theme.colorScheme.error.withValues(alpha: 0.4),
              ),
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
          label: Text(
            _submitting
                ? t.backups.wizard.saving
                : _mode == _SetupMode.generate
                ? t.backups.wizard.generateAndSave
                : t.backups.wizard.savePassphrase,
          ),
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
              Text(
                t.backups.encryption.random256bit,
                style: const TextStyle(fontWeight: FontWeight.w600),
              ),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            t.backups.wizard.generateHint,
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
        helperText: t.backups.wizard.helperRecommended,
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
        Icon(
          Icons.warning_amber_rounded,
          size: 48,
          color: Colors.amber.shade700,
        ),
        const SizedBox(height: 12),
        Text(
          t.backups.wizard.saveNowHeader,
          style: theme.textTheme.titleMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 6),
        Text(
          t.backups.wizard.saveNowBody,
          style: theme.textTheme.bodySmall?.copyWith(color: muted),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 20),
        Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: theme.colorScheme.surfaceContainerHighest.withValues(
              alpha: 0.4,
            ),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
              color: theme.colorScheme.primary.withValues(alpha: 0.5),
            ),
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
          title: Text(
            t.backups.savedConfirmCheckbox,
            style: const TextStyle(fontSize: 13),
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
// At-a-glance health strip: when the last good backup landed plus any
// counts that need attention (recent failures, failed verifications,
// overdue schedules). Green when all is well, red when not, neutral
// before the first successful backup.
class _HealthStrip extends StatelessWidget {
  const _HealthStrip({required this.health});
  final BackupHealth health;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final allClear = health.allClear;
    final neverBackedUp = health.neverBackedUp;
    final color = allClear
        ? Colors.green
        : neverBackedUp
        ? theme.colorScheme.outline
        : theme.colorScheme.error;
    final bg = color.withValues(alpha: 0.10);
    final border = color.withValues(alpha: 0.45);

    final attention = <String>[
      if (health.recentFailures > 0)
        t.backups.health.recentFailures(count: health.recentFailures),
      if (health.verifyFailures > 0)
        t.backups.health.verifyFailures(count: health.verifyFailures),
      if (health.overdueSchedules > 0)
        t.backups.health.overdueSchedules(count: health.overdueSchedules),
    ];

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
              Icon(
                allClear
                    ? Icons.verified_user_outlined
                    : Icons.error_outline,
                size: 18,
                color: color,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text.rich(
                  TextSpan(
                    children: [
                      TextSpan(
                        text: '${t.backups.health.lastSuccess}: ',
                        style: theme.textTheme.bodySmall,
                      ),
                      TextSpan(
                        text: neverBackedUp
                            ? t.backups.health.never
                            : _relTime(health.lastSuccessAt!),
                        style: TextStyle(
                          color: color,
                          fontWeight: FontWeight.w600,
                          fontSize: 13,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
          if (attention.isNotEmpty) ...[
            const SizedBox(height: 6),
            ...attention.map(
              (line) => Padding(
                padding: const EdgeInsets.only(top: 2),
                child: Text(
                  line,
                  style: theme.textTheme.bodySmall?.copyWith(color: color),
                ),
              ),
            ),
          ] else if (allClear)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Text(
                t.backups.health.allClear,
                style: theme.textTheme.bodySmall?.copyWith(color: color),
              ),
            ),
          if (health.schedules > 0) ...[
            const SizedBox(height: 6),
            Text(
              t.backups.health.scheduleSummary(
                enabled: health.enabledSchedules,
                total: health.schedules,
              ),
              style: theme.textTheme.bodySmall,
            ),
          ],
        ],
      ),
    );
  }
}

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
              Icon(
                ok ? Icons.check_circle_outline : Icons.error_outline,
                size: 18,
                color: color,
              ),
              const SizedBox(width: 8),
              Text(
                ok ? t.backups.statusReady : t.backups.statusCannot,
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
              status.pgDumpError ?? t.backups.pgDumpMissing,
              style: theme.textTheme.bodySmall?.copyWith(color: color),
            ),
        ],
      ),
    );
  }

  Widget _kvRow(
    BuildContext context,
    String label,
    String value, {
    bool mono = false,
  }) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(width: 110, child: Text(label, style: muted)),
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
    final totalBytes = liveRows.fold<int>(
      0,
      (acc, r) => acc + (r.bytes > 0 ? r.bytes : 0),
    );
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
            label: t.backups.overviewTargets,
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
            label: t.backups.overviewSchedules,
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
            label: t.backups.overviewBackups,
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
              Text(
                label,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.outline,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                value,
                style: const TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                ),
              ),
              if (sub != null)
                Text(
                  sub!,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.outline,
                  ),
                ),
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
            Icon(Icons.lock_outline, size: 13, color: muted?.color),
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
              t.backups.failedToLoad,
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

// Collapsible "what's actually backed up" card. Mirrors web's
// InventoryCard — defers loading until first tap so the main screen
// still paints instantly on a slow network. Re-fetches every time
// the operator collapses + re-expands (data is cheap; server-side
// it's a single COUNT(*) over each table).
class _InventoryCard extends ConsumerStatefulWidget {
  const _InventoryCard();

  @override
  ConsumerState<_InventoryCard> createState() => _InventoryCardState();
}

class _InventoryCardState extends ConsumerState<_InventoryCard> {
  bool _open = false;
  bool _loading = false;
  List<InventoryGroup>? _groups;

  Future<void> _toggle() async {
    setState(() => _open = !_open);
    if (!_open || _groups != null || _loading) return;
    setState(() => _loading = true);
    try {
      final groups = await ref.read(backupsApiProvider).inventory();
      if (mounted) setState(() => _groups = groups);
    } on Object catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            '${t.backups.inventory.loadFailedToast}: ${e is ApiException ? e.message : e}',
          ),
        ),
      );
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final groups = _groups;
    final totalRows = groups?.fold<int>(
      0,
      (acc, g) => acc + g.tables.fold<int>(0, (a, tbl) => a + tbl.count),
    );
    final totalTables = groups?.fold<int>(0, (acc, g) => acc + g.tables.length);
    final fmt = NumberFormat.decimalPattern();
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          InkWell(
            onTap: _toggle,
            borderRadius: BorderRadius.circular(8),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              child: Row(
                children: [
                  Icon(
                    _open ? Icons.expand_more : Icons.chevron_right,
                    size: 18,
                    color: theme.colorScheme.outline,
                  ),
                  const SizedBox(width: 8),
                  Icon(
                    Icons.inventory_2_outlined,
                    size: 16,
                    color: theme.colorScheme.primary,
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      t.backups.inventory.title,
                      style: theme.textTheme.titleSmall,
                    ),
                  ),
                  if (totalRows != null && totalTables != null)
                    Text(
                      t.backups.inventory.summary(
                        rows: fmt.format(totalRows),
                        tables: totalTables.toString(),
                      ),
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.outline,
                      ),
                    )
                  else
                    Text(
                      t.backups.inventory.tap,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.outline,
                      ),
                    ),
                ],
              ),
            ),
          ),
          if (_open)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    t.backups.inventory.description,
                    style: theme.textTheme.bodySmall,
                  ),
                  const SizedBox(height: 8),
                  if (_loading)
                    Text(
                      t.backups.inventory.loading,
                      style: theme.textTheme.bodySmall,
                    ),
                  if (groups != null)
                    ...groups.map(
                      (g) => Padding(
                        padding: const EdgeInsets.only(bottom: 10),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              children: [
                                Text(
                                  g.label,
                                  style: theme.textTheme.titleSmall,
                                ),
                                const SizedBox(width: 8),
                                Text(
                                  '${fmt.format(g.tables.fold<int>(0, (a, t) => a + t.count))} '
                                  '${t.backups.inventory.rowsLabel}',
                                  style: theme.textTheme.bodySmall?.copyWith(
                                    color: theme.colorScheme.outline,
                                  ),
                                ),
                              ],
                            ),
                            const SizedBox(height: 2),
                            Text(
                              g.description,
                              style: theme.textTheme.bodySmall,
                            ),
                            const SizedBox(height: 6),
                            Wrap(
                              spacing: 6,
                              runSpacing: 4,
                              children: g.tables.map((tbl) {
                                return Container(
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 8,
                                    vertical: 2,
                                  ),
                                  decoration: BoxDecoration(
                                    color: theme
                                        .colorScheme
                                        .surfaceContainerHighest,
                                    border: Border.all(
                                      color: theme.dividerColor,
                                      width: 0.5,
                                    ),
                                    borderRadius: BorderRadius.circular(4),
                                  ),
                                  child: Row(
                                    mainAxisSize: MainAxisSize.min,
                                    crossAxisAlignment:
                                        CrossAxisAlignment.baseline,
                                    textBaseline: TextBaseline.alphabetic,
                                    children: [
                                      Text(
                                        tbl.name,
                                        style: const TextStyle(
                                          fontFamily: 'monospace',
                                          fontSize: 11,
                                        ),
                                      ),
                                      const SizedBox(width: 6),
                                      Text(
                                        fmt.format(tbl.count),
                                        style: theme.textTheme.bodySmall
                                            ?.copyWith(
                                              color: theme.colorScheme.outline,
                                            ),
                                      ),
                                    ],
                                  ),
                                );
                              }).toList(),
                            ),
                          ],
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
}

// Bottom-sheet form for /backups/restore — file pick, target DSN,
// clean toggle, audit note, plus the "I understand" sentinel that's
// required when target_dsn is empty (i.e. restoring into opendray's
// own DB). Submits via api.restore() and pops with the RestoreResult
// on success so the parent can show the manifest dialog.
class _RestoreSheet extends ConsumerStatefulWidget {
  const _RestoreSheet();

  @override
  ConsumerState<_RestoreSheet> createState() => _RestoreSheetState();
}

class _RestoreSheetState extends ConsumerState<_RestoreSheet> {
  PlatformFile? _picked;
  final _targetDsnCtrl = TextEditingController();
  final _noteCtrl = TextEditingController();
  final _confirmCtrl = TextEditingController();
  bool _clean = true;
  bool _busy = false;
  // Plan from the most recent dry-run preview. Apply is gated on this
  // being non-null so an operator always sees what a restore would do
  // before it touches a live database. Any change to the bundle / DSN /
  // clean flag invalidates it (the plan is specific to those inputs).
  RestorePlan? _plan;

  @override
  void dispose() {
    _targetDsnCtrl.dispose();
    _noteCtrl.dispose();
    _confirmCtrl.dispose();
    super.dispose();
  }

  bool get _restoringOwn => _targetDsnCtrl.text.trim().isEmpty;
  bool get _confirmReady =>
      !_restoringOwn ||
      _confirmCtrl.text.trim() == t.backups.restore.confirmSentinel;

  Future<void> _pickBundle() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.any,
      withData: false,
    );
    if (result == null || result.files.isEmpty) return;
    final f = result.files.single;
    if (f.path == null || f.path!.isEmpty) return;
    setState(() {
      _picked = f;
      _plan = null;
    });
  }

  // Step 1: dry run — validates the bundle and reports a plan; changes
  // nothing on disk or in the database.
  Future<void> _preview() async {
    final picked = _picked;
    if (picked == null || picked.path == null) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(t.backups.restore.pickFileToast)));
      return;
    }
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final res = await ref
          .read(backupsApiProvider)
          .restore(
            bundle: File(picked.path!),
            targetDsn: _targetDsnCtrl.text.trim().isEmpty
                ? null
                : _targetDsnCtrl.text.trim(),
            clean: _clean,
            apply: false,
          );
      if (!mounted) return;
      setState(() => _plan = res.plan);
      messenger.showSnackBar(
        SnackBar(content: Text(t.backups.restore.dryRunToast)),
      );
    } on ApiException catch (e) {
      _showError(messenger, e.message);
    } on Object catch (e) {
      _showError(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  // Step 2: apply — commits the restore (safety snapshot + write +
  // pg_restore). Gated on a prior preview so the plan card is always
  // shown first.
  Future<void> _apply() async {
    final picked = _picked;
    if (picked == null || picked.path == null || _plan == null) return;
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final res = await ref
          .read(backupsApiProvider)
          .restore(
            bundle: File(picked.path!),
            targetDsn: _targetDsnCtrl.text.trim().isEmpty
                ? null
                : _targetDsnCtrl.text.trim(),
            clean: _clean,
            apply: true,
            confirm: _restoringOwn ? t.backups.restore.confirmSentinel : null,
            note: _noteCtrl.text.trim().isEmpty ? null : _noteCtrl.text.trim(),
          );
      if (!mounted) return;
      Navigator.of(context).pop(res);
    } on ApiException catch (e) {
      _showError(messenger, e.message);
    } on Object catch (e) {
      _showError(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _showError(ScaffoldMessengerState messenger, String msg) {
    if (!mounted) return;
    messenger.showSnackBar(
      SnackBar(
        content: Text('${t.backups.restore.failedTitle}: $msg'),
        backgroundColor: Theme.of(context).colorScheme.error,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                children: [
                  Icon(
                    Icons.restore_outlined,
                    color: theme.colorScheme.primary,
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      t.backups.restore.title,
                      style: theme.textTheme.titleMedium,
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: _busy ? null : () => Navigator.of(context).pop(),
                  ),
                ],
              ),
              Text(
                t.backups.restore.subtitle,
                style: theme.textTheme.bodySmall,
              ),
              const SizedBox(height: 16),
              Text(
                t.backups.restore.bundleLabel,
                style: theme.textTheme.labelMedium,
              ),
              const SizedBox(height: 4),
              Row(
                children: [
                  OutlinedButton.icon(
                    onPressed: _busy ? null : _pickBundle,
                    icon: const Icon(Icons.attach_file, size: 16),
                    label: Text(t.backups.restore.pickFile),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      _picked == null
                          ? t.backups.restore.noFile
                          : t.backups.restore.fileSelected(
                              name: _picked!.name,
                              size: _formatBytes(_picked!.size),
                            ),
                      style: theme.textTheme.bodySmall,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              Text(
                t.backups.restore.targetDsnLabel,
                style: theme.textTheme.labelMedium,
              ),
              const SizedBox(height: 2),
              Text(
                t.backups.restore.targetDsnHint,
                style: theme.textTheme.bodySmall,
              ),
              const SizedBox(height: 4),
              TextField(
                controller: _targetDsnCtrl,
                enabled: !_busy,
                style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
                decoration: InputDecoration(
                  isDense: true,
                  border: const OutlineInputBorder(),
                  hintText: t.backups.restore.targetDsnPlaceholder,
                ),
                // DSN change re-evaluates the own-DB confirm gate and
                // invalidates any plan previewed against the old target.
                onChanged: (_) => setState(() => _plan = null),
              ),
              const SizedBox(height: 12),
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                value: _clean,
                onChanged: _busy
                    ? null
                    : (v) => setState(() {
                        _clean = v;
                        _plan = null;
                      }),
                title: Text(t.backups.restore.cleanLabel),
                subtitle: Text(t.backups.restore.cleanHint),
              ),
              const SizedBox(height: 8),
              Text(
                t.backups.restore.auditNoteLabel,
                style: theme.textTheme.labelMedium,
              ),
              const SizedBox(height: 4),
              TextField(
                controller: _noteCtrl,
                enabled: !_busy,
                decoration: InputDecoration(
                  isDense: true,
                  border: const OutlineInputBorder(),
                  hintText: t.backups.restore.auditNotePlaceholder,
                ),
              ),
              if (_restoringOwn) ...[
                const SizedBox(height: 12),
                Container(
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.errorContainer.withValues(
                      alpha: 0.4,
                    ),
                    border: Border.all(color: theme.colorScheme.error),
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Icon(
                            Icons.shield_outlined,
                            size: 16,
                            color: theme.colorScheme.error,
                          ),
                          const SizedBox(width: 6),
                          Expanded(
                            child: Text(
                              t.backups.restore.ownDbWarning,
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.error,
                              ),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),
                      TextField(
                        controller: _confirmCtrl,
                        enabled: !_busy,
                        decoration: InputDecoration(
                          isDense: true,
                          border: const OutlineInputBorder(),
                          hintText: t.backups.restore.confirmPlaceholder,
                        ),
                        onChanged: (_) => setState(() {}),
                      ),
                    ],
                  ),
                ),
              ],
              if (_plan != null) ...[
                const SizedBox(height: 16),
                _RestorePlanCard(plan: _plan!),
              ] else if (_picked != null) ...[
                const SizedBox(height: 12),
                Text(
                  t.backups.restore.previewFirstHint,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.outline,
                  ),
                ),
              ],
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: _busy || _picked == null ? null : _preview,
                      child: Text(
                        _busy && _plan == null
                            ? t.backups.restore.previewing
                            : t.backups.restore.preview,
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton(
                      style: FilledButton.styleFrom(
                        backgroundColor: theme.colorScheme.error,
                      ),
                      onPressed:
                          _busy ||
                              _picked == null ||
                              _plan == null ||
                              !_confirmReady
                          ? null
                          : _apply,
                      child: Text(
                        _busy && _plan != null
                            ? t.backups.restore.restoring
                            : t.backups.restore.applyRestore,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 4),
            ],
          ),
        ),
      ),
    );
  }
}

// Read-only preview of what an apply-mode restore would write, shown
// after a dry-run. Mirrors the web restore plan card.
class _RestorePlanCard extends StatelessWidget {
  const _RestorePlanCard({required this.plan});
  final RestorePlan plan;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final muted = theme.textTheme.bodySmall?.copyWith(
      color: theme.colorScheme.outline,
    );
    final lines = <Widget>[
      if (plan.dumpPresent)
        Text(
          t.backups.restore.planDump(size: _formatBytes(plan.dumpBytes)),
          style: theme.textTheme.bodySmall,
        ),
      if (plan.configPath.isNotEmpty)
        Text(
          t.backups.restore.planConfig(path: plan.configPath),
          style: theme.textTheme.bodySmall,
        ),
      if (plan.secretsPath.isNotEmpty)
        Text(
          t.backups.restore.planSecrets(path: plan.secretsPath),
          style: theme.textTheme.bodySmall,
        ),
      if (plan.vaultFiles > 0)
        Text(
          t.backups.restore.planVault(
            files: plan.vaultFiles,
            roots: plan.vaultRoots.join(', '),
          ),
          style: theme.textTheme.bodySmall,
        ),
    ];
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(
          alpha: 0.4,
        ),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            t.backups.restore.planTitle.toUpperCase(),
            style: theme.textTheme.labelSmall?.copyWith(
              letterSpacing: 0.6,
              color: theme.colorScheme.outline,
            ),
          ),
          const SizedBox(height: 6),
          ...lines.expand(
            (w) => [w, const SizedBox(height: 2)],
          ),
          const SizedBox(height: 6),
          Text(t.backups.restore.planApplyHint, style: muted),
        ],
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
