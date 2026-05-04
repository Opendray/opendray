// V1 Hub layout — first iteration of the new prototype design ported to
// Flutter. Lives at /hub-v1 so reviewers can compare side-by-side with
// the existing dashboard at / before we swap.
//
// Real data wired:
//   - Active sessions count + the session-card grid (GET /api/sessions).
//
// Placeholder data (clearly labelled "—" so we don't lie to users about
// features we haven't built yet):
//   - Tokens 24h, PRs opened, Avg session length KPIs.
//   - Activity rail (will wire to /api/audit-log when that endpoint
//     ships in the upcoming HostProbe work).
//
// Follow-up screens (Workbench, Agents, Connections, Files) ship in
// subsequent PRs against the same prototype.

import 'dart:async';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/models/session.dart';
import '../../shared/session_launcher.dart';
import '../../shared/theme/app_theme.dart';

class HubV1Page extends StatefulWidget {
  const HubV1Page({super.key});
  @override
  State<HubV1Page> createState() => _HubV1PageState();
}

enum _SessionFilter { all, waiting, finished }

class _HubV1PageState extends State<HubV1Page> {
  List<Session> _sessions = [];
  bool _loading = true;
  String? _error;
  Timer? _pollTimer;
  _SessionFilter _filter = _SessionFilter.all;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
    _pollTimer = Timer.periodic(const Duration(seconds: 5), (_) => _load());
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final s = await _api.listSessions();
      if (!mounted) return;
      setState(() {
        _sessions = s;
        _loading = false;
        _error = null;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  List<Session> get _filteredSessions {
    return switch (_filter) {
      _SessionFilter.all => _sessions,
      _SessionFilter.waiting => _sessions
          .where((s) => s.status == 'idle' || s.status == 'waiting')
          .toList(),
      _SessionFilter.finished => _sessions
          .where((s) => s.status == 'stopped' || s.status == 'error')
          .toList(),
    };
  }

  int get _waitingCount =>
      _sessions.where((s) => s.status == 'idle' || s.status == 'waiting').length;

  int get _activeCount => _sessions.where((s) => s.status == 'running').length;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final greeting = _greeting();

    return Scaffold(
      backgroundColor: t.bg,
      body: _loading && _sessions.isEmpty
          ? Center(child: CircularProgressIndicator(color: t.accent))
          : SafeArea(
              child: Scrollbar(
                child: SingleChildScrollView(
                  padding: EdgeInsets.symmetric(
                      horizontal: t.sp5, vertical: t.sp4),
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 1400),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        _PageHeader(
                          greeting: greeting,
                          subtitle: _subtitle(),
                          onNewSession: _onNewSession,
                        ),
                        SizedBox(height: t.sp4),
                        _SummaryStrip(
                          active: _activeCount,
                          waiting: _waitingCount,
                          total: _sessions.length,
                        ),
                        SizedBox(height: t.sp4),
                        _QuickActionBar(
                          onNewSession: _onNewSession,
                          onAttachRepo: _onAttachRepo,
                        ),
                        SizedBox(height: t.sp4),
                        _SessionsCard(
                          sessions: _filteredSessions,
                          totalCount: _sessions.length,
                          filter: _filter,
                          waitingCount: _waitingCount,
                          onFilterChanged: (f) => setState(() => _filter = f),
                          onSessionTap: (s) =>
                              context.push('/session/${s.id}'),
                          onSessionStart: _onStartSession,
                          onSessionStop: _onStopSession,
                          onSessionDelete: _onDeleteSession,
                          loading: _loading,
                          error: _error,
                        ),
                        SizedBox(height: t.sp5),
                      ],
                    ),
                  ),
                ),
              ),
            ),
    );
  }

  String _greeting() {
    final h = DateTime.now().hour;
    if (h < 12) return 'Good morning';
    if (h < 18) return 'Good afternoon';
    return 'Good evening';
  }

  String _subtitle() {
    if (_sessions.isEmpty) {
      return 'No sessions yet — start one to attach an AI coding agent to a repo.';
    }
    final parts = <String>[];
    if (_activeCount > 0) parts.add('$_activeCount running');
    if (_waitingCount > 0) parts.add('$_waitingCount waiting on you');
    if (parts.isEmpty) {
      return '${_sessions.length} session${_sessions.length == 1 ? '' : 's'}';
    }
    return parts.join(' · ');
  }

  Future<void> _onNewSession() async {
    await launchNewSession(context);
    _load();
  }

  void _onAttachRepo() {
    context.go('/source-control');
  }

  Future<void> _onStartSession(Session s) async {
    try {
      await _api.startSession(s.id);
      if (mounted) _toast('Started ${_displayName(s)}');
      _load();
    } catch (e) {
      if (mounted) _toast('Start failed: $e', isError: true);
    }
  }

  Future<void> _onStopSession(Session s) async {
    try {
      await _api.stopSession(s.id);
      if (mounted) _toast('Stopped ${_displayName(s)}');
      _load();
    } catch (e) {
      if (mounted) _toast('Stop failed: $e', isError: true);
    }
  }

  Future<void> _onDeleteSession(Session s) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete this session?'),
        content: Text(
            'This removes "${_displayName(s)}" and its history. The agent CLI is stopped if running.'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          TextButton(
              onPressed: () => Navigator.pop(ctx, true),
              style: TextButton.styleFrom(foregroundColor: Colors.red),
              child: const Text('Delete')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.deleteSession(s.id);
      if (mounted) _toast('Deleted ${_displayName(s)}');
      _load();
    } catch (e) {
      if (mounted) _toast('Delete failed: $e', isError: true);
    }
  }

  String _displayName(Session s) =>
      s.name.isNotEmpty ? s.name : 'session ${s.id.substring(0, 8)}';

  void _toast(String msg, {bool isError = false}) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: isError ? t.danger : null,
      duration: const Duration(seconds: 3),
    ));
  }
}

