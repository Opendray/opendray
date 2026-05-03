// V1 app shell — sidebar nav rail + top bar from the prototype, used by
// the wide-viewport branch of [_Shell] in app.dart. Narrow viewports
// keep the existing BottomNavigationBar so the mobile feel is preserved.
//
// Inner page contents are delivered as `child` and rendered inside the
// content slot. Each inner page still owns its own AppBar today; once
// individual pages migrate to use V1TopBar's breadcrumbs we can drop
// their per-page bars.

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'theme/app_theme.dart';

class V1NavItem {
  final IconData icon;
  final String label;
  final String route;
  final int? badgeCount;
  final Color? badgeColor;
  final bool comingSoon;
  const V1NavItem({
    required this.icon,
    required this.label,
    required this.route,
    this.badgeCount,
    this.badgeColor,
    this.comingSoon = false,
  });
}

class V1NavSection {
  final String label;
  final List<V1NavItem> items;
  const V1NavSection({required this.label, required this.items});
}

class V1AppShell extends StatelessWidget {
  final Widget child;
  final List<V1NavSection> sections;
  final String currentRoute;
  final String breadcrumbTitle;
  final String workspaceName;
  final String userInitials;
  final String userName;
  final String userEmail;

  const V1AppShell({
    super.key,
    required this.child,
    required this.sections,
    required this.currentRoute,
    required this.breadcrumbTitle,
    this.workspaceName = 'Workspace',
    this.userInitials = '',
    this.userName = '',
    this.userEmail = '',
  });

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Scaffold(
      backgroundColor: t.bg,
      body: Row(
        children: [
          _V1Sidebar(
            sections: sections,
            currentRoute: currentRoute,
            workspaceName: workspaceName,
            userInitials: userInitials,
            userName: userName,
            userEmail: userEmail,
          ),
          Container(width: 1, color: t.border),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                _V1TopBar(
                  workspaceName: workspaceName,
                  current: breadcrumbTitle,
                ),
                Container(height: 1, color: t.border),
                Expanded(child: child),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Sidebar
// -----------------------------------------------------------------------------

class _V1Sidebar extends StatelessWidget {
  final List<V1NavSection> sections;
  final String currentRoute;
  final String workspaceName;
  final String userInitials;
  final String userName;
  final String userEmail;

  const _V1Sidebar({
    required this.sections,
    required this.currentRoute,
    required this.workspaceName,
    required this.userInitials,
    required this.userName,
    required this.userEmail,
  });

  bool _isActive(String route) {
    if (route == '/' && currentRoute == '/') return true;
    if (route == '/') return false;
    return currentRoute == route || currentRoute.startsWith('$route/');
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      width: 240,
      color: t.surface,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _Brand(),
          Divider(height: 1, color: t.border),
          _OrgSwitcher(workspaceName: workspaceName),
          Divider(height: 1, color: t.border),
          Expanded(
            child: SingleChildScrollView(
              padding: EdgeInsets.symmetric(vertical: t.sp3),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  for (final section in sections) ...[
                    _SectionLabel(label: section.label),
                    for (final item in section.items)
                      _NavTile(
                        item: item,
                        active: _isActive(item.route),
                      ),
                    SizedBox(height: t.sp3),
                  ],
                ],
              ),
            ),
          ),
          Divider(height: 1, color: t.border),
          _UserChip(
              initials: userInitials, name: userName, email: userEmail),
        ],
      ),
    );
  }
}

class _Brand extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      padding: EdgeInsets.symmetric(horizontal: t.sp4, vertical: t.sp4),
      child: Row(
        children: [
          Container(
            width: 32,
            height: 32,
            decoration: BoxDecoration(
              color: t.accent,
              borderRadius: BorderRadius.circular(t.rSm),
            ),
            alignment: Alignment.center,
            child: const Text('OD',
                style: TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.w800,
                    fontSize: 12,
                    letterSpacing: 0.5)),
          ),
          SizedBox(width: t.sp3),
          Text('Opendray',
              style: Theme.of(context).textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w700, fontSize: 16)),
        ],
      ),
    );
  }
}

