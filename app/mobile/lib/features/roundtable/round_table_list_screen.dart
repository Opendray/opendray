import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/widgets/brand_avatar.dart';
import 'package:opendray/features/roundtable/create_round_table_sheet.dart';
import 'package:opendray/features/roundtable/round_table_detail_screen.dart';

// Round Table list — mobile parity with
// app/web/src/components/roundtable/RoundTablePanel.tsx. Master list of
// cross-vendor group chats; FAB opens the create sheet; tapping a row opens
// the chat.
class RoundTableListScreen extends ConsumerWidget {
  const RoundTableListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final tables = ref.watch(roundTablesProvider);
    return Scaffold(
      appBar: AppBar(
        title: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(t.web.roundTable.title),
            const SizedBox(width: 8),
            _ExperimentalBadge(),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _openCreate(context, ref),
        icon: const Icon(Icons.add),
        label: Text(t.web.roundTable.kNew),
      ),
      body: tables.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorState(
          message: t.web.roundTable.detail.loadFailed,
          onRetry: () => ref.invalidate(roundTablesProvider),
        ),
        data: (rows) {
          if (rows.isEmpty) {
            return _EmptyState(
              onCreate: () => _openCreate(context, ref),
            );
          }
          return RefreshIndicator(
            onRefresh: () async => ref.invalidate(roundTablesProvider),
            child: ListView.separated(
              padding: const EdgeInsets.all(12),
              itemCount: rows.length,
              separatorBuilder: (_, __) => const SizedBox(height: 8),
              itemBuilder: (_, i) => _RoundTableCard(
                table: rows[i],
                onTap: () => _open(context, rows[i].id),
              ),
            ),
          );
        },
      ),
    );
  }

  Future<void> _openCreate(BuildContext context, WidgetRef ref) async {
    final created = await CreateRoundTableSheet.show(context);
    if (created == null || !context.mounted) return;
    ref.invalidate(roundTablesProvider);
    _open(context, created.id);
  }

  void _open(BuildContext context, String id) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => RoundTableDetailScreen(id: id),
      ),
    );
  }
}

class _RoundTableCard extends StatelessWidget {
  const _RoundTableCard({required this.table, required this.onTap});
  final RoundTable table;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final title =
        table.topic.isNotEmpty ? table.topic : t.web.roundTable.untitled;
    return Card(
      clipBehavior: Clip.antiAlias,
      child: ListTile(
        onTap: onTap,
        title: Text(title, maxLines: 1, overflow: TextOverflow.ellipsis),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 6),
          child: Row(
            children: [
              for (final s in table.seats)
                Padding(
                  padding: const EdgeInsets.only(right: 4),
                  child: BrandAvatar(providerId: s.provider, size: 20),
                ),
              const Spacer(),
              _StatusChip(status: table.status),
            ],
          ),
        ),
        trailing: Icon(
          Icons.chevron_right,
          color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
        ),
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.status});
  final String status;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final active = status == 'active';
    final color = active ? theme.colorScheme.primary : theme.colorScheme.outline;
    final label = active
        ? t.web.roundTable.status.active
        : t.web.roundTable.status.closed;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: color.withValues(alpha: 0.4)),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(color: color),
      ),
    );
  }
}

class _ExperimentalBadge extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: theme.colorScheme.tertiary.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        t.web.roundTable.experimental,
        style: theme.textTheme.labelSmall?.copyWith(
          color: theme.colorScheme.tertiary,
        ),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.onCreate});
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.groups_outlined,
              size: 48,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.3)),
          const SizedBox(height: 12),
          Text(t.web.roundTable.empty, style: theme.textTheme.bodyMedium),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: onCreate,
            icon: const Icon(Icons.add),
            label: Text(t.web.roundTable.kNew),
          ),
        ],
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.message, required this.onRetry});
  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(message),
          const SizedBox(height: 12),
          OutlinedButton(onPressed: onRetry, child: const Text('Retry')),
        ],
      ),
    );
  }
}
