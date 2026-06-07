import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/knowledge_api.dart';
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
      final nodes = await ref.read(knowledgeApiProvider).list();
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(t.web.knowledge.title)),
      body: Column(
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
