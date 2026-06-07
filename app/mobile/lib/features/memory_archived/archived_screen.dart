import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/memory_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:path/path.dart' as p;

// ArchivedMemoriesScreen surfaces the soft-archived memories the
// auto-cleaner / lifecycle pass removed across every project, grouped
// by scope_key, with one-click restore. Reachable from More → Memory
// → Archived.
//
// There is no approval queue anymore: the cleaner auto-applies its
// keep/stale/duplicate verdicts as reversible soft-archives. This
// screen is the read-only "undo" surface — restore any false positive
// before the 30-day grace window hard-purges it. Web parity:
// app/web/src/pages/Archived.tsx.
class ArchivedMemoriesScreen extends ConsumerStatefulWidget {
  const ArchivedMemoriesScreen({super.key});

  @override
  ConsumerState<ArchivedMemoriesScreen> createState() =>
      _ArchivedMemoriesScreenState();
}

class _ArchivedMemoriesScreenState
    extends ConsumerState<ArchivedMemoriesScreen> {
  AsyncValue<List<Memory>> _rows = const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _rows = const AsyncValue.loading());
    try {
      final list = await ref.read(memoryApiProvider).listArchived(limit: 500);
      if (!mounted) return;
      setState(() => _rows = AsyncValue.data(list));
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _rows = AsyncValue.error(e, StackTrace.current));
    }
  }

  Future<void> _restore(String id) async {
    try {
      await ref.read(memoryApiProvider).restore(id);
      if (mounted) await _load();
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(t.memoryArchived.restoreFailed(error: e.toString())),
          ),
        );
        await _load();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.memoryArchived.title),
        actions: [
          IconButton(
            tooltip: t.common.refresh,
            icon: const Icon(Icons.refresh),
            onPressed: _load,
          ),
        ],
      ),
      body: _rows.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text(t.memoryArchived.loadFailed(error: e.toString())),
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
                          Icons.inventory_2_outlined,
                          size: 48,
                          color: Theme.of(context)
                              .colorScheme
                              .onSurface
                              .withValues(alpha: 0.4),
                        ),
                        const SizedBox(height: 16),
                        Text(
                          t.memoryArchived.emptyTitle,
                          textAlign: TextAlign.center,
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),
                        Text(
                          t.memoryArchived.emptyBody,
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
          final grouped = <String, List<Memory>>{};
          for (final m in rows) {
            grouped.putIfAbsent(m.scopeKey, () => []).add(m);
          }
          final keys = grouped.keys.toList()..sort();
          return RefreshIndicator(
            onRefresh: _load,
            child: ListView.builder(
              padding: const EdgeInsets.fromLTRB(8, 8, 8, 16),
              itemCount: keys.length,
              itemBuilder: (_, i) {
                final k = keys[i];
                return _ScopeGroup(
                  scopeKey: k,
                  rows: grouped[k]!,
                  onRestore: _restore,
                );
              },
            ),
          );
        },
      ),
    );
  }
}

class _ScopeGroup extends StatelessWidget {
  const _ScopeGroup({
    required this.scopeKey,
    required this.rows,
    required this.onRestore,
  });

  final String scopeKey;
  final List<Memory> rows;
  final ValueChanged<String> onRestore;

  @override
  Widget build(BuildContext context) {
    final base = scopeKey.isEmpty
        ? t.memoryArchived.globalScope
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
                        color: Theme.of(context)
                            .colorScheme
                            .primary
                            .withValues(alpha: 0.12),
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Text(
                        t.memoryArchived.countBadge(count: rows.length),
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
          for (final m in rows)
            _ArchivedCard(
              memory: m,
              onRestore: () => onRestore(m.id),
            ),
        ],
      ),
    );
  }
}

class _ArchivedCard extends StatelessWidget {
  const _ArchivedCard({
    required this.memory,
    required this.onRestore,
  });

  final Memory memory;
  final VoidCallback onRestore;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final scheme = Theme.of(context).colorScheme;
    final reason = memory.archivedReason ?? '';
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                if (reason.isNotEmpty) ...[
                  Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                      color: scheme.surfaceContainerHighest,
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      reason.toUpperCase(),
                      style: TextStyle(
                        fontSize: 10,
                        color: scheme.onSurfaceVariant,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                ],
                if (memory.archivedAt != null)
                  Expanded(
                    child: Text(
                      DateFormat.MMMd()
                          .add_jm()
                          .format(memory.archivedAt!.toLocal()),
                      style: muted,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              memory.text,
              maxLines: 4,
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                FilledButton.tonalIcon(
                  onPressed: onRestore,
                  icon: const Icon(Icons.restore, size: 18),
                  label: Text(t.memoryArchived.restore),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
