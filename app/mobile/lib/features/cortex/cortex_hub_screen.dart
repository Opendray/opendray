import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/cortex_api.dart';
import 'package:opendray/core/api/project_docs_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/knowledge/knowledge_screen.dart';
import 'package:opendray/features/memory/memory_screen.dart';
import 'package:opendray/features/memory_quarantine/quarantine_screen.dart';
import 'package:opendray/features/notes/notes_screen.dart';
import 'package:path/path.dart' as p;

const _globalCwd = '__global__';

// CortexHubScreen — the unified Memory → Notes → Knowledge flywheel hub,
// with status badges and a cross-project pending-proposals inbox.
// Web parity: app/web/src/pages/Cortex.tsx.
class CortexHubScreen extends ConsumerStatefulWidget {
  const CortexHubScreen({super.key});

  @override
  ConsumerState<CortexHubScreen> createState() => _CortexHubScreenState();
}

class _CortexHubScreenState extends ConsumerState<CortexHubScreen> {
  AsyncValue<CortexStatus> _status = const AsyncValue.loading();
  AsyncValue<List<DocProposal>> _proposals = const AsyncValue.loading();
  final _expanded = <String>{};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _status = const AsyncValue.loading();
      _proposals = const AsyncValue.loading();
    });
    try {
      final status = await ref.read(cortexApiProvider).status();
      if (mounted) setState(() => _status = AsyncValue.data(status));
    } on ApiException catch (e) {
      if (mounted) setState(() => _status = AsyncValue.error(e, StackTrace.current));
    }
    try {
      final props =
          await ref.read(projectDocsApiProvider).listPendingProposals();
      if (mounted) setState(() => _proposals = AsyncValue.data(props));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _proposals = AsyncValue.error(e, StackTrace.current));
      }
    }
  }

  Future<void> _approve(String id) async {
    try {
      await ref.read(projectDocsApiProvider).approveProposal(id);
      if (mounted) {
        _snack(t.cortexHub.approvedToast);
        await _load();
      }
    } on ApiException catch (e) {
      _snack(t.cortexHub.actionFailed(error: e.toString()));
    }
  }

  Future<void> _reject(String id) async {
    try {
      await ref.read(projectDocsApiProvider).rejectProposal(id);
      if (mounted) {
        _snack(t.cortexHub.rejectedToast);
        await _load();
      }
    } on ApiException catch (e) {
      _snack(t.cortexHub.actionFailed(error: e.toString()));
    }
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(msg), behavior: SnackBarBehavior.floating),
    );
  }

  void _push(Widget screen) {
    Navigator.of(context)
        .push(MaterialPageRoute<void>(builder: (_) => screen));
  }

  @override
  Widget build(BuildContext context) {
    final status = _status.valueOrNull;
    final proposals = _proposals.valueOrNull ?? const <DocProposal>[];
    return Scaffold(
      appBar: AppBar(
        title: Text(t.cortexHub.title),
        actions: [
          IconButton(
            tooltip: t.common.refresh,
            icon: const Icon(Icons.refresh),
            onPressed: _load,
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          padding: const EdgeInsets.all(12),
          children: [
            Text(t.cortexHub.subtitle,
                style: Theme.of(context).textTheme.bodySmall),
            const SizedBox(height: 12),
            _RungCard(
              icon: Icons.psychology_outlined,
              title: t.cortexHub.memory,
              description: t.cortexHub.memoryDesc,
              badges: [
                if (status != null && !status.memoryEnabled)
                  _Badge(t.cortexHub.disabled, _BadgeTone.muted),
                if (status != null && status.quarantineCount > 0)
                  _Badge(t.cortexHub.quarantineBadge(count: status.quarantineCount),
                      _BadgeTone.warning),
              ],
              onTap: () => _push(const MemoryScreen()),
            ),
            _RungCard(
              icon: Icons.menu_book_outlined,
              title: t.cortexHub.notes,
              description: t.cortexHub.notesDesc,
              badges: [
                if (status != null && status.notesPendingProposals > 0)
                  _Badge(t.cortexHub.pendingBadge(count: status.notesPendingProposals),
                      _BadgeTone.danger),
              ],
              onTap: () => _push(const NotesScreen()),
            ),
            _RungCard(
              icon: Icons.hub_outlined,
              title: t.cortexHub.knowledge,
              description: t.cortexHub.knowledgeDesc,
              badges: [
                if (status != null && !status.knowledgeEnabled)
                  _Badge(t.cortexHub.disabled, _BadgeTone.muted),
                if (status != null && status.knowledgePendingProposals > 0)
                  _Badge(
                      t.cortexHub.pendingBadge(count: status.knowledgePendingProposals),
                      _BadgeTone.danger),
              ],
              onTap: () => _push(const KnowledgeScreen()),
            ),
            if (status != null && status.quarantineCount > 0) ...[
              const SizedBox(height: 4),
              OutlinedButton.icon(
                onPressed: () => _push(const QuarantineScreen()),
                icon: const Icon(Icons.shield_outlined, size: 18),
                label: Text(
                    t.cortexHub.quarantineBadge(count: status.quarantineCount)),
              ),
            ],
            const SizedBox(height: 16),
            if (_proposals.hasError)
              Text(
                t.cortexHub.loadFailed(error: _proposals.error.toString()),
                style: Theme.of(context).textTheme.bodySmall,
              )
            else if (proposals.isNotEmpty) ...[
              Row(
                children: [
                  const Icon(Icons.inbox_outlined, size: 18),
                  const SizedBox(width: 6),
                  Text(
                    t.cortexHub.inboxTitle(count: proposals.length),
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                ],
              ),
              const SizedBox(height: 2),
              Text(t.cortexHub.inboxHint,
                  style: Theme.of(context).textTheme.bodySmall),
              const SizedBox(height: 8),
              for (final pr in proposals)
                _ProposalCard(
                  proposal: pr,
                  expanded: _expanded.contains(pr.id),
                  onToggle: () => setState(() {
                    _expanded.contains(pr.id)
                        ? _expanded.remove(pr.id)
                        : _expanded.add(pr.id);
                  }),
                  onApprove: () => _approve(pr.id),
                  onReject: () => _reject(pr.id),
                ),
            ],
          ],
        ),
      ),
    );
  }
}

class _RungCard extends StatelessWidget {
  const _RungCard({
    required this.icon,
    required this.title,
    required this.description,
    required this.badges,
    required this.onTap,
  });
  final IconData icon;
  final String title;
  final String description;
  final List<Widget> badges;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Row(
            children: [
              Icon(icon, color: Theme.of(context).colorScheme.primary),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Text(title,
                            style: Theme.of(context).textTheme.titleSmall),
                        const SizedBox(width: 8),
                        ...badges,
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(description,
                        style: Theme.of(context).textTheme.bodySmall),
                  ],
                ),
              ),
              const Icon(Icons.chevron_right),
            ],
          ),
        ),
      ),
    );
  }
}

