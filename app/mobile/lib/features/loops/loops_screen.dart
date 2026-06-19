import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/loops_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Loops tab — the mobile mirror of the web Loops panel. Lists autonomous
// agent loops, lets the operator create one, and pause/resume/stop a live
// loop. Phase 1 has no per-loop WS, so the list polls every 4s (same cadence
// as the web panel) on top of pull-to-refresh.
class LoopsScreen extends ConsumerStatefulWidget {
  const LoopsScreen({super.key});

  @override
  ConsumerState<LoopsScreen> createState() => _LoopsScreenState();
}

class _LoopsScreenState extends ConsumerState<LoopsScreen> {
  Timer? _poll;

  @override
  void initState() {
    super.initState();
    _poll = Timer.periodic(const Duration(seconds: 4), (_) {
      if (mounted) ref.invalidate(loopsListProvider);
    });
  }

  @override
  void dispose() {
    _poll?.cancel();
    super.dispose();
  }

  Future<void> _act(
    Future<LoopSummary> Function() action,
  ) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      await action();
      ref.invalidate(loopsListProvider);
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(e.toString()),
          behavior: SnackBarBehavior.floating,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(loopsListProvider);
    return Scaffold(
      appBar: AppBar(title: Text(t.loops.title)),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _openCreate,
        icon: const Icon(Icons.add),
        label: Text(t.loops.create),
      ),
      body: RefreshIndicator(
        onRefresh: () => ref.refresh(loopsListProvider.future),
        child: async.when(
          data: (loops) {
            if (loops.isEmpty) {
              return ListView(
                children: [
                  const SizedBox(height: 120),
                  Center(
                    child: Padding(
                      padding: const EdgeInsets.all(32),
                      child: Text(
                        t.loops.empty,
                        textAlign: TextAlign.center,
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                              color: Theme.of(context).colorScheme.outline,
                            ),
                      ),
                    ),
                  ),
                ],
              );
            }
            return ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: loops.length,
              itemBuilder: (_, i) => _LoopCard(
                loop: loops[i],
                onPause: () =>
                    _act(() => ref.read(loopsApiProvider).pause(loops[i].id)),
                onResume: () =>
                    _act(() => ref.read(loopsApiProvider).resume(loops[i].id)),
                onStop: () =>
                    _act(() => ref.read(loopsApiProvider).stop(loops[i].id)),
                onDetails: () => _openRuns(loops[i]),
              ),
            );
          },
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) => _ErrorView(
            error: e.toString(),
            onRetry: () => ref.invalidate(loopsListProvider),
          ),
        ),
      ),
    );
  }

  Future<void> _openCreate() async {
    final created = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (_) => const _CreateLoopSheet(),
    );
    if (created ?? false) {
      ref.invalidate(loopsListProvider);
    }
  }

  Future<void> _openRuns(LoopSummary loop) async {
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (_) => _RunsSheet(loopId: loop.id),
    );
  }
}

Color _statusColor(BuildContext context, LoopStatus s) {
  final scheme = Theme.of(context).colorScheme;
  return switch (s) {
    LoopStatus.running => Colors.green,
    LoopStatus.paused => Colors.amber,
    LoopStatus.done => Colors.blue,
    LoopStatus.failed => scheme.error,
    LoopStatus.escalated => Colors.orange,
    _ => scheme.outline,
  };
}

String _statusLabel(LoopStatus s) => switch (s) {
      LoopStatus.pending => t.loops.status.pending,
      LoopStatus.running => t.loops.status.running,
      LoopStatus.paused => t.loops.status.paused,
      LoopStatus.done => t.loops.status.done,
      LoopStatus.stopped => t.loops.status.stopped,
      LoopStatus.failed => t.loops.status.failed,
      LoopStatus.escalated => t.loops.status.escalated,
      LoopStatus.unknown => s.name,
    };

class _LoopCard extends StatelessWidget {
  const _LoopCard({
    required this.loop,
    required this.onPause,
    required this.onResume,
    required this.onStop,
    required this.onDetails,
  });

