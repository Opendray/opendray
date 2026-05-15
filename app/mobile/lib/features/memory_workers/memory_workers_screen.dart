import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/memory_workers_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// MemoryWorkersScreen — mobile parity with the web MemoryWorkers
// page. One card per task with worker-kind picker, optional
// provider/account selectors, Enabled toggle, Test button, Save
// button + a 24h metrics rollup.
//
// Reachable from Memory screen AppBar → 🛠 Workers icon (added in
// the same PR).
class MemoryWorkersScreen extends ConsumerStatefulWidget {
  const MemoryWorkersScreen({super.key});

  @override
  ConsumerState<MemoryWorkersScreen> createState() =>
      _MemoryWorkersScreenState();
}

class _MemoryWorkersScreenState extends ConsumerState<MemoryWorkersScreen> {
  late Future<List<WorkerConfig>> _workersFuture;
  late Future<List<CallSummary>> _callsFuture;
  late Future<List<ClaudeAccountSummary>> _accountsFuture;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    setState(() {
      _workersFuture = ref.read(memoryWorkersApiProvider).list();
      _callsFuture =
          ref.read(memoryWorkersApiProvider).calls(limit: 200);
      _accountsFuture = ref.read(claudeAccountsApiProvider).list();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.memoryWorkers.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: _reload,
          ),
        ],
      ),
      body: FutureBuilder<List<WorkerConfig>>(
        future: _workersFuture,
        builder: (context, snap) {
          if (snap.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snap.hasError) {
            return Padding(
              padding: const EdgeInsets.all(16),
              child: Card(
                color: Theme.of(context).colorScheme.errorContainer,
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        t.memoryWorkers.errorTitle,
                        style: const TextStyle(fontWeight: FontWeight.bold),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        t.memoryWorkers.errorDetail,
                        style: const TextStyle(fontSize: 12),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        '${snap.error}',
                        style: const TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 11,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            );
          }
          final workers = snap.data ?? const <WorkerConfig>[];
          return ListView(
            padding: const EdgeInsets.all(12),
            children: [
              Card(
                color: Theme.of(context).colorScheme.surfaceContainerHighest,
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Text(
                    t.memoryWorkers.intro,
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
              ),
              const SizedBox(height: 8),
              for (final w in workers)
                _WorkerCard(
                  config: w,
                  callsFuture: _callsFuture,
                  accountsFuture: _accountsFuture,
                  onChanged: _reload,
                ),
            ],
          );
        },
      ),
    );
  }
}

class _WorkerCard extends ConsumerStatefulWidget {
  const _WorkerCard({
    required this.config,
    required this.callsFuture,
    required this.accountsFuture,
    required this.onChanged,
  });

  final WorkerConfig config;
  final Future<List<CallSummary>> callsFuture;
  final Future<List<ClaudeAccountSummary>> accountsFuture;
  final VoidCallback onChanged;

  @override
  ConsumerState<_WorkerCard> createState() => _WorkerCardState();
}

