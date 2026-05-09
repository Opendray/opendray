import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/auth/auth_state.dart';

// "More" tab — overflow list + signed-in identity. F1 ships only
// sign-out and the diagnostics card; sub-pages (Channels,
// Integrations, Providers, Backups, Settings) land in F7–F11.
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
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Signed in as',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  Text(
                    auth.username,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  const SizedBox(height: 12),
                  Text(
                    'Server',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  Text(
                    auth.serverUrl,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 12),
                  Text(
                    'Token expires',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  Text(
                    DateFormat.yMMMd().add_jm().format(
                          auth.expiresAt.toLocal(),
                        ),
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          OutlinedButton(
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
        ],
      ),
    );
  }
}
