import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/memory_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/project_docs_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/memory_workers/memory_workers_screen.dart';
import 'package:opendray/features/project/project_screen.dart';
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

  // Project state snapshot for the currently selected project key.
  // Surfaces goal / plan / latest-journal so operators see what the
  // project "is doing right now" alongside the discrete fact list.
  // The full editor for these still lives in More → Project — this
  // card is a glanceable summary, not a CRUD surface.
  AsyncValue<_ProjectStateSnapshot> _projectState =
      const AsyncValue.loading();

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
        // Drop the active selection if its scope_key was wiped by a
        // bulk delete (or otherwise vanished from the server's
        // distinct-keys list). Fall back to the first remaining
        // chip, or to null when nothing's left.
        if (_selectedKey != null && !keys.contains(_selectedKey)) {
          _selectedKey = keys.isEmpty ? null : keys.first;
        } else {
          _selectedKey ??= keys.isEmpty ? null : keys.first;
        }
      });
      if (_selectedKey != null) {
        await _loadProject();
      } else {
        // No keys left — clear the row state so the empty-view
        // renders instead of a stale list.
        setState(() => _projectRows = const AsyncValue.data([]));
      }
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
    setState(() {
      _projectRows = const AsyncValue.loading();
      _projectState = const AsyncValue.loading();
    });
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
    // Fetch goal / plan / journal snapshot in parallel-ish. These
    // calls are independent of the memory list above; failures are
    // non-fatal (snapshot just stays empty).
    try {
      final api = ref.read(projectDocsApiProvider);
      final docs = await api.listDocs(key);
      final logs = await api.listLogs(key, limit: 1);
      if (!mounted) return;
      var goal = '';
      var plan = '';
      for (final d in docs) {
        if (d.kind == 'goal') goal = d.content;
        if (d.kind == 'plan') plan = d.content;
      }
      setState(() => _projectState = AsyncValue.data(
            _ProjectStateSnapshot(
              goal: goal,
              plan: plan,
              latestLog: logs.isEmpty ? null : logs.first,
            ),
          ));
    } on Object catch (e, st) {
      if (mounted) setState(() => _projectState = AsyncValue.error(e, st));
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

  // Renders the AppBar overflow that exposes "delete every memory in
  // this scope". Hidden when there's no actionable target (no chip
  // picked on the Project tab).
  Widget _bulkDeleteMenu(BuildContext context) {
    final inProject = _tabs.index == 0;
    final activeKey = _selectedKey;
    if (inProject && (activeKey == null || activeKey.isEmpty)) {
      return const SizedBox.shrink();
    }
    return PopupMenuButton<String>(
      icon: const Icon(Icons.more_vert),
      tooltip: t.memory.more,
      onSelected: (v) {
        if (v == 'wipe') _confirmAndWipe();
      },
      itemBuilder: (_) => [
        PopupMenuItem<String>(
          value: 'wipe',
          child: Row(
            children: [
              Icon(
                Icons.delete_sweep_outlined,
                size: 18,
                color: Theme.of(context).colorScheme.error,
              ),
              const SizedBox(width: 8),
              Text(
                inProject
                    ? 'Delete all in this project'
                    : 'Delete all global memories',
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Future<void> _confirmAndWipe() async {
    final inProject = _tabs.index == 0;
    final scope = inProject ? MemoryScope.project : MemoryScope.global;
    final scopeKey = inProject ? _selectedKey : null;
    final visibleCount = inProject
        ? _projectRows.value?.length
        : _globalRows.value?.length;
    final ok = await showDialog<bool>(
      context: context,
      builder: (dialogCtx) => AlertDialog(
        title: Text(t.memory.deleteAllConfirm.title),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              inProject
                  ? 'Project: $scopeKey'
                  : 'Scope: Global',
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            ),
            const SizedBox(height: 8),
            Text(
              visibleCount != null
                  ? 'Currently visible: $visibleCount memory '
                      '${visibleCount == 1 ? 'item' : 'items'}.'
                  : 'Counts unknown until the list loads.',
              style: Theme.of(dialogCtx).textTheme.bodySmall,
            ),
            const SizedBox(height: 8),
            Text(
              'This cannot be undone. Memories that were ingested via the '
              'Claude mirror will reappear on the next sync.',
              style: Theme.of(dialogCtx).textTheme.bodySmall,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogCtx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(dialogCtx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(dialogCtx).pop(true),
            child: Text(t.memory.deleteAllConfirm.deleteAll),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      final deleted = await ref.read(memoryApiProvider).deleteByScope(
            scope: scope,
            scopeKey: scopeKey,
          );
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            deleted == 1
                ? t.memory.deletedSnackOne(n: deleted.toString())
                : t.memory.deletedSnackOther(n: deleted.toString()),
          ),
          behavior: SnackBarBehavior.floating,
        ),
      );
      if (inProject) {
        // Remove the now-empty scope_key from the chip strip if its
        // count went to zero, then reload.
        await _loadProjectKeys();
      } else {
        await _loadGlobal();
      }
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(t.memory.bulkDeleteFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(t.memory.bulkDeleteFailedGeneric(error: e.toString()))),
      );
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
        title: Text(t.memory.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.tune),
            tooltip: t.memory.workers,
            onPressed: () {
              Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => const MemoryWorkersScreen(),
                ),
              );
            },
          ),
          AnimatedBuilder(
            animation: _tabs,
            builder: (_, __) => _bulkDeleteMenu(context),
          ),
        ],
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
        label: Text(t.memory.kNew),
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
              : _ProjectSelector(
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
        if (_selectedKey != null)
          _ProjectStateCard(
            state: _projectState,
            onOpenProject: () {
              Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => const ProjectScreen(),
                ),
              );
            },
          ),
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
          child: TextField(
            controller: _projectSearch,
            onChanged: _onProjectQueryChanged,
            decoration: InputDecoration(
              hintText: t.memory.searchHint,
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
              hintText: t.memory.searchHint,
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

// _ProjectSelector replaced the horizontal-strip ChoiceChip row in
// PR #54. The strip was unusable with 15+ projects: finding the
// right entry meant thumb-scrolling past everything alphabetically
// after it, and the chips had no search affordance. The selector
// is a single tappable row showing the active project's basename
// + the total count; tapping opens a full-search modal picker.
//
// One row regardless of project count — the page no longer
// "expands" visually with more projects, leaving the memory list
// the same screen real estate either way.
class _ProjectSelector extends StatelessWidget {
  const _ProjectSelector({
    required this.keys,
    required this.selected,
    required this.onChanged,
  });

  final List<String> keys;
  final String? selected;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final activeLabel = selected == null
        ? 'Pick a project'
        : (p.basename(selected!).isEmpty ? selected! : p.basename(selected!));
    final activePath = selected;
    // Mirrors a Material dropdown form field — bordered surface,
    // floating-style label, chevron on the right. The original
    // single-row tile (no border, tiny chevron) didn't read as
    // interactive; operators saw it as a header. The
    // InputDecorator-style shell makes "tap me" unmistakable.
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 4),
      child: InkWell(
        onTap: () => _openPicker(context),
        borderRadius: BorderRadius.circular(10),
        child: InputDecorator(
          decoration: InputDecoration(
            labelText: t.memory.projectLabel,
            // The picker is always pre-selected at first project, so
            // there is always a value below the label — no need to
            // suppress hintText. Adding an explicit suffixIcon makes
            // the dropdown affordance obvious.
            suffixIcon: Padding(
              padding: const EdgeInsets.only(right: 4),
              child: Icon(
                Icons.unfold_more,
                size: 22,
                color: theme.colorScheme.outline,
              ),
            ),
            prefixIcon: Padding(
              padding: const EdgeInsets.only(left: 8, right: 4),
              child: Icon(
                Icons.folder_outlined,
                size: 20,
                color: theme.colorScheme.outline,
              ),
            ),
            isDense: false,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 12,
              vertical: 10,
            ),
          ),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      activeLabel,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontWeight: FontWeight.w600,
                        fontSize: 14,
                      ),
                    ),
                    if (activePath != null &&
                        activePath.isNotEmpty &&
                        activePath != activeLabel)
                      Padding(
                        padding: const EdgeInsets.only(top: 2),
                        child: Text(
                          activePath,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          // Don't override the body-small color —
                          // colorScheme.outline maps to a near-border
                          // tone in our dark theme and rendered the
                          // path nearly invisible. Default bodySmall
                          // already uses the readable muted token.
                          style: theme.textTheme.bodySmall?.copyWith(
                            fontFamily: 'monospace',
                            fontSize: 12,
                          ),
                        ),
                      ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              Text(
                '${keys.length}',
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

  Future<void> _openPicker(BuildContext context) async {
    final picked = await showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => _ProjectPickerSheet(keys: keys, selected: selected),
    );
    if (picked != null) onChanged(picked);
  }
}

// _ProjectPickerSheet renders a full-search modal list of every
// known project_key. Search filters by case-insensitive substring
// match against both the basename and the full path — operators
// who remember "the one under HomeLab" can type that and narrow
// down without scrolling.
class _ProjectPickerSheet extends StatefulWidget {
  const _ProjectPickerSheet({required this.keys, required this.selected});
  final List<String> keys;
  final String? selected;

  @override
  State<_ProjectPickerSheet> createState() => _ProjectPickerSheetState();
}

class _ProjectPickerSheetState extends State<_ProjectPickerSheet> {
  final _searchCtrl = TextEditingController();
  String _query = '';

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Sort alphabetically by basename so projects with the same
    // last segment (e.g. multiple "frontend" dirs under different
    // workspaces) cluster together — easier to disambiguate when
    // the operator can see them side-by-side.
    final sorted = [...widget.keys]..sort((a, b) {
        final ab = p.basename(a).toLowerCase();
        final bb = p.basename(b).toLowerCase();
        final cmp = ab.compareTo(bb);
        return cmp != 0 ? cmp : a.toLowerCase().compareTo(b.toLowerCase());
      });
    final q = _query.trim().toLowerCase();
    final filtered = q.isEmpty
        ? sorted
        : sorted.where((k) => k.toLowerCase().contains(q)).toList();
    final mq = MediaQuery.of(context);

    return Padding(
      padding: EdgeInsets.only(bottom: mq.viewInsets.bottom),
      child: SafeArea(
        top: false,
        child: SizedBox(
          height: mq.size.height * 0.7,
          child: Column(
            children: [
              const SizedBox(height: 8),
              Container(
                width: 36,
                height: 4,
                decoration: BoxDecoration(
                  color: Theme.of(context).dividerColor,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        'Pick project',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                    ),
                    Text(
                      '${filtered.length} / ${sorted.length}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                child: TextField(
                  controller: _searchCtrl,
                  autofocus: true,
                  onChanged: (v) => setState(() => _query = v),
                  decoration: InputDecoration(
                    hintText: t.memory.filterHint,
                    prefixIcon: const Icon(Icons.search, size: 18),
                    isDense: true,
                  ),
                ),
              ),
              Expanded(
                child: filtered.isEmpty
                    ? Center(
                        child: Padding(
                          padding: const EdgeInsets.all(24),
                          child: Text(
                            'No projects match "$q".',
                            textAlign: TextAlign.center,
                            style: Theme.of(context).textTheme.bodyMedium,
                          ),
                        ),
                      )
                    : ListView.separated(
                        itemCount: filtered.length,
                        separatorBuilder: (_, __) => Divider(
                          height: 1,
                          color: Theme.of(context).dividerColor,
                        ),
                        itemBuilder: (_, i) {
                          final k = filtered[i];
                          final isSelected = k == widget.selected;
                          final base =
                              p.basename(k).isEmpty ? k : p.basename(k);
                          return ListTile(
                            dense: true,
                            leading: Icon(
                              isSelected
                                  ? Icons.radio_button_checked
                                  : Icons.folder_outlined,
                              size: 18,
                              color: isSelected
                                  ? Theme.of(context).colorScheme.primary
                                  : Theme.of(context).colorScheme.outline,
                            ),
                            title: Text(
                              base,
                              style: TextStyle(
                                fontWeight: isSelected
                                    ? FontWeight.w600
                                    : FontWeight.w500,
                              ),
                            ),
                            subtitle: Text(
                              k,
                              // Inherit ListTile subtitle's default
                              // color (theme.textTheme.bodySmall →
                              // _darkMuted in dark mode), readable
                              // against the picker sheet surface.
                              style: Theme.of(context)
                                  .textTheme
                                  .bodySmall
                                  ?.copyWith(
                                    fontFamily: 'monospace',
                                    fontSize: 12,
                                  ),
                            ),
                            onTap: () => Navigator.of(context).pop(k),
                          );
                        },
                      ),
              ),
            ],
          ),
        ),
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

    // Extract metadata signals surfaced as chips.
    final meta = m.metadata ?? const {};
    final typeTag = meta['type']?.toString() ?? '';
    final dedupedCount = _readInt(meta['deduped_count']);
    final originLabel = _originLabel(m.sourceKind);

    final badges = <Widget>[];
    if (typeTag.isNotEmpty) {
      badges.add(_TileBadge(
        text: typeTag,
        color: _typeColor(context, typeTag),
      ));
    }
    if (originLabel != null) {
      badges.add(_TileBadge(
        text: originLabel,
        color: Theme.of(context).colorScheme.secondary,
      ));
    }
    if (dedupedCount > 0) {
      badges.add(_TileBadge(
        text: 'merged ×$dedupedCount',
        color: Theme.of(context).colorScheme.tertiary,
      ));
    }
    if (m.hitCount > 0) {
      badges.add(_TileBadge(
        text: '${m.hitCount} hits',
        color: Theme.of(context).colorScheme.outline,
      ));
    }

    return ListTile(
      onTap: onTap,
      title: Text(
        preview.isEmpty ? '(empty memory)' : preview,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
        style: Theme.of(context).textTheme.bodyMedium,
      ),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (badges.isNotEmpty) ...[
            const SizedBox(height: 4),
            Wrap(spacing: 6, runSpacing: 4, children: badges),
            const SizedBox(height: 2),
          ],
          Row(
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

  static int _readInt(Object? v) {
    if (v is int) return v;
    if (v is num) return v.toInt();
    if (v is String) return int.tryParse(v) ?? 0;
    return 0;
  }

  static String? _originLabel(String? sourceKind) {
    switch (sourceKind) {
      case 'mcp_call':
        return 'agent';
      case 'summarizer':
        return 'summarizer';
      case 'mirror_claude_md':
        return 'mirror';
      case 'imported':
        return 'imported';
      case 'manual':
      case null:
      case '':
        return null;
      default:
        return sourceKind;
    }
  }

  static Color _typeColor(BuildContext context, String type) {
    final scheme = Theme.of(context).colorScheme;
    switch (type) {
      case 'user_preference':
        return scheme.primary;
      case 'project_fact':
        return scheme.secondary;
      case 'feedback':
        return scheme.error;
      case 'reference':
        return scheme.tertiary;
      default:
        return scheme.outline;
    }
  }
}

// _TileBadge is a tight 11-px chip used in the memory list. Smaller
// than Material's Chip so multiple badges fit on a phone-sized row.
class _TileBadge extends StatelessWidget {
  const _TileBadge({required this.text, required this.color});
  final String text;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        border: Border.all(color: color.withValues(alpha: 0.4)),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        text,
        style: TextStyle(
          fontSize: 10,
          color: color,
          fontWeight: FontWeight.w600,
          height: 1.2,
        ),
      ),
    );
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
          TextButton(onPressed: onRetry, child: Text(t.common.retry)),
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
              child: Text(t.common.retry),
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
        title: Text(t.memory.deleteOne.title),
        content: Text(t.memory.deleteOne.body),
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
      SnackBar(
        content: Text(t.memory.copied),
        duration: const Duration(seconds: 2),
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
                    tooltip: t.memory.copyTooltip,
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
                            : Text(t.common.save),
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
                        label: Text(t.common.delete),
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
                segments: [
                  ButtonSegment(
                    value: MemoryScope.project,
                    label: Text(t.memory.scope.project),
                  ),
                  ButtonSegment(
                    value: MemoryScope.global,
                    label: Text(t.memory.scope.global),
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
                  labelText: t.memory.create.textLabel,
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
            labelText: t.memory.create.scopeKeyLabel,
            hintText: t.memory.create.scopeKeyHint,
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
            ),
          ),
        );
      },
    );
  }
}

// _ProjectStateSnapshot bundles the goal / plan / latest-journal
// triple shown on the Memory tab as a one-glance summary. Goal and
// plan are markdown bodies; latestLog is the newest session_logs
// row if any.
class _ProjectStateSnapshot {
  _ProjectStateSnapshot({
    required this.goal,
    required this.plan,
    required this.latestLog,
  });
  final String goal;
  final String plan;
  final SessionLogEntry? latestLog;

  bool get isEmpty => goal.isEmpty && plan.isEmpty && latestLog == null;
}

// _ProjectStateCard renders the snapshot. Tap → open the full
// Project screen for editing. We render two lines of preview per
// section so the card never grows huge on a phone screen.
class _ProjectStateCard extends StatelessWidget {
  const _ProjectStateCard({
    required this.state,
    required this.onOpenProject,
  });

  final AsyncValue<_ProjectStateSnapshot> state;
  final VoidCallback onOpenProject;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
      child: state.when(
        loading: () => const SizedBox.shrink(),
        error: (_, __) => const SizedBox.shrink(),
        data: (snap) {
          if (snap.isEmpty) {
            return _EmptySnapshot(onOpenProject: onOpenProject);
          }
          return Card(
            child: InkWell(
              onTap: onOpenProject,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(12, 12, 12, 8),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Icon(
                          Icons.flag_outlined,
                          size: 16,
                          color: Theme.of(context).colorScheme.primary,
                        ),
                        const SizedBox(width: 6),
                        Text(
                          'Project state',
                          style: Theme.of(context)
                              .textTheme
                              .labelMedium
                              ?.copyWith(
                                fontWeight: FontWeight.w600,
                                color:
                                    Theme.of(context).colorScheme.primary,
                              ),
                        ),
                        const Spacer(),
                        Icon(
                          Icons.chevron_right,
                          size: 18,
                          color: Theme.of(context).colorScheme.outline,
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    _SnapshotSection(label: 'Goal', body: snap.goal),
                    _SnapshotSection(label: 'Plan', body: snap.plan),
                    if (snap.latestLog != null)
                      _SnapshotSection(
                        label: 'Last journal',
                        body: snap.latestLog!.title.isNotEmpty
                            ? '${snap.latestLog!.title} — ${snap.latestLog!.content}'
                            : snap.latestLog!.content,
                      ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

class _SnapshotSection extends StatelessWidget {
  const _SnapshotSection({required this.label, required this.body});
  final String label;
  final String body;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final trimmed = body.replaceAll(RegExp(r'\s+'), ' ').trim();
    if (trimmed.isEmpty) {
      return Padding(
        padding: const EdgeInsets.only(bottom: 6),
        child: Row(
          children: [
            SizedBox(
              width: 70,
              child: Text(label, style: muted),
            ),
            Expanded(
              child: Text(
                '(not set)',
                style: muted?.copyWith(fontStyle: FontStyle.italic),
              ),
            ),
          ],
        ),
      );
    }
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 70,
            child: Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(label, style: muted),
            ),
          ),
          Expanded(
            child: Text(
              trimmed,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }
}

class _EmptySnapshot extends StatelessWidget {
  const _EmptySnapshot({required this.onOpenProject});
  final VoidCallback onOpenProject;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Card(
      color: scheme.primary.withValues(alpha: 0.04),
      child: InkWell(
        onTap: onOpenProject,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              Icon(Icons.flag_outlined, size: 20, color: scheme.primary),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'No goal / plan / journal yet for this project',
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Tap to seed them in the Project screen',
                      style: Theme.of(context)
                          .textTheme
                          .bodySmall
                          ?.copyWith(color: scheme.outline),
                    ),
                  ],
                ),
              ),
              Icon(Icons.chevron_right, color: scheme.outline),
            ],
          ),
        ),
      ),
    );
  }
}
