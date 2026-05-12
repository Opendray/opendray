// SettingsScreen — the operator's home for app-level preferences
// that aren't tied to a specific server resource. Reached via
// More → Settings.
//
// PR #51 ships the Appearance section. The Account section
// (change password / change username) lands in PR #52 — its
// placeholder is included here so the layout doesn't shift when
// it arrives.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/theme/theme_controller.dart';
import 'package:opendray/features/settings/change_credentials_screen.dart';
import 'package:opendray/features/settings/server_settings_screen.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final mode = ref.watch(themeControllerProvider);
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: SafeArea(
        bottom: false,
        child: ListView(
          padding: const EdgeInsets.symmetric(vertical: 8),
          children: [
            const _SectionHeader('Appearance'),
            // RadioGroup is the Flutter 3.32+ way to share group
            // state across Radio<T> descendants without each tile
            // duplicating groupValue/onChanged.
            RadioGroup<ThemeMode>(
              groupValue: mode,
              onChanged: (m) {
                if (m == null) return;
                ref.read(themeControllerProvider.notifier).setMode(m);
              },
              child: const Column(
                children: [
                  _ThemeOption(
                    title: 'System',
                    subtitle: "Follow your phone's appearance setting",
                    icon: Icons.brightness_auto_outlined,
                    value: ThemeMode.system,
                  ),
                  _ThemeOption(
                    title: 'Light',
                    subtitle: 'Always use the light palette',
                    icon: Icons.light_mode_outlined,
                    value: ThemeMode.light,
                  ),
                  _ThemeOption(
                    title: 'Dark',
                    subtitle: 'Always use the dark palette',
                    icon: Icons.dark_mode_outlined,
                    value: ThemeMode.dark,
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            const _SectionHeader('Account'),
            ListTile(
              leading: const Icon(Icons.lock_outline),
              title: const Text('Change credentials'),
              subtitle: Text(
                'Username and password',
                style: theme.textTheme.bodySmall,
              ),
              trailing: const Icon(Icons.chevron_right, size: 18),
              onTap: () => Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => const ChangeCredentialsScreen(),
                ),
              ),
            ),
            const SizedBox(height: 16),
            const _SectionHeader('Gateway'),
            ListTile(
              leading: const Icon(Icons.dns_outlined),
              title: const Text('Server settings'),
              subtitle: Text(
                'Listen address, logging, vault, memory, storage paths…',
                style: theme.textTheme.bodySmall,
              ),
              trailing: const Icon(Icons.chevron_right, size: 18),
              onTap: () => Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => const ServerSettingsScreen(),
                ),
              ),
            ),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader(this.label);
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 6),
      child: Text(
        label.toUpperCase(),
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.8,
          fontSize: 11,
        ),
      ),
    );
  }
}

class _ThemeOption extends StatelessWidget {
  const _ThemeOption({
    required this.title,
    required this.subtitle,
    required this.icon,
    required this.value,
  });

  final String title;
  final String subtitle;
  final IconData icon;
  final ThemeMode value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ListTile(
      leading: Icon(icon),
      title: Text(title),
      subtitle: Text(
        subtitle,
        style: theme.textTheme.bodySmall,
      ),
      trailing: Radio<ThemeMode>(value: value),
      // ListTile.onTap forwards through RadioGroup's onChanged
      // automatically via the Radio widget — no manual wiring
      // needed.
      onTap: () {
        // Find the ancestor RadioGroup and report this value as
        // the new selection. Reaching through context keeps the
        // option widget itself stateless.
        final group = RadioGroup.maybeOf<ThemeMode>(context);
        group?.onChanged(value);
      },
    );
  }
}
