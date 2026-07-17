import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/cortex_api.dart';
import 'package:opendray/core/api/knowledge_api.dart';
import 'package:opendray/core/api/memory_workers_api.dart';
import 'package:opendray/core/api/project_docs_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/cortex/curation_chat_screen.dart';
import 'package:opendray/features/knowledge/force_graph.dart';

// Knowledge tab — read-mostly browser over the M-KG knowledge graph.
// Lists + semantic-searches nodes; tap opens a detail sheet with the
// body, connections, and promote / skillify actions.
class KnowledgeScreen extends ConsumerStatefulWidget {
  const KnowledgeScreen({super.key});

  @override
  ConsumerState<KnowledgeScreen> createState() => _KnowledgeScreenState();
}

class _KnowledgeScreenState extends ConsumerState<KnowledgeScreen> {
  String _view = 'kb';
  // Force-graph state — lazily loaded the first time the Graph tab opens.
  AsyncValue<KnowledgeGraphSnapshot>? _graph;
  String? _graphSelId;

  Future<void> _loadGraph() async {
    setState(() => _graph = const AsyncValue.loading());
    try {
      final snap = await ref.read(knowledgeApiProvider).graphAll();
      if (mounted) setState(() => _graph = AsyncValue.data(snap));
    } on Object catch (e, st) {
      if (mounted) setState(() => _graph = AsyncValue.error(e, st));
    }
  }

  Future<void> _openDetail(KnowledgeNode node) async {
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => _DetailSheet(node: node, onChanged: _loadGraph),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(t.web.knowledge.title)),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
            child: SegmentedButton<String>(
              showSelectedIcon: false,
              segments: [
                ButtonSegment(value: 'kb', label: Text(t.web.knowledge.kb.tab)),
                ButtonSegment(
                  value: 'distill',
                  label: Text(t.web.knowledge.distill.tab),
                ),
                ButtonSegment(
                  value: 'graph',
                  label: Text(t.web.knowledge.kb.graphTab),
                ),
              ],
              selected: {_view},
              onSelectionChanged: (s) {
                setState(() => _view = s.first);
                if (s.first == 'graph' && _graph == null) _loadGraph();
              },
            ),
          ),
          Expanded(
            child: switch (_view) {
              'kb' => const _KbView(),
              'distill' => const _DistillView(),
              _ => _graphView(context),
            },
          ),
        ],
      ),
    );
  }

  // Obsidian-style force-directed graph. Tap a node → its detail sheet;
  // pinch to zoom, drag to pan. Lazy-loaded via _loadGraph().
  Widget _graphView(BuildContext context) {
    final g = _graph;
    if (g == null || g.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (g.hasError) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('${g.error}', textAlign: TextAlign.center),
              const SizedBox(height: 12),
              FilledButton(
                onPressed: _loadGraph,
                child: Text(t.common.retry),
              ),
            ],
          ),
        ),
      );
    }
    final snap = g.value!;
    if (snap.nodes.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(t.web.knowledge.empty,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium),
        ),
      );
    }
    return Stack(
      children: [
        Positioned.fill(
          child: ForceGraphView(
            nodes: snap.nodes,
            edges: snap.edges,
            selectedId: _graphSelId,
            onSelect: (id) {
              setState(() => _graphSelId = id);
              if (id == null) return;
              final node = snap.nodes.firstWhere(
                (n) => n.id == id,
                orElse: () => snap.nodes.first,
              );
              _openDetail(node);
            },
          ),
        ),
        // Legend + node/edge count overlay.
        Positioned(
          left: 12,
          top: 12,
          child: _GraphLegend(
            nodeCount: snap.nodes.length,
            edgeCount: snap.edges.length,
          ),
        ),
        Positioned(
          right: 12,
          top: 12,
          child: Material(
            color: Colors.transparent,
            child: IconButton.filledTonal(
              tooltip: t.common.refresh,
              icon: const Icon(Icons.refresh, size: 18),
              onPressed: _loadGraph,
            ),
          ),
        ),
      ],
    );
  }
}

// Compact kind-color legend + counts, floated over the graph canvas.
class _GraphLegend extends StatelessWidget {
  const _GraphLegend({required this.nodeCount, required this.edgeCount});
  final int nodeCount;
  final int edgeCount;

