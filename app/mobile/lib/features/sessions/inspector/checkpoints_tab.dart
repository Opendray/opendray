import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/checkpoints_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Checkpoints surface inside the session inspector — the mobile mirror of
// the web CheckpointsPanel. Lists a session's context checkpoints
// (uncommitted git diff + untracked files + input history), can capture one
// on demand, and per checkpoint can view the stored diff or restore it back
// onto the cwd under the gateway's strict guards (a 409 guard failure shows
// its reason verbatim).
class CheckpointsTab extends ConsumerStatefulWidget {
  const CheckpointsTab({required this.sessionId, super.key});

  final String sessionId;

  @override
  ConsumerState<CheckpointsTab> createState() => _CheckpointsTabState();
}

class _CheckpointsTabState extends ConsumerState<CheckpointsTab>
    with AutomaticKeepAliveClientMixin {
  AsyncValue<List<Checkpoint>> _state = const AsyncValue.loading();
  bool _capturing = false;

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final res = await ref.read(checkpointsApiProvider).list(widget.sessionId);
      if (!mounted) return;
      setState(() => _state = AsyncValue.data(res));
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _capture() async {
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _capturing = true);
    try {
      final cp = await ref.read(checkpointsApiProvider).capture(widget.sessionId);
      final clean = cp.isGit && cp.diffBytes == 0 && cp.untrackedFiles == 0;
      messenger.showSnackBar(
        SnackBar(
          behavior: SnackBarBehavior.floating,
          content: Text(
            !cp.isGit
                ? t.sessions.inspector.checkpoints.capturedNonGit
                : clean
                    ? t.sessions.inspector.checkpoints.capturedClean
                    : t.sessions.inspector.checkpoints.capturedGit(
                        diff: cp.diffBytes,
                        files: cp.untrackedFiles,
                      ),
          ),
        ),
      );
      await _load();
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(behavior: SnackBarBehavior.floating, content: Text(e.message)),
      );
    } finally {
      if (mounted) setState(() => _capturing = false);
    }
  }

  Future<void> _delete(Checkpoint cp) async {
    final messenger = ScaffoldMessenger.of(context);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.sessions.inspector.checkpoints.deleteTitle),
        content: Text(t.sessions.inspector.checkpoints.deleteConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.sessions.inspector.checkpoints.delete),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ref.read(checkpointsApiProvider).delete(cp.id);
      await _load();
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(behavior: SnackBarBehavior.floating, content: Text(e.message)),
      );
    }
  }

  Future<void> _restore(Checkpoint cp) async {
    final messenger = ScaffoldMessenger.of(context);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.sessions.inspector.checkpoints.restoreTitle),
        content: Text(t.sessions.inspector.checkpoints.restoreWarn),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.sessions.inspector.checkpoints.restoreConfirm),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      final res = await ref.read(checkpointsApiProvider).restore(cp.id);
      messenger.showSnackBar(
        SnackBar(
          behavior: SnackBarBehavior.floating,
          content: Text(
            t.sessions.inspector.checkpoints.restored(
              files: res.untrackedRestored,
              skipped: res.untrackedSkipped.length,
            ),
          ),
        ),
      );
    } on ApiException catch (e) {
      // Guard failures (409) carry their reason in the message.
      messenger.showSnackBar(
        SnackBar(behavior: SnackBarBehavior.floating, content: Text(e.message)),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  t.sessions.inspector.checkpoints.blurb,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ),
              const SizedBox(width: 8),
              FilledButton.tonalIcon(
                onPressed: _capturing ? null : _capture,
                icon: _capturing
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.camera_alt_outlined, size: 18),
                label: Text(t.sessions.inspector.checkpoints.capture),
              ),
            ],
          ),
        ),
        Expanded(
          child: _state.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => _ErrorView(
              message: e is ApiException ? e.message : e.toString(),
              onRetry: _load,
            ),
            data: (list) => list.isEmpty
                ? _EmptyView(onRefresh: _load)
                : RefreshIndicator(
                    onRefresh: _load,
                    child: ListView.separated(
                      padding: const EdgeInsets.fromLTRB(12, 4, 12, 24),
                      itemCount: list.length,
                      separatorBuilder: (_, __) => const SizedBox(height: 8),
                      itemBuilder: (_, i) => _CheckpointCard(
                        cp: list[i],
                        onRestore: () => _restore(list[i]),
                        onDelete: () => _delete(list[i]),
                        loadDiff: () =>
                            ref.read(checkpointsApiProvider).diff(list[i].id),
                      ),
                    ),
                  ),
          ),
        ),
      ],
    );
  }
}

