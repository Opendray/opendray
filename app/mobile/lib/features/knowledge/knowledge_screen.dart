import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/knowledge_api.dart';
import 'package:opendray/core/api/project_docs_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Knowledge tab — read-mostly browser over the M-KG knowledge graph.
// Lists + semantic-searches nodes; tap opens a detail sheet with the
// body, connections, and promote / skillify actions.
class KnowledgeScreen extends ConsumerStatefulWidget {
  const KnowledgeScreen({super.key});

  @override
  ConsumerState<KnowledgeScreen> createState() => _KnowledgeScreenState();
}

class _KnowledgeScreenState extends ConsumerState<KnowledgeScreen> {
  AsyncValue<List<KnowledgeNode>> _state = const AsyncValue.loading();
  final _searchCtrl = TextEditingController();
  bool _searching = false;
  String _kind = 'entity';
  String _scope = 'all';
  String _view = 'kb';

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _state = const AsyncValue.loading();
      _searching = false;
    });
    try {
      final nodes = await ref
          .read(knowledgeApiProvider)
          .list(
            kind: _kind == 'all' ? null : _kind,
            scope: _scope == 'all' ? null : _scope,
          );
      if (!mounted) return;
      setState(() => _state = AsyncValue.data(nodes));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _runSearch() async {
    final q = _searchCtrl.text.trim();
    if (q.isEmpty) {
      await _load();
      return;
    }
    setState(() => _state = const AsyncValue.loading());
    try {
      final hits = await ref.read(knowledgeApiProvider).search(query: q);
      if (!mounted) return;
      setState(() {
        _state = AsyncValue.data(hits.map((h) => h.node).toList());
        _searching = true;
      });
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
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
      builder: (_) => _DetailSheet(node: node, onChanged: _load),
    );
  }

  String _kindLabel(String k) {
    switch (k) {
      case 'entity':
        return t.web.knowledge.kinds.entity;
      case 'fact':
        return t.web.knowledge.kinds.fact;
      case 'playbook':
        return t.web.knowledge.kinds.playbook;
      case 'skill':
        return t.web.knowledge.kinds.skill;
      default:
        return t.web.knowledge.kinds.all;
    }
  }

  String _scopeLabel(String s) {
    switch (s) {
      case 'global':
        return t.web.knowledge.scopes.global;
      case 'project':
        return t.web.knowledge.scopes.project;
      case 'domain':
        return t.web.knowledge.scopes.domain;
      default:
        return t.web.knowledge.scopes.all;
    }
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
              onSelectionChanged: (s) => setState(() => _view = s.first),
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

  Widget _graphView(BuildContext context) {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(12),
          child: TextField(
            controller: _searchCtrl,
            textInputAction: TextInputAction.search,
            onSubmitted: (_) => _runSearch(),
            decoration: InputDecoration(
              hintText: t.web.knowledge.searchPlaceholder,
              prefixIcon: const Icon(Icons.search),
              suffixIcon: _searching
                  ? IconButton(
                      icon: const Icon(Icons.clear),
                      onPressed: () {
                        _searchCtrl.clear();
                        _load();
                      },
                    )
                  : null,
              border: const OutlineInputBorder(),
              isDense: true,
            ),
          ),
        ),
        SizedBox(
          height: 44,
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(
              children: [
                // 'fact' retired (P-G): facts live in Memory, not the graph.
                for (final k in const [
                  'all',
                  'entity',
                  'playbook',
                  'skill',
                ])
                  Padding(
                    padding: const EdgeInsets.only(right: 6),
                    child: ChoiceChip(
                      label: Text(_kindLabel(k)),
                      selected: _kind == k,
                      onSelected: (_) {
                        setState(() => _kind = k);
                        if (!_searching) _load();
                      },
                    ),
                  ),
              ],
            ),
          ),
        ),
        SizedBox(
          height: 44,
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(
              children: [
                Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: Center(child: Text(t.web.knowledge.scope)),
                ),
                for (final s in const ['all', 'global', 'project', 'domain'])
                  Padding(
                    padding: const EdgeInsets.only(right: 6),
                    child: ChoiceChip(
                      label: Text(_scopeLabel(s)),
                      selected: _scope == s,
                      onSelected: (_) {
                        setState(() => _scope = s);
                        if (!_searching) _load();
                      },
                    ),
                  ),
              ],
            ),
          ),
        ),
        Expanded(
          child: _state.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Text('$e', textAlign: TextAlign.center),
              ),
            ),
            data: (nodes) => nodes.isEmpty
                ? Center(
                    child: Padding(
                      padding: const EdgeInsets.all(24),
                      child: Text(
                        _searching
                            ? t.web.knowledge.noResults
                            : t.web.knowledge.empty,
                        textAlign: TextAlign.center,
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                    ),
                  )
                : RefreshIndicator(
                    onRefresh: _load,
                    child: ListView.separated(
                      itemCount: nodes.length,
                      separatorBuilder: (_, __) => const Divider(height: 1),
                      itemBuilder: (_, i) {
                        final n = nodes[i];
                        return ListTile(
                          leading: _KindChip(kind: n.kind),
                          title: Text(
                            n.title,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                          ),
                          subtitle: Text(
                            n.scopeKey.isNotEmpty
                                ? '${n.scope} · ${n.scopeKey}'
                                : n.scope,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                          onTap: () => _openDetail(n),
                        );
                      },
                    ),
                  ),
          ),
        ),
      ],
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

  bool get _isFoundational => _foundational.contains(_kind);

  @override
  void initState() {
    super.initState();
    _load();
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
      default:
        return t.web.knowledge.kb.kinds.kb_infrastructure;
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
        SizedBox(
          height: 44,
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(
              children: [
                _sectionLabel(t.web.knowledge.kb.foundational),
                for (final k in _foundational) _chip(k),
                _sectionLabel(t.web.knowledge.kb.emergent),
                for (final k in _emergent) _chip(k),
              ],
            ),
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
              return Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Padding(
                    padding: const EdgeInsets.fromLTRB(12, 4, 12, 0),
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
                        if (!_editing) ...[
                          TextButton(
                            onPressed: () {
                              _editCtrl.text = content;
                              setState(() => _editing = true);
                            },
                            child: Text(t.web.knowledge.kb.edit),
                          ),
                          if (locked)
                            TextButton(
                              onPressed: () => _unlock(d),
                              child: Text(t.web.knowledge.kb.unlock),
                            ),
                          TextButton(
                            onPressed: _regen,
                            child: Text(t.web.knowledge.kb.regenerate),
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
