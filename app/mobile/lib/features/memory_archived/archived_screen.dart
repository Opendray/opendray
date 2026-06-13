import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/memory_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:path/path.dart' as p;

// ArchivedMemoriesScreen surfaces the soft-archived memories across every
// project AND global scope, grouped per project as COLLAPSED rows
// (project name + count); expand one to see and restore/delete its
// memories. Sources: the auto-cleaner's verdicts, manual per-memory
// archive, and whole projects you archive. Restorable until the 30-day
// grace window purges them — or delete now to skip the wait. Web parity:
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
  final _expanded = <String>{};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _rows = const AsyncValue.loading());
    try {
      final api = ref.read(memoryApiProvider);
      // Both scopes: project-scoped rows (all cwds) AND global-scope rows.
      final results = await Future.wait([
        api.listArchived(scope: MemoryScope.project, limit: 500),
        api.listArchived(scope: MemoryScope.global, limit: 500),
      ]);
      if (!mounted) return;
      setState(() => _rows = AsyncValue.data([...results[0], ...results[1]]));
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
      _snack(t.memoryArchived.restoreFailed(error: e.toString()));
      if (mounted) await _load();
    }
  }

  Future<void> _delete(String id) async {
    final ok = await _confirm(t.memoryArchived.deleteConfirm);
    if (ok != true) return;
    try {
      await ref.read(memoryApiProvider).delete(id);
      if (mounted) {
        _snack(t.memoryArchived.deletedToast);
        await _load();
      }
    } on ApiException catch (e) {
      _snack(t.memoryArchived.deleteFailed(error: e.toString()));
      if (mounted) await _load();
    }
  }

  // Bulk restore / delete fan out over the group's rows (counts are
  // small; there's no general restore-by-scope endpoint, and delete-all
  // must touch only the archived rows shown, not active memories).
  Future<void> _restoreAll(_ArchivedGroup g) async {
    final ok = await _confirm(t.memoryArchived
        .restoreAllConfirm(count: g.rows.length, project: g.label));
    if (ok != true) return;
    final api = ref.read(memoryApiProvider);
    var done = 0;
    for (final m in g.rows) {
      try {
        await api.restore(m.id);
        done++;
      } on Object {/* keep going */}
    }
    if (mounted) {
      _snack(t.memoryArchived.restoredAllToast(count: done));
      await _load();
    }
  }

  Future<void> _deleteAll(_ArchivedGroup g) async {
    final ok = await _confirm(t.memoryArchived
        .deleteAllConfirm(count: g.rows.length, project: g.label));
    if (ok != true) return;
    final api = ref.read(memoryApiProvider);
    var done = 0;
    for (final m in g.rows) {
      try {
        await api.delete(m.id);
        done++;
      } on Object {/* keep going */}
    }
    if (mounted) {
      _snack(t.memoryArchived.deletedAllToast(count: done));
      await _load();
    }
  }

  Future<bool?> _confirm(String message) {
    return showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(message),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.common.ok),
          ),
        ],
      ),
    );
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(msg), behavior: SnackBarBehavior.floating),
    );
  }

  List<_ArchivedGroup> _group(List<Memory> rows) {
    final m = <String, _ArchivedGroup>{};
    for (final r in rows) {
      final key = '${r.scope.wire}:${r.scopeKey}';
      final g = m.putIfAbsent(
        key,
        () => _ArchivedGroup(
          key: key,
          isGlobal: r.scope == MemoryScope.global,
          scopeKey: r.scopeKey,
          rows: [],
        ),
      );
      g.rows.add(r);
    }
    final out = m.values.toList()
      ..sort((a, b) => b.rows.length.compareTo(a.rows.length));
    return out;
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
          if (rows.isEmpty) return _empty();
          final groups = _group(rows);
          return RefreshIndicator(
            onRefresh: _load,
            child: ListView.builder(
              padding: const EdgeInsets.fromLTRB(8, 8, 8, 16),
              itemCount: groups.length + 1,
              itemBuilder: (_, i) {
                if (i == 0) {
                  return Padding(
                    padding: const EdgeInsets.fromLTRB(8, 4, 8, 8),
                    child: Text(
                      t.memoryArchived.summary(
                        projects: groups.length,
                        memories: rows.length,
                      ),
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  );
                }
                final g = groups[i - 1];
                return _GroupTile(
                  group: g,
                  expanded: _expanded.contains(g.key),
                  onToggle: () => setState(() {
                    _expanded.contains(g.key)
                        ? _expanded.remove(g.key)
                        : _expanded.add(g.key);
                  }),
                  onRestoreAll: () => _restoreAll(g),
                  onDeleteAll: () => _deleteAll(g),
                  onRestore: _restore,
                  onDelete: _delete,
                );
              },
            ),
          );
        },
      ),
    );
  }

  Widget _empty() {
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
}