class _CheckpointCard extends StatefulWidget {
  const _CheckpointCard({
    required this.cp,
    required this.onRestore,
    required this.onDelete,
    required this.loadDiff,
  });

  final Checkpoint cp;
  final VoidCallback onRestore;
  final VoidCallback onDelete;
  final Future<String> Function() loadDiff;

  @override
  State<_CheckpointCard> createState() => _CheckpointCardState();
}

class _CheckpointCardState extends State<_CheckpointCard> {
  bool _showDiff = false;
  AsyncValue<String>? _diff;

  Future<void> _toggleDiff() async {
    if (_showDiff) {
      setState(() => _showDiff = false);
      return;
    }
    setState(() {
      _showDiff = true;
      _diff = const AsyncValue.loading();
    });
    try {
      final text = await widget.loadDiff();
      if (mounted) setState(() => _diff = AsyncValue.data(text));
    } on Object catch (e, st) {
      if (mounted) setState(() => _diff = AsyncValue.error(e, st));
    }
  }

  @override
  Widget build(BuildContext context) {
    final cp = widget.cp;
    final theme = Theme.of(context);
    final muted = theme.colorScheme.onSurface.withValues(alpha: 0.6);
    final isManual = cp.trigger == 'manual';
    return Card(
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 10, 8, 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: (isManual ? theme.colorScheme.primary : Colors.amber)
                        .withValues(alpha: 0.15),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    isManual
                        ? t.sessions.inspector.checkpoints.triggerManual
                        : t.sessions.inspector.checkpoints.triggerInterrupted,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color:
                          isManual ? theme.colorScheme.primary : Colors.amber,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    DateFormat.yMMMd().add_Hm().format(cp.createdAt.toLocal()),
                    style: theme.textTheme.bodySmall,
                  ),
                ),
                if (cp.truncated)
                  Tooltip(
                    message: t.sessions.inspector.checkpoints.truncatedHint,
                    child: const Icon(Icons.warning_amber_rounded,
                        size: 16, color: Colors.amber),
                  ),
              ],
            ),
            const SizedBox(height: 6),
            Builder(
              builder: (_) {
                final head = cp.gitHead != null
                    ? cp.gitHead!.substring(
                        0, cp.gitHead!.length < 8 ? cp.gitHead!.length : 8)
                    : '—';
                // A git checkpoint with no diff and no untracked files
                // captured nothing restorable — the tree was clean. Say so
                // explicitly so an empty diff doesn't read as broken.
                final clean =
                    cp.isGit && cp.diffBytes == 0 && cp.untrackedFiles == 0;
                if (clean) {
                  return Row(
                    children: [
                      Icon(Icons.check_circle_outline,
                          size: 14, color: Colors.green.shade600),
                      const SizedBox(width: 4),
                      Expanded(
                        child: Text(
                          '$head · ${t.sessions.inspector.checkpoints.clean}   ⌨ ${cp.inputBytes}B',
                          style: theme.textTheme.bodySmall?.copyWith(
                              fontFamily: 'monospace', color: muted),
                        ),
                      ),
                    ],
                  );
                }
                return Text(
                  cp.isGit
                      ? '$head   Δ ${cp.diffBytes}B   +${cp.untrackedFiles}f   ⌨ ${cp.inputBytes}B'
                      : '${t.sessions.inspector.checkpoints.nonGit}   ⌨ ${cp.inputBytes}B',
                  style: theme.textTheme.bodySmall
                      ?.copyWith(fontFamily: 'monospace', color: muted),
                );
              },
            ),
            if (cp.note != null) ...[
              const SizedBox(height: 4),
              Text(
                cp.note!.split('\n').first,
                style: theme.textTheme.bodySmall
                    ?.copyWith(fontStyle: FontStyle.italic, color: muted),
              ),
            ],
            Row(
              children: [
                if (cp.diffBytes > 0)
                  TextButton.icon(
                    onPressed: _toggleDiff,
                    icon: const Icon(Icons.difference_outlined, size: 16),
                    label: Text(
                      _showDiff
                          ? t.sessions.inspector.checkpoints.hideDiff
                          : t.sessions.inspector.checkpoints.viewDiff,
                    ),
                  ),
                if (cp.isGit)
                  TextButton.icon(
                    onPressed: widget.onRestore,
                    icon: const Icon(Icons.restore, size: 16),
                    label: Text(t.sessions.inspector.checkpoints.restore),
                  ),
                const Spacer(),
                IconButton(
                  onPressed: widget.onDelete,
                  icon: Icon(Icons.delete_outline,
                      size: 18, color: theme.colorScheme.error),
                  tooltip: t.sessions.inspector.checkpoints.delete,
                ),
              ],
            ),
            if (_showDiff && cp.diffBytes > 0)
              Container(
                width: double.infinity,
                constraints: const BoxConstraints(maxHeight: 260),
                decoration: BoxDecoration(
                  color: theme.colorScheme.surfaceContainerHighest
                      .withValues(alpha: 0.4),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: (_diff ?? const AsyncValue<String>.loading()).when(
                  loading: () => const Padding(
                    padding: EdgeInsets.all(12),
                    child: Center(
                      child: SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      ),
                    ),
                  ),
                  error: (e, _) => Padding(
                    padding: const EdgeInsets.all(8),
                    child: Text(
                      e is ApiException ? e.message : e.toString(),
                      style: TextStyle(color: theme.colorScheme.error),
                    ),
                  ),
                  data: (text) => SingleChildScrollView(
                    padding: const EdgeInsets.all(8),
                    child: _DiffText(text: text),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

// _DiffText renders a unified diff with per-line color coding, prefix-only.
class _DiffText extends StatelessWidget {
  const _DiffText({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    const maxLines = 2000;
    final all = text.split('\n');
    final lines = all.length > maxLines ? all.sublist(0, maxLines) : all;
    final theme = Theme.of(context);
    Color? colorFor(String line) {
      if (line.startsWith('+++') || line.startsWith('---')) return null;
      if (line.startsWith('@@')) return Colors.blue;
      if (line.startsWith('+')) return Colors.green.shade600;
      if (line.startsWith('-')) return theme.colorScheme.error;
      return null;
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final line in lines)
          Text(
            line.isEmpty ? ' ' : line,
            softWrap: false,
            style: TextStyle(
              fontFamily: 'monospace',
              fontSize: 11,
              height: 1.4,
              color: colorFor(line),
            ),
          ),
        if (all.length > maxLines)
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: Text(
              '… ${all.length - maxLines} more lines',
              style: theme.textTheme.labelSmall,
            ),
          ),
      ],
    );
  }
}

class _EmptyView extends StatelessWidget {
  const _EmptyView({required this.onRefresh});
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView(
        children: [
          const SizedBox(height: 80),
          Icon(Icons.archive_outlined,
              size: 48,
              color:
                  Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4)),
          const SizedBox(height: 12),
          Center(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 32),
              child: Text(
                t.sessions.inspector.checkpoints.empty,
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.message, required this.onRetry});
  final String message;
  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              message,
              textAlign: TextAlign.center,
              style: Theme.of(context)
                  .textTheme
                  .bodySmall
                  ?.copyWith(color: Theme.of(context).colorScheme.error),
            ),
            const SizedBox(height: 12),
            OutlinedButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
