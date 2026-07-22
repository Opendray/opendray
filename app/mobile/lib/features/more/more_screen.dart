import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/releases_api.dart';
import 'package:opendray/core/api/version_api.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/activity/activity_screen.dart';
import 'package:opendray/features/backups/backups_screen.dart';
import 'package:opendray/features/channels/channels_screen.dart';
import 'package:opendray/features/custom_tasks/custom_tasks_screen.dart';
import 'package:opendray/features/data_export/data_export_screen.dart';
import 'package:opendray/features/githosts/githosts_screen.dart';
import 'package:opendray/features/mcp/mcp_screen.dart';
import 'package:opendray/features/memory_archived/archived_screen.dart';
import 'package:opendray/features/memory_quarantine/quarantine_screen.dart';
import 'package:opendray/features/more/about_screen.dart';
import 'package:opendray/features/more/updates_sheet.dart';
import 'package:opendray/features/notes/notes_screen.dart';
import 'package:opendray/features/project/project_screen.dart';
import 'package:opendray/features/providers/providers_screen.dart';
import 'package:opendray/features/settings/settings_screen.dart';
import 'package:opendray/features/skills/skills_screen.dart';
import 'package:url_launcher/url_launcher.dart';

// "More" tab — the app's settings / overflow surface for everything that
// doesn't earn its own bottom-nav slot.
//
// Layout: a compact profile header + a pinned search field on top, then a
// launcher-style 2-column grid of icon tiles grouped by section (Gateway /
// Plugins / Memory / System / Resources). The search filters every tile
// across all sections at once, so an 18-entry menu is one keystroke away.
// Sub-pages route via Navigator.push (owned by this tab, no deep-linking).
const _docsUrl = 'https://opendray.dev/docs/';
const _communityUrl = 'https://t.me/opendraycommunity';
const _sponsorUrl = 'https://opendray.dev/sponsors/';

class MoreScreen extends ConsumerStatefulWidget {
  const MoreScreen({super.key});

  @override
  ConsumerState<MoreScreen> createState() => _MoreScreenState();
}

class _MoreScreenState extends ConsumerState<MoreScreen> {
  String _query = '';