// -----------------------------------------------------------------------------
// Page header
// -----------------------------------------------------------------------------

class _PageHeader extends StatelessWidget {
  final String greeting;
  final String subtitle;
  final VoidCallback onNewSession;
  const _PageHeader({
    required this.greeting,
    required this.subtitle,
    required this.onNewSession,
  });

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    return LayoutBuilder(builder: (ctx, c) {
      final wide = c.maxWidth > 720;
      final headlineCol = Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(greeting, style: theme.textTheme.displaySmall),
          SizedBox(height: t.sp2),
          Text(subtitle,
              style:
                  theme.textTheme.bodyLarge?.copyWith(color: t.textMuted)),
        ],
      );
      final actions = Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          OutlinedButton.icon(
            onPressed: null, // attach-repo flow ships in a future iteration
            icon: const Icon(Icons.upload_outlined, size: 16),
            label: const Text('Import repo'),
          ),
          SizedBox(width: t.sp3),
          ElevatedButton.icon(
            onPressed: onNewSession,
            icon: const Icon(Icons.add, size: 16),
            label: const Text('New session'),
          ),
        ],
      );
      if (wide) {
        return Row(
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [Expanded(child: headlineCol), actions],
        );
      }
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [headlineCol, SizedBox(height: t.sp4), actions],
      );
    });
  }
}

// -----------------------------------------------------------------------------
// KPI grid
// -----------------------------------------------------------------------------

/// Compact one-line summary bar that replaces the 4-up KPI grid. Three
/// of those cards rendered "—" placeholders today; once we have real
/// metrics the cards can come back. For now: a single subtle strip
/// avoids the "lying-by-omission" look of empty KPI placeholders.
class _SummaryStrip extends StatelessWidget {
  final int active;
  final int waiting;
  final int total;
  const _SummaryStrip(
      {required this.active, required this.waiting, required this.total});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    Widget chip(String label, int value, Color color) => Container(
          padding: EdgeInsets.symmetric(horizontal: t.sp3, vertical: t.sp2),
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(t.rMd),
            border: Border.all(color: color.withValues(alpha: 0.25)),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 8, height: 8,
                decoration:
                    BoxDecoration(color: color, shape: BoxShape.circle),
              ),
              SizedBox(width: t.sp2),
              Text('$value $label',
                  style: theme.textTheme.bodyMedium?.copyWith(
                      color: t.text,
                      fontWeight: FontWeight.w600,
                      fontSize: 13)),
            ],
          ),
        );
    return Wrap(
      spacing: t.sp3,
      runSpacing: t.sp2,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        chip('active', active, t.success),
        chip('waiting', waiting, t.warning),
        chip('total', total, t.accent),
      ],
    );
  }
}

// -----------------------------------------------------------------------------
// Sessions card
// -----------------------------------------------------------------------------

