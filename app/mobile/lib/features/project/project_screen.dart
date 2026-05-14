import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/memory_api.dart';
import 'package:opendray/core/api/memory_cleanup_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/project_docs_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:path/path.dart' as p;

// Project screen — surfaces memory layers 2-4 (goal / plan / journal)
// plus the proposal inbox. cwd picker reuses the project scope-keys
// endpoint from memory so the project list stays consistent with the
// Memory tab.
//
// Layout: cwd picker at top, 4 tabs underneath (Goal / Plan / Journal
// / Inbox). Wide tabs save the user a button-press compared to a
// scrollable list of sub-pages.
class ProjectScreen extends ConsumerStatefulWidget {
  const ProjectScreen({super.key, this.initialCwd});

  /// When set, the screen jumps straight to that cwd's project view
  /// instead of letting the operator pick from the dropdown. Used
  /// when entering from a Session detail screen that already knows
  /// which project the session is bound to.
  final String? initialCwd;

  @override
  ConsumerState<ProjectScreen> createState() => _ProjectScreenState();
}

class _ProjectScreenState extends ConsumerState<ProjectScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  AsyncValue<List<String>> _projectKeys = const AsyncValue.loading();
  String? _selectedKey;

  // Per-tab state. Kept in the parent so tab swipes don't tear down.
  AsyncValue<List<ProjectDoc>> _docs = const AsyncValue.loading();
  AsyncValue<List<DocProposal>> _proposals = const AsyncValue.loading();
  AsyncValue<List<SessionLogEntry>> _logs = const AsyncValue.loading();
  AsyncValue<List<CleanupDecision>> _cleanupDecisions =
      const AsyncValue.loading();
  bool _cleanupRunning = false;

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 7, vsync: this);
    _loadKeys();
  }

  @override
  void dispose() {
    _tabs.dispose();
    super.dispose();
  }

  Future<void> _loadKeys() async {
    setState(() => _projectKeys = const AsyncValue.loading());
    try {
      final keys = await ref
          .read(memoryApiProvider)
          .scopeKeys(MemoryScope.project);
      if (!mounted) return;
      keys.sort();
      // When the caller passed an initialCwd that's not yet in the
      // memory scope_keys list (a brand-new session whose project
      // has no L5 memories yet), inject it so the picker can still
      // anchor on it. Tech-stack + journal + plan all live in their
      // own tables and don't need a memory row to exist.
      final initial = widget.initialCwd;
      var mergedKeys = keys;
      if (initial != null && !keys.contains(initial)) {
        mergedKeys = [...keys, initial]..sort();
      }
      setState(() {
        _projectKeys = AsyncValue.data(mergedKeys);
        if (initial != null && initial.isNotEmpty) {
          // initialCwd wins on the first call. Subsequent _loadKeys
          // calls (refresh, pull-to-reload) don't reset it because
          // we leave _selectedKey alone when already set.
          _selectedKey ??= initial;
        }
        if (_selectedKey != null && !mergedKeys.contains(_selectedKey)) {
          _selectedKey = null;
        }
        _selectedKey ??=
            mergedKeys.isEmpty ? null : mergedKeys.first;
      });
      if (_selectedKey != null) {
        await _loadAll(_selectedKey!);
      }
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _projectKeys = AsyncValue.error(e, StackTrace.current));
    }
  }

  Future<void> _loadAll(String cwd) async {
    setState(() {
      _docs = const AsyncValue.loading();
      _proposals = const AsyncValue.loading();
      _logs = const AsyncValue.loading();
      _cleanupDecisions = const AsyncValue.loading();
    });
    final api = ref.read(projectDocsApiProvider);
    final cleanupApi = ref.read(memoryCleanupApiProvider);
    try {
      final docs = await api.listDocs(cwd);
      if (!mounted) return;
      setState(() => _docs = AsyncValue.data(docs));
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _docs = AsyncValue.error(e, StackTrace.current));
    }
    try {
      final props = await api.listPendingProposals(cwd: cwd);
      if (!mounted) return;
      setState(() => _proposals = AsyncValue.data(props));
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _proposals = AsyncValue.error(e, StackTrace.current));
    }
    try {
      final logs = await api.listLogs(cwd);
      if (!mounted) return;
      setState(() => _logs = AsyncValue.data(logs));
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _logs = AsyncValue.error(e, StackTrace.current));
    }
    try {
      final decisions = await cleanupApi.list(
        scope: 'project',
        scopeKey: cwd,
        status: 'pending',
      );
      if (!mounted) return;
      setState(() => _cleanupDecisions = AsyncValue.data(decisions));
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() =>
          _cleanupDecisions = AsyncValue.error(e, StackTrace.current));
    }
  }

  Widget _cwdPicker() {
    return _projectKeys.when(
      loading: () => const Padding(
        padding: EdgeInsets.all(16),
        child: LinearProgressIndicator(),
      ),
      error: (e, _) => Padding(
        padding: const EdgeInsets.all(16),
        child: Text(t.project.projectsLoadFailed(error: e.toString())),
      ),
      data: (keys) {
        if (keys.isEmpty) {
          return const Padding(
            padding: EdgeInsets.all(16),
            child: Text(
              'No projects yet — spawn a session in a working directory '
              'to register it.',
            ),
          );
        }
        return Padding(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 0),
          child: InkWell(
            onTap: () => _openProjectPicker(keys),
            child: InputDecorator(
              decoration: InputDecoration(
                labelText: t.project.projectLabel,
                border: const OutlineInputBorder(),
                suffixIcon: const Icon(Icons.unfold_more),
              ),
              child: Text(
                _selectedKey == null
                    ? 'Select a project'
                    : '${p.basename(_selectedKey!)}  ·  ${_selectedKey!}',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ),
        );
      },
    );
  }

  void _openProjectPicker(List<String> keys) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) {
        return SafeArea(
          child: ListView(
            shrinkWrap: true,
            children: [
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Text(
                  'Select a project',
                  style: TextStyle(fontWeight: FontWeight.w600),
                ),
              ),
              for (final k in keys)
                ListTile(
                  title: Text(p.basename(k)),
                  subtitle: Text(
                    k,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  selected: k == _selectedKey,
                  onTap: () {
                    Navigator.of(ctx).pop();
                    setState(() => _selectedKey = k);
                    _loadAll(k);
                  },
                ),
              const SizedBox(height: 8),
            ],
          ),
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.project.title),
        actions: [
          if (_selectedKey != null && _selectedKey!.isNotEmpty)
            IconButton(
              icon: const Icon(Icons.restart_alt),
              tooltip: t.project.resetTooltip,
              onPressed: () => _confirmReset(_selectedKey!),
            ),
        ],
        bottom: TabBar(
          controller: _tabs,
          isScrollable: true,
          tabs: const [
            Tab(text: 'Goal'),
            Tab(text: 'Plan'),
            Tab(text: 'Tech'),
            Tab(text: 'Activity'),
            Tab(text: 'Journal'),
            Tab(text: 'Inbox'),
            Tab(text: 'Cleanup'),
          ],
        ),
      ),
      body: Column(
        children: [
          _cwdPicker(),
          const SizedBox(height: 8),
          Expanded(
            child: TabBarView(
              controller: _tabs,
              children: [
                _docTab('goal'),
                _docTab('plan'),
                _readonlyDocTab(
                  'tech_stack',
                  emptyText: 'No tech_stack scan yet. '
                      'Triggered automatically on every session spawn '
                      '(refreshes every 6h). You can force a refresh '
                      'via POST /api/v1/project-scan/run.',
                ),
                _readonlyDocTab(
                  'recent_activity',
                  emptyText: 'No git activity summary yet. '
                      'Generated by the LLM librarian — refreshes '
                      'automatically every 12h, or you can force it '
                      'via POST /api/v1/git-activity/run.',
                ),
                _journalTab(),
                _inboxTab(),
                _cleanupTab(),
              ],
            ),
          ),
        ],
      ),
    );
  }

  // ── Goal / Plan editor ─────────────────────────────────────────

  Widget _docTab(String kind) {
    if (_selectedKey == null) {
      return Center(child: Text(t.project.pickFirst));
    }
    return _docs.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Text(t.project.loadFailed(error: e.toString())),
        ),
      ),
      data: (docs) {
        final current = docs.firstWhere(
          (d) => d.kind == kind,
          orElse: () => ProjectDoc(
            id: '',
            cwd: _selectedKey!,
            kind: kind,
            content: '',
            updatedBy: 'operator',
          ),
        );
        return _DocEditor(
          key: ValueKey('$kind-${current.id}-${_selectedKey!}'),
          doc: current,
          onSaved: () => _loadAll(_selectedKey!),
        );
      },
    );
  }

  // _readonlyDocTab renders scanner-managed docs (tech_stack /
  // recent_activity) — these are NEVER hand-edited; the scanner /
  // git activity summariser rewrites them on each refresh. The UI
  // only shows the content + last-updated metadata.
  Widget _readonlyDocTab(String kind, {required String emptyText}) {
    if (_selectedKey == null) {
      return Center(child: Text(t.project.pickFirst));
    }
    return _docs.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Text(t.project.loadFailed(error: e.toString())),
        ),
      ),
      data: (docs) {
        final current = docs.firstWhere(
          (d) => d.kind == kind,
          orElse: () => ProjectDoc(
            id: '',
            cwd: _selectedKey!,
            kind: kind,
            content: '',
            updatedBy: 'scanner',
          ),
        );
        return RefreshIndicator(
          onRefresh: () async => _loadAll(_selectedKey!),
          child: ListView(
            padding: const EdgeInsets.all(12),
            children: [
              if (current.content.trim().isEmpty)
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Text(
                    emptyText,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                )
              else
                SelectableText(
                  current.content,
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
            ],
          ),
        );
      },
    );
  }

  // ── Journal ────────────────────────────────────────────────────

  Widget _journalTab() {
    if (_selectedKey == null) {
      return Center(child: Text(t.project.pickFirst));
    }
    return _logs.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) =>
          Center(child: Text(t.project.loadFailed(error: e.toString()))),
      data: (logs) {
        return RefreshIndicator(
          onRefresh: () async {
            await _loadAll(_selectedKey!);
          },
          child: Stack(
            children: [
              if (logs.isEmpty)
                Center(
                  child: Padding(
                    padding: const EdgeInsets.all(32),
                    child: Text(
                      'No journal entries yet. Sessions write '
                      'auto-summaries on end, and the Append button '
                      'lets you add notes by hand.',
                      textAlign: TextAlign.center,
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                  ),
                ),
              if (logs.isNotEmpty)
                ListView.separated(
                  padding: const EdgeInsets.fromLTRB(8, 8, 8, 96),
                  itemCount: logs.length,
                  separatorBuilder: (_, __) => const SizedBox(height: 4),
                  itemBuilder: (_, i) => _LogTile(
                    entry: logs[i],
                    onDelete: () async {
                      await ref
                          .read(projectDocsApiProvider)
                          .deleteLog(logs[i].id);
                      if (mounted) await _loadAll(_selectedKey!);
                    },
                  ),
                ),
              Positioned(
                right: 16,
                bottom: 16,
                child: FloatingActionButton.extended(
                  heroTag: 'project_journal_fab',
                  onPressed: _openAppendJournal,
                  icon: const Icon(Icons.add),
                  label: Text(t.project.append),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _openAppendJournal() async {
    final cwd = _selectedKey;
    if (cwd == null) return;
    final titleCtl = TextEditingController();
    final bodyCtl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.project.appendDialogTitle),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: titleCtl,
                decoration: InputDecoration(
                  labelText: t.project.titleFieldLabel,
                ),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: bodyCtl,
                minLines: 3,
                maxLines: 8,
                decoration: InputDecoration(
                  labelText: t.project.contentFieldLabel,
                  border: const OutlineInputBorder(),
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.project.append),
          ),
        ],
      ),
    );
    if (ok != true) return;
    final content = bodyCtl.text.trim();
    if (content.isEmpty) return;
    try {
      await ref.read(projectDocsApiProvider).appendLog(
            cwd: cwd,
            content: content,
            title: titleCtl.text.trim(),
          );
      if (!mounted) return;
      await _loadAll(cwd);
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(t.project.appendFailed(error: e.toString())),
        ),
      );
    }
  }

  // _currentContentFor returns the live doc body for the given
  // kind (goal / plan) so the proposal card can render a before /
  // after diff. Returns empty string when nothing is loaded or the
  // kind has no doc yet — _DiffBlock will render "(not set)".
  String _currentContentFor(String kind) {
    final docs = _docs.asData?.value ?? const <ProjectDoc>[];
    for (final d in docs) {
      if (d.kind == kind) return d.content;
    }
    return '';
  }

  // ── Proposal inbox ─────────────────────────────────────────────

  Widget _inboxTab() {
    if (_selectedKey == null) {
      return Center(child: Text(t.project.pickFirst));
    }
    return _proposals.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) =>
          Center(child: Text(t.project.loadFailed(error: e.toString()))),
      data: (props) {
        if (props.isEmpty) {
          return RefreshIndicator(
            onRefresh: () async => _loadAll(_selectedKey!),
            child: ListView(
              children: [
                const SizedBox(height: 80),
                Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text(
                    'No pending proposals. Agents file these via the '
                    'opendray-memory MCP tools — they will land here for '
                    'your review before any goal / plan rewrite goes '
                    'live.',
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                ),
              ],
            ),
          );
        }
        return RefreshIndicator(
          onRefresh: () async => _loadAll(_selectedKey!),
          child: ListView.separated(
            padding: const EdgeInsets.all(8),
            itemCount: props.length,
            separatorBuilder: (_, __) => const SizedBox(height: 4),
            itemBuilder: (_, i) => _ProposalCard(
              proposal: props[i],
              currentContent: _currentContentFor(props[i].kind),
              onApprove: () async {
                try {
                  await ref
                      .read(projectDocsApiProvider)
                      .approveProposal(props[i].id);
                  if (mounted) await _loadAll(_selectedKey!);
                } on ApiException catch (e) {
                  if (mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(
                        content: Text(
                          t.project.approveFailed(error: e.toString()),
                        ),
                      ),
                    );
                    // Stale UI: the proposal was already decided
                    // (CLI / web / another phone). Refresh so this
                    // user stops seeing it as pending.
                    await _loadAll(_selectedKey!);
                  }
                }
              },
              onReject: () async {
                try {
                  await ref
                      .read(projectDocsApiProvider)
                      .rejectProposal(props[i].id);
                  if (mounted) await _loadAll(_selectedKey!);
                } on ApiException catch (e) {
                  if (mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(
                        content: Text(
                          t.project.rejectFailed(error: e.toString()),
                        ),
                      ),
                    );
                    await _loadAll(_selectedKey!);
                  }
                }
              },
            ),
          ),
        );
      },
    );
  }

  // ── Cleanup tab ───────────────────────────────────────────────

  Widget _cleanupTab() {
    if (_selectedKey == null) {
      return Center(child: Text(t.project.pickFirst));
    }
    return _cleanupDecisions.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) =>
          Center(child: Text(t.project.loadFailed(error: e.toString()))),
      data: (decisions) {
        return RefreshIndicator(
          onRefresh: () async => _loadAll(_selectedKey!),
          child: Stack(
            children: [
              if (decisions.isEmpty)
                ListView(
                  children: [
                    const SizedBox(height: 60),
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
                          const SizedBox(height: 12),
                          Text(
                            'No pending cleanup decisions.',
                            style: Theme.of(context).textTheme.titleMedium,
                            textAlign: TextAlign.center,
                          ),
                          const SizedBox(height: 8),
                          Text(
                            'Tap Run cleanup below to have the LLM '
                            "librarian review this project's memories and "
                            'propose deletions / merges. Each proposal '
                            'lands here for your approval.',
                            textAlign: TextAlign.center,
                            style: Theme.of(context).textTheme.bodyMedium,
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              if (decisions.isNotEmpty)
                ListView.separated(
                  padding: const EdgeInsets.fromLTRB(8, 8, 8, 96),
                  itemCount: decisions.length,
                  separatorBuilder: (_, __) => const SizedBox(height: 4),
                  itemBuilder: (_, i) => _CleanupCard(
                    decision: decisions[i],
                    onApprove: () => _approveCleanup(decisions[i].id),
                    onReject: () => _rejectCleanup(decisions[i].id),
                  ),
                ),
              Positioned(
                right: 16,
                bottom: 16,
                child: FloatingActionButton.extended(
                  heroTag: 'project_cleanup_fab',
                  onPressed: _cleanupRunning ? null : _runCleanup,
                  icon: _cleanupRunning
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.play_arrow_outlined),
                  label: Text(_cleanupRunning ? 'Running…' : 'Run cleanup'),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _runCleanup() async {
    final cwd = _selectedKey;
    if (cwd == null) return;
    setState(() => _cleanupRunning = true);
    try {
      final res = await ref.read(memoryCleanupApiProvider).run(
            scope: 'project',
            scopeKey: cwd,
          );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            'Cleanup run: ${res.memoriesIn} reviewed, '
            '${res.decisionsOut} decisions filed.',
          ),
        ),
      );
      await _loadAll(cwd);
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(t.project.cleanupFailed(error: e.toString())),
        ),
      );
    } finally {
      if (mounted) setState(() => _cleanupRunning = false);
    }
  }

  Future<void> _approveCleanup(String id) async {
    try {
      await ref.read(memoryCleanupApiProvider).approve(id);
      if (mounted) await _loadAll(_selectedKey!);
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(t.project.approveFailed(error: e.toString())),
          ),
        );
        // Stale UI — re-pull to show the real status.
        await _loadAll(_selectedKey!);
      }
    }
  }

  Future<void> _rejectCleanup(String id) async {
    try {
      await ref.read(memoryCleanupApiProvider).reject(id);
      if (mounted) await _loadAll(_selectedKey!);
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(t.project.rejectFailed(error: e.toString())),
          ),
        );
        await _loadAll(_selectedKey!);
      }
    }
  }

  /// Shows the reset confirmation dialog, then on confirm calls the
  /// reset endpoint + (optionally) deleteMemoriesByScope. Mirrors
  /// the web ResetButton flow in app/web.../ProjectScreen.tsx.
  Future<void> _confirmReset(String cwd) async {
    var includeScanner = false;
    var includeMemories = false;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) {
        return StatefulBuilder(
          builder: (ctx, setDialogState) {
            return AlertDialog(
              title: Text(t.project.resetConfirmTitle),
              content: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: Theme.of(ctx).colorScheme.errorContainer,
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Icon(Icons.warning_amber_rounded,
                            size: 18,
                            color: Theme.of(ctx).colorScheme.onErrorContainer),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              SelectableText(
                                cwd,
                                style: const TextStyle(
                                  fontFamily: 'monospace',
                                  fontSize: 11,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                              const SizedBox(height: 4),
                              const Text(
                                'Always deleted: goal, plan, proposals, '
                                'journal, cleanup decisions.',
                                style: TextStyle(fontSize: 11),
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 12),
                  CheckboxListTile(
                    contentPadding: EdgeInsets.zero,
                    dense: true,
                    value: includeScanner,
                    title: Text(t.project.alsoDeleteScanner,
                        style: const TextStyle(fontSize: 13)),
                    subtitle: const Text(
                      'tech_stack + recent_activity (auto-rebuild on next spawn).',
                      style: TextStyle(fontSize: 11),
                    ),
                    onChanged: (v) =>
                        setDialogState(() => includeScanner = v ?? false),
                  ),
                  CheckboxListTile(
                    contentPadding: EdgeInsets.zero,
                    dense: true,
                    value: includeMemories,
                    title: Text(t.project.alsoDeletePgvector,
                        style: const TextStyle(fontSize: 13)),
                    subtitle: const Text(
                      'Long-term facts for this scope_key. Cannot be recovered.',
                      style: TextStyle(fontSize: 11),
                    ),
                    onChanged: (v) =>
                        setDialogState(() => includeMemories = v ?? false),
                  ),
                ],
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.of(ctx).pop(false),
                  child: Text(t.common.cancel),
                ),
                FilledButton.tonal(
                  style: FilledButton.styleFrom(
                    backgroundColor: Theme.of(ctx).colorScheme.errorContainer,
                    foregroundColor: Theme.of(ctx).colorScheme.onErrorContainer,
                  ),
                  onPressed: () => Navigator.of(ctx).pop(true),
                  child: Text(t.project.deleteForever),
                ),
              ],
            );
          },
        );
      },
    );
    if (confirmed != true) return;
    try {
      final counts = await ref.read(projectDocsApiProvider).resetCwd(
            cwd: cwd,
            includeScannerDocs: includeScanner,
          );
      var memCount = 0;
      if (includeMemories) {
        memCount = await ref.read(memoryApiProvider).deleteByScope(
              scope: MemoryScope.project,
              scopeKey: cwd,
            );
      }
      if (!mounted) return;
      final parts = <String>[
        '${counts['project_docs'] ?? 0} doc',
        '${counts['session_logs'] ?? 0} journal',
        '${counts['memory_cleanup_decisions'] ?? 0} cleanup',
      ];
      if (includeMemories) parts.add('$memCount memories');
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            t.project.resetDoneSnack(parts: parts.join(' · ')),
          ),
        ),
      );
      await _loadAll(cwd);
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(t.project.resetFailed(error: e.toString())),
          ),
        );
      }
    }
  }
}

