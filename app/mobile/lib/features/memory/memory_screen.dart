import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/memory_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:path/path.dart' as p;

// Global Memory tab. Browses the cross-session pgvector memory
// store across the two scopes a phone-shaped UI can sensibly
// surface:
//
//   • Project — memories scoped to a specific cwd (project_key).
//     A horizontally-scrollable chip row picks the active project;
//     the chips come from /memory/scope-keys?scope=project.
//   • Global  — single flat list, no scope_key required.
//
// Session-scoped memories live alongside their session and are
// reached via the Sessions tab → Inspector (future). They're not
// browsable here because picking the right session id without
// session context is a worse UX than just opening the session.
class MemoryScreen extends ConsumerStatefulWidget {
  const MemoryScreen({super.key});

  @override
  ConsumerState<MemoryScreen> createState() => _MemoryScreenState();
}

class _MemoryScreenState extends ConsumerState<MemoryScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  // Project sub-state — kept here so it survives swipes between
  // tabs without tearing down the whole screen.
  AsyncValue<List<String>> _projectKeys = const AsyncValue.loading();
  String? _selectedKey;
  AsyncValue<List<_RowEntry>> _projectRows = const AsyncValue.loading();
  final _projectSearch = TextEditingController();
  String _projectQuery = '';

  AsyncValue<List<_RowEntry>> _globalRows = const AsyncValue.loading();
  final _globalSearch = TextEditingController();
  String _globalQuery = '';

  Timer? _debounce;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 2, vsync: this);
    _loadProjectKeys();
    _loadGlobal();
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _projectSearch.dispose();
    _globalSearch.dispose();
    _tabs.dispose();
    super.dispose();
  }

  Future<void> _loadProjectKeys() async {
    setState(() => _projectKeys = const AsyncValue.loading());
    try {
      final keys = await ref
          .read(memoryApiProvider)
          .scopeKeys(MemoryScope.project);
      if (!mounted) return;
      keys.sort();
      setState(() {
        _projectKeys = AsyncValue.data(keys);
        _selectedKey ??= keys.isEmpty ? null : keys.first;
      });
      if (_selectedKey != null) await _loadProject();
    } on ApiException catch (e) {
      if (mounted) {
        setState(() =>
            _projectKeys = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _projectKeys = AsyncValue.error(e, st));
    }
  }

  Future<void> _loadProject() async {
    final key = _selectedKey;
    if (key == null) return;
    setState(() => _projectRows = const AsyncValue.loading());
    try {
      final rows = _projectQuery.isEmpty
          ? await _listAsRows(MemoryScope.project, key)
          : await _searchAsRows(MemoryScope.project, key, _projectQuery);
      if (!mounted) return;
      setState(() => _projectRows = AsyncValue.data(rows));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() =>
            _projectRows = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _projectRows = AsyncValue.error(e, st));
    }
  }

  Future<void> _loadGlobal() async {
    setState(() => _globalRows = const AsyncValue.loading());
    try {
      final rows = _globalQuery.isEmpty
          ? await _listAsRows(MemoryScope.global, null)
          : await _searchAsRows(MemoryScope.global, null, _globalQuery);
      if (!mounted) return;
      setState(() => _globalRows = AsyncValue.data(rows));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _globalRows = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _globalRows = AsyncValue.error(e, st));
    }
  }

  Future<List<_RowEntry>> _listAsRows(
    MemoryScope scope,
    String? scopeKey,
  ) async {
    final memories = await ref.read(memoryApiProvider).list(
          scope: scope,
          scopeKey: scopeKey,
          limit: 200,
        );
    return memories.map((m) => _RowEntry(memory: m)).toList();
  }

  Future<List<_RowEntry>> _searchAsRows(
    MemoryScope scope,
    String? scopeKey,
    String query,
  ) async {
    final hits = await ref.read(memoryApiProvider).search(
          query: query,
          scope: scope,
          scopeKey: scopeKey,
          topK: 50,
          minSimilarity: -1,
        );
    return hits
        .map((h) => _RowEntry(memory: h.memory, similarity: h.similarity))
        .toList();
  }

  void _onProjectQueryChanged(String value) {
    _projectQuery = value.trim();
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), _loadProject);
  }

  void _onGlobalQueryChanged(String value) {
    _globalQuery = value.trim();
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), _loadGlobal);
  }

  Future<void> _openDetail(Memory mem) async {
    final result = await _MemoryDetailSheet.show(context: context, memory: mem);
    if (!mounted) return;
    if (result == _DetailResult.changed) {
      await Future.wait([
        if (mem.scope == MemoryScope.project) _loadProject(),
        if (mem.scope == MemoryScope.global) _loadGlobal(),
      ]);
    }
  }

  Future<void> _newMemory() async {
    final tabIndex = _tabs.index;
    final initialScope =
        tabIndex == 0 ? MemoryScope.project : MemoryScope.global;
    final initialKey = tabIndex == 0 ? _selectedKey : null;
    final created = await _NewMemorySheet.show(
      context: context,
      initialScope: initialScope,
      initialScopeKey: initialKey,
      knownProjectKeys: _projectKeys.value ?? const [],
    );
    if (!mounted || created != true) return;
    if (initialScope == MemoryScope.project) {
      await _loadProjectKeys(); // pick up brand new scope_key
    } else {
      await _loadGlobal();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Memory'),
        bottom: TabBar(
          controller: _tabs,
          tabs: const [Tab(text: 'Project'), Tab(text: 'Global')],
        ),
      ),
      body: TabBarView(
        controller: _tabs,
        children: [_projectTab(), _globalTab()],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _newMemory,
        icon: const Icon(Icons.add),
        label: const Text('New'),
      ),
    );
  }

  Widget _projectTab() {
    return Column(
      children: [
        _projectKeys.when(
          data: (keys) => keys.isEmpty
              ? const _EmptyHeader(
                  text: 'No project-scoped memories yet. Use + to create one.',
                )
              : _ProjectKeyChips(
                  keys: keys,
                  selected: _selectedKey,
                  onChanged: (k) {
                    setState(() => _selectedKey = k);
                    _loadProject();
                  },
                ),
          loading: () => const _LoadingStrip(),
          error: (e, _) => _ErrorStrip(error: e, onRetry: _loadProjectKeys),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
          child: TextField(
            controller: _projectSearch,
            onChanged: _onProjectQueryChanged,
            decoration: InputDecoration(
              hintText: 'Search…',
              prefixIcon: const Icon(Icons.search, size: 18),
              isDense: true,
              suffixIcon: _projectQuery.isEmpty
                  ? null
                  : IconButton(
                      icon: const Icon(Icons.clear, size: 18),
                      onPressed: () {
                        _projectSearch.clear();
                        _projectQuery = '';
                        _loadProject();
                      },
                    ),
            ),
          ),
        ),
        Expanded(
          child: _selectedKey == null
              ? const _EmptyView(text: 'Pick a project to browse its memories.')
              : _MemoryList(
                  state: _projectRows,
                  onTap: _openDetail,
                  onRefresh: _loadProject,
                  searching: _projectQuery.isNotEmpty,
                ),
        ),
      ],
    );
  }

  Widget _globalTab() {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
          child: TextField(
            controller: _globalSearch,
            onChanged: _onGlobalQueryChanged,
            decoration: InputDecoration(
              hintText: 'Search…',
              prefixIcon: const Icon(Icons.search, size: 18),
              isDense: true,
              suffixIcon: _globalQuery.isEmpty
                  ? null
                  : IconButton(
                      icon: const Icon(Icons.clear, size: 18),
                      onPressed: () {
                        _globalSearch.clear();
                        _globalQuery = '';
                        _loadGlobal();
                      },
                    ),
            ),
          ),
        ),
        Expanded(
          child: _MemoryList(
            state: _globalRows,
            onTap: _openDetail,
            onRefresh: _loadGlobal,
            searching: _globalQuery.isNotEmpty,
          ),
        ),
      ],
    );
  }
}