class _OrgSwitcher extends StatelessWidget {
  final String workspaceName;
  const _OrgSwitcher({required this.workspaceName});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return InkWell(
      onTap: null,
      child: Padding(
        padding: EdgeInsets.symmetric(horizontal: t.sp4, vertical: t.sp3),
        child: Row(
          children: [
            Container(
              width: 28,
              height: 28,
              decoration: BoxDecoration(
                color: t.surface3,
                borderRadius: BorderRadius.circular(t.rSm),
              ),
              alignment: Alignment.center,
              child: Text(
                workspaceName.isNotEmpty ? workspaceName[0].toUpperCase() : 'W',
                style: TextStyle(
                    color: t.text,
                    fontWeight: FontWeight.w700,
                    fontSize: 12),
              ),
            ),
            SizedBox(width: t.sp3),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(workspaceName,
                      style: Theme.of(context)
                          .textTheme
                          .titleMedium
                          ?.copyWith(fontWeight: FontWeight.w600),
                      overflow: TextOverflow.ellipsis),
                  Text('Self-hosted',
                      style: Theme.of(context)
                          .textTheme
                          .bodySmall
                          ?.copyWith(color: t.textSubtle)),
                ],
              ),
            ),
            Icon(Icons.unfold_more,
                size: 14, color: t.textSubtle),
          ],
        ),
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  final String label;
  const _SectionLabel({required this.label});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Padding(
      padding:
          EdgeInsets.fromLTRB(t.sp4, t.sp3, t.sp4, t.sp1),
      child: Text(label.toUpperCase(),
          style: Theme.of(context).textTheme.labelSmall?.copyWith(
              color: t.textSubtle,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.0)),
    );
  }
}

