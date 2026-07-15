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

// "More" tab — overflow menu for everything that doesn't earn its
// own bottom-nav slot. Three sections: identity card, navigation
// list, destructive sign-out. Sub-pages route via Navigator.push
// (not go_router) because they're owned by this tab and don't need
// deep-linking from outside.
//
// Sub-pages still ship as PlaceholderScreen until F8–F11 fill them
// in — Integrations first (highest signal: every operator wants
// "who's calling me right now"), then Channels, Providers, Backups,
// About.
// External Resources links — mirror app/web/src/components/SidebarNav.tsx.
const _docsUrl = 'https://opendray.dev/docs/';
const _communityUrl = 'https://t.me/opendraycommunity';
const _sponsorUrl = 'https://opendray.dev/sponsors/';

class MoreScreen extends ConsumerWidget {
  const MoreScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
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
    return Scaffold(
      appBar: AppBar(title: Text(t.more.title)),
      body: ListView(
        children: [
          _IdentityCard(auth: auth),
          const SizedBox(height: 8),
          // Round Table + Integrations are top-level bottom-nav tabs now;
          // Activity (per-call integration audit) lives here in the gateway
          // section alongside the lower-frequency destinations.
          _SectionHeader(label: t.more.sections.gateway),
          _MenuTile(
            icon: Icons.timeline_outlined,
            title: t.more.items.activity.title,
            subtitle: t.more.items.activity.subtitle,
            onTap: () => _push(context, const ActivityScreen()),
          ),
          _MenuTile(
            icon: Icons.notifications_outlined,
            title: t.more.items.channels.title,
            subtitle: t.more.items.channels.subtitle,
            onTap: () => _push(context, const ChannelsScreen()),
          ),
          _MenuTile(
            icon: Icons.psychology_outlined,
            title: t.more.items.providers.title,
            subtitle: t.more.items.providers.subtitle,
            onTap: () => _push(context, const ProvidersScreen()),
          ),
          const SizedBox(height: 8),
          _SectionHeader(label: t.more.sections.plugins),
          _MenuTile(
            icon: Icons.extension_outlined,
            title: t.more.items.mcp.title,
            subtitle: t.more.items.mcp.subtitle,
            onTap: () => _push(context, const McpScreen()),
          ),
          _MenuTile(
            icon: Icons.auto_awesome_outlined,
            title: t.more.items.skills.title,
            subtitle: t.more.items.skills.subtitle,
            onTap: () => _push(context, const SkillsScreen()),
          ),
          _MenuTile(
            icon: Icons.account_tree_outlined,
            title: t.more.items.gitHosts.title,
            subtitle: t.more.items.gitHosts.subtitle,
            onTap: () => _push(context, const GitHostsScreen()),
          ),
          _MenuTile(
            icon: Icons.terminal_outlined,
            title: t.more.items.customTasks.title,
            subtitle: t.more.items.customTasks.subtitle,
            onTap: () => _push(context, const CustomTasksScreen()),
          ),
          const SizedBox(height: 8),
          // Cortex hub is the bottom-nav "Cortex" tab and its ⚙ opens the
          // unified Cortex settings (workers + capture/injection +
          // providers) — so capture/injection no longer needs its own More
          // entry. This section keeps the deeper, lower-frequency tools.
          _SectionHeader(label: t.more.sections.memory),
          _MenuTile(
            icon: Icons.flag_outlined,
            title: t.more.items.projectMemory.title,
            subtitle: t.more.items.projectMemory.subtitle,
            onTap: () => _push(context, const ProjectScreen()),
          ),
          _MenuTile(
            icon: Icons.inventory_2_outlined,
            title: t.more.items.archived.title,
            subtitle: t.more.items.archived.subtitle,
            onTap: () => _push(context, const ArchivedMemoriesScreen()),
          ),
          _MenuTile(
            icon: Icons.shield_outlined,
            title: t.more.items.quarantine.title,
            subtitle: t.more.items.quarantine.subtitle,
            onTap: () => _push(context, const QuarantineScreen()),
          ),
          _MenuTile(
            icon: Icons.folder_outlined,
            title: t.more.items.vault.title,
            subtitle: t.more.items.vault.subtitle,
            onTap: () => _push(context, const NotesVaultScreen()),
          ),
          const SizedBox(height: 8),
          _SectionHeader(label: t.more.sections.system),
          _MenuTile(
            icon: Icons.backup_outlined,
            title: t.more.items.backups.title,
            subtitle: t.more.items.backups.subtitle,
            onTap: () => _push(context, const BackupsScreen()),
          ),
          _MenuTile(
            icon: Icons.import_export_outlined,
            title: t.more.items.dataExport.title,
            subtitle: t.more.items.dataExport.subtitle,
            onTap: () => _push(context, const DataExportScreen()),
          ),
          _MenuTile(
            icon: Icons.tune_outlined,
            title: t.more.items.settings.title,
            subtitle: t.more.items.settings.subtitle,
            badge: updateAvailable,
            onTap: () => _push(context, const SettingsScreen()),
          ),
          _MenuTile(
            icon: Icons.info_outline,
            title: t.more.items.about.title,
            subtitle: t.more.items.about.subtitle,
            onTap: () => _push(context, const AboutScreen()),
          ),
          const SizedBox(height: 8),
          _SectionHeader(label: t.nav.resources),
          _MenuTile(
            icon: Icons.auto_awesome_outlined,
            title: t.nav.updates.title,
            badge: showUpdatesBadge,
            // Accent-primary when only release notes are unread; the
            // system error red is reserved for a waiting binary update
            // (matches web's accent-vs-primary dot).
            badgeColor: updateAvailable
                ? Theme.of(context).colorScheme.error
                : Theme.of(context).colorScheme.primary,
            trailingText: (notesUnread && highlightCount > 0)
                ? '· $highlightCount'
                : null,
            onTap: () => showUpdatesSheet(context),
          ),
          _MenuTile(
            icon: Icons.menu_book_outlined,
            title: t.nav.docs,
            external: true,
            onTap: () => _openUrl(_docsUrl),
          ),
          _MenuTile(
            icon: Icons.forum_outlined,
            title: t.nav.community,
            external: true,
            onTap: () => _openUrl(_communityUrl),
          ),
          _MenuTile(
            icon: Icons.favorite_outline,
            title: t.nav.sponsor,
            external: true,
            onTap: () => _openUrl(_sponsorUrl),
          ),
          const Divider(height: 32),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
            child: OutlinedButton(
              style: OutlinedButton.styleFrom(
                foregroundColor: Theme.of(context).colorScheme.error,
                side: BorderSide(
                  color: Theme.of(
                    context,
                  ).colorScheme.error.withValues(alpha: 0.4),
                ),
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
              onPressed: () =>
                  ref.read(authControllerProvider.notifier).logout(),
              child: Text(t.more.signOut),
            ),
          ),
        ],
      ),
    );
  }

  void _push(BuildContext context, Widget page) {
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
}