class _RowEntry {
  _RowEntry({required this.memory, this.similarity});
  final Memory memory;
  final double? similarity;
}

class _ProjectKeyChips extends StatelessWidget {
  const _ProjectKeyChips({
    required this.keys,
    required this.selected,
    required this.onChanged,
  });

  final List<String> keys;
  final String? selected;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 44,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        separatorBuilder: (_, __) => const SizedBox(width: 6),
        itemCount: keys.length,
        itemBuilder: (_, i) {
          final k = keys[i];
          final isSelected = k == selected;
          final label = p.basename(k).isEmpty ? k : p.basename(k);
          return ChoiceChip(
            label: Text(label),
            tooltip: k,
            selected: isSelected,
            onSelected: (_) => onChanged(k),
          );
        },
      ),
    );
  }
}

class _MemoryList extends StatelessWidget {
  const _MemoryList({
    required this.state,
    required this.onTap,
    required this.onRefresh,
    required this.searching,
  });

  final AsyncValue<List<_RowEntry>> state;
  final ValueChanged<Memory> onTap;
  final Future<void> Function() onRefresh;
  final bool searching;

  @override
  Widget build(BuildContext context) {
    return state.when(
      data: (rows) {
        if (rows.isEmpty) {
          return _EmptyView(
            text: searching ? 'No matches.' : 'No memories yet.',
          );
        }
        return RefreshIndicator(
          onRefresh: onRefresh,
          child: ListView.separated(
            itemCount: rows.length,
            separatorBuilder: (_, __) => Divider(
              height: 1,
              color: Theme.of(context).dividerColor,
            ),
            itemBuilder: (_, i) =>
                _MemoryTile(row: rows[i], onTap: () => onTap(rows[i].memory)),
          ),
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _ErrorView(error: e, onRetry: onRefresh),
    );
  }
}

class _MemoryTile extends StatelessWidget {
  const _MemoryTile({required this.row, required this.onTap});
  final _RowEntry row;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final m = row.memory;
    final preview = m.text.replaceAll(RegExp(r'\s+'), ' ').trim();
    return ListTile(
      onTap: onTap,
      title: Text(
        preview.isEmpty ? '(empty memory)' : preview,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
        style: Theme.of(context).textTheme.bodyMedium,
      ),
      subtitle: Row(
        children: [
          if (m.scopeKey.isNotEmpty)
            Flexible(
              child: Text(
                p.basename(m.scopeKey).isEmpty
                    ? m.scopeKey
                    : p.basename(m.scopeKey),
                style: Theme.of(context).textTheme.bodySmall,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          if (m.scopeKey.isNotEmpty) const Text('  ·  '),
          Text(
            _relTime(m.updatedAt),
            style: Theme.of(context).textTheme.bodySmall,
          ),
          if (row.similarity != null) ...[
            const Text('  ·  '),
            Text(
              row.similarity!.toStringAsFixed(2),
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context).colorScheme.primary,
                    fontWeight: FontWeight.w600,
                  ),
            ),
          ],
        ],
      ),
      trailing: const Icon(Icons.chevron_right),
    );
  }

  static String _relTime(DateTime ts) {
    final diff = DateTime.now().toUtc().difference(ts.toUtc());
    if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays < 7) return '${diff.inDays}d ago';
    return DateFormat.yMMMd().format(ts.toLocal());
  }
}