  static const _legend = <String, Color>{
    'entity': Color(0xFF60A5FA),
    'project': Color(0xFFA78BFA),
    'playbook': Color(0xFFFBBF24),
    'skill': Color(0xFF34D399),
  };

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      decoration: BoxDecoration(
        color: scheme.surface.withValues(alpha: 0.82),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: scheme.outline.withValues(alpha: 0.3)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            t.web.knowledge.kb.graphCounts(nodes: nodeCount, edges: edgeCount),
            style: Theme.of(context).textTheme.labelSmall?.copyWith(
                  color: scheme.onSurface.withValues(alpha: 0.7),
                ),
          ),
          const SizedBox(height: 6),
          for (final e in _legend.entries)
            Padding(
              padding: const EdgeInsets.only(bottom: 2),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Container(
                    width: 9,
                    height: 9,
                    decoration:
                        BoxDecoration(color: e.value, shape: BoxShape.circle),
                  ),
                  const SizedBox(width: 6),
                  Text(e.key, style: Theme.of(context).textTheme.labelSmall),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

class _KindChip extends StatelessWidget {
  const _KindChip({required this.kind});
  final String kind;

  @override
  Widget build(BuildContext context) {
    return Chip(
      label: Text(kind, style: const TextStyle(fontSize: 10)),
      visualDensity: VisualDensity.compact,
      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
      padding: const EdgeInsets.symmetric(horizontal: 4),
    );
  }
}

// Secondary per-page actions, collapsed into a ⋮ overflow menu so the KB
// header stays compact on a phone screen (Edit is the visible primary).
enum _KbAction { discuss, unlock, regenerate, settings }

// _KbView — the curated Knowledge Base pages (the human-readable surface),
// fused with the note system (projectdoc kb_* docs). Global pages + per-project
// handbook; AI-drafted, human edit locks a page from AI overwrite.
class _KbView extends ConsumerStatefulWidget {
  const _KbView();

  @override
  ConsumerState<_KbView> createState() => _KbViewState();
}

class _KbViewState extends ConsumerState<_KbView> {
  static const _global = '__global__';
  static const _foundational = ['kb_infrastructure', 'kb_conventions'];
  static const _emergent = ['kb_lessons', 'kb_reusable'];
  String _kind = 'kb_infrastructure';
  final _editCtrl = TextEditingController();
  bool _editing = false;
  bool _busy = false;
  bool _showProposal = false;
  AsyncValue<ProjectDoc> _doc = const AsyncValue.loading();
  List<DocProposal> _proposals = const [];
  // Custom kb_* pages beyond the four classics, pulled from the global
  // blueprint so operator/AI-added pages show up + are selectable.
  List<BlueprintSection> _extra = const [];

  static const _classic = {
    'kb_infrastructure',
    'kb_conventions',
    'kb_lessons',
    'kb_reusable',
  };

  bool get _isFoundational =>
      _foundational.contains(_kind) ||
      _extra.any((s) => s.slug == _kind && s.nature == 'foundational');

  @override
  void initState() {
    super.initState();
    _load();
    _loadSections();
  }

  Future<void> _loadSections() async {
    try {
      final secs = await ref.read(cortexApiProvider).listSections(_global);
      if (!mounted) return;
      setState(() => _extra =
          secs.where((s) => !_classic.contains(s.slug)).toList()
            ..sort((a, b) => a.position.compareTo(b.position)));
    } on Object catch (_) {
      // Non-fatal: the four classics still render without the extras.
    }
  }

  // Create a new kb_* knowledge page (a global blueprint section), then
  // select it. Mirrors the web PageSettingsDialog (create mode).
  Future<void> _newPage() async {
    final created = await showDialog<BlueprintSection>(
      context: context,
      builder: (_) => const _KbPageDialog(),
    );
    if (created == null || !mounted) return;
    await _loadSections();
    if (!mounted) return;
    setState(() => _kind = created.slug);
    await _load();
  }

  // Edit the selected custom page's config (title/description/nature/inject).
  // Only offered for operator-added pages (the classic four aren't editable
  // here). Mirrors the web PageSettingsDialog (edit mode).
  Future<void> _editPageSettings(BlueprintSection section) async {
    final saved = await showDialog<BlueprintSection>(
      context: context,
      builder: (_) => _KbPageDialog(section: section),
    );
    if (saved == null || !mounted) return;
    await _loadSections();
    if (mounted) setState(() {});
  }

  // Dispatch a ⋮ overflow-menu action for the current page.
  void _onKbAction(_KbAction action, ProjectDoc d, BlueprintSection? custom) {
    switch (action) {
      case _KbAction.discuss:
        Navigator.of(context).push(
          MaterialPageRoute<void>(
            builder: (_) => CurationChatScreen(
              targetKind: 'kb_page',
              targetCwd: _global,
              targetSlug: _kind,
              onRevision: _load,
            ),
          ),
        );
      case _KbAction.unlock:
        _unlock(d);
      case _KbAction.regenerate:
        _regen();
      case _KbAction.settings:
        if (custom != null) _editPageSettings(custom);
    }
  }

  // Pick the cloud agent + model (+ Claude account) for the KB Librarian,
  // then launch the session and open it.
  Future<void> _launchLibrarian() async {
    final choice = await _LibrarianLaunchSheet.show(context);
    if (choice == null || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      final sid = await ref.read(cortexApiProvider).launchLibrarian(
            provider: choice.provider,
            model: choice.model,
            claudeAccountId: choice.account,
          );
      if (!mounted || sid.isEmpty) return;
      messenger.showSnackBar(
        SnackBar(content: Text(t.web.knowledge.kb.librarian.launchedToast)),
      );
      unawaited(context.push('/session/$sid'));
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text('${t.web.knowledge.actionFailed}: $e')),
      );
    }
  }

  @override
  void dispose() {
    _editCtrl.dispose();
    super.dispose();
  }

  String _stripSig(String s) =>
      s.split('\n').where((l) => !l.contains('kb-sig:')).join('\n').trim();

  String _kindLabel(String k) {
    switch (k) {
      case 'kb_conventions':
        return t.web.knowledge.kb.kinds.kb_conventions;
      case 'kb_lessons':
        return t.web.knowledge.kb.kinds.kb_lessons;
      case 'kb_reusable':
        return t.web.knowledge.kb.kinds.kb_reusable;
      case 'kb_infrastructure':
        return t.web.knowledge.kb.kinds.kb_infrastructure;
      default:
        // Custom page → its blueprint title (slug minus kb_ as fallback).
        for (final s in _extra) {
          if (s.slug == k) {
            return s.title.isNotEmpty ? s.title : k.replaceFirst('kb_', '');
          }
        }
        return k.replaceFirst('kb_', '');
    }
  }

  Future<void> _load() async {
    setState(() {
      _doc = const AsyncValue.loading();
      _editing = false;
      _showProposal = false;
    });
    try {
      final api = ref.read(projectDocsApiProvider);
      final d = await api.getDoc(_global, _kind);
      final props = await api.listPendingProposals(cwd: _global);
      if (mounted) {
        setState(() {
          _doc = AsyncValue.data(d);
          _proposals = props;
        });
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _doc = AsyncValue.error(e, st));
    }
  }

  DocProposal? get _pending {
    for (final p in _proposals) {
      if (p.kind == _kind) return p;
    }
    return null;
  }

  void _select(String kind) {
    setState(() => _kind = kind);
    _load();
  }