class _ArchivedGroup {
  _ArchivedGroup({
    required this.key,
    required this.isGlobal,
    required this.scopeKey,
    required this.rows,
  });
  final String key;
  final bool isGlobal;
  final String scopeKey;
  final List<Memory> rows;

  String get label {
    if (isGlobal) return t.memoryArchived.globalScope;
    final base = p.basename(scopeKey);
    return base.isEmpty ? scopeKey : base;
  }
}

class _GroupTile extends StatelessWidget {
  const _GroupTile({
    required this.group,
    required this.expanded,
    required this.onToggle,
    required this.onRestoreAll,
    required this.onDeleteAll,
    required this.onRestore,
    required this.onDelete,
  });

  final _ArchivedGroup group;
  final bool expanded;
  final VoidCallback onToggle;
  final VoidCallback onRestoreAll;
  final VoidCallback onDeleteAll;
  final ValueChanged<String> onRestore;
  final ValueChanged<String> onDelete;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          InkWell(
            onTap: onToggle,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(8, 8, 4, 8),
              child: Row(
                children: [
                  Icon(expanded ? Icons.expand_more : Icons.chevron_right,
                      size: 20, color: scheme.outline),
                  Icon(group.isGlobal ? Icons.public : Icons.folder_outlined,
                      size: 18, color: scheme.outline),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      group.label,
                      style: Theme.of(context).textTheme.titleSmall,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: scheme.surfaceContainerHighest,
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Text(
                      t.memoryArchived.countBadge(count: group.rows.length),
                      style: TextStyle(
                          fontSize: 11, color: scheme.onSurfaceVariant),
                    ),
                  ),
                  PopupMenuButton<String>(
                    icon: const Icon(Icons.more_vert, size: 20),
                    onSelected: (v) {
                      if (v == 'restoreAll') onRestoreAll();
                      if (v == 'deleteAll') onDeleteAll();
                    },
                    itemBuilder: (_) => [
                      PopupMenuItem(
                        value: 'restoreAll',
                        child: Row(children: [
                          const Icon(Icons.restore, size: 18),
                          const SizedBox(width: 8),
                          Text(t.memoryArchived.restoreAll),
                        ]),
                      ),
                      PopupMenuItem(
                        value: 'deleteAll',
                        child: Row(children: [
                          Icon(Icons.delete_forever,
                              size: 18, color: scheme.error),
                          const SizedBox(width: 8),
                          Text(t.memoryArchived.deleteAll,
                              style: TextStyle(color: scheme.error)),
                        ]),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
          if (expanded)
            for (final m in group.rows)
              _ArchivedCard(
                memory: m,
                onRestore: () => onRestore(m.id),
                onDelete: () => onDelete(m.id),
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
    required this.onDelete,
  });

  final Memory memory;
  final VoidCallback onRestore;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final scheme = Theme.of(context).colorScheme;
    final reason = memory.archivedReason ?? '';
    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 0, 8, 8),
      child: Container(
        decoration: BoxDecoration(
          color: scheme.surfaceContainerHighest.withValues(alpha: 0.3),
          borderRadius: BorderRadius.circular(8),
        ),
        padding: const EdgeInsets.all(10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                if (reason.isNotEmpty) ...[
                  Flexible(
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: scheme.surfaceContainerHighest,
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(
                        reason.toUpperCase(),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 10,
                          color: scheme.onSurfaceVariant,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                ],
                if (memory.archivedAt != null)
                  Text(
                    DateFormat.MMMd()
                        .add_jm()
                        .format(memory.archivedAt!.toLocal()),
                    style: muted,
                  ),
              ],
            ),
            const SizedBox(height: 8),
            Text(memory.text, maxLines: 4, overflow: TextOverflow.ellipsis),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                FilledButton.tonalIcon(
                  onPressed: onRestore,
                  icon: const Icon(Icons.restore, size: 18),
                  label: Text(t.memoryArchived.restore),
                ),
                const SizedBox(width: 8),
                TextButton.icon(
                  onPressed: onDelete,
                  style: TextButton.styleFrom(foregroundColor: scheme.error),
                  icon: const Icon(Icons.delete_forever, size: 18),
                  label: Text(t.memoryArchived.deletePermanently),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