class _LoadingStrip extends StatelessWidget {
  const _LoadingStrip();
  @override
  Widget build(BuildContext context) =>
      const Padding(padding: EdgeInsets.all(12), child: LinearProgressIndicator());
}

class _ErrorStrip extends StatelessWidget {
  const _ErrorStrip({required this.error, required this.onRetry});
  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      color: Theme.of(context).colorScheme.error.withValues(alpha: 0.08),
      child: Row(
        children: [
          Expanded(
            child: Text(
              error.toString(),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
          TextButton(onPressed: onRetry, child: const Text('Retry')),
        ],
      ),
    );
  }
}

class _EmptyHeader extends StatelessWidget {
  const _EmptyHeader({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
      child: Text(text, style: Theme.of(context).textTheme.bodySmall),
    );
  }
}

class _EmptyView extends StatelessWidget {
  const _EmptyView({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Text(
          text,
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.bodyMedium,
        ),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final Object error;
  final Future<void> Function() onRetry;

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
              error.toString(),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: onRetry,
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
    );
  }
}

// ─── Detail sheet ─────────────────────────────────────────────────

// `null` from the sheet means "no change" (user dismissed); `changed`
// means a save / delete went through and the caller should reload.
enum _DetailResult { changed }

class _MemoryDetailSheet extends ConsumerStatefulWidget {
  const _MemoryDetailSheet({required this.memory});
  final Memory memory;