  void _push(Widget page) {
    Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => page));
  }

  Future<void> _openUrl(String url) async {
    final uri = Uri.tryParse(url);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } on Object {
      // Best-effort convenience link.
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authControllerProvider);
    final version = ref.watch(versionInfoProvider).asData?.value;
    final updateAvailable = version?.updateAvailable ?? false;
    // Updates "unread" badge — mirror the web sidebar: prefer the fetched
    // GitHub release version, fall back to the gateway's `latest` so a
    // failed notes fetch still badges when a binary update is waiting.
    final release = ref.watch(latestReleaseProvider).asData?.value;
    final lastRead = ref.watch(lastReadReleaseProvider);
    final latestForUnread =
        release?.version ?? normalizeReleaseVersion(version?.latest);
    final notesUnread = isReleaseUnread(latestForUnread, lastRead);
    final showUpdatesBadge = notesUnread || updateAvailable;
    final highlightCount = release?.highlights.length ?? 0;
    if (auth is! AuthLoggedIn) {
      return const Scaffold(body: SizedBox.shrink());
    }

    final sections = _buildSections(
      updateAvailable: updateAvailable,
      showUpdatesBadge: showUpdatesBadge,
      notesUnread: notesUnread,
      highlightCount: highlightCount,
    );
    final q = _query.trim().toLowerCase();
    final visible = q.isEmpty
        ? sections
        : [
            for (final s in sections)
              if (s.items.any((it) => it.matches(q)))
                _Section(
                  s.label,
                  s.items.where((it) => it.matches(q)).toList(),
                ),
          ];

    return Scaffold(
      appBar: AppBar(title: Text(t.more.title)),
      body: Column(
        children: [
          _ProfileHeader(auth: auth),
          _SearchField(
            onChanged: (v) => setState(() => _query = v),
          ),
          Expanded(
            child: visible.isEmpty
                ? _EmptyResults(query: _query.trim())
                : ListView(
                    padding: const EdgeInsets.only(top: 4, bottom: 8),
                    children: [
                      for (final section in visible) ...[
                        _SectionHeader(label: section.label),
                        _TileGrid(items: section.items),
                      ],
                      // Actions only belong on the full (unsearched) page.
                      if (q.isEmpty) ...[
                        const SizedBox(height: 20),
                        _SignOutButton(
                          onPressed: () => ref
                              .read(authControllerProvider.notifier)
                              .logout(),
                        ),
                        if (version != null && version.current.isNotEmpty)
                          _VersionStamp(version: version.current),
                      ],
                    ],
                  ),
          ),
        ],
      ),
    );
  }

  // The full menu, grouped by section. Built here (not const) because the
  // Updates entry's badge/count depend on live release state.
  List<_Section> _buildSections({
    required bool updateAvailable,
    required bool showUpdatesBadge,
    required bool notesUnread,
    required int highlightCount,
  }) {
    final scheme = Theme.of(context).colorScheme;
    return [
      _Section(t.more.sections.gateway, [
        _MenuItem(
          icon: Icons.timeline_outlined,
          title: t.more.items.activity.title,
          subtitle: t.more.items.activity.subtitle,
          onTap: () => _push(const ActivityScreen()),
        ),
        _MenuItem(
          icon: Icons.notifications_outlined,
          title: t.more.items.channels.title,
          subtitle: t.more.items.channels.subtitle,
          onTap: () => _push(const ChannelsScreen()),
        ),
        _MenuItem(
          icon: Icons.psychology_outlined,
          title: t.more.items.providers.title,
          subtitle: t.more.items.providers.subtitle,
          onTap: () => _push(const ProvidersScreen()),
        ),
      ]),
      _Section(t.more.sections.plugins, [
        _MenuItem(
          icon: Icons.extension_outlined,
          title: t.more.items.mcp.title,
          subtitle: t.more.items.mcp.subtitle,
          onTap: () => _push(const McpScreen()),
        ),
        _MenuItem(
          icon: Icons.auto_awesome_outlined,
          title: t.more.items.skills.title,
          subtitle: t.more.items.skills.subtitle,
          onTap: () => _push(const SkillsScreen()),
        ),
        _MenuItem(
          icon: Icons.account_tree_outlined,
          title: t.more.items.gitHosts.title,
          subtitle: t.more.items.gitHosts.subtitle,
          onTap: () => _push(const GitHostsScreen()),
        ),
        _MenuItem(
          icon: Icons.terminal_outlined,
          title: t.more.items.customTasks.title,
          subtitle: t.more.items.customTasks.subtitle,
          onTap: () => _push(const CustomTasksScreen()),
        ),
      ]),
      _Section(t.more.sections.memory, [
        _MenuItem(
          icon: Icons.flag_outlined,
          title: t.more.items.projectMemory.title,
          subtitle: t.more.items.projectMemory.subtitle,
          onTap: () => _push(const ProjectScreen()),
        ),
        _MenuItem(
          icon: Icons.inventory_2_outlined,
          title: t.more.items.archived.title,
          subtitle: t.more.items.archived.subtitle,
          onTap: () => _push(const ArchivedMemoriesScreen()),
        ),
        _MenuItem(
          icon: Icons.shield_outlined,
          title: t.more.items.quarantine.title,
          subtitle: t.more.items.quarantine.subtitle,
          onTap: () => _push(const QuarantineScreen()),
        ),
        _MenuItem(
          icon: Icons.folder_outlined,
          title: t.more.items.vault.title,
          subtitle: t.more.items.vault.subtitle,
          onTap: () => _push(const NotesVaultScreen()),
        ),
      ]),
      _Section(t.more.sections.system, [
        _MenuItem(
          icon: Icons.backup_outlined,
          title: t.more.items.backups.title,
          subtitle: t.more.items.backups.subtitle,
          onTap: () => _push(const BackupsScreen()),
        ),
        _MenuItem(
          icon: Icons.import_export_outlined,
          title: t.more.items.dataExport.title,
          subtitle: t.more.items.dataExport.subtitle,
          onTap: () => _push(const DataExportScreen()),
        ),
        _MenuItem(
          icon: Icons.tune_outlined,
          title: t.more.items.settings.title,
          subtitle: t.more.items.settings.subtitle,
          badge: updateAvailable,
          onTap: () => _push(const SettingsScreen()),
        ),
        _MenuItem(
          icon: Icons.info_outline,
          title: t.more.items.about.title,
          subtitle: t.more.items.about.subtitle,
          onTap: () => _push(const AboutScreen()),
        ),
      ]),
      _Section(t.nav.resources, [
        _MenuItem(
          icon: Icons.auto_awesome_outlined,
          title: t.nav.updates.title,
          badge: showUpdatesBadge,
          // Accent-primary when only release notes are unread; the system
          // error red is reserved for a waiting binary update (matches
          // web's accent-vs-primary dot).
          badgeColor: updateAvailable ? scheme.error : scheme.primary,
          trailingText:
              (notesUnread && highlightCount > 0) ? '$highlightCount' : null,
          onTap: () => showUpdatesSheet(context),
        ),
        _MenuItem(
          icon: Icons.menu_book_outlined,
          title: t.nav.docs,
          external: true,
          onTap: () => _openUrl(_docsUrl),
        ),
        _MenuItem(
          icon: Icons.forum_outlined,
          title: t.nav.community,
          external: true,
          onTap: () => _openUrl(_communityUrl),
        ),
        _MenuItem(
          icon: Icons.favorite_outline,
          title: t.nav.sponsor,
          external: true,
          onTap: () => _openUrl(_sponsorUrl),
        ),
      ]),
    ];
  }
}

