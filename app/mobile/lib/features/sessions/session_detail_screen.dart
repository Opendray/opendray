import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:opendray/core/api/antigravity_accounts_api.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/providers/provider_visual.dart';
import 'package:opendray/core/widgets/brand_avatar.dart';
import 'package:opendray/features/sessions/account_switch_sheet.dart';
import 'package:opendray/features/sessions/session_action_sheet.dart';
import 'package:opendray/features/sessions/session_terminal_view.dart';
import 'package:opendray/features/sessions/session_tool_dock.dart';

// Session detail surface.
//
// The terminal is the hero and now owns the full height of the body —
// the old expandable metadata header is gone. Identity + lifecycle
// state live in a compact two-line AppBar title (name on top, provider
// + state beneath with a colour dot), and every cwd-scoped project tool
// is one tap away on the SessionToolDock pinned under the terminal
// (Files / Git / Tasks / Project memory / More). The AppBar "⋮" keeps
// only the low-frequency, session-level actions (refresh, account
// switch, lifecycle) — Inspector/Memory no longer hide in there.
class SessionDetailScreen extends ConsumerStatefulWidget {
  const SessionDetailScreen({required this.sessionId, super.key});

  final String sessionId;

  @override
  ConsumerState<SessionDetailScreen> createState() =>
      _SessionDetailScreenState();
}

class _SessionDetailScreenState extends ConsumerState<SessionDetailScreen> {
  @override
  Widget build(BuildContext context) {
    final async = ref.watch(sessionByIdProvider(widget.sessionId));
    return Scaffold(
      appBar: AppBar(
        titleSpacing: 0,
        title: async.when(
          data: _TitleBar.new,
          loading: () => Text(t.sessions.detail.fallbackTitle),
          error: (_, __) => Text(t.sessions.detail.fallbackTitle),
        ),
        actions: [
          PopupMenuButton<String>(
            tooltip: MaterialLocalizations.of(context).showMenuTooltip,
            icon: const Icon(Icons.more_vert),
            onSelected: (v) => _onMenu(v, async.valueOrNull),
            itemBuilder: (_) => _menuItems(async.valueOrNull),
          ),
        ],
      ),
      body: async.when(
        data: (session) => Column(
          children: [
            Expanded(child: SessionTerminalView(sessionId: widget.sessionId)),
            SessionToolDock(session: session),
          ],
        ),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(
          error: e,
          onRetry: () => ref.invalidate(sessionByIdProvider(widget.sessionId)),
        ),
      ),
    );
  }

  // Overflow menu — only the low-frequency, session-level actions now
  // that the tools moved to the dock. The account switcher is
  // Claude/Antigravity-live only (nothing to rebind otherwise).
  List<PopupMenuEntry<String>> _menuItems(SessionSummary? s) {
    final canAccount = s != null &&
        (s.providerId == 'claude' || s.providerId == 'antigravity') &&
        s.isLive;
    return [
      _menuItem('refresh', Icons.refresh, t.sessions.detail.refreshMetadata),
      if (canAccount)
        _menuItem(
          'account',
          Icons.manage_accounts_outlined,
          s.providerId == 'antigravity'
              ? t.sessions.detail.accountSwitcher.tooltipAgy
              : t.sessions.detail.accountSwitcher.tooltip,
        ),
      if (s != null) ...[
        const PopupMenuDivider(),
        _menuItem('actions', Icons.tune, t.sessions.detail.actions),
      ],
    ];
  }

  PopupMenuItem<String> _menuItem(String value, IconData icon, String label) {
    return PopupMenuItem<String>(
      value: value,
      child: Row(
        children: [
          Icon(icon, size: 20),
          const SizedBox(width: 12),
          Text(label),
        ],
      ),
    );
  }

  void _onMenu(String value, SessionSummary? s) {
    switch (value) {
      case 'refresh':
        ref.invalidate(sessionByIdProvider(widget.sessionId));
      case 'account':
        if (s != null) _switchAccount(s);
      case 'actions':
        if (s != null) _openActions(s);
    }
  }

  Future<void> _switchAccount(SessionSummary s) async {
    final isAgy = s.providerId == 'antigravity';
    final switched = await AccountSwitchSheet.show(context, session: s);
    if (!switched || !mounted) return;
    ref
      ..invalidate(sessionByIdProvider(widget.sessionId))
      ..invalidate(sessionsListProvider)
      ..invalidate(
        isAgy ? antigravityAccountsListProvider : claudeAccountsListProvider,
      );
  }

  Future<void> _openActions(SessionSummary s) async {
    final result = await SessionActionSheet.show(context, session: s);
    if (result == null || !mounted) return;
    if (result == SessionActionResult.deleted) {
      ref.invalidate(sessionsListProvider);
      context.pop();
      return;
    }
    ref
      ..invalidate(sessionByIdProvider(widget.sessionId))
      ..invalidate(sessionsListProvider);
  }
}

// Compact two-line AppBar title: brand mark + session name on top, and
// "provider · state" with a lifecycle colour dot beneath.
class _TitleBar extends StatelessWidget {
  const _TitleBar(this.session);
  final SessionSummary session;

  @override
  Widget build(BuildContext context) {
    final visual = providerVisualFor(session.providerId);
    final muted = Theme.of(context).textTheme.bodySmall;
    return Row(
      children: [
        BrandAvatar(providerId: session.providerId, size: 28),
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                session.displayName,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context)
                    .textTheme
                    .titleSmall
                    ?.copyWith(fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 1),
              Row(
                children: [
                  _StateDot(state: session.state),
                  const SizedBox(width: 6),
                  Flexible(
                    child: Text(
                      '${visual.label} · ${session.state.wire}',
                      overflow: TextOverflow.ellipsis,
                      style: muted,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _StateDot extends StatelessWidget {
  const _StateDot({required this.state});
  final SessionState state;

  @override
  Widget build(BuildContext context) {
    final color = switch (state) {
      SessionState.running => Colors.greenAccent,
      SessionState.idle => Colors.amberAccent,
      _ => Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4),
    };
    return Container(
      width: 8,
      height: 8,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
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
              t.sessions.detail.errorTitle,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              error.toString(),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
