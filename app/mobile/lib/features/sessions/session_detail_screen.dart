import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/features/sessions/session_action_sheet.dart';

// F2 placeholder for the session detail surface. Renders metadata
// + state badge + Stop / Restart / Delete actions, but the actual
// terminal stream lands in F3 once xterm.dart is wired up.
//
// This screen also handles the case where the session was deleted
// (or stopped, or just-spawned) by listening to mutations on
// sessionByIdProvider — we stay live without manual refresh.
class SessionDetailScreen extends ConsumerWidget {
  const SessionDetailScreen({required this.sessionId, super.key});

  final String sessionId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(sessionByIdProvider(sessionId));
    return Scaffold(
      appBar: AppBar(
        title: const Text('Session'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () => ref.invalidate(sessionByIdProvider(sessionId)),
          ),
        ],
      ),
      body: async.when(
        data: (session) => _Body(
          session: session,
          onChanged: () {
            ref
              ..invalidate(sessionByIdProvider(sessionId))
              ..invalidate(sessionsListProvider);
          },
          onDeleted: () {
            ref.invalidate(sessionsListProvider);
            if (context.mounted) context.pop();
          },
        ),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e, onRetry: () {
          ref.invalidate(sessionByIdProvider(sessionId));
        }),
      ),
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({
    required this.session,
    required this.onChanged,
    required this.onDeleted,
  });

  final SessionSummary session;
  final VoidCallback onChanged;
  final VoidCallback onDeleted;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _MetadataCard(session: session),
        const SizedBox(height: 16),
        _ActionBar(
          session: session,
          onChanged: onChanged,
          onDeleted: onDeleted,
        ),
        const SizedBox(height: 24),
        _TerminalPlaceholder(),
      ],
    );
  }
}

class _MetadataCard extends StatelessWidget {
  const _MetadataCard({required this.session});
  final SessionSummary session;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    session.displayName,
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                _StateBadge(state: session.state),
              ],
            ),
            const SizedBox(height: 12),
            _InfoRow(label: 'Provider', value: session.providerId),
            _InfoRow(label: 'Working dir', value: session.cwd),
            _InfoRow(label: 'Session id', value: session.id),
            _InfoRow(
              label: 'Started',
              value: DateFormat.yMMMd()
                  .add_Hm()
                  .format(session.startedAt.toLocal()),
            ),
            if (session.endedAt != null)
              _InfoRow(
                label: 'Ended',
                value: DateFormat.yMMMd()
                    .add_Hm()
                    .format(session.endedAt!.toLocal()),
              ),
          ],
        ),
      ),
    );
  }
}

class _ActionBar extends ConsumerWidget {
  const _ActionBar({
    required this.session,
    required this.onChanged,
    required this.onDeleted,
  });

  final SessionSummary session;
  final VoidCallback onChanged;
  final VoidCallback onDeleted;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Row(
      children: [
        Expanded(
          child: OutlinedButton.icon(
            onPressed: () async {
              final result = await SessionActionSheet.show(
                context,
                session: session,
              );
              if (result == null) return;
              if (result == SessionActionResult.deleted) {
                onDeleted();
              } else {
                onChanged();
              }
            },
            icon: const Icon(Icons.tune),
            label: const Text('Actions'),
            style: OutlinedButton.styleFrom(
              padding: const EdgeInsets.symmetric(vertical: 14),
            ),
          ),
        ),
      ],
    );
  }
}

class _TerminalPlaceholder extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border.all(color: Theme.of(context).dividerColor),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        children: [
          Icon(
            Icons.terminal_outlined,
            size: 48,
            color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.3),
          ),
          const SizedBox(height: 12),
          Text(
            'Terminal view',
            style: Theme.of(context).textTheme.titleSmall,
          ),
          const SizedBox(height: 4),
          Text(
            'xterm.dart wiring lands in F3 — for now the session '
            'is fully manageable from the Actions sheet above.',
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ],
      ),
    );
  }
}

class _StateBadge extends StatelessWidget {
  const _StateBadge({required this.state});
  final SessionState state;

  @override
  Widget build(BuildContext context) {
    final (bg, fg) = switch (state) {
      SessionState.running => (Colors.green.shade900, Colors.greenAccent),
      SessionState.idle => (Colors.amber.shade900, Colors.amberAccent),
      SessionState.pending => (Colors.grey.shade800, Colors.grey.shade300),
      SessionState.stopped ||
      SessionState.ended =>
        (Colors.grey.shade800, Colors.grey.shade400),
      SessionState.unknown => (Colors.grey.shade800, Colors.grey.shade400),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: bg.withValues(alpha: 0.45),
        border: Border.all(color: fg.withValues(alpha: 0.4)),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        state.wire,
        style: TextStyle(
          color: fg,
          fontSize: 11,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.5,
        ),
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({required this.label, required this.value});
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 92,
            child: Text(
              label,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
          Expanded(
            child: SelectableText(
              value,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final Object error;
  final VoidCallback onRetry;

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
              'Failed to load session',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              error.toString(),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: const Text('Retry')),
          ],
        ),
      ),
    );
  }
}
