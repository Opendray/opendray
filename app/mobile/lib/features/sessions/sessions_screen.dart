import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/features/sessions/session_action_sheet.dart';
import 'package:opendray/features/sessions/spawn_session_sheet.dart';
import 'package:path/path.dart' as p;

// Sessions home — full CRUD over the multi-CLI session pool.
//   • Filter chips above the list narrow by lifecycle state.
//   • FAB (bottom-right) opens the spawn-new-session sheet.
//   • Tap a card → /session/:id (detail / actions / future
//     terminal view).
//   • Long-press OR "..." menu on a card → state-aware action
//     sheet (Stop / Restart / Delete).
class SessionsScreen extends ConsumerStatefulWidget {
  const SessionsScreen({super.key});

  @override
  ConsumerState<SessionsScreen> createState() => _SessionsScreenState();
}

class _SessionsScreenState extends ConsumerState<SessionsScreen> {
  _Filter _filter = _Filter.all;

  Future<void> _onSpawn() async {
    final created = await SpawnSessionSheet.show(context);
    if (!mounted) return;
    if (created != null) {
      ref.invalidate(sessionsListProvider);
      unawaited(context.push('/session/${created.id}'));
    }
  }

  Future<void> _onAction(SessionSummary session) async {
    final result = await SessionActionSheet.show(context, session: session);
    if (!mounted) return;
    if (result != null) {
      ref.invalidate(sessionsListProvider);
    }
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(sessionsListProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Sessions'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: async.isLoading
                ? null
                : () => ref.invalidate(sessionsListProvider),
          ),
        ],
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(48),
          child: _FilterBar(
            value: _filter,
            onChanged: (f) => setState(() => _filter = f),
          ),
        ),
      ),
      body: RefreshIndicator(
        onRefresh: () async => ref.refresh(sessionsListProvider.future),
        child: async.when(
          data: (sessions) {
            final visible =
                sessions.where((s) => _filter.matches(s)).toList();
            if (visible.isEmpty) {
              return _EmptyState(
                hasAny: sessions.isNotEmpty,
                filter: _filter,
              );
            }
            return ListView.separated(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 96),
              itemCount: visible.length,
              separatorBuilder: (_, __) => const SizedBox(height: 8),
              itemBuilder: (_, i) {
                final s = visible[i];
                return _SessionCard(
                  session: s,
                  onTap: () => context.push('/session/${s.id}'),
                  onLongPress: () => _onAction(s),
                  onMore: () => _onAction(s),
                );
              },
            );
          },
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) => _ErrorView(
            error: e,
            onRetry: () => ref.invalidate(sessionsListProvider),
          ),
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _onSpawn,
        icon: const Icon(Icons.add),
        label: const Text('Spawn'),
      ),
    );
  }
}

enum _Filter {
  all('All'),
  running('Running'),
  idle('Idle'),
  ended('Ended');

  const _Filter(this.label);
  final String label;

  bool matches(SessionSummary s) => switch (this) {
        _Filter.all => true,
        _Filter.running =>
          s.state == SessionState.running || s.state == SessionState.pending,
        _Filter.idle => s.state == SessionState.idle,
        _Filter.ended =>
          s.state == SessionState.stopped || s.state == SessionState.ended,
      };
}

class _FilterBar extends StatelessWidget {
  const _FilterBar({required this.value, required this.onChanged});
  final _Filter value;
  final ValueChanged<_Filter> onChanged;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 48,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        itemCount: _Filter.values.length,
        separatorBuilder: (_, __) => const SizedBox(width: 8),
        itemBuilder: (_, i) {
          final f = _Filter.values[i];
          final selected = f == value;
          return ChoiceChip(
            label: Text(f.label),
            selected: selected,
            onSelected: (_) => onChanged(f),
          );
        },
      ),
    );
  }
}