  Future<void> _save() async {
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    try {
      await ref
          .read(projectDocsApiProvider)
          .putDoc(cwd: _global, kind: _kind, content: _editCtrl.text);
      await _load();
      messenger.showSnackBar(SnackBar(content: Text(t.web.knowledge.kb.saved)));
    } on Object {
      messenger.showSnackBar(
        SnackBar(content: Text(t.web.knowledge.actionFailed)),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _unlock(ProjectDoc d) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(projectDocsApiProvider).putDoc(
            cwd: _global,
            kind: _kind,
            content: _stripSig(d.content),
            updatedBy: 'agent',
          );
      await _load();
      messenger.showSnackBar(
        SnackBar(content: Text(t.web.knowledge.kb.unlocked)),
      );
    } on Object {
      messenger.showSnackBar(
        SnackBar(content: Text(t.web.knowledge.actionFailed)),
      );
    }
  }

  Future<void> _decide(bool approve) async {
    final p = _pending;
    if (p == null) return;
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    try {
      final api = ref.read(projectDocsApiProvider);
      if (approve) {
        await api.approveProposal(p.id);
      } else {
        await api.rejectProposal(p.id);
      }
      await _load();
      messenger.showSnackBar(SnackBar(
        content: Text(approve
            ? t.web.knowledge.kb.proposal.approved
            : t.web.knowledge.kb.proposal.rejected),
      ));
    } on Object {
      messenger.showSnackBar(
        SnackBar(content: Text(t.web.knowledge.actionFailed)),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _regen() async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(knowledgeApiProvider).draftKb();
      messenger.showSnackBar(
        SnackBar(content: Text(t.web.knowledge.kb.regenerating)),
      );
    } on Object {
      messenger.showSnackBar(
        SnackBar(content: Text(t.web.knowledge.actionFailed)),
      );
    }
  }

  Widget _sectionLabel(String text) => Padding(
        padding: const EdgeInsets.only(right: 6, left: 2),
        child: Text(
          text,
          style: TextStyle(
            fontSize: 10,
            color: Theme.of(context).colorScheme.onSurfaceVariant,
          ),
        ),
      );

  Widget _chip(String k) => Padding(
        padding: const EdgeInsets.only(right: 6),
        child: ChoiceChip(
          label: Text(_kindLabel(k)),
          selected: _kind == k,
          onSelected: (_) => _select(k),
        ),
      );

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Column(
      children: [
        // Page navigation: a horizontal scroll of section chips (nav only),
        // with the two global actions (new page / Librarian) as trailing
        // icon buttons so their long labels don't crowd out the chips on a
        // narrow screen.
        SizedBox(
          height: 48,
          child: Row(
            children: [
              Expanded(
                child: SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  padding: const EdgeInsets.only(left: 12),
                  child: Row(
                    children: [
                      _sectionLabel(t.web.knowledge.kb.foundational),
                      for (final k in _foundational) _chip(k),
                      for (final s in _extra)
                        if (s.nature == 'foundational') _chip(s.slug),
                      _sectionLabel(t.web.knowledge.kb.emergent),
                      for (final k in _emergent) _chip(k),
                      for (final s in _extra)
                        if (s.nature != 'foundational') _chip(s.slug),
                    ],
                  ),
                ),
              ),
              IconButton(
                icon: const Icon(Icons.add, size: 22),
                tooltip: t.web.knowledge.kb.newPage.button,
                onPressed: _newPage,
              ),
              IconButton(
                icon: const Icon(Icons.auto_awesome, size: 20),
                tooltip: t.web.knowledge.kb.librarian.button,
                onPressed: _launchLibrarian,
              ),
              const SizedBox(width: 4),
            ],
          ),
        ),
        Expanded(
          child: _doc.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Text('$e', textAlign: TextAlign.center),
              ),
            ),
            data: (d) {
              final content = _stripSig(d.content);
              final locked = d.updatedBy == 'operator';
              final pending = _pending;
              // The custom (non-classic) page for the current selection, if any
              // — drives whether "Settings" is offered.
              BlueprintSection? custom;
              for (final s in _extra) {
                if (s.slug == _kind) {
                  custom = s;
                  break;
                }
              }
              return Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Compact action bar: the page's nature + lock badges on the
                  // left, then the primary Edit action + a ⋮ menu for the rest
                  // (Discuss / Unlock / Regenerate / Settings). Keeps everything
                  // on one line instead of a wrapping button pile.
                  Padding(
                    padding: const EdgeInsets.fromLTRB(12, 4, 4, 0),
                    child: Row(
                      children: [
                        Expanded(
                          child: Wrap(
                            spacing: 6,
                            runSpacing: 4,
                            crossAxisAlignment: WrapCrossAlignment.center,
                            children: [
                              Chip(
                                label: Text(
                                  _isFoundational
                                      ? t.web.knowledge.kb.bindingBadge
                                      : t.web.knowledge.kb.referenceBadge,
                                  style: const TextStyle(fontSize: 10),
                                ),
                                backgroundColor: _isFoundational
                                    ? scheme.tertiaryContainer
                                    : scheme.secondaryContainer,
                                visualDensity: VisualDensity.compact,
                                materialTapTargetSize:
                                    MaterialTapTargetSize.shrinkWrap,
                              ),
                              if (d.isPersisted)
                                Chip(
                                  label: Text(
                                    locked
                                        ? t.web.knowledge.kb.locked
                                        : t.web.knowledge.kb.aiDrafted,
                                    style: const TextStyle(fontSize: 10),
                                  ),
                                  visualDensity: VisualDensity.compact,
                                  materialTapTargetSize:
                                      MaterialTapTargetSize.shrinkWrap,
                                ),
                            ],
                          ),
                        ),
                        if (!_editing) ...[
                          TextButton(
                            onPressed: () {
                              _editCtrl.text = content;
                              setState(() => _editing = true);
                            },
                            child: Text(t.web.knowledge.kb.edit),
                          ),
                          PopupMenuButton<_KbAction>(
                            icon: const Icon(Icons.more_vert, size: 20),
                            tooltip: t.common.more,
                            onSelected: (a) => _onKbAction(a, d, custom),
                            itemBuilder: (_) => [
                              PopupMenuItem(
                                value: _KbAction.discuss,
                                child: Text(t.web.cortex.chat.show),
                              ),
                              if (locked)
                                PopupMenuItem(
                                  value: _KbAction.unlock,
                                  child: Text(t.web.knowledge.kb.unlock),
                                ),
                              PopupMenuItem(
                                value: _KbAction.regenerate,
                                child: Text(t.web.knowledge.kb.regenerate),
                              ),
                              if (custom != null)
                                PopupMenuItem(
                                  value: _KbAction.settings,
                                  child:
                                      Text(t.web.knowledge.kb.pageSettings.button),
                                ),
                            ],
                          ),
                        ],
                      ],
                    ),
                  ),
                  if (pending != null && !_editing)
                    Container(
                      margin: const EdgeInsets.fromLTRB(12, 6, 12, 0),
                      padding: const EdgeInsets.all(10),
                      decoration: BoxDecoration(
                        color: scheme.tertiaryContainer,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            t.web.knowledge.kb.proposal.text,
                            style: const TextStyle(fontSize: 12),
                          ),
                          Row(
                            children: [
                              TextButton(
                                onPressed: () => setState(
                                    () => _showProposal = !_showProposal),
                                child: Text(_showProposal
                                    ? t.web.knowledge.kb.proposal.hide
                                    : t.web.knowledge.kb.proposal.preview),
                              ),
                              const Spacer(),
                              TextButton(
                                onPressed: _busy ? null : () => _decide(false),
                                child: Text(t.web.knowledge.kb.proposal.reject),
                              ),
                              FilledButton(
                                onPressed: _busy ? null : () => _decide(true),
                                child: Text(t.web.knowledge.kb.proposal.approve),
                              ),
                            ],
                          ),
                          if (_showProposal)
                            Container(
                              constraints: const BoxConstraints(maxHeight: 240),
                              margin: const EdgeInsets.only(top: 6),
                              child: SingleChildScrollView(
                                child: SelectableText(
                                  _stripSig(pending.proposedContent),
                                  style: const TextStyle(fontSize: 12),
                                ),
                              ),
                            ),
                        ],
                      ),
                    ),
                  Expanded(
                    child: _editing
                        ? Padding(
                            padding: const EdgeInsets.all(12),
                            child: Column(
                              children: [
                                Expanded(
                                  child: TextField(
                                    controller: _editCtrl,
                                    maxLines: null,
                                    expands: true,
                                    textAlignVertical: TextAlignVertical.top,
                                    style: const TextStyle(
                                      fontFamily: 'monospace',
                                      fontSize: 13,
                                    ),
                                    decoration: const InputDecoration(
                                      border: OutlineInputBorder(),
                                      alignLabelWithHint: true,
                                    ),
                                  ),
                                ),
                                const SizedBox(height: 8),
                                Row(
                                  children: [
                                    FilledButton(
                                      onPressed: _busy ? null : _save,
                                      child: Text(t.web.knowledge.kb.save),
                                    ),
                                    const SizedBox(width: 8),
                                    TextButton(
                                      onPressed: () =>
                                          setState(() => _editing = false),
                                      child: Text(t.web.knowledge.kb.cancel),
                                    ),
                                  ],
                                ),
                              ],
                            ),
                          )
                        : SingleChildScrollView(
                            padding: const EdgeInsets.all(12),
                            child: SelectableText(
                              content.isEmpty
                                  ? t.web.knowledge.kb.empty
                                  : content,
                            ),
                          ),
                  ),
                ],
              );
            },
          ),
        ),
      ],
    );
  }
}


