import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/vault_git_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Vault sync screen — the phone-side counterpart of the web's vault
// sync dialog, minus anything that reshapes the repository.
//
// Scope is deliberate: status, the three manual actions (commit, push,
// pull) and the auto-sync settings. Repository init, remote
// configuration, abort and reset-to-remote stay web-only — the last of
// those discards local work, and a phone is the worst place to hit it
// by accident.

// Interval presets, matching the web picker. The sync loop can't tick
// faster than 30s (vaultgit.minTickInterval), so nothing shorter is
// offered.
const _commitPresets = ['30s', '1m', '5m', '10m', '15m', '30m', '1h', '6h'];
const _pullPresets = ['5m', '15m', '30m', '1h', '6h', '24h'];

class VaultSyncScreen extends ConsumerStatefulWidget {
  const VaultSyncScreen({super.key});

  @override
  ConsumerState<VaultSyncScreen> createState() => _VaultSyncScreenState();
}

class _VaultSyncScreenState extends ConsumerState<VaultSyncScreen> {
  AsyncValue<VaultStatus> _status = const AsyncValue.loading();
  AsyncValue<VaultSyncConfig> _config = const AsyncValue.loading();

  // Pending edits, applied over the server config. Kept separate from
  // the loaded value so a refresh can never overwrite something the
  // operator is part-way through changing — the web client had exactly
  // that bug, where an 8s poll wiped the field mid-edit.
  bool? _draftEnabled;
  String? _draftCommitInterval;
  String? _draftPullInterval;
  bool? _draftPushEnabled;
  bool? _draftPullEnabled;

  final Set<String> _busy = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  bool get _dirty =>
      _draftEnabled != null ||
      _draftCommitInterval != null ||
      _draftPullInterval != null ||
      _draftPushEnabled != null ||
      _draftPullEnabled != null;

  Future<void> _load() async {
    final api = ref.read(vaultGitApiProvider);
    setState(() {
      _status = const AsyncValue.loading();
      _config = const AsyncValue.loading();
    });
    try {
      final results = await Future.wait([api.status(), api.syncConfig()]);
      if (!mounted) return;
      setState(() {
        _status = AsyncValue.data(results[0] as VaultStatus);
        _config = AsyncValue.data(results[1] as VaultSyncConfig);
      });
    } on Object catch (e, st) {
      if (!mounted) return;
      setState(() {
        _status = AsyncValue.error(e, st);
        _config = AsyncValue.error(e, st);
      });
    }
  }

  /// Refresh that leaves unsaved edits alone.
  Future<void> _refresh() async {
    final api = ref.read(vaultGitApiProvider);
    try {
      final results = await Future.wait([api.status(), api.syncConfig()]);
      if (!mounted) return;
      setState(() {
        _status = AsyncValue.data(results[0] as VaultStatus);
        _config = AsyncValue.data(results[1] as VaultSyncConfig);
      });
    } on Object catch (e, st) {
      if (!mounted) return;
      setState(() => _status = AsyncValue.error(e, st));
    }
  }