class _SessionCard extends StatelessWidget {
  const _SessionCard({
    required this.session,
    required this.onTap,
    required this.onLongPress,
    required this.onMore,
  });

  final SessionSummary session;
  final VoidCallback onTap;
  final VoidCallback onLongPress;
  final VoidCallback onMore;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: InkWell(
        onTap: onTap,
        onLongPress: onLongPress,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(14, 12, 6, 12),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            _projectTitle(session),
                            style: Theme.of(context)
                                .textTheme
                                .titleSmall
                                ?.copyWith(fontWeight: FontWeight.w600),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const SizedBox(width: 8),
                        _StateBadge(state: session.state),
                      ],
                    ),
                    const SizedBox(height: 6),
                    Text(
                      _shortCwd(session.cwd),
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                            fontFamily: 'monospace',
                            fontSize: 11,
                          ),
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '${session.providerId} · started ${_formatRelative(session.startedAt)}',
                      style: Theme.of(context).textTheme.bodySmall,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ),
              ),
              IconButton(
                icon: const Icon(Icons.more_vert),
                tooltip: 'Actions',
                onPressed: onMore,
              ),
            ],
          ),
        ),
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

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.hasAny, required this.filter});
  final bool hasAny;
  final _Filter filter;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.3);
    return ListView(
      padding: const EdgeInsets.fromLTRB(24, 80, 24, 24),
      children: [
        Icon(Icons.terminal_outlined, size: 64, color: muted),
        const SizedBox(height: 16),
        Center(
          child: Text(
            hasAny
                ? 'No sessions match the "${filter.label}" filter.'
                : 'No sessions yet',
            style: Theme.of(context).textTheme.titleMedium,
          ),
        ),
        const SizedBox(height: 4),
        Center(
          child: Text(
            hasAny
                ? 'Try a different filter or pull to refresh.'
                : 'Tap the Spawn button to create one.',
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ),
      ],
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.fromLTRB(24, 80, 24, 24),
      children: [
        Icon(
          Icons.error_outline,
          size: 48,
          color: Theme.of(context).colorScheme.error,
        ),
        const SizedBox(height: 12),
        Center(
          child: Text(
            'Failed to load sessions',
            style: Theme.of(context).textTheme.titleMedium,
          ),
        ),
        const SizedBox(height: 8),
        Center(
          child: Text(
            error.toString(),
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ),
        const SizedBox(height: 16),
        Center(
          child: FilledButton(onPressed: onRetry, child: const Text('Retry')),
        ),
      ],
    );
  }
}

String _formatRelative(DateTime ts) {
  final diff = DateTime.now().toUtc().difference(ts.toUtc());
  if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
  if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
  if (diff.inHours < 24) return '${diff.inHours}h ago';
  if (diff.inDays < 7) return '${diff.inDays}d ago';
  return DateFormat.yMMMd().format(ts.toLocal());
}

// Headline shown on the session card. Prefer the operator-given
// session name, then the cwd's basename (so a session in
// `.../Claude_Workspace/opendray-v2` reads as `opendray-v2` at a
// glance), and only fall back to the opaque `ses_…` id when both
// are missing.
String _projectTitle(SessionSummary session) {
  final name = session.name;
  if (name != null && name.isNotEmpty) return name;
  final base = p.basename(session.cwd);
  if (base.isNotEmpty && base != '/') return base;
  return session.id;
}

// Compact a long cwd to head-ellipsis + last two segments, e.g.
// `/Users/linivek/Documents/HomeLab/Claude_Workspace/opendray-v2`
// → `…/Claude_Workspace/opendray-v2`. Short paths pass through.
// Keeps the project-identifying tail visible inside the card's
// fixed width.
String _shortCwd(String cwd) {
  if (cwd.isEmpty) return '';
  final parts = p.split(cwd).where((s) => s.isNotEmpty).toList();
  if (parts.length <= 2) return cwd;
  return '…/${parts[parts.length - 2]}/${parts.last}';
}