class _IdentityCard extends StatelessWidget {
  const _IdentityCard({required this.auth});
  final AuthLoggedIn auth;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 0),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(t.more.identity.signedInAs, style: muted),
              Text(
                auth.username,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 12),
              Text(t.more.identity.server, style: muted),
              Text(
                auth.serverUrl,
                style: Theme.of(context).textTheme.bodyMedium,
              ),
              const SizedBox(height: 12),
              Text(t.more.identity.tokenExpires, style: muted),
              Text(
                DateFormat.yMMMd().add_jm().format(auth.expiresAt.toLocal()),
                style: Theme.of(context).textTheme.bodyMedium,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 12, 16, 6),
      child: Text(
        label.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
          letterSpacing: 0.8,
          color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
        ),
      ),
    );
  }
}

class _MenuTile extends StatelessWidget {
  const _MenuTile({
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
  // Optional secondary line; Resources links render title-only.
  final String? subtitle;
  final VoidCallback onTap;
  // When true, shows a status dot before the trailing icon.
  final bool badge;
  // Dot colour; defaults to the error colour (waiting binary update).
  final Color? badgeColor;
  // Small muted count chip (e.g. "· 3") shown before the trailing icon.
  final String? trailingText;
  // External links show an open-in-new glyph instead of a chevron.
  final bool external;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final subtitle = this.subtitle;
    return ListTile(
      onTap: onTap,
      leading: Icon(icon, color: theme.colorScheme.primary),
      title: Text(title),
      subtitle: subtitle == null
          ? null
          : Text(subtitle, style: theme.textTheme.bodySmall),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (trailingText != null)
            Padding(
              padding: const EdgeInsets.only(right: 8),
              child: Text(
                trailingText!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
                ),
              ),
            ),
          if (badge)
            Container(
              width: 8,
              height: 8,
              margin: const EdgeInsets.only(right: 8),
              decoration: BoxDecoration(
                color: badgeColor ?? theme.colorScheme.error,
                shape: BoxShape.circle,
              ),
            ),
          Icon(external ? Icons.open_in_new : Icons.chevron_right,
              size: external ? 18 : null,
              color: external
                  ? theme.colorScheme.onSurface.withValues(alpha: 0.4)
                  : null),
        ],
      ),
    );
  }
}