class _SessionsCard extends StatelessWidget {
  final List<Session> sessions;
  final int totalCount;
  final _SessionFilter filter;
  final int waitingCount;
  final ValueChanged<_SessionFilter> onFilterChanged;
  final ValueChanged<Session> onSessionTap;
  final ValueChanged<Session> onSessionStart;
  final ValueChanged<Session> onSessionStop;
  final ValueChanged<Session> onSessionDelete;
  final bool loading;
  final String? error;
  const _SessionsCard({
    required this.sessions,
    required this.totalCount,
    required this.filter,
    required this.waitingCount,
    required this.onFilterChanged,
    required this.onSessionTap,
    required this.onSessionStart,
    required this.onSessionStop,
    required this.onSessionDelete,
    required this.loading,
    required this.error,
  });

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    return _Card(
      header: _CardHeader(
        title: 'Active sessions',
        subtitle:
            'Tap a card to open the terminal · use the menu to start, stop, or delete',
        actions: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            _FilterChip(
              label: 'All',
              selected: filter == _SessionFilter.all,
              onTap: () => onFilterChanged(_SessionFilter.all),
            ),
            SizedBox(width: t.sp2),
            _FilterChip(
              label: 'Waiting on me',
              selected: filter == _SessionFilter.waiting,
              badgeCount: waitingCount,
              badgeColor: t.warning,
              onTap: () => onFilterChanged(_SessionFilter.waiting),
            ),
            SizedBox(width: t.sp2),
            _FilterChip(
              label: 'Finished',
              selected: filter == _SessionFilter.finished,
              onTap: () => onFilterChanged(_SessionFilter.finished),
            ),
          ],
        ),
      ),
      body: error != null
          ? Padding(
              padding: EdgeInsets.all(t.sp5),
              child: Text(error!,
                  style: TextStyle(color: t.danger)),
            )
          : sessions.isEmpty
              ? Padding(
                  padding: EdgeInsets.all(t.sp8),
                  child: Center(
                      child: Text(
                          totalCount == 0
                              ? 'No sessions yet — click "New session" to get started.'
                              : 'No sessions match this filter.',
                          style: theme.textTheme.bodyMedium)),
                )
              : LayoutBuilder(builder: (ctx, c) {
                  // Cap at 2 columns so single-session views don't render
                  // a tile at 1/3 width with 2/3 of the band sitting empty.
                  // 1-col on narrow viewports keeps tiles legible.
                  final cols = c.maxWidth > 720 ? 2 : 1;
                  return GridView.count(
                    crossAxisCount: cols,
                    shrinkWrap: true,
                    physics: const NeverScrollableScrollPhysics(),
                    crossAxisSpacing: t.sp4,
                    mainAxisSpacing: t.sp4,
                    childAspectRatio: cols == 1 ? 5.0 : 3.5,
                    padding: EdgeInsets.all(t.sp4),
                    children: sessions
                        .map((s) => _SessionTile(
                              session: s,
                              onTap: () => onSessionTap(s),
                              onStart: () => onSessionStart(s),
                              onStop: () => onSessionStop(s),
                              onDelete: () => onSessionDelete(s),
                            ))
                        .toList(),
                  );
                }),
    );
  }
}

class _SessionTile extends StatelessWidget {
  final Session session;
  final VoidCallback onTap;
  final VoidCallback onStart;
  final VoidCallback onStop;
  final VoidCallback onDelete;
  const _SessionTile({
    required this.session,
    required this.onTap,
    required this.onStart,
    required this.onStop,
    required this.onDelete,
  });

  bool get _isRunning => session.status == 'running' || session.status == 'idle' || session.status == 'waiting';

  Color _statusColor(OpendrayTokens t) => switch (session.status) {
        'running' => t.success,
        'error' => t.danger,
        'idle' || 'waiting' => t.warning,
        _ => t.textSubtle,
      };
  String _statusLabel() => switch (session.status) {
        'running' => 'Running',
        'idle' => 'Idle',
        'waiting' => 'Waiting on you',
        'error' => 'Error',
        'stopped' => 'Stopped',
        _ => session.status,
      };
  String _agentInitial() {
    final type = session.sessionType;
    if (type.isEmpty) return '?';
    return type[0].toUpperCase();
  }

  String _shortPath(String p) {
    final parts = p.split('/').where((s) => s.isNotEmpty).toList();
    if (parts.length <= 3) return p;
    return '…/${parts.sublist(parts.length - 2).join('/')}';
  }