enum _BadgeTone { muted, warning, danger }

class _Badge extends StatelessWidget {
  const _Badge(this.text, this.tone);
  final String text;
  final _BadgeTone tone;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final color = switch (tone) {
      _BadgeTone.muted => scheme.outline,
      _BadgeTone.warning => Colors.amber.shade700,
      _BadgeTone.danger => scheme.error,
    };
    return Container(
      margin: const EdgeInsets.only(right: 4),
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        border: Border.all(color: color.withValues(alpha: 0.4)),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(text,
          style: TextStyle(fontSize: 10, color: color, fontWeight: FontWeight.w600)),
    );
  }
}

class _ProposalCard extends StatelessWidget {
  const _ProposalCard({
    required this.proposal,
    required this.expanded,
    required this.onToggle,
    required this.onApprove,
    required this.onReject,
  });
  final DocProposal proposal;
  final bool expanded;
  final VoidCallback onToggle;
  final VoidCallback onApprove;
  final VoidCallback onReject;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final projectLabel = proposal.cwd == _globalCwd
        ? t.cortexHub.kbLabel
        : (p.basename(proposal.cwd).isEmpty
            ? proposal.cwd
            : p.basename(proposal.cwd));
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 10, 12, 6),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    _Badge(projectLabel, _BadgeTone.muted),
                    Expanded(
                      child: Text(proposal.kind,
                          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              fontFamily: 'monospace')),
                    ),
                    Text(
                      DateFormat.MMMd().format(proposal.createdAt.toLocal()),
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                ),
                if (proposal.reason.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 4),
                    child: Text(proposal.reason,
                        style: Theme.of(context).textTheme.bodySmall),
                  ),
              ],
            ),
          ),
          if (expanded)
            Container(
              margin: const EdgeInsets.fromLTRB(12, 0, 12, 8),
              padding: const EdgeInsets.all(8),
              constraints: const BoxConstraints(maxHeight: 240),
              decoration: BoxDecoration(
                color: scheme.surfaceContainerHighest.withValues(alpha: 0.3),
                borderRadius: BorderRadius.circular(8),
              ),
              child: SingleChildScrollView(
                child: Text(proposal.proposedContent,
                    style: const TextStyle(fontSize: 12, height: 1.4)),
              ),
            ),
          OverflowBar(
            alignment: MainAxisAlignment.end,
            children: [
              TextButton(
                onPressed: onToggle,
                child: Text(expanded ? t.cortexHub.hide : t.cortexHub.preview),
              ),
              TextButton(
                onPressed: onReject,
                style: TextButton.styleFrom(foregroundColor: scheme.error),
                child: Text(t.cortexHub.reject),
              ),
              FilledButton.tonal(
                onPressed: onApprove,
                child: Text(t.cortexHub.approve),
              ),
              const SizedBox(width: 8),
            ],
          ),
        ],
      ),
    );
  }
}
