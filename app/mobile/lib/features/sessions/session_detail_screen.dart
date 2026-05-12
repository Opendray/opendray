import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/features/project/project_screen.dart';
import 'package:opendray/features/sessions/session_action_sheet.dart';
import 'package:opendray/features/sessions/session_terminal_view.dart';

// Session detail surface. The terminal eats most of the screen;
// metadata sits in a collapsible header that defaults to one
// compact row (provider + state badge), expanding on tap to show
// the long fields (cwd, timestamps). The connection-state line
// from earlier iterations is gone — its full strip now appears
// only when the WS is *not* connected (handled inside
// SessionTerminalView), so a healthy live session shows just a
// thin colored accent.
class SessionDetailScreen extends ConsumerStatefulWidget {
  const SessionDetailScreen({required this.sessionId, super.key});

  final String sessionId;

  @override
  ConsumerState<SessionDetailScreen> createState() =>
      _SessionDetailScreenState();
}

class _SessionDetailScreenState extends ConsumerState<SessionDetailScreen> {
  bool _metadataExpanded = false;

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(sessionByIdProvider(widget.sessionId));
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
            onPressed: () =>
                ref.invalidate(sessionByIdProvider(widget.sessionId)),
          ),
          IconButton(
            icon: const Icon(Icons.dashboard_customize_outlined),
            tooltip: 'Inspector (Files / Git / Tasks / History / Notes)',
            onPressed: () =>
                context.push('/session/${widget.sessionId}/inspector'),
          ),
          // Project memory shortcut — open the goal / plan / journal
          // / inbox view pre-filtered to this session's cwd. Saves
          // operators from going More → Project → picking the cwd by
          // hand.
          async.maybeWhen(
            data: (s) => IconButton(
              icon: const Icon(Icons.flag_outlined),
              tooltip: 'Project memory (goal / plan / journal / inbox)',
              onPressed: () {
                Navigator.of(context).push(
                  MaterialPageRoute<void>(
                    builder: (_) => ProjectScreen(initialCwd: s.cwd),
                  ),
                );
              },
            ),
            orElse: SizedBox.shrink,
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
                  ..invalidate(sessionByIdProvider(widget.sessionId))
                  ..invalidate(sessionsListProvider);
              },
            ),
            orElse: SizedBox.shrink,
          ),
        ],
      ),
      body: async.when(
        data: (session) => Column(
          children: [
            _MetadataHeader(
              session: session,
              expanded: _metadataExpanded,
              onToggle: () =>
                  setState(() => _metadataExpanded = !_metadataExpanded),
            ),
            Expanded(child: SessionTerminalView(sessionId: widget.sessionId)),
          ],
        ),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(
          error: e,
          onRetry: () =>
              ref.invalidate(sessionByIdProvider(widget.sessionId)),
        ),
      ),
    );
  }
}

class _MetadataHeader extends StatelessWidget {
  const _MetadataHeader({
    required this.session,
    required this.expanded,
    required this.onToggle,
  });

  final SessionSummary session;
  final bool expanded;
  final VoidCallback onToggle;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final muted = Theme.of(context).textTheme.bodySmall;
    return Material(
      color: scheme.surface,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Always-visible compact row.
          InkWell(
            onTap: onToggle,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(14, 6, 8, 6),
              child: Row(
                children: [
                  Text(
                    session.providerId,
                    style: muted?.copyWith(fontWeight: FontWeight.w600),
                  ),
                  const SizedBox(width: 8),
                  _StateBadge(state: session.state),
                  const Spacer(),
                  Icon(
                    expanded ? Icons.expand_less : Icons.expand_more,
                    color: scheme.onSurface.withValues(alpha: 0.6),
                    size: 20,
                  ),
                ],
              ),
            ),
          ),
          AnimatedSize(
            duration: const Duration(milliseconds: 180),
            curve: Curves.easeOut,
            alignment: Alignment.topCenter,
            child: expanded
                ? _ExpandedDetail(session: session)
                : const SizedBox(width: double.infinity),
          ),
          Divider(
            height: 1,
            thickness: 1,
            color: Theme.of(context).dividerColor,
          ),
        ],
      ),
    );
  }
}

class _ExpandedDetail extends StatelessWidget {
  const _ExpandedDetail({required this.session});
  final SessionSummary session;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final started =
        DateFormat.yMMMd().add_Hm().format(session.startedAt.toLocal());
    final ended = session.endedAt != null
        ? DateFormat.yMMMd().add_Hm().format(session.endedAt!.toLocal())
        : null;
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 0, 14, 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          SelectableText(session.cwd, style: muted),
          const SizedBox(height: 2),
          Text(
            ended == null
                ? 'started $started'
                : 'started $started  ·  ended $ended',
            style: muted,
          ),
          const SizedBox(height: 2),
          SelectableText(
            'id: ${session.id}',
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