// ── data model ─────────────────────────────────────────────────────────

class _MenuItem {
  const _MenuItem({
    required this.icon,
    required this.title,
    required this.onTap,
    this.subtitle,
    this.badge = false,
    this.badgeColor,
    this.trailingText,
    this.external = false,
  });

  final IconData icon;
  final String title;
  final String? subtitle;
  final VoidCallback onTap;
  final bool badge;
  final Color? badgeColor;
  final String? trailingText;
  final bool external;

  bool matches(String q) =>
      title.toLowerCase().contains(q) ||
      (subtitle?.toLowerCase().contains(q) ?? false);
}

class _Section {
  const _Section(this.label, this.items);
  final String label;
  final List<_MenuItem> items;
}

// ── header + search ────────────────────────────────────────────────────

// Compact account block pinned above the grid: monogram avatar, username,
// and server / token-expiry on muted meta lines.
class _ProfileHeader extends StatelessWidget {
  const _ProfileHeader({required this.auth});
  final AuthLoggedIn auth;

  String get _initials {
    final name = auth.username.trim();
    if (name.isEmpty) return '?';
    final parts = name.split(RegExp(r'[\s._-]+')).where((p) => p.isNotEmpty);
    if (parts.length >= 2) {
      return (parts.first[0] + parts.elementAt(1)[0]).toUpperCase();
    }
    return name.substring(0, name.length >= 2 ? 2 : 1).toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final muted = scheme.onSurface.withValues(alpha: 0.6);
    final expires = DateFormat.yMMMd().format(auth.expiresAt.toLocal());
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Row(
            children: [
              Container(
                width: 46,
                height: 46,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: scheme.primary.withValues(alpha: 0.14),
                  shape: BoxShape.circle,
                ),
                child: Text(
                  _initials,
                  style: theme.textTheme.titleMedium?.copyWith(
                    color: scheme.primary,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              const SizedBox(width: 13),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      auth.username,
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.w600),
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      auth.serverUrl,
                      style: theme.textTheme.bodySmall?.copyWith(color: muted),
                      overflow: TextOverflow.ellipsis,
                    ),
                    Text(
                      '${t.more.identity.tokenExpires} $expires',
                      style: theme.textTheme.bodySmall?.copyWith(color: muted),
                      overflow: TextOverflow.ellipsis,
                    ),
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

class _SearchField extends StatelessWidget {
  const _SearchField({required this.onChanged});
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 4, 12, 8),
      child: TextField(
        onChanged: onChanged,
        textInputAction: TextInputAction.search,
        decoration: InputDecoration(
          prefixIcon: const Icon(Icons.search, size: 20),
          hintText: t.more.searchHint,
          isDense: true,
        ),
      ),
    );
  }
}

class _EmptyResults extends StatelessWidget {
  const _EmptyResults({required this.query});
  final String query;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final muted = theme.colorScheme.onSurface.withValues(alpha: 0.55);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.search_off, size: 40, color: muted),
            const SizedBox(height: 12),
            Text(
              t.more.noResults(query: query),
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyMedium?.copyWith(color: muted),
            ),
          ],
        ),
      ),
    );
  }
}