class _DocEditor extends StatefulWidget {
  const _DocEditor({
    required this.doc,
    required this.onSaved,
    super.key,
  });

  final ProjectDoc doc;
  final VoidCallback onSaved;

  @override
  State<_DocEditor> createState() => _DocEditorState();
}

class _DocEditorState extends State<_DocEditor> {
  late TextEditingController _ctl;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _ctl = TextEditingController(text: widget.doc.content);
  }

  @override
  void dispose() {
    _ctl.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    final api = ProviderScope.containerOf(context, listen: false)
        .read(projectDocsApiProvider);
    try {
      await api.putDoc(
        cwd: widget.doc.cwd,
        kind: widget.doc.kind,
        content: _ctl.text,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content:
              Text(t.project.docSavedSnack(kind: widget.doc.kind)),
        ),
      );
      widget.onSaved();
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(t.project.docSaveFailed(error: e.toString())),
        ),
      );
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (widget.doc.isPersisted)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                'last updated by ${widget.doc.updatedBy}',
                style: muted,
              ),
            ),
          Expanded(
            child: TextField(
              controller: _ctl,
              maxLines: null,
              expands: true,
              textAlignVertical: TextAlignVertical.top,
              keyboardType: TextInputType.multiline,
              decoration: InputDecoration(
                hintText: t.project.docHintTemplate(kind: widget.doc.kind),
                border: const OutlineInputBorder(),
              ),
            ),
          ),
          const SizedBox(height: 8),
          FilledButton.icon(
            onPressed: _saving ? null : _save,
            icon: _saving
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.save_outlined),
            label: Text(_saving ? 'Saving…' : 'Save ${widget.doc.kind}'),
          ),
        ],
      ),
    );
  }
}