class _NavTile extends StatelessWidget {
  final V1NavItem item;
  final bool active;
  const _NavTile({required this.item, required this.active});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    final fg = active ? t.accentText : t.textMuted;
    final bg = active ? t.accentSoft : Colors.transparent;
    return Padding(
      padding: EdgeInsets.symmetric(horizontal: t.sp2, vertical: 1),
      child: Material(
        color: bg,
        borderRadius: BorderRadius.circular(t.rMd),
        child: InkWell(
          borderRadius: BorderRadius.circular(t.rMd),
          onTap: item.comingSoon
              ? () => ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                      content: Text(
                          '${item.label} — coming in a future iteration')))
              : () => context.go(item.route),
          child: Padding(
            padding: EdgeInsets.symmetric(
                horizontal: t.sp3, vertical: 9),
            child: Row(
              children: [
                Icon(item.icon, size: 16, color: fg),
                SizedBox(width: t.sp3),
                Expanded(
                  child: Text(item.label,
                      style: TextStyle(
                          fontSize: 13,
                          fontWeight:
                              active ? FontWeight.w600 : FontWeight.w500,
                          color: fg)),
                ),
                if (item.comingSoon)
                  Container(
                    padding: EdgeInsets.symmetric(
                        horizontal: 6, vertical: 1),
                    decoration: BoxDecoration(
                      color: t.surface3,
                      borderRadius: BorderRadius.circular(t.rXs),
                    ),
                    child: Text('soon',
                        style: TextStyle(
                            fontSize: 9,
                            color: t.textSubtle,
                            fontWeight: FontWeight.w700,
                            letterSpacing: 0.5)),
                  )
                else if (item.badgeCount != null && item.badgeCount! > 0)
                  Container(
                    padding: EdgeInsets.symmetric(
                        horizontal: 6, vertical: 1),
                    decoration: BoxDecoration(
                      color: (item.badgeColor ?? t.accent)
                          .withValues(alpha: 0.18),
                      borderRadius: BorderRadius.circular(t.rXs),
                    ),
                    child: Text('${item.badgeCount}',
                        style: TextStyle(
                            fontSize: 10,
                            color: item.badgeColor ?? t.accentText,
                            fontWeight: FontWeight.w700)),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _UserChip extends StatelessWidget {
  final String initials;
  final String name;
  final String email;
  const _UserChip(
      {required this.initials, required this.name, required this.email});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return InkWell(
      onTap: () => context.go('/settings'),
      child: Padding(
        padding: EdgeInsets.symmetric(horizontal: t.sp4, vertical: t.sp3),
        child: Row(
          children: [
            Container(
              width: 30,
              height: 30,
              decoration: BoxDecoration(
                  color: t.surface3,
                  borderRadius: BorderRadius.circular(15)),
              alignment: Alignment.center,
              child: Text(initials.isEmpty ? '?' : initials.toUpperCase(),
                  style: TextStyle(
                      color: t.text,
                      fontWeight: FontWeight.w700,
                      fontSize: 11)),
            ),
            SizedBox(width: t.sp3),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(name.isEmpty ? 'Signed in' : name,
                      style: Theme.of(context)
                          .textTheme
                          .titleMedium
                          ?.copyWith(
                              fontWeight: FontWeight.w600, fontSize: 13),
                      overflow: TextOverflow.ellipsis),
                  if (email.isNotEmpty)
                    Text(email,
                        style: Theme.of(context)
                            .textTheme
                            .bodySmall
                            ?.copyWith(color: t.textSubtle, fontSize: 11),
                        overflow: TextOverflow.ellipsis),
                ],
              ),
            ),
            Icon(Icons.unfold_more, size: 14, color: t.textSubtle),
          ],
        ),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Top bar
// -----------------------------------------------------------------------------

class _V1TopBar extends StatelessWidget {
  final String workspaceName;
  final String current;
  const _V1TopBar(
      {required this.workspaceName, required this.current});

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Container(
      height: 52,
      color: t.bg,
      padding: EdgeInsets.symmetric(horizontal: t.sp5),
      child: Row(
        children: [
          Text(workspaceName,
              style: Theme.of(context)
                  .textTheme
                  .bodyMedium
                  ?.copyWith(color: t.textMuted)),
          SizedBox(width: t.sp2),
          Text('/',
              style: Theme.of(context)
                  .textTheme
                  .bodyMedium
                  ?.copyWith(color: t.textSubtle)),
          SizedBox(width: t.sp2),
          Text(current,
              style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                  fontWeight: FontWeight.w600, fontSize: 14)),
          const Spacer(),
          _SearchTrigger(),
          SizedBox(width: t.sp2),
          _IconBtn(
              icon: Icons.notifications_none,
              tooltip: 'Notifications',
              onTap: null),
          _IconBtn(
              icon: Icons.help_outline,
              tooltip: 'Help',
              onTap: null),
        ],
      ),
    );
  }
}

class _SearchTrigger extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return InkWell(
      onTap: () => ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
              content: Text('Command palette — coming soon (⌘K)'))),
      borderRadius: BorderRadius.circular(t.rMd),
      child: Container(
        height: 30,
        padding: EdgeInsets.symmetric(horizontal: t.sp3),
        decoration: BoxDecoration(
          color: t.surface,
          borderRadius: BorderRadius.circular(t.rMd),
          border: Border.all(color: t.border),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.search, size: 14, color: t.textSubtle),
            SizedBox(width: t.sp2),
            Text('Jump to session, file, or command',
                style: TextStyle(fontSize: 12, color: t.textSubtle)),
            SizedBox(width: t.sp3),
            Container(
              padding:
                  EdgeInsets.symmetric(horizontal: 6, vertical: 1),
              decoration: BoxDecoration(
                color: t.surface3,
                borderRadius: BorderRadius.circular(t.rXs),
                border: Border.all(color: t.border),
              ),
              child: Text('⌘K',
                  style: mono(size: 10, color: t.textSubtle)),
            ),
          ],
        ),
      ),
    );
  }
}

class _IconBtn extends StatelessWidget {
  final IconData icon;
  final String tooltip;
  final VoidCallback? onTap;
  const _IconBtn(
      {required this.icon, required this.tooltip, required this.onTap});
  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).extension<OpendrayTokens>()!;
    return Tooltip(
      message: tooltip,
      child: IconButton(
        onPressed: onTap,
        icon: Icon(icon, size: 16),
        color: t.textMuted,
        splashRadius: 16,
        constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
        padding: EdgeInsets.zero,
      ),
    );
  }
}
