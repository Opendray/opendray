import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/features/backups/backups_screen.dart';
import 'package:opendray/features/channels/channels_screen.dart';
import 'package:opendray/features/custom_tasks/custom_tasks_screen.dart';
import 'package:opendray/features/githosts/githosts_screen.dart';
import 'package:opendray/features/integrations/integrations_screen.dart';
import 'package:opendray/features/mcp/mcp_screen.dart';
import 'package:opendray/features/memory_cleanup/cleanup_inbox_screen.dart';
import 'package:opendray/features/more/about_screen.dart';
import 'package:opendray/features/project/project_screen.dart';
import 'package:opendray/features/providers/providers_screen.dart';
import 'package:opendray/features/settings/settings_screen.dart';
import 'package:opendray/features/skills/skills_screen.dart';

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
class MoreScreen extends ConsumerWidget {
  const MoreScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authControllerProvider);
    if (auth is! AuthLoggedIn) {
      return const Scaffold(body: SizedBox.shrink());
    }
    return Scaffold(
      appBar: AppBar(title: const Text('More')),
      body: ListView(
        children: [
          _IdentityCard(auth: auth),
          const SizedBox(height: 8),
          const _SectionHeader(label: 'Gateway'),
          _MenuTile(
            icon: Icons.api_outlined,
            title: 'Integrations',
            subtitle: 'API callers — recent activity & error rates',
            onTap: () => _push(context, const IntegrationsScreen()),
          ),
          _MenuTile(
            icon: Icons.notifications_outlined,
            title: 'Channels',
            subtitle: 'Notification destinations',
            onTap: () => _push(context, const ChannelsScreen()),
          ),
          _MenuTile(
            icon: Icons.psychology_outlined,
            title: 'Providers',
            subtitle: 'Claude / Codex / Gemini CLI status',
            onTap: () => _push(context, const ProvidersScreen()),
          ),
          _MenuTile(
            icon: Icons.extension_outlined,
            title: 'MCP',
            subtitle: 'Model Context Protocol servers & secrets',
            onTap: () => _push(context, const McpScreen()),
          ),
          _MenuTile(
            icon: Icons.auto_awesome_outlined,
            title: 'Skills',
            subtitle: 'Agent SKILL.md library (built-in + vault)',
            onTap: () => _push(context, const SkillsScreen()),
          ),
          _MenuTile(
            icon: Icons.account_tree_outlined,
            title: 'Git hosts',
            subtitle: 'PAT credentials for GitHub / GitLab / etc.',
            onTap: () => _push(context, const GitHostsScreen()),
          ),
          _MenuTile(
            icon: Icons.terminal_outlined,
            title: 'Custom tasks',
            subtitle: 'Slash commands shown in the session task picker',
            onTap: () => _push(context, const CustomTasksScreen()),
          ),
          const SizedBox(height: 8),
          const _SectionHeader(label: 'Memory'),
          _MenuTile(
            icon: Icons.flag_outlined,
            title: 'Project goal / plan / journal',
            subtitle: 'Per-cwd memory layers 2-4 + agent proposals',
            onTap: () => _push(context, const ProjectScreen()),
          ),
          _MenuTile(
            icon: Icons.cleaning_services_outlined,
            title: 'Cleanup inbox',
            subtitle:
                'LLM-proposed deletions / merges across all projects',
            onTap: () => _push(context, const CleanupInboxScreen()),
          ),
          const SizedBox(height: 8),
          const _SectionHeader(label: 'System'),
          _MenuTile(
            icon: Icons.backup_outlined,
            title: 'Backups',
            subtitle: 'Latest backup status & run-now',
            onTap: () => _push(context, const BackupsScreen()),
          ),
          _MenuTile(
            icon: Icons.tune_outlined,
            title: 'Settings',
            subtitle: 'Appearance, account',
            onTap: () => _push(context, const SettingsScreen()),
          ),
          _MenuTile(
            icon: Icons.info_outline,
            title: 'About',
            subtitle: 'Build version & server info',
            onTap: () => _push(context, const AboutScreen()),
          ),
          const Divider(height: 32),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
            child: OutlinedButton(
              style: OutlinedButton.styleFrom(
                foregroundColor: Theme.of(context).colorScheme.error,
                side: BorderSide(
                  color: Theme.of(context)
                      .colorScheme
                      .error
                      .withValues(alpha: 0.4),
                ),
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
              onPressed: () =>
                  ref.read(authControllerProvider.notifier).logout(),
              child: const Text('Sign out'),
            ),
          ),
        ],
      ),
    );
  }

  void _push(BuildContext context, Widget page) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(builder: (_) => page),
    );
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
              Text('Signed in as', style: muted),
              Text(
                auth.username,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 12),
              Text('Server', style: muted),
              Text(
                auth.serverUrl,
                style: Theme.of(context).textTheme.bodyMedium,
              ),
              const SizedBox(height: 12),
              Text('Token expires', style: muted),
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
              color: Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.6),
            ),
      ),
    );
  }
}

class _MenuTile extends StatelessWidget {
  const _MenuTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      onTap: onTap,
      leading: Icon(icon, color: Theme.of(context).colorScheme.primary),
      title: Text(title),
      subtitle: Text(
        subtitle,
        style: Theme.of(context).textTheme.bodySmall,
      ),
      trailing: const Icon(Icons.chevron_right),
    );
  }
}