// ── grid ───────────────────────────────────────────────────────────────

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 8),
      child: Text(
        label.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
              letterSpacing: 0.8,
              fontWeight: FontWeight.w600,
              color:
                  Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
            ),
      ),
    );
  }
}

class _TileGrid extends StatelessWidget {
  const _TileGrid({required this.items});
  final List<_MenuItem> items;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: GridView.builder(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        padding: EdgeInsets.zero,
        itemCount: items.length,
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 2,
          mainAxisExtent: 96,
          crossAxisSpacing: 10,
          mainAxisSpacing: 10,
        ),
        itemBuilder: (_, i) => _GridTile(item: items[i]),
      ),
    );
  }
}

// One launcher tile: tinted icon chip top-left (with an optional status
// dot / external glyph), then title + one-line subtitle beneath. Kept on
// the single gold accent so the grid reads as one system, not a mosaic.
class _GridTile extends StatelessWidget {
  const _GridTile({required this.item});
  final _MenuItem item;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: item.onTap,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(13, 12, 12, 12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    width: 34,
                    height: 34,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: scheme.primary.withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(9),
                    ),
                    child: Icon(item.icon, size: 19, color: scheme.primary),
                  ),
                  const Spacer(),
                  if (item.trailingText != null)
                    Padding(
                      padding: const EdgeInsets.only(right: 2),
                      child: Text(
                        item.trailingText!,
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: scheme.onSurface.withValues(alpha: 0.55),
                        ),
                      ),
                    ),
                  if (item.badge)
                    Container(
                      width: 8,
                      height: 8,
                      decoration: BoxDecoration(
                        color: item.badgeColor ?? scheme.error,
                        shape: BoxShape.circle,
                      ),
                    )
                  else if (item.external)
                    Icon(Icons.open_in_new,
                        size: 15,
                        color: scheme.onSurface.withValues(alpha: 0.35)),
                ],
              ),
              const Spacer(),
              Text(
                item.title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodyMedium
                    ?.copyWith(fontWeight: FontWeight.w600),
              ),
              if (item.subtitle != null)
                Padding(
                  padding: const EdgeInsets.only(top: 1),
                  child: Text(
                    item.subtitle!,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: scheme.onSurface.withValues(alpha: 0.55),
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

// ── footer ─────────────────────────────────────────────────────────────

class _SignOutButton extends StatelessWidget {
  const _SignOutButton({required this.onPressed});
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
      child: OutlinedButton.icon(
        style: OutlinedButton.styleFrom(
          foregroundColor: scheme.error,
          side: BorderSide(color: scheme.error.withValues(alpha: 0.4)),
          padding: const EdgeInsets.symmetric(vertical: 14),
        ),
        onPressed: onPressed,
        icon: const Icon(Icons.logout, size: 18),
        label: Text(t.more.signOut),
      ),
    );
  }
}

class _VersionStamp extends StatelessWidget {
  const _VersionStamp({required this.version});
  final String version;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 4, 16, 28),
      child: Center(
        child: Text(
          'opendray $version',
          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color:
                    Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4),
              ),
        ),
      ),
    );
  }
}