  void _toast(String msg, {bool error = false}) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(msg),
        behavior: SnackBarBehavior.floating,
        duration: Duration(seconds: error ? 5 : 2),
        backgroundColor:
            error ? Theme.of(context).colorScheme.errorContainer : null,
      ),
    );
  }

  Future<void> _run({
    required String key,
    required Future<String?> Function() op,
    required String Function(String result) ok,
  }) async {
    setState(() => _busy.add(key));
    try {
      final out = await op();
      if (!mounted) return;
      _toast(ok(out ?? ''));
      await _refresh();
    } on ApiException catch (e) {
      // git's own stderr is the most useful thing we can show — e.g.
      // "nothing to commit" arrives as a 422 with the real reason.
      _toast(e.message, error: true);
    } on Object catch (e) {
      _toast('$e', error: true);
    } finally {
      if (mounted) setState(() => _busy.remove(key));
    }
  }

  Future<void> _save() async {
    setState(() => _busy.add('save'));
    try {
      final next = await ref.read(vaultGitApiProvider).setSyncConfig(
            enabled: _draftEnabled,
            commitInterval: _draftCommitInterval,
            pullInterval: _draftPullInterval,
            pushEnabled: _draftPushEnabled,
            pullEnabled: _draftPullEnabled,
          );
      if (!mounted) return;
      setState(() {
        _config = AsyncValue.data(next);
        _draftEnabled = null;
        _draftCommitInterval = null;
        _draftPullInterval = null;
        _draftPushEnabled = null;
        _draftPullEnabled = null;
      });
      _toast(t.vaultSync.savedToast);
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    } on Object catch (e) {
      _toast('$e', error: true);
    } finally {
      if (mounted) setState(() => _busy.remove('save'));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.vaultSync.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.vaultSync.refresh,
            onPressed: _refresh,
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 32),
          children: [
            _statusCard(),
            const SizedBox(height: 12),
            _actionsCard(),
            const SizedBox(height: 12),
            _autoSyncCard(),
          ],
        ),
      ),
    );
  }

  // ── Status ──────────────────────────────────────────────────────

  Widget _statusCard() {
    final theme = Theme.of(context);
    return _card(
      title: t.vaultSync.statusTitle,
      child: _status.when(
        loading: () => const Padding(
          padding: EdgeInsets.symmetric(vertical: 12),
          child: Center(child: CircularProgressIndicator.adaptive()),
        ),
        error: (e, _) => Text(
          e is ApiException ? e.message : '$e',
          style: TextStyle(color: theme.colorScheme.error),
        ),
        data: (s) {
          if (!s.isRepo) {
            return Text(t.vaultSync.notARepo, style: theme.textTheme.bodySmall);
          }
          final mid = s.state?.isMidOperation ?? false;
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Wrap(
                spacing: 8,
                runSpacing: 6,
                crossAxisAlignment: WrapCrossAlignment.center,
                children: [
                  _chip(Icons.call_split, s.branch.isEmpty ? '—' : s.branch),
                  if (s.ahead > 0)
                    _chip(Icons.arrow_upward,
                        t.vaultSync.ahead(n: s.ahead)),
                  if (s.behind > 0)
                    _chip(Icons.arrow_downward,
                        t.vaultSync.behind(n: s.behind)),
                  _chip(
                    s.isClean ? Icons.check_circle_outline : Icons.edit_note,
                    s.isClean
                        ? t.vaultSync.clean
                        : t.vaultSync.changedFiles(n: s.files.length),
                  ),
                  if (!s.hasRemote)
                    _chip(Icons.cloud_off, t.vaultSync.noRemote,
                        tone: theme.colorScheme.error),
                ],
              ),
              if (mid) ...[
                const SizedBox(height: 8),
                _notice(t.vaultSync.midOperation, theme.colorScheme.error),
              ],
              const SizedBox(height: 8),
              Text(s.root,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontFamily: 'monospace',
                    color: theme.colorScheme.onSurfaceVariant,
                  )),
            ],
          );
        },
      ),
    );
  }

  // ── Manual actions ──────────────────────────────────────────────

  Widget _actionsCard() {
    final s = _status.valueOrNull;
    final blocked = s == null || !s.isRepo || (s.state?.isMidOperation ?? false);
    final noRemote = s == null || !s.hasRemote;
    return _card(
      title: t.vaultSync.actionsTitle,
      child: Wrap(
        spacing: 8,
        runSpacing: 8,
        children: [
          _actionButton(
            key_: 'commit',
            icon: Icons.save_outlined,
            label: t.vaultSync.commit,
            // Nothing staged means git exits non-zero; don't offer it.
            enabled: !blocked && !s.isClean,
            onPressed: () => _run(
              key: 'commit',
              op: () => ref.read(vaultGitApiProvider).commit(),
              ok: (hash) => hash.isEmpty
                  ? t.vaultSync.committedToast
                  : t.vaultSync.committedToastWithHash(hash: hash),
            ),
          ),
          _actionButton(
            key_: 'push',
            icon: Icons.cloud_upload_outlined,
            label: t.vaultSync.push,
            enabled: !blocked && !noRemote,
            onPressed: () => _run(
              key: 'push',
              op: () => ref.read(vaultGitApiProvider).push(),
              ok: (_) => t.vaultSync.pushedToast,
            ),
          ),
          _actionButton(
            key_: 'pull',
            icon: Icons.cloud_download_outlined,
            label: t.vaultSync.pull,
            enabled: !blocked && !noRemote,
            onPressed: () => _run(
              key: 'pull',
              op: () => ref.read(vaultGitApiProvider).pull(),
              ok: (_) => t.vaultSync.pulledToast,
            ),
          ),
          _actionButton(
            key_: 'runNow',
            icon: Icons.play_arrow,
            label: t.vaultSync.runNow,
            enabled: !blocked,
            onPressed: () => _run(
              key: 'runNow',
              op: () async {
                await ref.read(vaultGitApiProvider).syncRunNow();
                return '';
              },
              ok: (_) => t.vaultSync.runNowToast,
            ),
          ),
        ],
      ),
    );
  }

  Widget _actionButton({
    required String key_,
    required IconData icon,
    required String label,
    required bool enabled,
    required VoidCallback onPressed,
  }) {
    final busy = _busy.contains(key_);
    return FilledButton.tonalIcon(
      onPressed: enabled && !busy ? onPressed : null,
      icon: busy
          ? const SizedBox(
              width: 14,
              height: 14,
              child: CircularProgressIndicator.adaptive(strokeWidth: 2),
            )
          : Icon(icon, size: 18),
      label: Text(label),
    );
  }

  // ── Auto-sync ───────────────────────────────────────────────────

  Widget _autoSyncCard() {
    final theme = Theme.of(context);
    return _card(
      title: t.vaultSync.autoTitle,
      child: _config.when(
        loading: () => const Padding(
          padding: EdgeInsets.symmetric(vertical: 12),
          child: Center(child: CircularProgressIndicator.adaptive()),
        ),
        error: (e, _) => Text(
          e is ApiException ? e.message : '$e',
          style: TextStyle(color: theme.colorScheme.error),
        ),
        data: (c) {
          final enabled = _draftEnabled ?? c.enabled;
          final commitInterval =
              _draftCommitInterval ?? normaliseGoDuration(c.commitInterval);
          final pullInterval = _draftPullInterval ?? normaliseGoDuration(c.pullInterval);
          final pushEnabled = _draftPushEnabled ?? c.pushEnabled;
          final pullEnabled = _draftPullEnabled ?? c.pullEnabled;
          final hasRemote = _status.valueOrNull?.hasRemote ?? false;

          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              SwitchListTile.adaptive(
                contentPadding: EdgeInsets.zero,
                value: enabled,
                // Auto-sync with no remote would commit locally forever
                // and never publish — misleading rather than useful.
                onChanged: !hasRemote && !enabled
                    ? null
                    : (v) => setState(() => _draftEnabled = v),
                title: Text(t.vaultSync.autoEnabled),
                subtitle: !hasRemote
                    ? Text(t.vaultSync.autoNoRemoteHint,
                        style: theme.textTheme.bodySmall)
                    : null,
              ),
              _intervalRow(
                label: t.vaultSync.commitEvery,
                value: commitInterval,
                presets: _commitPresets,
                onChanged: (v) => setState(() => _draftCommitInterval = v),
              ),
              _intervalRow(
                label: t.vaultSync.pullEvery,
                value: pullInterval,
                presets: _pullPresets,
                enabled: pullEnabled,
                onChanged: (v) => setState(() => _draftPullInterval = v),
              ),
              SwitchListTile.adaptive(
                contentPadding: EdgeInsets.zero,
                dense: true,
                value: pushEnabled,
                onChanged: (v) => setState(() => _draftPushEnabled = v),
                title: Text(t.vaultSync.pushAfterCommit),
              ),
              SwitchListTile.adaptive(
                contentPadding: EdgeInsets.zero,
                dense: true,
                value: pullEnabled,
                onChanged: (v) => setState(() => _draftPullEnabled = v),
                title: Text(t.vaultSync.pullPeriodically),
              ),
              const SizedBox(height: 4),
              Row(
                children: [
                  FilledButton(
                    onPressed: _dirty && !_busy.contains('save') ? _save : null,
                    child: _busy.contains('save')
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator.adaptive(
                                strokeWidth: 2),
                          )
                        : Text(t.vaultSync.save),
                  ),
                  if (_dirty) ...[
                    const SizedBox(width: 8),
                    TextButton(
                      onPressed: () => setState(() {
                        _draftEnabled = null;
                        _draftCommitInterval = null;
                        _draftPullInterval = null;
                        _draftPushEnabled = null;
                        _draftPullEnabled = null;
                      }),
                      child: Text(t.vaultSync.discard),
                    ),
                  ],
                ],
              ),
              const Divider(height: 24),
              _timestamp(t.vaultSync.lastCommit, c.lastCommitAt,
                  extra: c.lastCommitHash),
              _timestamp(t.vaultSync.lastPush, c.lastPushAt),
              _timestamp(t.vaultSync.lastPull, c.lastPullAt),
              if (c.lastError.isNotEmpty) ...[
                const SizedBox(height: 8),
                _notice(c.lastError, theme.colorScheme.error),
              ],
            ],
          );
        },
      ),
    );
  }

  Widget _intervalRow({
    required String label,
    required String value,
    required List<String> presets,
    required ValueChanged<String> onChanged,
    bool enabled = true,
  }) {
    // A value the server holds that isn't one of our presets (set from
    // the web's Custom field) must still be selectable, or opening this
    // screen would silently propose changing it.
    final items = presets.contains(value) ? presets : [...presets, value];
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Expanded(child: Text(label)),
          DropdownButton<String>(
            value: value,
            underline: const SizedBox.shrink(),
            onChanged: enabled ? (v) => v == null ? null : onChanged(v) : null,
            items: [
              for (final p in items)
                DropdownMenuItem(value: p, child: Text(_presetLabel(p))),
            ],
          ),
        ],
      ),
    );
  }

  Widget _timestamp(String label, DateTime? at, {String extra = ''}) {
    final theme = Theme.of(context);
    final text = at == null
        ? t.vaultSync.never
        : DateFormat.yMMMd().add_Hm().format(at);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        children: [
          Expanded(
            child: Text(label, style: theme.textTheme.bodySmall),
          ),
          Text(
            extra.isEmpty ? text : '$text · $extra',
            style: theme.textTheme.bodySmall?.copyWith(
              fontFamily: 'monospace',
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }

  // ── Small shared pieces ─────────────────────────────────────────

  Widget _card({required String title, required Widget child}) {
    final theme = Theme.of(context);
    return Card(
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(title.toUpperCase(),
                style: theme.textTheme.labelSmall?.copyWith(
                  letterSpacing: 1.1,
                  color: theme.colorScheme.onSurfaceVariant,
                )),
            const SizedBox(height: 8),
            child,
          ],
        ),
      ),
    );
  }

  Widget _chip(IconData icon, String label, {Color? tone}) {
    final theme = Theme.of(context);
    final color = tone ?? theme.colorScheme.onSurfaceVariant;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        border: Border.all(color: theme.dividerColor),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 13, color: color),
          const SizedBox(width: 5),
          Text(label,
              style: theme.textTheme.bodySmall?.copyWith(color: color)),
        ],
      ),
    );
  }

  Widget _notice(String text, Color color) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.08),
        border: Border.all(color: color.withValues(alpha: 0.35)),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(text,
          style: Theme.of(context).textTheme.bodySmall?.copyWith(color: color)),
    );
  }
}