  static Future<_DetailResult?> show({
    required BuildContext context,
    required Memory memory,
  }) {
    return showModalBottomSheet<_DetailResult>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      constraints: BoxConstraints(
        maxHeight: MediaQuery.of(context).size.height * 0.92,
      ),
      builder: (_) => _MemoryDetailSheet(memory: memory),
    );
  }

  @override
  ConsumerState<_MemoryDetailSheet> createState() =>
      _MemoryDetailSheetState();
}

class _MemoryDetailSheetState extends ConsumerState<_MemoryDetailSheet> {
  late final TextEditingController _ctrl;
  bool _editing = false;
  bool _busy = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.memory.text);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    final body = _ctrl.text.trim();
    if (body.isEmpty) {
      setState(() => _error = 'Text cannot be empty');
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref.read(memoryApiProvider).update(
            id: widget.memory.id,
            text: body,
          );
      if (!mounted) return;
      Navigator.of(context).pop(_DetailResult.changed);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _busy = false;
          _error = e.message;
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
        _busy = false;
        _error = e.toString();
      });
      }
    }
  }

  Future<void> _delete() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (dialogCtx) => AlertDialog(
        title: const Text('Delete memory?'),
        content: const Text('This cannot be undone.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogCtx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(dialogCtx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(dialogCtx).pop(true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref.read(memoryApiProvider).delete(widget.memory.id);
      if (!mounted) return;
      Navigator.of(context).pop(_DetailResult.changed);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
        _busy = false;
        _error = e.message;
      });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
        _busy = false;
        _error = e.toString();
      });
      }
    }
  }

  Future<void> _copyText() async {
    await Clipboard.setData(ClipboardData(text: widget.memory.text));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('Copied'),
        duration: Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final m = widget.memory;
    return SafeArea(
      top: false,
      child: Padding(
        padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              Center(
                child: Container(
                  width: 36,
                  height: 4,
                  margin: const EdgeInsets.only(bottom: 8),
                  decoration: BoxDecoration(
                    color: Theme.of(context).dividerColor,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              Row(
                children: [
                  _ScopeBadge(scope: m.scope),
                  const SizedBox(width: 6),
                  if (m.scopeKey.isNotEmpty)
                    Expanded(
                      child: Text(
                        m.scopeKey,
                        style: Theme.of(context).textTheme.bodySmall,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  IconButton(
                    icon: const Icon(Icons.copy, size: 18),
                    tooltip: 'Copy text',
                    onPressed: _busy ? null : _copyText,
                  ),
                  IconButton(
                    icon: Icon(_editing ? Icons.close : Icons.edit, size: 18),
                    tooltip: _editing ? 'Cancel edit' : 'Edit',
                    onPressed: _busy
                        ? null
                        : () => setState(() {
                              if (_editing) {
                                _ctrl.text = widget.memory.text;
                                _error = null;
                              }
                              _editing = !_editing;
                            }),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Flexible(
                child: SingleChildScrollView(
                  child: _editing
                      ? TextField(
                          controller: _ctrl,
                          maxLines: null,
                          minLines: 6,
                          autofocus: true,
                          style: const TextStyle(fontSize: 13, height: 1.5),
                          decoration: InputDecoration(
                            border: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(8),
                            ),
                          ),
                        )
                      : SelectableText(
                          m.text,
                          style: const TextStyle(fontSize: 13, height: 1.5),
                        ),
                ),
              ),
              const SizedBox(height: 12),
              _ProvenanceBlock(memory: m),
              if (_error != null) ...[
                const SizedBox(height: 8),
                Text(
                  _error!,
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.error,
                    fontSize: 12,
                  ),
                ),
              ],
              const SizedBox(height: 12),
              Row(
                children: [
                  if (_editing)
                    Expanded(
                      child: FilledButton(
                        onPressed: _busy ? null : _save,
                        child: _busy
                            ? const SizedBox(
                                height: 16,
                                width: 16,
                                child: CircularProgressIndicator(strokeWidth: 2),
                              )
                            : const Text('Save'),
                      ),
                    )
                  else
                    Expanded(
                      child: OutlinedButton.icon(
                        onPressed: _busy ? null : _delete,
                        style: OutlinedButton.styleFrom(
                          foregroundColor: Theme.of(context).colorScheme.error,
                          side: BorderSide(
                            color: Theme.of(context)
                                .colorScheme
                                .error
                                .withValues(alpha: 0.4),
                          ),
                        ),
                        icon: const Icon(Icons.delete_outline, size: 18),
                        label: const Text('Delete'),
                      ),
                    ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ScopeBadge extends StatelessWidget {
  const _ScopeBadge({required this.scope});
  final MemoryScope scope;

  @override
  Widget build(BuildContext context) {
    final color = switch (scope) {
      MemoryScope.session => Colors.amberAccent,
      MemoryScope.project => Colors.blueAccent,
      MemoryScope.global => Colors.greenAccent,
      MemoryScope.unknown => Colors.grey,
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        border: Border.all(color: color.withValues(alpha: 0.5)),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        scope.label,
        style: TextStyle(
          color: color,
          fontSize: 10,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.5,
        ),
      ),
    );
  }
}

class _ProvenanceBlock extends StatelessWidget {
  const _ProvenanceBlock({required this.memory});
  final Memory memory;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final fmt = DateFormat.yMMMd().add_Hm();
    final lines = <String>[
      'created ${fmt.format(memory.createdAt.toLocal())}',
      'updated ${fmt.format(memory.updatedAt.toLocal())}',
      if (memory.embedder.isNotEmpty) 'embedder: ${memory.embedder}',
      if (memory.sourceKind != null && memory.sourceKind!.isNotEmpty)
        memory.sourceRef != null && memory.sourceRef!.isNotEmpty
            ? 'source: ${memory.sourceKind} (${memory.sourceRef})'
            : 'source: ${memory.sourceKind}',
      if (memory.confidence != null)
        'confidence: ${memory.confidence!.toStringAsFixed(2)}',
      if (memory.hitCount > 0) 'hits: ${memory.hitCount}',
    ];
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: Theme.of(context).dividerColor.withValues(alpha: 0.18),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final l in lines)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 1),
              child: Text(l, style: muted),
            ),
        ],
      ),
    );
  }
}

// ─── New-memory sheet ─────────────────────────────────────────────

class _NewMemorySheet extends ConsumerStatefulWidget {
  const _NewMemorySheet({
    required this.initialScope,
    required this.initialScopeKey,
    required this.knownProjectKeys,
  });

  final MemoryScope initialScope;
  final String? initialScopeKey;
  final List<String> knownProjectKeys;

  static Future<bool?> show({
    required BuildContext context,
    required MemoryScope initialScope,
    String? initialScopeKey,
    List<String> knownProjectKeys = const [],
  }) {
    return showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => _NewMemorySheet(
        initialScope: initialScope,
        initialScopeKey: initialScopeKey,
        knownProjectKeys: knownProjectKeys,
      ),
    );
  }

  @override
  ConsumerState<_NewMemorySheet> createState() => _NewMemorySheetState();
}

class _NewMemorySheetState extends ConsumerState<_NewMemorySheet> {
  late MemoryScope _scope;
  late final TextEditingController _scopeKeyCtrl;
  late final TextEditingController _textCtrl;
  bool _busy = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _scope = widget.initialScope;
    _scopeKeyCtrl = TextEditingController(
      text: widget.initialScopeKey ?? '',
    );
    _textCtrl = TextEditingController();
  }

  @override
  void dispose() {
    _scopeKeyCtrl.dispose();
    _textCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final text = _textCtrl.text.trim();
    final scopeKey = _scopeKeyCtrl.text.trim();
    if (text.isEmpty) {
      setState(() => _error = 'Text is required');
      return;
    }
    if (_scope != MemoryScope.global && scopeKey.isEmpty) {
      setState(() => _error = 'Scope key is required for ${_scope.label}');
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref.read(memoryApiProvider).store(
            text: text,
            scope: _scope,
            scopeKey: _scope == MemoryScope.global ? null : scopeKey,
          );
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
        _busy = false;
        _error = e.message;
      });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
        _busy = false;
        _error = e.toString();
      });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Padding(
        padding:
            EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              Center(
                child: Container(
                  width: 36,
                  height: 4,
                  decoration: BoxDecoration(
                    color: Theme.of(context).dividerColor,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              const SizedBox(height: 12),
              Text(
                'New memory',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 16),
              SegmentedButton<MemoryScope>(
                segments: const [
                  ButtonSegment(
                    value: MemoryScope.project,
                    label: Text('Project'),
                  ),
                  ButtonSegment(
                    value: MemoryScope.global,
                    label: Text('Global'),
                  ),
                ],
                selected: {_scope},
                onSelectionChanged: (s) {
                  setState(() {
                    _scope = s.first;
                    if (_scope == MemoryScope.global) _scopeKeyCtrl.clear();
                  });
                },
              ),
              const SizedBox(height: 12),
              if (_scope != MemoryScope.global)
                _ScopeKeyField(
                  controller: _scopeKeyCtrl,
                  knownKeys: widget.knownProjectKeys,
                ),
              const SizedBox(height: 12),
              TextField(
                controller: _textCtrl,
                maxLines: null,
                minLines: 5,
                autofocus: true,
                decoration: InputDecoration(
                  labelText: 'Text',
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                  ),
                ),
              ),
              if (_error != null) ...[
                const SizedBox(height: 8),
                Text(
                  _error!,
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.error,
                    fontSize: 12,
                  ),
                ),
              ],
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed:
                          _busy ? null : () => Navigator.of(context).pop(false),
                      child: const Text('Cancel'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: FilledButton(
                      onPressed: _busy ? null : _submit,
                      child: _busy
                          ? const SizedBox(
                              height: 16,
                              width: 16,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Text('Create'),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ScopeKeyField extends StatelessWidget {
  const _ScopeKeyField({required this.controller, required this.knownKeys});
  final TextEditingController controller;
  final List<String> knownKeys;

  @override
  Widget build(BuildContext context) {
    return Autocomplete<String>(
      initialValue: TextEditingValue(text: controller.text),
      optionsBuilder: (textEditingValue) {
        final q = textEditingValue.text.trim().toLowerCase();
        if (q.isEmpty) return knownKeys;
        return knownKeys
            .where((k) => k.toLowerCase().contains(q))
            .toList(growable: false);
      },
      fieldViewBuilder: (context, controllerInner, focusNode, onSubmitted) {
        // Wire the autocomplete's controller back to ours.
        controllerInner.addListener(() {
          controller.text = controllerInner.text;
        });
        return TextField(
          controller: controllerInner,
          focusNode: focusNode,
          autocorrect: false,
          decoration: InputDecoration(
            labelText: 'Scope key (project cwd)',
            hintText: '/Users/you/projects/foo',
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
            ),
          ),
        );
      },
    );
  }
}
