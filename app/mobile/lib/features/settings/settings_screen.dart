// SettingsScreen — the operator's home for app-level preferences
// that aren't tied to a specific server resource. Reached via
// More → Settings.
//
// All visible strings flow through slang (`t.settings.*`). The
// Language section at the top lets the user override the system
// locale; LocaleController persists the pick to shared_preferences.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/locale/locale_controller.dart';
import 'package:opendray/core/theme/theme_controller.dart';
import 'package:opendray/features/settings/change_credentials_screen.dart';
import 'package:opendray/features/settings/log_viewer_screen.dart';
import 'package:opendray/features/settings/server_settings_screen.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final mode = ref.watch(themeControllerProvider);
    final locale = ref.watch(localeControllerProvider);
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(t.settings.title)),
      body: SafeArea(
        bottom: false,
        child: ListView(
          padding: const EdgeInsets.symmetric(vertical: 8),
          children: [
            _SectionHeader(t.settings.language.section),
            RadioGroup<LocalePreference>(
              groupValue: locale,
              onChanged: (pref) {
                if (pref == null) return;
                ref
                    .read(localeControllerProvider.notifier)
                    .setPreference(pref);
              },
              child: Column(
                children: [
                  _LocaleOption(
                    title: t.settings.language.system,
                    subtitle: t.settings.language.systemSubtitle,
                    icon: Icons.language_outlined,
                    value: LocalePreference.system,
                  ),
                  _LocaleOption(
                    title: t.settings.language.english,
                    subtitle: 'English',
                    icon: Icons.translate_outlined,
                    value: LocalePreference.english,
                  ),
                  _LocaleOption(
                    title: t.settings.language.chinese,
                    subtitle: '中文',
                    icon: Icons.translate_outlined,
                    value: LocalePreference.chinese,
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            _SectionHeader(t.settings.appearance.section),
            // RadioGroup is the Flutter 3.32+ way to share group
            // state across Radio<T> descendants without each tile
            // duplicating groupValue/onChanged.
            RadioGroup<ThemeMode>(
              groupValue: mode,
              onChanged: (m) {
                if (m == null) return;
                ref.read(themeControllerProvider.notifier).setMode(m);
              },
              child: Column(
                children: [
                  _ThemeOption(
                    title: t.settings.appearance.system,
                    subtitle: t.settings.appearance.systemSubtitle,
                    icon: Icons.brightness_auto_outlined,
                    value: ThemeMode.system,
                  ),
                  _ThemeOption(
                    title: t.settings.appearance.light,
                    subtitle: t.settings.appearance.lightSubtitle,
                    icon: Icons.light_mode_outlined,
                    value: ThemeMode.light,
                  ),
                  _ThemeOption(
                    title: t.settings.appearance.dark,
                    subtitle: t.settings.appearance.darkSubtitle,
                    icon: Icons.dark_mode_outlined,
                    value: ThemeMode.dark,
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            _SectionHeader(t.settings.account.section),
            ListTile(
              leading: const Icon(Icons.lock_outline),
              title: Text(t.settings.account.changeCredentials),
              subtitle: Text(
                t.settings.account.changeCredentialsSubtitle,
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
            _SectionHeader(t.settings.gateway.section),
            ListTile(
              leading: const Icon(Icons.dns_outlined),
              title: Text(t.settings.gateway.serverSettings),
              subtitle: Text(
                t.settings.gateway.serverSettingsSubtitle,
                style: theme.textTheme.bodySmall,
              ),
              trailing: const Icon(Icons.chevron_right, size: 18),
              onTap: () => Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => const ServerSettingsScreen(),
                ),
              ),
            ),
            ListTile(
              leading: const Icon(Icons.subject_outlined),
              title: Text(t.settings.gateway.liveLogs),
              subtitle: Text(
                t.settings.gateway.liveLogsSubtitle,
                style: theme.textTheme.bodySmall,
              ),
              trailing: const Icon(Icons.chevron_right, size: 18),
              onTap: () => Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => const LogViewerScreen(),
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
      onTap: () {
        final group = RadioGroup.maybeOf<ThemeMode>(context);
        group?.onChanged(value);
      },
    );
  }
}

class _LocaleOption extends StatelessWidget {
  const _LocaleOption({
    required this.title,
    required this.subtitle,
    required this.icon,
    required this.value,
  });

  final String title;
  final String subtitle;
  final IconData icon;
  final LocalePreference value;

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
      trailing: Radio<LocalePreference>(value: value),
      onTap: () {
        final group = RadioGroup.maybeOf<LocalePreference>(context);
        group?.onChanged(value);
      },
    );
  }
}
