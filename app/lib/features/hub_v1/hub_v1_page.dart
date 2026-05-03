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
                      horizontal: t.sp8, vertical: t.sp6),
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
                        SizedBox(height: t.sp8),
                        _KpiGrid(activeSessions: _activeCount),
                        SizedBox(height: t.sp8),
                        _SessionsCard(
                          sessions: _filteredSessions,
                          totalCount: _sessions.length,
                          filter: _filter,
                          waitingCount: _waitingCount,
                          onFilterChanged: (f) => setState(() => _filter = f),
                          onSessionTap: (s) => context.go('/?sid=${s.id}'),
                          loading: _loading,
                          error: _error,
                        ),
                        SizedBox(height: t.sp8),
                        _ActivityAndQuickActions(
                          onNewSession: _onNewSession,
                          onAttachRepo: _onAttachRepo,
                        ),
                        SizedBox(height: t.sp8),
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

class _KpiGrid extends StatelessWidget {
  final int activeSessions;
  const _KpiGrid({required this.activeSessions});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final items = [
      _KpiData(label: 'Active sessions', value: '$activeSessions', accent: t.accent),
      _KpiData(label: 'Tokens (24h)', value: '—', accent: t.success),
      _KpiData(label: 'PRs this week', value: '—', accent: t.info),
      _KpiData(label: 'Avg session length', value: '—', accent: t.warning),
    ];
    return LayoutBuilder(builder: (ctx, c) {
      final cols = c.maxWidth > 1100 ? 4 : c.maxWidth > 700 ? 2 : 1;
      return GridView.count(
        crossAxisCount: cols,
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        crossAxisSpacing: t.sp4,
        mainAxisSpacing: t.sp4,
        childAspectRatio: cols == 1 ? 4 : (cols == 2 ? 2.5 : 1.9),
        children: items.map((d) => _KpiCard(data: d)).toList(),
      );
    });
  }
}

class _KpiData {
  final String label;
  final String value;
  final Color accent;
  const _KpiData({required this.label, required this.value, required this.accent});
}