class _WorkerCardState extends ConsumerState<_WorkerCard> {
  late WorkerKind _kind;
  late String _summarizerId;
  late String _providerId;
  late String _accountId;
  late bool _enabled;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _kind = widget.config.kind;
    _summarizerId = widget.config.summarizerId ?? '';
    _providerId = widget.config.providerId ?? '';
    _accountId = widget.config.accountId ?? '';
    _enabled = widget.config.enabled;
  }

  bool get _dirty =>
      _kind != widget.config.kind ||
      _summarizerId != (widget.config.summarizerId ?? '') ||
      _providerId != (widget.config.providerId ?? '') ||
      _accountId != (widget.config.accountId ?? '') ||
      _enabled != widget.config.enabled;

  Future<void> _save() async {
    setState(() => _busy = true);
    try {
      await ref.read(memoryWorkersApiProvider).upsert(
            task: widget.config.task,
            kind: _kind,
            summarizerId: _kind == WorkerKind.summarizer ? _summarizerId : '',
            providerId: _kind == WorkerKind.agent ? _providerId : '',
            accountId: _kind == WorkerKind.agent && _providerId == 'claude'
                ? _accountId
                : '',
            enabled: _enabled,
          );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(t.memoryWorkers
              .savedSnack(label: _taskLabel(widget.config.task))),
        ),
      );
      widget.onChanged();
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              t.memoryWorkers.saveFailed(error: e.toString()),
            ),
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _test() async {
    setState(() => _busy = true);
    try {
      final res =
          await ref.read(memoryWorkersApiProvider).test(widget.config.task);
      if (!mounted) return;
      if (res.ok) {
        final base = t.memoryWorkers.testOkSnack(
          label: _taskLabel(widget.config.task),
          duration: res.durationMs.toString(),
        );
        final preview = (res.preview != null && res.preview!.isNotEmpty)
            ? '\n${_truncate(res.preview!, 200)}'
            : '';
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('$base$preview'),
            duration: const Duration(seconds: 6),
          ),
        );
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              t.memoryWorkers.testFailedReturnedSnack(
                label: _taskLabel(widget.config.task),
                error: res.error ?? t.memoryWorkers.unknownError,
              ),
            ),
          ),
        );
      }
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              t.memoryWorkers.testFailed(error: e.toString()),
            ),
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  String _taskLabel(WorkerTaskKind k) => switch (k) {
        WorkerTaskKind.gatekeeper => t.memoryWorkers.tasks.gatekeeper.label,
        WorkerTaskKind.cleaner => t.memoryWorkers.tasks.cleaner.label,
        WorkerTaskKind.gitactivity => t.memoryWorkers.tasks.gitactivity.label,
        WorkerTaskKind.transcript => t.memoryWorkers.tasks.transcript.label,
        WorkerTaskKind.planDrift => t.memoryWorkers.tasks.planDrift.label,
        WorkerTaskKind.conflictDetector =>
          t.memoryWorkers.tasks.conflictDetector.label,
        WorkerTaskKind.capture => t.memoryWorkers.tasks.capture.label,
      };

  String _taskDescription(WorkerTaskKind k) => switch (k) {
        WorkerTaskKind.gatekeeper =>
          t.memoryWorkers.tasks.gatekeeper.description,
        WorkerTaskKind.cleaner => t.memoryWorkers.tasks.cleaner.description,
        WorkerTaskKind.gitactivity =>
          t.memoryWorkers.tasks.gitactivity.description,
        WorkerTaskKind.transcript =>
          t.memoryWorkers.tasks.transcript.description,
        WorkerTaskKind.planDrift =>
          t.memoryWorkers.tasks.planDrift.description,
        WorkerTaskKind.conflictDetector =>
          t.memoryWorkers.tasks.conflictDetector.description,
        WorkerTaskKind.capture => t.memoryWorkers.tasks.capture.description,
      };

  @override
  Widget build(BuildContext context) {
    final agentAllowed = widget.config.task.agentSupported;
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    _taskLabel(widget.config.task),
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                  ),
                ),
                if (!agentAllowed)
                  Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                      color: Theme.of(context).colorScheme.tertiaryContainer,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(
                      t.memoryWorkers.summarizerOnlyBadge,
                      style: TextStyle(
                        fontSize: 9,
                        color:
                            Theme.of(context).colorScheme.onTertiaryContainer,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              _taskDescription(widget.config.task),
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context).hintColor,
                  ),
            ),
            const SizedBox(height: 12),
            _kindSelector(agentAllowed),
            if (_kind == WorkerKind.summarizer) ...[
              const SizedBox(height: 8),
              _summarizerSelector(),
            ],
            if (_kind == WorkerKind.agent) ...[
              const SizedBox(height: 8),
              _providerSelector(),
              if (_providerId == 'claude') ...[
                const SizedBox(height: 8),
                _accountSelector(),
              ],
              const SizedBox(height: 8),
              _agentWarning(),
            ],
            const SizedBox(height: 12),
            Row(
              children: [
                Checkbox(
                  value: _enabled,
                  onChanged: (v) =>
                      setState(() => _enabled = v ?? true),
                ),
                Text(t.common.enabled, style: const TextStyle(fontSize: 12)),
                const Spacer(),
                OutlinedButton.icon(
                  onPressed: _busy ? null : _test,
                  icon: const Icon(Icons.play_arrow, size: 14),
                  label: Text(t.memoryWorkers.test),
                ),
                const SizedBox(width: 8),
                FilledButton.icon(
                  onPressed: (_busy || !_dirty) ? null : _save,
                  icon: const Icon(Icons.save, size: 14),
                  label: Text(t.common.save),
                ),
              ],
            ),
            const SizedBox(height: 8),
            _metricsRollup(),
          ],
        ),
      ),
    );
  }

  Widget _kindSelector(bool agentAllowed) {
    return DropdownButtonFormField<WorkerKind>(
      initialValue: _kind,
      decoration: InputDecoration(
        labelText: t.memoryWorkers.workerLabel,
        border: const OutlineInputBorder(),
        isDense: true,
      ),
      items: [
        DropdownMenuItem(
          value: WorkerKind.summarizer,
          child: Text(t.memoryWorkers.summarizerHttp),
        ),
        if (agentAllowed)
          DropdownMenuItem(
            value: WorkerKind.agent,
            child: Text(t.memoryWorkers.agentCliPrint),
          ),
      ],
      onChanged: agentAllowed ? (v) => setState(() => _kind = v!) : null,
    );
  }

  Widget _summarizerSelector() {
    // Mobile keeps it simple: always uses the registry default
    // summarizer provider. Operator who wants to pin a specific
    // row uses the web UI. Display informational text instead of
    // a dropdown to avoid the round-trip to fetch provider rows
    // that mobile doesn't otherwise need.
    return Container(
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHigh,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(
        children: [
          const Icon(Icons.info_outline, size: 14),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              t.memoryWorkers.summarizerInfo,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        ],
      ),
    );
  }

  Widget _providerSelector() {
    return DropdownButtonFormField<String>(
      initialValue: _providerId.isEmpty ? null : _providerId,
      decoration: InputDecoration(
        labelText: t.memoryWorkers.cliLabel,
        border: const OutlineInputBorder(),
        isDense: true,
      ),
      items: [
        DropdownMenuItem(
          value: 'claude',
          child: Text(t.memoryWorkers.cliClaude),
        ),
        DropdownMenuItem(
          value: 'gemini',
          child: Text(t.memoryWorkers.cliGemini),
        ),
      ],
      onChanged: (v) => setState(() => _providerId = v ?? ''),
    );
  }

  Widget _accountSelector() {
    return FutureBuilder<List<ClaudeAccountSummary>>(
      future: widget.accountsFuture,
      builder: (_, snap) {
        final accounts = snap.data ?? const <ClaudeAccountSummary>[];
        return DropdownButtonFormField<String>(
          initialValue: _accountId.isEmpty ? '__default__' : _accountId,
          decoration: InputDecoration(
            labelText: t.memoryWorkers.claudeAccountLabel,
            border: const OutlineInputBorder(),
            isDense: true,
          ),
          items: [
            DropdownMenuItem(
              value: '__default__',
              child: Text(t.memoryWorkers.claudeAccountDefault),
            ),
            for (final a in accounts)
              DropdownMenuItem(
                value: a.id,
                child: Text(a.displayName),
              ),
          ],
          onChanged: (v) => setState(
              () => _accountId = v == '__default__' ? '' : (v ?? '')),
        );
      },
    );
  }

  Widget _agentWarning() {
    return Container(
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHigh,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.warning_amber_rounded, size: 14),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              t.memoryWorkers.agentWarning,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        ],
      ),
    );
  }

  Widget _metricsRollup() {
    return FutureBuilder<List<CallSummary>>(
      future: widget.callsFuture,
      builder: (_, snap) {
        final all = snap.data ?? const <CallSummary>[];
        final cutoff =
            DateTime.now().subtract(const Duration(hours: 24));
        final recent = all
            .where((c) =>
                c.task == widget.config.task &&
                c.startedAt.isAfter(cutoff))
            .toList();
        if (recent.isEmpty) {
          return Text(
            t.memoryWorkers.noCalls24h,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: Theme.of(context).hintColor,
                  fontSize: 11,
                ),
          );
        }
        final avgMs =
            recent.fold<int>(0, (sum, c) => sum + c.durationMs) ~/
                recent.length;
        final errors = recent.where((c) => !c.success).length;
        return Text(
          '${recent.length} calls · avg ${avgMs}ms'
          '${errors > 0 ? " · $errors errors" : ""}',
          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color: errors > 0
                    ? Theme.of(context).colorScheme.error
                    : Theme.of(context).hintColor,
                fontSize: 11,
              ),
        );
      },
    );
  }

  String _truncate(String s, int n) =>
      s.length > n ? '${s.substring(0, n)}…' : s;
}