  String _ago(DateTime dt) {
    final diff = DateTime.now().difference(dt);
    if (diff.inSeconds < 60) return 'now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${diff.inDays}d ago';
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    final statusC = _statusColor(t);
    return Material(
      color: t.surface,
      borderRadius: BorderRadius.circular(t.rLg),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(t.rLg),
        child: Container(
          decoration: BoxDecoration(
            color: t.surface,
            borderRadius: BorderRadius.circular(t.rLg),
            border: Border.all(color: t.border),
          ),
          padding: EdgeInsets.fromLTRB(t.sp4, t.sp3, t.sp2, t.sp3),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                children: [
                  Container(
                    width: 26, height: 26,
                    decoration: BoxDecoration(
                      color: t.accentSoft,
                      borderRadius: BorderRadius.circular(t.rSm),
                    ),
                    alignment: Alignment.center,
                    child: Text(_agentInitial(),
                        style: TextStyle(
                            color: t.accentText,
                            fontWeight: FontWeight.w700,
                            fontSize: 11)),
                  ),
                  SizedBox(width: t.sp3),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          session.name.isEmpty ? '(unnamed)' : session.name,
                          style: theme.textTheme.titleMedium?.copyWith(
                              fontWeight: FontWeight.w600, fontSize: 13),
                          overflow: TextOverflow.ellipsis,
                          maxLines: 1,
                        ),
                        Text(_shortPath(session.cwd),
                            style: theme.textTheme.bodySmall
                                ?.copyWith(color: t.textSubtle, fontSize: 11),
                            overflow: TextOverflow.ellipsis,
                            maxLines: 1),
                      ],
                    ),
                  ),
                  _StatusPill(color: statusC, label: _statusLabel(), pulsing: session.status == 'running'),
                  // Kebab menu — start/stop/delete. Stop click propagation
                  // so the underlying card-tap doesn't navigate too.
                  PopupMenuButton<String>(
                    tooltip: 'Actions',
                    icon: Icon(Icons.more_vert, size: 16, color: t.textMuted),
                    splashRadius: 16,
                    padding: EdgeInsets.zero,
                    iconSize: 16,
                    constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
                    onSelected: (v) {
                      switch (v) {
                        case 'start': onStart(); break;
                        case 'stop':  onStop();  break;
                        case 'open':  onTap();   break;
                        case 'delete': onDelete(); break;
                      }
                    },
                    itemBuilder: (_) => [
                      const PopupMenuItem(
                          value: 'open',
                          child: ListTile(
                              dense: true,
                              leading: Icon(Icons.open_in_new, size: 16),
                              title: Text('Open terminal'))),
                      if (_isRunning)
                        const PopupMenuItem(
                            value: 'stop',
                            child: ListTile(
                                dense: true,
                                leading: Icon(Icons.stop_circle_outlined, size: 16),
                                title: Text('Stop')))
                      else
                        const PopupMenuItem(
                            value: 'start',
                            child: ListTile(
                                dense: true,
                                leading: Icon(Icons.play_circle_outline, size: 16),
                                title: Text('Start'))),
                      const PopupMenuDivider(),
                      const PopupMenuItem(
                          value: 'delete',
                          child: ListTile(
                              dense: true,
                              leading: Icon(Icons.delete_outline, size: 16, color: Colors.redAccent),
                              title: Text('Delete', style: TextStyle(color: Colors.redAccent)))),
                    ],
                  ),
                ],
              ),
              SizedBox(height: t.sp2),
              DefaultTextStyle(
                style: theme.textTheme.bodySmall!.copyWith(color: t.textSubtle, fontSize: 11),
                child: Row(
                  children: [
                    Text(session.sessionType),
                    if (session.model.isNotEmpty) ...[
                      const Text('  ·  '),
                      Flexible(
                        child: Text(session.model, overflow: TextOverflow.ellipsis),
                      ),
                    ],
                    const Spacer(),
                    Text(_ago(session.lastActiveAt)),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  final Color color;
  final String label;
  final bool pulsing;
  const _StatusPill({required this.color, required this.label, required this.pulsing});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      padding: EdgeInsets.symmetric(horizontal: t.sp2, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(t.rXl),
        border: Border.all(color: color.withValues(alpha: 0.35)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6, height: 6,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
          SizedBox(width: t.sp1),
          Text(label,
              style: TextStyle(
                  fontSize: 10,
                  color: color,
                  fontWeight: FontWeight.w600)),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Quick actions — inline horizontal bar (was a right-column card).
// Activity card is dropped until /api/audit-log exists; reinstate it
// once the wizard PR lands the endpoint.
// -----------------------------------------------------------------------------

class _QuickActionBar extends StatelessWidget {
  final VoidCallback onNewSession;
  final VoidCallback onAttachRepo;
  const _QuickActionBar(
      {required this.onNewSession, required this.onAttachRepo});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Wrap(
      spacing: t.sp2,
      runSpacing: t.sp2,
      children: [
        _QuickActionPill(
          icon: Icons.account_tree_outlined,
          label: 'Attach GitHub repo',
          onTap: onAttachRepo,
        ),
        _QuickActionPill(
          icon: Icons.link,
          label: 'Connect MCP server',
          onTap: () => GoRouter.of(context).go('/settings/llm-endpoints'),
        ),
        _QuickActionPill(
          icon: Icons.terminal,
          label: 'Playground terminal',
          onTap: onNewSession,
        ),
      ],
    );
  }
}

class _QuickActionPill extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;
  const _QuickActionPill(
      {required this.icon, required this.label, required this.onTap});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return OutlinedButton.icon(
      onPressed: onTap,
      icon: Icon(icon, size: 14),
      label: Text(label, style: const TextStyle(fontSize: 12)),
      style: OutlinedButton.styleFrom(
        backgroundColor: t.surface,
        side: BorderSide(color: t.border),
        padding: EdgeInsets.symmetric(horizontal: t.sp3, vertical: 6),
        minimumSize: const Size(0, 32),
        shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(t.rXl)),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Generic card chrome (header / body / footer)
// -----------------------------------------------------------------------------

class _Card extends StatelessWidget {
  final Widget header;
  final Widget body;
  final Widget? footer;
  const _Card({required this.header, required this.body, this.footer});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      decoration: BoxDecoration(
        color: t.surface,
        borderRadius: BorderRadius.circular(t.rLg),
        border: Border.all(color: t.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          header,
          Divider(height: 1, color: t.border),
          body,
          if (footer != null) Divider(height: 1, color: t.border),
          if (footer != null) footer!,
        ],
      ),
    );
  }
}

class _CardHeader extends StatelessWidget {
  final String title;
  final String? subtitle;
  final Widget? actions;
  const _CardHeader({required this.title, this.subtitle, this.actions});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    return Padding(
      padding: EdgeInsets.symmetric(horizontal: t.sp5, vertical: t.sp4),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    style: theme.textTheme.headlineSmall
                        ?.copyWith(fontSize: 15, fontWeight: FontWeight.w600)),
                if (subtitle != null) ...[
                  SizedBox(height: t.sp1),
                  Text(subtitle!,
                      style: theme.textTheme.bodySmall
                          ?.copyWith(color: t.textMuted)),
                ],
              ],
            ),
          ),
          if (actions != null) actions!,
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Filter pill
// -----------------------------------------------------------------------------

class _FilterChip extends StatelessWidget {
  final String label;
  final bool selected;
  final int? badgeCount;
  final Color? badgeColor;
  final VoidCallback onTap;
  const _FilterChip({
    required this.label,
    required this.selected,
    this.badgeCount,
    this.badgeColor,
    required this.onTap,
  });
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(t.rXl),
      child: Container(
        padding: EdgeInsets.symmetric(horizontal: t.sp3, vertical: 6),
        decoration: BoxDecoration(
          color: selected ? t.accentSoft : Colors.transparent,
          borderRadius: BorderRadius.circular(t.rXl),
          border: Border.all(
              color: selected ? t.accentBorder : t.border),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(label,
                style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    color: selected ? t.accentText : t.textMuted)),
            if (badgeCount != null && badgeCount! > 0) ...[
              SizedBox(width: t.sp2),
              Container(
                padding:
                    EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                decoration: BoxDecoration(
                  color: (badgeColor ?? t.warning).withValues(alpha: 0.18),
                  borderRadius: BorderRadius.circular(t.rXs),
                ),
                child: Text('$badgeCount',
                    style: TextStyle(
                        fontSize: 10,
                        fontWeight: FontWeight.w700,
                        color: badgeColor ?? t.warning)),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