class _DetailSheet extends ConsumerWidget {
  const _DetailSheet({required this.node, required this.onChanged});
  final KnowledgeNode node;
  final Future<void> Function() onChanged;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final api = ref.read(knowledgeApiProvider);
    return DraggableScrollableSheet(
      expand: false,
      initialChildSize: 0.6,
      maxChildSize: 0.92,
      builder: (_, scroll) => SafeArea(
        top: false,
        child: ListView(
          controller: scroll,
          padding: const EdgeInsets.all(16),
          children: [
            Row(
              children: [
                _KindChip(kind: node.kind),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    node.title,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              '${node.scopeKey.isNotEmpty ? '${node.scope} · ${node.scopeKey}' : node.scope} · ${node.maturity}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
            if (node.body.isNotEmpty) ...[
              const SizedBox(height: 12),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(node.body),
              ),
            ],
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              children: [
                if (node.kind == 'playbook')
                  FilledButton.icon(
                    icon: const Icon(Icons.auto_awesome, size: 16),
                    label: Text(t.web.knowledge.skillify),
                    onPressed: () async {
                      final nav = Navigator.of(context);
                      final messenger = ScaffoldMessenger.of(context);
                      try {
                        final s = await api.skillify(node.id);
                        nav.pop();
                        await onChanged();
                        messenger.showSnackBar(
                          SnackBar(
                            content: Text(
                              t.web.knowledge.skillified(title: s.title),
                            ),
                          ),
                        );
                      } on Object {
                        messenger.showSnackBar(
                          SnackBar(content: Text(t.web.knowledge.actionFailed)),
                        );
                      }
                    },
                  ),
                if (node.scope != 'global')
                  OutlinedButton.icon(
                    icon: const Icon(Icons.public, size: 16),
                    label: Text(t.web.knowledge.promote),
                    onPressed: () async {
                      final nav = Navigator.of(context);
                      final messenger = ScaffoldMessenger.of(context);
                      try {
                        await api.promote(node.id);
                        nav.pop();
                        await onChanged();
                        messenger.showSnackBar(
                          SnackBar(content: Text(t.web.knowledge.promoted)),
                        );
                      } on Object {
                        messenger.showSnackBar(
                          SnackBar(content: Text(t.web.knowledge.actionFailed)),
                        );
                      }
                    },
                  ),
                OutlinedButton.icon(
                  icon: const Icon(Icons.delete_outline, size: 16),
                  label: Text(t.web.knowledge.delete),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: Theme.of(context).colorScheme.error,
                  ),
                  onPressed: () async {
                    final nav = Navigator.of(context);
                    final messenger = ScaffoldMessenger.of(context);
                    final ok = await showDialog<bool>(
                      context: context,
                      builder: (c) => AlertDialog(
                        content: Text(t.web.knowledge.deleteConfirm),
                        actions: [
                          TextButton(
                            onPressed: () => Navigator.pop(c, false),
                            child: Text(t.common.cancel),
                          ),
                          TextButton(
                            onPressed: () => Navigator.pop(c, true),
                            child: Text(t.web.knowledge.delete),
                          ),
                        ],
                      ),
                    );
                    if (ok != true) return;
                    try {
                      await api.delete(node.id);
                      nav.pop();
                      await onChanged();
                      messenger.showSnackBar(
                        SnackBar(content: Text(t.web.knowledge.deleted)),
                      );
                    } on Object {
                      messenger.showSnackBar(
                        SnackBar(content: Text(t.web.knowledge.actionFailed)),
                      );
                    }
                  },
                ),
              ],
            ),
            const SizedBox(height: 16),
            Text(
              t.web.knowledge.neighbors,
              style: Theme.of(context).textTheme.titleSmall,
            ),
            const SizedBox(height: 4),
            FutureBuilder<List<KnowledgeNeighbor>>(
              future: api.graph(node.id),
              builder: (_, snap) {
                if (snap.connectionState != ConnectionState.done) {
                  return const Padding(
                    padding: EdgeInsets.all(8),
                    child: LinearProgressIndicator(),
                  );
                }
                final ns = snap.data ?? const <KnowledgeNeighbor>[];
                if (ns.isEmpty) return const Text('—');
                return Column(
                  children: [
                    for (final nb in ns)
                      ListTile(
                        dense: true,
                        contentPadding: EdgeInsets.zero,
                        leading: Icon(
                          nb.direction == 'out'
                              ? Icons.arrow_forward
                              : Icons.arrow_back,
                          size: 16,
                        ),
                        title: Text(
                          nb.node.title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        subtitle: Text(
                          nb.edgeType,
                          style: Theme.of(context).textTheme.bodySmall,
                        ),
                      ),
                  ],
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}

// _KbPageDialog creates OR edits a kb_* knowledge page's config (a global
// blueprint section): slug (kb_ prefixed, fixed once created), title, optional
// description, nature (foundational/emergent), inject toggle. Mirrors the web
// PageSettingsDialog. Passing [section] switches it to edit mode; the page
// body is edited separately. Returns the saved section on success.
class _KbPageDialog extends ConsumerStatefulWidget {
  const _KbPageDialog({this.section});
  // Present = edit that section's settings; absent = create a new page.
  final BlueprintSection? section;

  @override
  ConsumerState<_KbPageDialog> createState() => _KbPageDialogState();
}

class _KbPageDialogState extends ConsumerState<_KbPageDialog> {
  late final BlueprintSection? _section = widget.section;
  bool get _editing => _section != null;
  late final _slug = TextEditingController(
    text: _section?.slug.replaceFirst('kb_', '') ?? '',
  );
  late final _title = TextEditingController(text: _section?.title ?? '');
  late final _desc = TextEditingController(text: _section?.description ?? '');
  late String _nature = _section?.nature == 'foundational'
      ? 'foundational'
      : 'emergent';
  late bool _inject = _section?.inject ?? false;
  bool _busy = false;

  static final _slugRe = RegExp(r'^kb_[a-z0-9][a-z0-9_]{0,44}$');

  @override
  void dispose() {
    _slug.dispose();
    _title.dispose();
    _desc.dispose();
    super.dispose();
  }

  bool get _valid =>
      _slugRe.hasMatch('kb_${_slug.text.trim()}') && _title.text.trim().isNotEmpty;

  Future<void> _save() async {
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final sec = await ref.read(cortexApiProvider).putBlueprintSection(
            BlueprintSection(
              cwd: '__global__',
              // The slug is the page's primary key — fixed once created.
              slug: _editing ? _section!.slug : 'kb_${_slug.text.trim()}',
              title: _title.text.trim(),
              description: _desc.text.trim(),
              // Preserve fields the config editor doesn't expose so an edit
              // never silently resets them; new pages get sensible defaults.
              position: _section?.position ?? 99,
              maintainerMode: _section?.maintainerMode ?? 'ai',
              writePolicy: _section?.writePolicy ?? 'proposal',
              promptHint: _section?.promptHint ?? '',
              pinned: _section?.pinned ?? false,
              inject: _inject,
              nature: _nature,
            ),
          );
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(_editing
              ? t.web.knowledge.kb.pageSettings.savedToast
              : t.web.knowledge.kb.newPage.createdToast),
        ),
      );
      Navigator.of(context).pop(sec);
    } on Object catch (e) {
      if (mounted) {
        setState(() => _busy = false);
        messenger.showSnackBar(
          SnackBar(content: Text('${t.web.knowledge.actionFailed}: $e')),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(_editing
          ? t.web.knowledge.kb.pageSettings.title
          : t.web.knowledge.kb.newPage.title),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
                _editing
                    ? t.web.knowledge.kb.pageSettings.description
                    : t.web.knowledge.kb.newPage.description,
                style: Theme.of(context).textTheme.bodySmall),
            const SizedBox(height: 12),
            Row(
              children: [
                const Text('kb_', style: TextStyle(fontFamily: 'monospace')),
                Expanded(
                  child: TextField(
                    controller: _slug,
                    autofocus: !_editing,
                    // The slug is fixed once created.
                    readOnly: _editing,
                    enabled: !_editing,
                    decoration: InputDecoration(
                      hintText: t.web.knowledge.kb.newPage.slugPlaceholder,
                      isDense: true,
                    ),
                    style: const TextStyle(fontFamily: 'monospace'),
                    onChanged: (_) => setState(() {}),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _title,
              decoration: InputDecoration(
                hintText: t.web.knowledge.kb.newPage.titlePlaceholder,
                isDense: true,
              ),
              onChanged: (_) => setState(() {}),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _desc,
              decoration: InputDecoration(
                hintText: t.web.knowledge.kb.newPage.descPlaceholder,
                isDense: true,
              ),
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              initialValue: _nature,
              decoration: const InputDecoration(isDense: true),
              items: [
                DropdownMenuItem(
                    value: 'foundational',
                    child: Text(t.web.knowledge.kb.foundational)),
                DropdownMenuItem(
                    value: 'emergent',
                    child: Text(t.web.knowledge.kb.emergent)),
              ],
              onChanged: (v) => setState(() => _nature = v ?? 'emergent'),
            ),
            const SizedBox(height: 4),
            CheckboxListTile(
              contentPadding: EdgeInsets.zero,
              dense: true,
              controlAffinity: ListTileControlAffinity.leading,
              value: _inject,
              onChanged: (v) => setState(() => _inject = v ?? false),
              title: Text(t.web.knowledge.kb.newPage.inject),
              subtitle: Text(t.web.knowledge.kb.newPage.injectHint,
                  style: Theme.of(context).textTheme.bodySmall),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.of(context).pop(),
          child: Text(t.web.knowledge.kb.cancel),
        ),
        FilledButton(
          onPressed: (_valid && !_busy) ? _save : null,
          child: Text(_editing
              ? t.web.knowledge.kb.pageSettings.save
              : t.web.knowledge.kb.newPage.create),
        ),
      ],
    );
  }
}

// _LibrarianLaunchSheet picks the cloud agent + model (+ Claude account) that
// backs the KB Librarian session. Mirrors the web LaunchLibrarianDialog.
typedef LibrarianChoice = ({String provider, String model, String account});

class _LibrarianLaunchSheet extends ConsumerStatefulWidget {
  const _LibrarianLaunchSheet();

  static Future<LibrarianChoice?> show(BuildContext context) {
    return showModalBottomSheet<LibrarianChoice>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => const _LibrarianLaunchSheet(),
    );
  }

  @override
  ConsumerState<_LibrarianLaunchSheet> createState() =>
      _LibrarianLaunchSheetState();
}

class _LibrarianLaunchSheetState
    extends ConsumerState<_LibrarianLaunchSheet> {
  // Every worker-backed CLI. grok/opencode use a single host login, so only
  // claude shows an account picker.
  static const _providers = <({String id, String label})>[
    (id: 'claude', label: 'Claude'),
    (id: 'codex', label: 'Codex'),
    (id: 'antigravity', label: 'Antigravity'),
    (id: 'grok', label: 'Grok'),
    (id: 'opencode', label: 'OpenCode'),
  ];
  String _provider = 'claude';
  String _model = '';
  String _account = '';
  List<ModelOption> _models = const [];

  @override
  void initState() {
    super.initState();
    _loadModels();
  }

  Future<void> _loadModels() async {
    try {
      final m =
          await ref.read(memoryWorkersApiProvider).listAgentModels(_provider);
      if (mounted) setState(() => _models = m);
    } on Object {
      if (mounted) setState(() => _models = const []);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final accounts = ref.watch(claudeAccountsListProvider).asData?.value
            .where((a) => a.enabled)
            .toList() ??
        const [];
    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(t.web.knowledge.kb.librarian.title,
                  style: theme.textTheme.titleMedium),
              const SizedBox(height: 4),
              Text(t.web.knowledge.kb.librarian.dialogHint,
                  style: theme.textTheme.bodySmall),
              const SizedBox(height: 16),
              DropdownButtonFormField<String>(
                initialValue: _provider,
                isExpanded: true,
                decoration: InputDecoration(
                  labelText: t.web.knowledge.kb.librarian.provider,
                  isDense: true,
                  border: const OutlineInputBorder(),
                ),
                items: [
                  for (final p in _providers)
                    DropdownMenuItem(value: p.id, child: Text(p.label)),
                ],
                onChanged: (v) {
                  if (v == null) return;
                  setState(() {
                    _provider = v;
                    _model = '';
                    _account = '';
                    _models = const [];
                  });
                  _loadModels();
                },
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: _models.any((m) => m.id == _model) ? _model : '',
                isExpanded: true,
                decoration: InputDecoration(
                  labelText: t.web.cortex.chat.modelLabel,
                  isDense: true,
                  border: const OutlineInputBorder(),
                ),
                items: [
                  DropdownMenuItem(
                      value: '', child: Text(t.web.cortex.chat.modelCliDefault)),
                  for (final m in _models)
                    DropdownMenuItem(value: m.id, child: Text(m.label)),
                ],
                onChanged: (v) => setState(() => _model = v ?? ''),
              ),
              if (_provider == 'claude' && accounts.isNotEmpty) ...[
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  initialValue:
                      accounts.any((a) => a.id == _account) ? _account : '',
                  isExpanded: true,
                  decoration: InputDecoration(
                    labelText: t.web.knowledge.kb.librarian.account,
                    isDense: true,
                    border: const OutlineInputBorder(),
                  ),
                  items: [
                    DropdownMenuItem(
                        value: '',
                        child: Text(t.web.cortex.chat.accountDefault)),
                    for (final a in accounts)
                      DropdownMenuItem(
                          value: a.id, child: Text(a.displayName)),
                  ],
                  onChanged: (v) => setState(() => _account = v ?? ''),
                ),
              ],
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () => Navigator.of(context).pop((
                    provider: _provider,
                    model: _model,
                    account: _provider == 'claude' ? _account : '',
                  )),
                  child: Text(t.web.knowledge.kb.librarian.launch),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ─── Distillation workbench ───────────────────────────────────────
//
// Playbooks (distilled candidates) → promote to Skills (injected into
// every spawn). Web parity: app/web/src/pages/Knowledge.tsx DistillationView.
class _DistillView extends ConsumerStatefulWidget {
  const _DistillView();

  @override
  ConsumerState<_DistillView> createState() => _DistillViewState();
}

class _DistillViewState extends ConsumerState<_DistillView> {
  AsyncValue<List<KnowledgeNode>> _playbooks = const AsyncValue.loading();
  AsyncValue<List<KnowledgeNode>> _skills = const AsyncValue.loading();
  AsyncValue<List<RetirementCandidate>> _retirement =
      const AsyncValue.loading();
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _playbooks = const AsyncValue.loading();
      _skills = const AsyncValue.loading();
    });
    final api = ref.read(knowledgeApiProvider);
    try {
      final pb = await api.list(kind: 'playbook');
      if (mounted) setState(() => _playbooks = AsyncValue.data(pb));
    } on Object catch (e, st) {
      if (mounted) setState(() => _playbooks = AsyncValue.error(e, st));
    }
    try {
      final sk = await api.list(kind: 'skill');
      if (mounted) setState(() => _skills = AsyncValue.data(sk));
    } on Object catch (e, st) {
      if (mounted) setState(() => _skills = AsyncValue.error(e, st));
    }
    try {
      final rc = await api.retirementCandidates();
      if (mounted) setState(() => _retirement = AsyncValue.data(rc));
    } on Object catch (e, st) {
      if (mounted) setState(() => _retirement = AsyncValue.error(e, st));
    }
  }

  Future<void> _disableCandidate(String id) async {
    setState(() => _busy = true);
    try {
      await ref.read(knowledgeApiProvider).setEnabled(id, enabled: false);
      _snack(t.web.knowledge.distill.disabledToast);
    } on ApiException catch (_) {
      _snack(t.web.knowledge.actionFailed);
    } finally {
      if (mounted) {
        setState(() => _busy = false);
        await _load();
      }
    }
  }

  Future<void> _skillify(String id) async {
    setState(() => _busy = true);
    try {
      await ref.read(knowledgeApiProvider).skillify(id);
      _snack(t.web.knowledge.distill.skillifiedToast);
    } on ApiException catch (e) {
      _snack(t.web.knowledge.actionFailed);
      _snack(e.message);
    } finally {
      if (mounted) {
        setState(() => _busy = false);
        await _load();
      }
    }
  }

  Future<void> _toggle(KnowledgeNode n) async {
    setState(() => _busy = true);
    try {
      final next = !n.enabled;
      await ref.read(knowledgeApiProvider).setEnabled(n.id, enabled: next);
      _snack(next
          ? t.web.knowledge.distill.enabledToast
          : t.web.knowledge.distill.disabledToast);
    } on ApiException catch (_) {
      _snack(t.web.knowledge.actionFailed);
    } finally {
      if (mounted) {
        setState(() => _busy = false);
        await _load();
      }
    }
  }

  Future<void> _remove(String id) async {
    setState(() => _busy = true);
    try {
      await ref.read(knowledgeApiProvider).delete(id);
      _snack(t.web.knowledge.distill.removedToast);
    } on ApiException catch (_) {
      _snack(t.web.knowledge.actionFailed);
    } finally {
      if (mounted) {
        setState(() => _busy = false);
        await _load();
      }
    }
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(msg), behavior: SnackBarBehavior.floating),
    );
  }

  void _preview(KnowledgeNode n) {
    final body = n.body.replaceFirst(RegExp(r'^---\n[\s\S]*?\n---\n?'), '');
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => DraggableScrollableSheet(
        expand: false,
        initialChildSize: 0.7,
        maxChildSize: 0.95,
        builder: (_, controller) => Padding(
          padding: const EdgeInsets.all(16),
          child: ListView(
            controller: controller,
            children: [
              Text(n.title,
                  style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 12),
              SelectableText(body,
                  style: const TextStyle(fontSize: 13, height: 1.5)),
            ],
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        padding: const EdgeInsets.all(12),
        children: [
          Text(t.web.knowledge.distill.intro,
              style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 16),
          _SectionHeader(
            title: t.web.knowledge.distill.playbooks,
            hint: t.web.knowledge.distill.playbooksHint,
            count: _playbooks.valueOrNull?.length,
          ),
          _playbooks.when(
            loading: () => const Padding(
                padding: EdgeInsets.all(16),
                child: Center(child: CircularProgressIndicator())),
            error: (e, _) => Padding(
                padding: const EdgeInsets.all(8), child: Text(e.toString())),
            data: (rows) => rows.isEmpty
                ? _empty(t.web.knowledge.distill.playbooksEmpty)
                : Column(
                    children: [
                      for (final n in rows)
                        _PlaybookCard(
                          node: n,
                          busy: _busy,
                          onPreview: () => _preview(n),
                          onSkillify: () => _skillify(n.id),
                          onDiscard: () => _remove(n.id),
                        ),
                    ],
                  ),
          ),
          const SizedBox(height: 16),
          _SectionHeader(
            title: t.web.knowledge.distill.skills,
            hint: t.web.knowledge.distill.skillsHint,
            count: _skills.valueOrNull?.length,
          ),
          _skills.when(
            loading: () => const Padding(
                padding: EdgeInsets.all(16),
                child: Center(child: CircularProgressIndicator())),
            error: (e, _) => Padding(
                padding: const EdgeInsets.all(8), child: Text(e.toString())),
            data: (rows) => rows.isEmpty
                ? _empty(t.web.knowledge.distill.skillsEmpty)
                : Column(
                    children: [
                      for (final n in rows)
                        _SkillCard(
                          node: n,
                          busy: _busy,
                          onPreview: () => _preview(n),
                          onToggle: () => _toggle(n),
                          onRetire: () => _remove(n.id),
                        ),
                    ],
                  ),
          ),
          const SizedBox(height: 16),
          _SectionHeader(
            title: t.web.knowledge.distill.retirementTitle,
            hint: t.web.knowledge.distill.retirementHint,
            count: _retirement.valueOrNull?.length,
          ),
          _retirement.when(
            loading: () => const Padding(
                padding: EdgeInsets.all(16),
                child: Center(child: CircularProgressIndicator())),
            error: (e, _) => Padding(
                padding: const EdgeInsets.all(8), child: Text(e.toString())),
            data: (rows) => rows.isEmpty
                ? _empty(t.web.knowledge.distill.retirementEmpty)
                : Column(
                    children: [
                      for (final c in rows)
                        _RetirementCard(
                          candidate: c,
                          busy: _busy,
                          onPreview: () => _preview(c.node),
                          onDisable: () => _disableCandidate(c.node.id),
                        ),
                    ],
                  ),
          ),
        ],
      ),
    );
  }

  Widget _empty(String text) => Padding(
        padding: const EdgeInsets.all(16),
        child: Text(text,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodySmall),
      );
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.title, required this.hint, this.count});
  final String title;
  final String hint;
  final int? count;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(title, style: Theme.of(context).textTheme.titleSmall),
            if (count != null) ...[
              const SizedBox(width: 6),
              Text('$count', style: Theme.of(context).textTheme.bodySmall),
            ],
          ],
        ),
        const SizedBox(height: 2),
        Text(hint, style: Theme.of(context).textTheme.bodySmall),
        const SizedBox(height: 6),
      ],
    );
  }
}