  final LoopSummary loop;
  final VoidCallback onPause;
  final VoidCallback onResume;
  final VoidCallback onStop;
  final VoidCallback onDetails;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final kindLabel =
        loop.kind == LoopKind.goal ? t.loops.kind.goal : t.loops.kind.interval;
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  loop.kind == LoopKind.goal ? Icons.flag_outlined : Icons.repeat,
                  size: 18,
                  color: theme.colorScheme.outline,
                ),
                const SizedBox(width: 6),
                Text(kindLabel, style: theme.textTheme.titleSmall),
                const SizedBox(width: 8),
                _Chip(
                  label: _statusLabel(loop.status),
                  color: _statusColor(context, loop.status),
                ),
                const Spacer(),
                Text(
                  t.loops.iterationOf(
                    n: loop.iteration.toString(),
                    max: loop.maxIterations.toString(),
                  ),
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.outline),
                ),
              ],
            ),
            const SizedBox(height: 6),
            Text(
              loop.sessionId,
              style: theme.textTheme.bodySmall?.copyWith(
                fontFamily: 'monospace',
                color: theme.colorScheme.outline,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            if (loop.goal != null && loop.goal!.isNotEmpty) ...[
              const SizedBox(height: 4),
              Text(loop.goal!, maxLines: 2, overflow: TextOverflow.ellipsis),
            ],
            if (loop.lastReason != null && loop.lastReason!.isNotEmpty) ...[
              const SizedBox(height: 4),
              Text(
                loop.lastVerdict != null && loop.lastVerdict!.isNotEmpty
                    ? '${loop.lastVerdict} — ${loop.lastReason}'
                    : loop.lastReason!,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.outline),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
            ],
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              children: [
                if (loop.status == LoopStatus.running)
                  TextButton.icon(
                    onPressed: onPause,
                    icon: const Icon(Icons.pause, size: 18),
                    label: Text(t.loops.action.pause),
                  ),
                if (loop.status == LoopStatus.paused)
                  TextButton.icon(
                    onPressed: onResume,
                    icon: const Icon(Icons.play_arrow, size: 18),
                    label: Text(t.loops.action.resume),
                  ),
                if (!loop.status.isTerminal)
                  TextButton.icon(
                    onPressed: onStop,
                    icon: const Icon(Icons.stop, size: 18),
                    label: Text(t.loops.action.stop),
                  ),
                TextButton.icon(
                  onPressed: onDetails,
                  icon: const Icon(Icons.list_alt, size: 18),
                  label: Text(t.loops.action.details),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _Chip extends StatelessWidget {
  const _Chip({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        label,
        style: TextStyle(fontSize: 11, color: color, fontWeight: FontWeight.w600),
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
    return ListView(
      children: [
        const SizedBox(height: 120),
        Center(
          child: Column(
            children: [
              Text(error, textAlign: TextAlign.center),
              const SizedBox(height: 12),
              OutlinedButton(onPressed: onRetry, child: Text(t.common.retry)),
            ],
          ),
        ),
      ],
    );
  }
}

class _CreateLoopSheet extends ConsumerStatefulWidget {
  const _CreateLoopSheet();

  @override
  ConsumerState<_CreateLoopSheet> createState() => _CreateLoopSheetState();
}

class _CreateLoopSheetState extends ConsumerState<_CreateLoopSheet> {
  final _session = TextEditingController();
  final _prompt = TextEditingController();
  final _goal = TextEditingController();
  final _interval = TextEditingController(text: '60');
  final _maxIter = TextEditingController(text: '20');
  final _duration = TextEditingController(text: '60');
  final _failCap = TextEditingController(text: '3');
  LoopKind _kind = LoopKind.goal;
  bool _busy = false;

  @override
  void dispose() {
    _session.dispose();
    _prompt.dispose();
    _goal.dispose();
    _interval.dispose();
    _maxIter.dispose();
    _duration.dispose();
    _failCap.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    final minutes = int.tryParse(_duration.text) ?? 60;
    final req = CreateLoopRequest(
      sessionId: _session.text.trim(),
      kind: _kind,
      prompt: _prompt.text.trim(),
      maxIterations: int.tryParse(_maxIter.text) ?? 20,
      deadlineAt: DateTime.now().add(Duration(minutes: minutes < 1 ? 1 : minutes)),
      failureCap: int.tryParse(_failCap.text) ?? 3,
      goal: _kind == LoopKind.goal ? _goal.text.trim() : null,
      intervalSeconds:
          _kind == LoopKind.interval ? int.tryParse(_interval.text) ?? 60 : null,
    );
    setState(() => _busy = true);
    try {
      await ref.read(loopsApiProvider).create(req);
      navigator.pop(true);
    } on Object catch (e) {
      if (mounted) setState(() => _busy = false);
      messenger.showSnackBar(
        SnackBar(
          content: Text(e.toString()),
          behavior: SnackBarBehavior.floating,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final insets = MediaQuery.of(context).viewInsets.bottom;
    return Padding(
      padding: EdgeInsets.fromLTRB(16, 16, 16, insets + 16),
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(t.loops.create, style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 12),
            SegmentedButton<LoopKind>(
              segments: [
                ButtonSegment(value: LoopKind.goal, label: Text(t.loops.kind.goal)),
                ButtonSegment(
                  value: LoopKind.interval,
                  label: Text(t.loops.kind.interval),
                ),
              ],
              selected: {_kind},
              onSelectionChanged: (s) => setState(() => _kind = s.first),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _session,
              decoration: InputDecoration(
                labelText: t.loops.field.session,
                hintText: t.loops.field.sessionPlaceholder,
              ),
            ),
            if (_kind == LoopKind.goal) ...[
              const SizedBox(height: 12),
              TextField(
                controller: _goal,
                minLines: 2,
                maxLines: 3,
                decoration: InputDecoration(
                  labelText: t.loops.field.goal,
                  hintText: t.loops.field.goalPlaceholder,
                ),
              ),
            ],
            const SizedBox(height: 12),
            TextField(
              controller: _prompt,
              minLines: 2,
              maxLines: 3,
              decoration: InputDecoration(
                labelText: _kind == LoopKind.goal
                    ? t.loops.field.seedPrompt
                    : t.loops.field.prompt,
                hintText: t.loops.field.promptPlaceholder,
              ),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                if (_kind == LoopKind.interval) ...[
                  Expanded(
                    child: TextField(
                      controller: _interval,
                      keyboardType: TextInputType.number,
                      decoration: InputDecoration(
                        labelText: t.loops.field.intervalSeconds,
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                ],
                Expanded(
                  child: TextField(
                    controller: _maxIter,
                    keyboardType: TextInputType.number,
                    decoration: InputDecoration(
                      labelText: t.loops.field.maxIterations,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _duration,
                    keyboardType: TextInputType.number,
                    decoration: InputDecoration(
                      labelText: t.loops.field.durationMinutes,
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: TextField(
                    controller: _failCap,
                    keyboardType: TextInputType.number,
                    decoration: InputDecoration(
                      labelText: t.loops.field.failureCap,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 20),
            FilledButton(
              onPressed: _busy ? null : _submit,
              child: _busy
                  ? const SizedBox(
                      height: 18,
                      width: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(t.loops.action.create),
            ),
          ],
        ),
      ),
    );
  }
}

class _RunsSheet extends ConsumerWidget {
  const _RunsSheet({required this.loopId});

  final String loopId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(loopRunsProvider(loopId));
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(t.loops.runs.title, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 12),
          async.when(
            data: (runs) {
              if (runs.isEmpty) {
                return Padding(
                  padding: const EdgeInsets.symmetric(vertical: 24),
                  child: Center(child: Text(t.loops.runs.empty)),
                );
              }
              return ConstrainedBox(
                constraints: BoxConstraints(
                  maxHeight: MediaQuery.of(context).size.height * 0.6,
                ),
                child: ListView.builder(
                  shrinkWrap: true,
                  itemCount: runs.length,
                  itemBuilder: (_, i) {
                    final run = runs[i];
                    return ListTile(
                      dense: true,
                      title: Text(t.loops.runs.iteration(n: run.iteration.toString())),
                      subtitle: run.reason != null && run.reason!.isNotEmpty
                          ? Text(run.reason!)
                          : null,
                      trailing: run.verdict != null
                          ? Text(
                              run.verdict!,
                              style: Theme.of(context).textTheme.bodySmall,
                            )
                          : null,
                    );
                  },
                ),
              );
            },
            loading: () => const Padding(
              padding: EdgeInsets.symmetric(vertical: 24),
              child: Center(child: CircularProgressIndicator()),
            ),
            error: (e, _) => Padding(
              padding: const EdgeInsets.symmetric(vertical: 24),
              child: Center(child: Text(e.toString())),
            ),
          ),
        ],
      ),
    );
  }
}