class _KpiCard extends StatelessWidget {
  final _KpiData data;
  const _KpiCard({required this.data});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final theme = Theme.of(context);
    return Container(
      padding: EdgeInsets.all(t.sp5),
      decoration: BoxDecoration(
        color: t.surface,
        borderRadius: BorderRadius.circular(t.rLg),
        border: Border.all(color: t.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(data.label,
              style: theme.textTheme.labelSmall?.copyWith(
                  color: t.textSubtle, letterSpacing: 0.6)),
          SizedBox(height: t.sp2),
          Text(data.value,
              style: theme.textTheme.displayMedium
                  ?.copyWith(fontFeatures: const [FontFeature.tabularFigures()])),
          const Spacer(),
          Container(
            height: 4,
            decoration: BoxDecoration(
              color: data.accent.withValues(alpha: 0.18),
              borderRadius: BorderRadius.circular(2),
            ),
            child: FractionallySizedBox(
              alignment: Alignment.centerLeft,
              widthFactor: data.value == '—' ? 0 : 0.55,
              child: Container(
                decoration: BoxDecoration(
                  color: data.accent,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
          ),
        ],
      ),
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
  final bool loading;
  final String? error;
  const _SessionsCard({
    required this.sessions,
    required this.totalCount,
    required this.filter,
    required this.waitingCount,
    required this.onFilterChanged,
    required this.onSessionTap,
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
            'Tap a session to attach a terminal, or send a command from Telegram',
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
                  final cols =
                      c.maxWidth > 1100 ? 3 : c.maxWidth > 700 ? 2 : 1;
                  return GridView.count(
                    crossAxisCount: cols,
                    shrinkWrap: true,
                    physics: const NeverScrollableScrollPhysics(),
                    crossAxisSpacing: t.sp4,
                    mainAxisSpacing: t.sp4,
                    childAspectRatio: cols == 1 ? 2.4 : 1.55,
                    padding: EdgeInsets.all(t.sp5),
                    children: sessions
                        .map((s) =>
                            _SessionTile(session: s, onTap: () => onSessionTap(s)))
                        .toList(),
                  );
                }),
      footer: Padding(
        padding: EdgeInsets.symmetric(horizontal: t.sp5, vertical: t.sp3),
        child: Row(
          children: [
            Text(
              '${sessions.length} of $totalCount session${totalCount == 1 ? '' : 's'} shown · max 20 per workspace',
              style: theme.textTheme.bodySmall,
            ),
            const Spacer(),
            TextButton(
              onPressed: () => GoRouter.of(context).go('/'),
              child: const Text('View classic dashboard →'),
            ),
          ],
        ),
      ),
    );
  }
}

class _SessionTile extends StatelessWidget {
  final Session session;
  final VoidCallback onTap;
  const _SessionTile({required this.session, required this.onTap});

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
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(t.rLg),
      child: Container(
        decoration: BoxDecoration(
          color: t.surface,
          borderRadius: BorderRadius.circular(t.rLg),
          border: Border.all(color: t.border),
        ),
        padding: EdgeInsets.all(t.sp4),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  width: 28,
                  height: 28,
                  decoration: BoxDecoration(
                    color: t.accentSoft,
                    borderRadius: BorderRadius.circular(t.rSm),
                  ),
                  alignment: Alignment.center,
                  child: Text(_agentInitial(),
                      style: TextStyle(
                          color: t.accentText,
                          fontWeight: FontWeight.w700,
                          fontSize: 12)),
                ),
                SizedBox(width: t.sp3),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        session.name.isEmpty ? '(unnamed)' : session.name,
                        style: theme.textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.w600),
                        overflow: TextOverflow.ellipsis,
                      ),
                      Text(_shortPath(session.cwd),
                          style: theme.textTheme.bodySmall
                              ?.copyWith(color: t.textSubtle),
                          overflow: TextOverflow.ellipsis),
                    ],
                  ),
                ),
                Container(
                  padding: EdgeInsets.symmetric(
                      horizontal: t.sp2, vertical: 2),
                  decoration: BoxDecoration(
                    color: statusC.withValues(alpha: 0.14),
                    borderRadius: BorderRadius.circular(t.rXl),
                    border: Border.all(color: statusC.withValues(alpha: 0.35)),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Container(
                        width: 6,
                        height: 6,
                        decoration: BoxDecoration(
                          color: statusC,
                          shape: BoxShape.circle,
                        ),
                      ),
                      SizedBox(width: t.sp1),
                      Text(_statusLabel(),
                          style: TextStyle(
                              fontSize: 11,
                              color: statusC,
                              fontWeight: FontWeight.w600)),
                    ],
                  ),
                ),
              ],
            ),
            SizedBox(height: t.sp3),
            Expanded(
              child: Container(
                width: double.infinity,
                padding: EdgeInsets.all(t.sp3),
                decoration: BoxDecoration(
                  color: t.bgRaised,
                  borderRadius: BorderRadius.circular(t.rMd),
                  border: Border.all(color: t.border),
                ),
                child: Text(
                  session.status == 'running'
                      ? '▸ session active · pid ${session.pid ?? '—'}'
                      : session.status == 'stopped'
                          ? '◼ stopped'
                          : '… ${_statusLabel().toLowerCase()}',
                  maxLines: 4,
                  overflow: TextOverflow.fade,
                  style: mono(size: 11, color: t.textMuted),
                ),
              ),
            ),
            SizedBox(height: t.sp2),
            DefaultTextStyle(
              style: theme.textTheme.bodySmall!.copyWith(color: t.textSubtle),
              child: Row(
                children: [
                  Text(session.sessionType),
                  Text('  ·  '),
                  Text(session.model.isEmpty ? '—' : session.model,
                      overflow: TextOverflow.ellipsis),
                  const Spacer(),
                  Text(_ago(session.lastActiveAt)),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Activity rail + Quick actions row
// -----------------------------------------------------------------------------

class _ActivityAndQuickActions extends StatelessWidget {
  final VoidCallback onNewSession;
  final VoidCallback onAttachRepo;
  const _ActivityAndQuickActions({
    required this.onNewSession,
    required this.onAttachRepo,
  });

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return LayoutBuilder(builder: (ctx, c) {
      final wide = c.maxWidth > 900;
      final activity = _Card(
        header: _CardHeader(
            title: 'Activity',
            subtitle: 'Last 24 hours across this workspace'),
        body: Padding(
          padding: EdgeInsets.all(t.sp5),
          child: Text(
            'Activity timeline ships next iteration — needs the audit-log endpoint.',
            style:
                Theme.of(context).textTheme.bodyMedium?.copyWith(color: t.textMuted),
          ),
        ),
      );
      final quick = _Card(
        header: _CardHeader(title: 'Quick actions'),
        body: Padding(
          padding: EdgeInsets.all(t.sp4),
          child: Column(
            children: [
              _QuickAction(
                icon: Icons.add_circle_outline,
                label: 'Start new session',
                shortcut: '⌘N',
                onTap: onNewSession,
              ),
              SizedBox(height: t.sp2),
              _QuickAction(
                icon: Icons.account_tree_outlined,
                label: 'Attach to GitHub repo',
                onTap: onAttachRepo,
              ),
              SizedBox(height: t.sp2),
              _QuickAction(
                icon: Icons.link,
                label: 'Connect MCP server',
                onTap: () => GoRouter.of(context).go('/'),
              ),
              SizedBox(height: t.sp2),
              _QuickAction(
                icon: Icons.terminal,
                label: 'Open playground terminal',
                onTap: onNewSession,
              ),
            ],
          ),
        ),
        footer: Padding(
          padding: EdgeInsets.symmetric(
              horizontal: t.sp5, vertical: t.sp3),
          child: Row(
            children: [
              Container(
                width: 8,
                height: 8,
                decoration: BoxDecoration(
                    color: t.success, shape: BoxShape.circle),
              ),
              SizedBox(width: t.sp2),
              Text('All systems operational',
                  style: Theme.of(context).textTheme.bodySmall),
            ],
          ),
        ),
      );
      if (wide) {
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(flex: 2, child: activity),
            SizedBox(width: t.sp4),
            Expanded(flex: 1, child: quick),
          ],
        );
      }
      return Column(children: [activity, SizedBox(height: t.sp4), quick]);
    });
  }
}

class _QuickAction extends StatelessWidget {
  final IconData icon;
  final String label;
  final String? shortcut;
  final VoidCallback onTap;
  const _QuickAction(
      {required this.icon,
      required this.label,
      this.shortcut,
      required this.onTap});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return SizedBox(
      width: double.infinity,
      child: OutlinedButton.icon(
        onPressed: onTap,
        icon: Icon(icon, size: 16),
        label: Row(
          children: [
            Expanded(
                child: Text(label, style: const TextStyle(fontSize: 13))),
            if (shortcut != null)
              Container(
                padding: EdgeInsets.symmetric(
                    horizontal: t.sp2, vertical: 2),
                decoration: BoxDecoration(
                  color: t.surface3,
                  borderRadius: BorderRadius.circular(t.rXs),
                  border: Border.all(color: t.border),
                ),
                child: Text(shortcut!,
                    style: mono(size: 11, color: t.textSubtle)),
              ),
          ],
        ),
        style: OutlinedButton.styleFrom(
          alignment: Alignment.centerLeft,
          minimumSize: const Size(0, 40),
        ),
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
