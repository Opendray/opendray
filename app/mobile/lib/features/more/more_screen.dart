import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/features/_shared/placeholder_screen.dart';

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
            onTap: () => _push(
              context,
              const PlaceholderScreen(
                title: 'Integrations',
                icon: Icons.api_outlined,
                body: 'Per-integration call history and error rates '
                    'land in F8.',
              ),
            ),
          ),
          _MenuTile(
            icon: Icons.notifications_outlined,
            title: 'Channels',
            subtitle: 'Notification destinations',
            onTap: () => _push(
              context,
              const PlaceholderScreen(
                title: 'Channels',
                icon: Icons.notifications_outlined,
                body: 'Slack / email / webhook destinations land in F9.',
              ),
            ),
          ),
          _MenuTile(
            icon: Icons.psychology_outlined,
            title: 'Providers',
            subtitle: 'Claude / Codex / Gemini CLI status',
            onTap: () => _push(
              context,
              const PlaceholderScreen(
                title: 'Providers',
                icon: Icons.psychology_outlined,
                body: 'Read-only view of provider credentials and '
                    'reachability lands in F10.',
              ),
            ),
          ),
          const SizedBox(height: 8),
          const _SectionHeader(label: 'System'),
          _MenuTile(
            icon: Icons.backup_outlined,
            title: 'Backups',
            subtitle: 'Latest backup status',
            onTap: () => _push(
              context,
              const PlaceholderScreen(
                title: 'Backups',
                icon: Icons.backup_outlined,
                body: 'Backup status (read-only) lands in F11.',
              ),
            ),
          ),
          _MenuTile(
            icon: Icons.info_outline,
            title: 'About',
            subtitle: 'Build version & diagnostics',
            onTap: () => _push(
              context,
              const PlaceholderScreen(
                title: 'About',
                icon: Icons.info_outline,
                body: 'Version, build date, and diagnostics land later.',
              ),
            ),
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
