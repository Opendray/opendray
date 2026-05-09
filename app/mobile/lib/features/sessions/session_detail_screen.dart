import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/features/sessions/session_action_sheet.dart';
import 'package:opendray/features/sessions/session_terminal_view.dart';

// Session detail surface. Top: a compact metadata strip + Actions
// menu. Body: the live terminal (xterm.dart over WebSocket). When
// the user mutates state via the action sheet (stop / restart /
// delete) we invalidate the watched providers so the strip and
// terminal both reflect the new state immediately.
class SessionDetailScreen extends ConsumerWidget {
  const SessionDetailScreen({required this.sessionId, super.key});

  final String sessionId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(sessionByIdProvider(sessionId));
    return Scaffold(
      appBar: AppBar(
        title: async.when(
          data: (s) => Text(
            s.displayName,
            overflow: TextOverflow.ellipsis,
          ),
          loading: () => const Text('Session'),
          error: (_, __) => const Text('Session'),
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh metadata',
            onPressed: () => ref.invalidate(sessionByIdProvider(sessionId)),
          ),
          async.maybeWhen(
            data: (s) => IconButton(
              icon: const Icon(Icons.tune),
              tooltip: 'Actions',
              onPressed: () async {
                final result = await SessionActionSheet.show(
                  context,
                  session: s,
                );
                if (result == null) return;
                if (!context.mounted) return;
                if (result == SessionActionResult.deleted) {
                  ref.invalidate(sessionsListProvider);
                  context.pop();
                  return;
                }
                ref
                  ..invalidate(sessionByIdProvider(sessionId))
                  ..invalidate(sessionsListProvider);
              },
            ),
            orElse: SizedBox.shrink,
          ),
        ],
      ),
      body: async.when(
        data: (session) => _Body(session: session, sessionId: sessionId),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(
          error: e,
          onRetry: () => ref.invalidate(sessionByIdProvider(sessionId)),
        ),
      ),
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({required this.session, required this.sessionId});

  final SessionSummary session;
  final String sessionId;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        _MetadataStrip(session: session),
        Expanded(child: SessionTerminalView(sessionId: sessionId)),
      ],
    );
  }
}

class _MetadataStrip extends StatelessWidget {
  const _MetadataStrip({required this.session});
  final SessionSummary session;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(14, 8, 14, 8),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border(
          bottom: BorderSide(color: Theme.of(context).dividerColor),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  session.providerId,
                  style: muted?.copyWith(fontWeight: FontWeight.w600),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              const SizedBox(width: 8),
              _StateBadge(state: session.state),
            ],
          ),
          const SizedBox(height: 2),
          Text(session.cwd, style: muted, overflow: TextOverflow.ellipsis),
          const SizedBox(height: 2),
          Text(
            'started ${DateFormat.yMMMd().add_Hm().format(session.startedAt.toLocal())}'
            '${session.endedAt != null ? '  ·  ended ${DateFormat.yMMMd().add_Hm().format(session.endedAt!.toLocal())}' : ''}',
            style: muted,
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
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: bg.withValues(alpha: 0.45),
        border: Border.all(color: fg.withValues(alpha: 0.4)),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        state.wire,
        style: TextStyle(
          color: fg,
          fontSize: 10,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.5,
        ),
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