class _PlaybookCard extends StatelessWidget {
  const _PlaybookCard({
    required this.node,
    required this.busy,
    required this.onPreview,
    required this.onSkillify,
    required this.onDiscard,
  });
  final KnowledgeNode node;
  final bool busy;
  final VoidCallback onPreview;
  final VoidCallback onSkillify;
  final VoidCallback onDiscard;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final summary = node.provenance['summary'];
    final recurrence = (node.provenance['recurrence'] as num?)?.toInt() ?? 0;
    final estMinutes = (node.provenance['est_minutes'] as num?)?.toInt() ?? 0;
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            InkWell(
              onTap: onPreview,
              child: Text(node.title,
                  style: Theme.of(context).textTheme.titleSmall),
            ),
            if (recurrence > 0)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(
                  '${t.web.knowledge.distill.recurrence(count: recurrence)} · '
                  '${t.web.knowledge.distill.timeCost(minutes: estMinutes)}',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ),
            if (summary is String && summary.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(summary,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodySmall),
              ),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: busy ? null : onDiscard,
                  style: TextButton.styleFrom(foregroundColor: scheme.error),
                  child: Text(t.web.knowledge.distill.discard),
                ),
                const SizedBox(width: 8),
                FilledButton.tonal(
                  onPressed: busy ? null : onSkillify,
                  child: Text(t.web.knowledge.distill.skillify),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _SkillCard extends StatelessWidget {
  const _SkillCard({
    required this.node,
    required this.busy,
    required this.onPreview,
    required this.onToggle,
    required this.onRetire,
  });
  final KnowledgeNode node;
  final bool busy;
  final VoidCallback onPreview;
  final VoidCallback onToggle;
  final VoidCallback onRetire;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final outcomes = node.successCount + node.failureCount;
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Opacity(
        opacity: node.enabled ? 1 : 0.6,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: InkWell(
                      onTap: onPreview,
                      child: Text(node.title,
                          style: Theme.of(context).textTheme.titleSmall),
                    ),
                  ),
                  Container(
                    margin: const EdgeInsets.only(right: 8),
                    padding:
                        const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                    decoration: BoxDecoration(
                      color: (node.enabled ? Colors.green : scheme.outline)
                          .withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      node.enabled
                          ? t.web.knowledge.distill.injectedBadge
                          : t.web.knowledge.distill.disabledBadge,
                      style: TextStyle(
                          fontSize: 10,
                          color: node.enabled ? Colors.green : scheme.outline),
                    ),
                  ),
                  Switch(
                    value: node.enabled,
                    onChanged: busy ? null : (_) => onToggle(),
                  ),
                ],
              ),
              Text(
                t.web.knowledge.distill.usage(count: node.useCount) +
                    (outcomes > 0
                        ? ' · ${t.web.knowledge.distill.outcomes(ok: node.successCount, failed: node.failureCount)}'
                        : ''),
                style: Theme.of(context).textTheme.bodySmall,
              ),
              Align(
                alignment: Alignment.centerLeft,
                child: TextButton(
                  onPressed: busy ? null : onRetire,
                  style: TextButton.styleFrom(foregroundColor: scheme.error),
                  child: Text(t.web.knowledge.distill.retire),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

String _retireReasonLabel(String reason) => switch (reason) {
      'never_used' => t.web.knowledge.distill.retirement.never_used,
      'low_success' => t.web.knowledge.distill.retirement.low_success,
      'dormant' => t.web.knowledge.distill.retirement.dormant,
      _ => reason,
    };

String _retireReasonHint(String reason) => switch (reason) {
      'never_used' => t.web.knowledge.distill.retirement.never_usedHint,
      'low_success' => t.web.knowledge.distill.retirement.low_successHint,
      'dormant' => t.web.knowledge.distill.retirement.dormantHint,
      _ => '',
    };

// _RetirementCard surfaces a skill the outcome loop flags for retirement:
// the WHY (reason badge + hint) plus a one-tap disable. The operator can
// also preview the body before deciding.
class _RetirementCard extends StatelessWidget {
  const _RetirementCard({
    required this.candidate,
    required this.busy,
    required this.onPreview,
    required this.onDisable,
  });
  final RetirementCandidate candidate;
  final bool busy;
  final VoidCallback onPreview;
  final VoidCallback onDisable;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final node = candidate.node;
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: InkWell(
                    onTap: onPreview,
                    child: Text(node.title,
                        style: Theme.of(context).textTheme.titleSmall),
                  ),
                ),
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                  decoration: BoxDecoration(
                    color: Colors.amber.withValues(alpha: 0.15),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    _retireReasonLabel(candidate.reason),
                    style: const TextStyle(fontSize: 10, color: Colors.amber),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              _retireReasonHint(candidate.reason),
              style: Theme.of(context).textTheme.bodySmall,
            ),
            Align(
              alignment: Alignment.centerLeft,
              child: TextButton(
                onPressed: busy ? null : onDisable,
                style: TextButton.styleFrom(foregroundColor: scheme.error),
                child: Text(t.web.knowledge.distill.retire),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