/// Go prints durations in full ("10m0s", "1h0m0s"); the presets are the
/// short form, and a dropdown value has to match one exactly.
String normaliseGoDuration(String goDuration) {
  // replaceAllMapped semantics, not replaceAll: Dart does NOT expand
  // `$1` in a replacement string — it substitutes those two characters
  // verbatim, which turned every interval into the literal "$1".
  final whole = RegExp(r'^(\d+h)0m0s$|^(\d+m)0s$|^(\d+s)$');
  final m = whole.firstMatch(goDuration);
  if (m == null) return goDuration;
  return m.group(1) ?? m.group(2) ?? m.group(3) ?? goDuration;
}
String _presetLabel(String preset) {
  switch (preset) {
    case '30s':
      return t.vaultSync.every.sec30;
    case '1m':
      return t.vaultSync.every.min1;
    case '5m':
      return t.vaultSync.every.min5;
    case '10m':
      return t.vaultSync.every.min10;
    case '15m':
      return t.vaultSync.every.min15;
    case '30m':
      return t.vaultSync.every.min30;
    case '1h':
      return t.vaultSync.every.hour1;
    case '6h':
      return t.vaultSync.every.hour6;
    case '24h':
      return t.vaultSync.every.hour24;
    default:
      return preset;
  }
}

/// Compact AppBar entry point for the Notes screen: a sync icon
/// carrying the number of uncommitted files, so the state of the vault
/// is visible while writing rather than only after going looking for
/// it. Tapping opens the full sync screen.
class VaultSyncBadge extends ConsumerWidget {
  const VaultSyncBadge({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final status = ref.watch(vaultStatusProvider);
    final s = status.valueOrNull;
    final count = s?.files.length ?? 0;
    final dirty = s != null && s.isRepo && count > 0;

    final button = IconButton(
      icon: const Icon(Icons.sync),
      tooltip: t.vaultSync.title,
      onPressed: () async {
        await Navigator.of(context).push(
          MaterialPageRoute<void>(builder: (_) => const VaultSyncScreen()),
        );
        // The sync screen is where commits and pushes happen, so the
        // badge is stale the moment it closes.
        ref.invalidate(vaultStatusProvider);
      },
    );

    if (!dirty) return button;
    return Badge.count(count: count, child: button);
  }
}