class _LogTile extends StatelessWidget {
  const _LogTile({required this.entry, required this.onDelete});

  final SessionLogEntry entry;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Card(
      child: ListTile(
        title: Text(
          entry.title.isEmpty ? '(untitled)' : entry.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                entry.content,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 4),
              Text(
                '${entry.kind} · ${entry.updatedBy} · '
                '${DateFormat.yMMMd().add_jm().format(entry.createdAt.toLocal())}',
                style: muted,
              ),
            ],
          ),
        ),
        trailing: IconButton(
          icon: const Icon(Icons.delete_outline),
          tooltip: t.project.deleteEntryTooltip,
          onPressed: onDelete,
        ),
        onTap: () {
          showModalBottomSheet<void>(
            context: context,
            isScrollControlled: true,
            builder: (_) => DraggableScrollableSheet(
              expand: false,
              builder: (_, controller) => SingleChildScrollView(
                controller: controller,
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      entry.title.isEmpty ? '(untitled)' : entry.title,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 8),
                    Text(
                      '${entry.kind} · ${entry.updatedBy} · '
                      '${DateFormat.yMMMd().add_jm().format(entry.createdAt.toLocal())}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                    const SizedBox(height: 16),
                    SelectableText(entry.content),
                    const SizedBox(height: 24),
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

// _ProposalCard shows ONE pending goal/plan proposal. Critical UX
// decision: agents propose REPLACEMENT bodies (not patches), so an
// Approve click overwrites the live doc. To prevent operators
// shipping an "improvement" that drops half the existing plan, the
// card always shows both the current live doc and the proposed
// replacement side by side, and Approve goes through a confirm
// dialog with the same diff.
class _ProposalCard extends StatelessWidget {
  const _ProposalCard({
    required this.proposal,
    required this.currentContent,
    required this.onApprove,
    required this.onReject,
  });

  final DocProposal proposal;

  /// The live doc body for (cwd, proposal.kind) right now. Used to
  /// render the "Current" half of the before/after view so operators
  /// can decide whether the proposal genuinely extends the existing
  /// plan or accidentally drops content.
  final String currentContent;
  final VoidCallback onApprove;
  final VoidCallback onReject;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final scheme = Theme.of(context).colorScheme;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Chip(label: Text(proposal.kind)),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    'agent proposal · '
                    '${DateFormat.MMMd().add_jm().format(proposal.createdAt.toLocal())}',
                    style: muted,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            // Loud warning so the operator can't miss the semantics.
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: scheme.error.withValues(alpha: 0.08),
                border: Border.all(
                  color: scheme.error.withValues(alpha: 0.3),
                ),
                borderRadius: BorderRadius.circular(4),
              ),
              child: Row(
                children: [
                  Icon(Icons.warning_amber_outlined,
                      size: 16, color: scheme.error),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(
                      'Approve will REPLACE the current ${proposal.kind} '
                      'with this content — not merge.',
                      style: TextStyle(
                        fontSize: 12,
                        color: scheme.error,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 8),
            if (proposal.reason.isNotEmpty) ...[
              Text(t.project.agentReason, style: muted),
              const SizedBox(height: 2),
              Text(proposal.reason),
              const SizedBox(height: 8),
            ],
            _DiffBlock(
              label: 'Current ${proposal.kind}',
              body: currentContent,
              empty: '(not set)',
              tint: scheme.outline,
            ),
            const SizedBox(height: 6),
            _DiffBlock(
              label: 'After approve',
              body: proposal.proposedContent,
              empty: '(empty — approving would clear the doc)',
              tint: scheme.primary,
            ),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: onReject,
                  child: Text(t.project.reject),
                ),
                const SizedBox(width: 8),
                FilledButton(
                  onPressed: () => _confirmAndApprove(context),
                  child: Text(t.project.approve),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _confirmAndApprove(BuildContext context) async {
    final scheme = Theme.of(context).colorScheme;
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) {
        return AlertDialog(
          title: Text(t.project.replaceConfirmTitle(kind: proposal.kind)),
          content: SizedBox(
            width: double.maxFinite,
            child: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    'The agent proposed replacing the entire '
                    '${proposal.kind} body. Make sure the new content '
                    'covers everything you still want.',
                    style: Theme.of(ctx).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 12),
                  _DiffBlock(
                    label: 'Current',
                    body: currentContent,
                    empty: '(not set)',
                    tint: scheme.outline,
                    maxLines: 8,
                  ),
                  const SizedBox(height: 8),
                  _DiffBlock(
                    label: 'After approve',
                    body: proposal.proposedContent,
                    empty: '(empty)',
                    tint: scheme.primary,
                    maxLines: 8,
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: Text(t.common.cancel),
            ),
            FilledButton(
              onPressed: () => Navigator.of(ctx).pop(true),
              child: Text(t.project.replaceKind(kind: proposal.kind)),
            ),
          ],
        );
      },
    );
    if (ok ?? false) {
      onApprove();
    }
  }
}

class _DiffBlock extends StatelessWidget {
  const _DiffBlock({
    required this.label,
    required this.body,
    required this.empty,
    required this.tint,
    this.maxLines = 6,
  });

  final String label;
  final String body;
  final String empty;
  final Color tint;
  final int maxLines;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final trimmed = body.trim();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Container(
              width: 3,
              height: 12,
              color: tint,
              margin: const EdgeInsets.only(right: 6),
            ),
            Text(label,
                style: muted?.copyWith(fontWeight: FontWeight.w600)),
          ],
        ),
        const SizedBox(height: 4),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: tint.withValues(alpha: 0.05),
            border: Border.all(color: tint.withValues(alpha: 0.2)),
            borderRadius: BorderRadius.circular(4),
          ),
          child: Text(
            trimmed.isEmpty ? empty : trimmed,
            maxLines: maxLines,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              fontSize: 13,
              fontFamily: 'monospace',
              fontStyle: trimmed.isEmpty ? FontStyle.italic : null,
              color: trimmed.isEmpty ? muted?.color : null,
            ),
          ),
        ),
      ],
    );
  }
}

// _CleanupCard renders one pending memory_cleanup_decisions row.
// Color-codes the verdict so operators can skim a long list and
// approve "stale" / "duplicate" rows quickly without reading every
// reason field.
class _CleanupCard extends StatelessWidget {
  const _CleanupCard({
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
                Chip(
                  label: Text(
                    decision.verdict.toUpperCase(),
                    style: TextStyle(
                      fontSize: 11,
                      color: _verdictColor(context),
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  visualDensity: VisualDensity.compact,
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    decision.memoryId,
                    style: muted?.copyWith(
                      fontFamily: 'monospace',
                      fontFeatures: const [FontFeature.tabularFigures()],
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              decision.memoryTextSnapshot,
              maxLines: 4,
              overflow: TextOverflow.ellipsis,
            ),
            if (decision.reason.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(t.project.reason, style: muted),
              const SizedBox(height: 2),
              Text(decision.reason),
            ],
            if (decision.mergeInto.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(t.project.willMergeInto, style: muted),
              const SizedBox(height: 2),
              Text(
                decision.mergeInto,
                style: const TextStyle(fontFamily: 'monospace'),
              ),
            ],
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: onReject,
                  child: Text(t.project.reject),
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
