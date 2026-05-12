import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/memory_cleanup_api.dart';
import 'package:path/path.dart' as p;

// CleanupInboxScreen surfaces pending memory_cleanup_decisions
// across every project in one list. Reachable from More → Memory
// → Cleanup so operators don't have to drill into a specific
// Project tab to find what the librarian has proposed.
//
// The per-project view (Project → Cleanup tab) still works for
// focused review, but this inbox is the "everything I owe a
// decision on" entry point — same pattern as the M9 proposal
// inbox.
class CleanupInboxScreen extends ConsumerStatefulWidget {
  const CleanupInboxScreen({super.key});

  @override
  ConsumerState<CleanupInboxScreen> createState() =>
      _CleanupInboxScreenState();
}

class _CleanupInboxScreenState extends ConsumerState<CleanupInboxScreen> {
  AsyncValue<List<CleanupDecision>> _decisions = const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _decisions = const AsyncValue.loading());
    try {
      final list = await ref
          .read(memoryCleanupApiProvider)
          .list(status: 'pending', limit: 200);
      if (!mounted) return;
      setState(() => _decisions = AsyncValue.data(list));
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _decisions = AsyncValue.error(e, StackTrace.current));
    }
  }

  Future<void> _approve(String id) async {
    try {
      await ref.read(memoryCleanupApiProvider).approve(id);
      if (mounted) await _load();
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Approve failed: $e')),
        );
        // Stale UI — server already decided this row out from under
        // us (CLI / web / another phone). Re-pull so the user sees
        // the real state.
        await _load();
      }
    }
  }

  Future<void> _reject(String id) async {
    try {
      await ref.read(memoryCleanupApiProvider).reject(id);
      if (mounted) await _load();
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Reject failed: $e')),
        );
        await _load();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Memory cleanup'),
        actions: [
          IconButton(
            tooltip: 'Refresh',
            icon: const Icon(Icons.refresh),
            onPressed: _load,
          ),
        ],
      ),
      body: _decisions.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text('Failed to load: $e'),
          ),
        ),
        data: (rows) {
          if (rows.isEmpty) {
            return RefreshIndicator(
              onRefresh: _load,
              child: ListView(
                children: [
                  const SizedBox(height: 80),
                  Padding(
                    padding: const EdgeInsets.all(24),
                    child: Column(
                      children: [
                        Icon(
                          Icons.cleaning_services_outlined,
                          size: 48,
                          color: Theme.of(context)
                              .colorScheme
                              .onSurface
                              .withValues(alpha: 0.4),
                        ),
                        const SizedBox(height: 16),
                        Text(
                          'No pending cleanup decisions across any project.',
                          textAlign: TextAlign.center,
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        Text(
                          'Open a specific project from More → Project '
                          'to run cleanup on its memories, or wait for '
                          'the scheduler to fire automatically.',
                          textAlign: TextAlign.center,
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            );
          }
          // Group by project scope_key for readability.
          final grouped = <String, List<CleanupDecision>>{};
          for (final d in rows) {
            grouped.putIfAbsent(d.memoryScopeKey, () => []).add(d);
          }
          final keys = grouped.keys.toList()..sort();
          return RefreshIndicator(
            onRefresh: _load,
            child: ListView.builder(
              padding: const EdgeInsets.fromLTRB(8, 8, 8, 16),
              itemCount: keys.length,
              itemBuilder: (_, i) {
                final k = keys[i];
                final decisions = grouped[k]!;
                return _ProjectGroup(
                  scopeKey: k,
                  decisions: decisions,
                  onApprove: _approve,
                  onReject: _reject,
                );
              },
            ),
          );
        },
      ),
    );
  }
}

class _ProjectGroup extends StatelessWidget {
  const _ProjectGroup({
    required this.scopeKey,
    required this.decisions,
    required this.onApprove,
    required this.onReject,
  });

  final String scopeKey;
  final List<CleanupDecision> decisions;
  final ValueChanged<String> onApprove;
  final ValueChanged<String> onReject;

  @override
  Widget build(BuildContext context) {
    final base = scopeKey.isEmpty
        ? '(no scope key)'
        : (p.basename(scopeKey).isEmpty ? scopeKey : p.basename(scopeKey));
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(8, 8, 8, 4),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(Icons.folder_outlined,
                        size: 18,
                        color: Theme.of(context).colorScheme.primary),
                    const SizedBox(width: 6),
                    Expanded(
                      child: Text(
                        base,
                        style: Theme.of(context).textTheme.titleSmall,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: Theme.of(context).colorScheme.primary
                            .withValues(alpha: 0.12),
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Text(
                        '${decisions.length} pending',
                        style: TextStyle(
                          fontSize: 11,
                          color: Theme.of(context).colorScheme.primary,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                  ],
                ),
                if (scopeKey.isNotEmpty)
                  Text(
                    scopeKey,
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          fontFamily: 'monospace',
                          fontSize: 11,
                        ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
              ],
            ),
          ),
          for (final d in decisions)
            _InboxDecisionCard(
              decision: d,
              onApprove: () => onApprove(d.id),
              onReject: () => onReject(d.id),
            ),
        ],
      ),
    );
  }
}

class _InboxDecisionCard extends StatelessWidget {
  const _InboxDecisionCard({
    required this.decision,
    required this.onApprove,
    required this.onReject,
  });

  final CleanupDecision decision;
  final VoidCallback onApprove;
  final VoidCallback onReject;

  Color _verdictColor(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    switch (decision.verdict) {
      case 'keep':
        return scheme.primary;
      case 'stale':
        return scheme.error;
      case 'duplicate':
        return scheme.tertiary;
      default:
        return scheme.outline;
    }
  }

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: _verdictColor(context).withValues(alpha: 0.12),
                    border: Border.all(
                      color: _verdictColor(context).withValues(alpha: 0.4),
                    ),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    decision.verdict.toUpperCase(),
                    style: TextStyle(
                      fontSize: 10,
                      color: _verdictColor(context),
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    decision.memoryId,
                    style: muted?.copyWith(fontFamily: 'monospace'),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                Text(
                  DateFormat.MMMd().format(decision.createdAt.toLocal()),
                  style: muted,
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              decision.memoryTextSnapshot,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
            ),
            if (decision.reason.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(
                decision.reason,
                style: muted?.copyWith(fontStyle: FontStyle.italic),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
            ],
            if (decision.mergeInto.isNotEmpty) ...[
              const SizedBox(height: 6),
              Row(
                children: [
                  Icon(Icons.merge_outlined,
                      size: 14,
                      color:
                          Theme.of(context).colorScheme.tertiary),
                  const SizedBox(width: 4),
                  Text(
                    'will merge into ${decision.mergeInto}',
                    style: muted?.copyWith(fontFamily: 'monospace'),
                  ),
                ],
              ),
            ],
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: onReject,
                  child: const Text('Reject'),
                ),
                const SizedBox(width: 8),
                FilledButton(
                  onPressed: onApprove,
                  child: Text(
                    decision.verdict == 'keep'
                        ? 'Confirm keep'
                        : decision.verdict == 'stale'
                            ? 'Delete'
                            : 'Merge',
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
