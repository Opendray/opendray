import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/antigravity_accounts_api.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/providers/provider_visual.dart';
import 'package:opendray/core/widgets/brand_avatar.dart';
import 'package:opendray/features/project/project_screen.dart';
import 'package:opendray/features/sessions/account_switch_sheet.dart';
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
        // Brand mark next to the title so the operator can tell at
        // a glance which CLI is driving the session, matching the
        // sessions list and the web admin's workbench header. Kept
        // inside `title` (rather than the leading slot) so the
        // system back arrow stays in place.
        titleSpacing: 0,
        title: async.when(
          data: (s) => Row(
            children: [
              BrandAvatar(providerId: s.providerId, size: 28),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  s.displayName,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
          loading: () => Text(t.sessions.detail.fallbackTitle),
          error: (_, __) => Text(t.sessions.detail.fallbackTitle),
        ),
        // Phone bars are narrow and a row of unlabeled icons is easy to
        // mis-tap — collapse every action into ONE labelled overflow menu so
        // each item shows its name. The lifecycle actions (stop / restart /
        // delete) keep their richer SessionActionSheet, opened from the menu.
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

  // Overflow-menu items. Session-dependent entries (project memory, account
  // switch, lifecycle actions) only appear once the session has loaded; the
  // account switcher is Claude/Antigravity-live only (nothing to rebind
  // otherwise), mirroring the web header.
  List<PopupMenuEntry<String>> _menuItems(SessionSummary? s) {
    final canAccount = s != null &&
        (s.providerId == 'claude' || s.providerId == 'antigravity') &&
        s.isLive;
    return [
      _menuItem('refresh', Icons.refresh, t.sessions.detail.refreshMetadata),
      _menuItem('inspector', Icons.dashboard_customize_outlined,
          t.sessions.detail.inspector),
      if (s != null)
        _menuItem(
            'project', Icons.flag_outlined, t.sessions.detail.projectMemory),
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
      case 'inspector':
        context.push('/session/${widget.sessionId}/inspector');
      case 'project':
        if (s != null) {
          Navigator.of(context).push(
            MaterialPageRoute<void>(
              builder: (_) => ProjectScreen(initialCwd: s.cwd),
            ),
          );
        }
      case 'account':
        if (s != null) _switchAccount(s);
      case 'actions':
        if (s != null) _openActions(s);
    }
  }

  // Rebind the running session to a different account (mirrors the web header
  // AccountSwitcher).
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

  // Lifecycle actions (stop / restart / delete) keep their richer sheet, which
  // carries per-action descriptions and a delete confirmation.
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
                    providerVisualFor(session.providerId).label,
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
                ? t.sessions.detail.started(when: started)
                : t.sessions.detail.startedEnded(
                    started: started,
                    ended: ended,
                  ),
            style: muted,
          ),
          const SizedBox(height: 2),
          SelectableText(
            t.sessions.detail.idPrefix(id: session.id),
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
