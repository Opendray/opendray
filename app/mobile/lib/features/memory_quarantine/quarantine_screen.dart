import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/cortex_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:path/path.dart' as p;

// QuarantineScreen — the review queue for memories that need vetting
// before they count as durable: third-party-integration captures land
// here by policy, and the operator can quarantine any memory by hand
// from the Memory inspector. Promote what's true; discard the rest —
// unreviewed rows expire on their own. Web parity:
// app/web/src/components/cortex/QuarantinePanel.tsx.
class QuarantineScreen extends ConsumerStatefulWidget {
  const QuarantineScreen({super.key});

  @override
  ConsumerState<QuarantineScreen> createState() => _QuarantineScreenState();
}

class _QuarantineScreenState extends ConsumerState<QuarantineScreen> {
  AsyncValue<List<Memory>> _rows = const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _rows = const AsyncValue.loading());
    try {
      final (rows, _) = await ref.read(cortexApiProvider).listQuarantined();
      if (!mounted) return;
      setState(() => _rows = AsyncValue.data(rows));
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _rows = AsyncValue.error(e, StackTrace.current));
    }
  }

  Future<void> _promote(String id) async {
    try {
      await ref.read(cortexApiProvider).promoteQuarantined(id);
      if (mounted) {
        _snack(t.memoryQuarantine.promotedToast);
        await _load();
      }
    } on ApiException catch (e) {
      _snack(t.memoryQuarantine.actionFailed(error: e.toString()));
    }
  }

  Future<void> _discard(String id) async {
    try {
      await ref.read(cortexApiProvider).discardQuarantined(id);
      if (mounted) {
        _snack(t.memoryQuarantine.discardedToast);
        await _load();
      }
    } on ApiException catch (e) {
      _snack(t.memoryQuarantine.actionFailed(error: e.toString()));
    }
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(msg), behavior: SnackBarBehavior.floating),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.memoryQuarantine.title),
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
            child: Text(t.memoryQuarantine.loadFailed(error: e.toString())),
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
                          Icons.shield_outlined,
                          size: 48,
                          color: Theme.of(context)
                              .colorScheme
                              .onSurface
                              .withValues(alpha: 0.4),
                        ),
                        const SizedBox(height: 16),
                        Text(
                          t.memoryQuarantine.empty,
                          textAlign: TextAlign.center,
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            );
          }
          return RefreshIndicator(
            onRefresh: _load,
            child: ListView.builder(
              padding: const EdgeInsets.fromLTRB(8, 8, 8, 16),
              itemCount: rows.length + 1,
              itemBuilder: (_, i) {
                if (i == 0) {
                  return Padding(
                    padding: const EdgeInsets.fromLTRB(8, 4, 8, 8),
                    child: Text(
                      t.memoryQuarantine.subtitle,
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  );
                }
                final m = rows[i - 1];
                return _QuarantineCard(
                  memory: m,
                  onPromote: () => _promote(m.id),
                  onDiscard: () => _discard(m.id),
                );
              },
            ),
          );
        },
      ),
    );
  }
}

class _QuarantineCard extends StatelessWidget {
  const _QuarantineCard({
    required this.memory,
    required this.onPromote,
    required this.onDiscard,
  });

  final Memory memory;
  final VoidCallback onPromote;
  final VoidCallback onDiscard;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final muted = Theme.of(context).textTheme.bodySmall;
    final integrationId = memory.metadata?['integration_id']?.toString();
    final scopeLabel = memory.scope == MemoryScope.global
        ? 'global'
        : (p.basename(memory.scopeKey).isEmpty
            ? memory.scopeKey
            : p.basename(memory.scopeKey));
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Wrap(
              spacing: 8,
              runSpacing: 4,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [
                _chip(context, scopeLabel, scheme.primary),
                if (integrationId != null && integrationId.isNotEmpty)
                  _chip(context, integrationId, scheme.secondary),
                if (memory.quarantineExpiresAt != null)
                  Text(
                    t.memoryQuarantine.expires(
                      date: DateFormat.MMMd()
                          .format(memory.quarantineExpiresAt!.toLocal()),
                    ),
                    style: muted,
                  ),
              ],
            ),
            const SizedBox(height: 8),
            Text(memory.text, maxLines: 6, overflow: TextOverflow.ellipsis),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton.icon(
                  onPressed: onDiscard,
                  style: TextButton.styleFrom(foregroundColor: scheme.error),
                  icon: const Icon(Icons.delete_outline, size: 18),
                  label: Text(t.memoryQuarantine.discard),
                ),
                const SizedBox(width: 8),
                FilledButton.tonalIcon(
                  onPressed: onPromote,
                  icon: const Icon(Icons.check, size: 18),
                  label: Text(t.memoryQuarantine.promote),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _chip(BuildContext context, String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        border: Border.all(color: color.withValues(alpha: 0.4)),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        text,
        style: TextStyle(fontSize: 10, color: color, fontWeight: FontWeight.w600),
      ),
    );
  }
}
