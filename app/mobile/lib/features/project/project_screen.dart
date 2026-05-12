import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/memory_api.dart';
import 'package:opendray/core/api/memory_cleanup_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/project_docs_api.dart';
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
  const ProjectScreen({super.key});

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
    _tabs = TabController(length: 5, vsync: this);
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
      setState(() {
        _projectKeys = AsyncValue.data(keys);
        if (_selectedKey != null && !keys.contains(_selectedKey)) {
          _selectedKey = null;
        }
        _selectedKey ??= keys.isEmpty ? null : keys.first;
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
        child: Text('Failed to load projects: $e'),
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
              decoration: const InputDecoration(
                labelText: 'Project',
                border: OutlineInputBorder(),
                suffixIcon: Icon(Icons.unfold_more),
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
        title: const Text('Project'),
        bottom: TabBar(
          controller: _tabs,
          isScrollable: true,
          tabs: const [
            Tab(text: 'Goal'),
            Tab(text: 'Plan'),
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
      return const Center(child: Text('Pick a project first.'));
    }
    return _docs.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Text('Failed to load: $e'),
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

  // ── Journal ────────────────────────────────────────────────────

  Widget _journalTab() {
    if (_selectedKey == null) {
      return const Center(child: Text('Pick a project first.'));
    }
    return _logs.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text('Failed to load: $e')),
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
                  onPressed: _openAppendJournal,
                  icon: const Icon(Icons.add),
                  label: const Text('Append'),
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
        title: const Text('Append journal entry'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: titleCtl,
                decoration: const InputDecoration(
                  labelText: 'Title (optional)',
                ),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: bodyCtl,
                minLines: 3,
                maxLines: 8,
                decoration: const InputDecoration(
                  labelText: 'Content (markdown)',
                  border: OutlineInputBorder(),
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Append'),
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
        SnackBar(content: Text('Failed: $e')),
      );
    }
  }

  // ── Proposal inbox ─────────────────────────────────────────────

  Widget _inboxTab() {
    if (_selectedKey == null) {
      return const Center(child: Text('Pick a project first.'));
    }
    return _proposals.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text('Failed to load: $e')),
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
              onApprove: () async {
                try {
                  await ref
                      .read(projectDocsApiProvider)
                      .approveProposal(props[i].id);
                  if (mounted) await _loadAll(_selectedKey!);
                } on ApiException catch (e) {
                  if (mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text('Approve failed: $e')),
                    );
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
                      SnackBar(content: Text('Reject failed: $e')),
                    );
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
      return const Center(child: Text('Pick a project first.'));
    }
    return _cleanupDecisions.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text('Failed to load: $e')),
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
        SnackBar(content: Text('Cleanup failed: $e')),
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
          SnackBar(content: Text('Approve failed: $e')),
        );
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
          SnackBar(content: Text('Reject failed: $e')),
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
        SnackBar(content: Text('${widget.doc.kind} saved')),
      );
      widget.onSaved();
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Save failed: $e')),
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
                hintText: 'Write the ${widget.doc.kind} as markdown…',
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
          tooltip: 'Delete entry',
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

class _ProposalCard extends StatelessWidget {
  const _ProposalCard({
    required this.proposal,
    required this.onApprove,
    required this.onReject,
  });

  final DocProposal proposal;
  final VoidCallback onApprove;
  final VoidCallback onReject;

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
            const SizedBox(height: 8),
            if (proposal.reason.isNotEmpty) ...[
              Text('Reason', style: muted),
              const SizedBox(height: 2),
              Text(proposal.reason),
              const SizedBox(height: 8),
            ],
            Text('Proposed content', style: muted),
            const SizedBox(height: 2),
            Text(
              proposal.proposedContent,
              maxLines: 6,
              overflow: TextOverflow.ellipsis,
            ),
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
                  child: const Text('Approve'),
                ),
              ],
            ),
          ],
        ),
      ),
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
              Text('Reason', style: muted),
              const SizedBox(height: 2),
              Text(decision.reason),
            ],
            if (decision.mergeInto.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text('Will merge into', style: muted),
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
