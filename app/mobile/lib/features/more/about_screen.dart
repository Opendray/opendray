import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:package_info_plus/package_info_plus.dart';

// Diagnostics screen for "what version is this and which server am
// I pointed at" — exactly the information that gets lost when bug
// reports come in over chat. Pulls package version from
// PackageInfo (so it tracks pubspec.yaml without a manual rebump)
// and the rest from auth state.
class AboutScreen extends ConsumerStatefulWidget {
  const AboutScreen({super.key});

  @override
  ConsumerState<AboutScreen> createState() => _AboutScreenState();
}

class _AboutScreenState extends ConsumerState<AboutScreen> {
  PackageInfo? _info;

  @override
  void initState() {
    super.initState();
    _loadInfo();
  }

  Future<void> _loadInfo() async {
    final info = await PackageInfo.fromPlatform();
    if (!mounted) return;
    setState(() => _info = info);
  }

  Future<void> _copy(String label, String value) async {
    await Clipboard.setData(ClipboardData(text: value));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('Copied $label'),
        duration: const Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authControllerProvider);
    final loggedIn = auth is AuthLoggedIn ? auth : null;
    final info = _info;
    return Scaffold(
      appBar: AppBar(title: const Text('About')),
      body: ListView(
        padding: const EdgeInsets.symmetric(vertical: 8),
        children: [
          const _SectionHeader(label: 'App'),
          if (info == null)
            const ListTile(
              leading: SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
              title: Text('Loading…'),
            )
          else ...[
            _kv(
              context,
              label: 'App',
              value: info.appName,
            ),
            _kv(
              context,
              label: 'Version',
              value: '${info.version} (build ${info.buildNumber})',
              mono: true,
              onCopy: () => _copy(
                'version',
                '${info.version}+${info.buildNumber}',
              ),
            ),
            _kv(
              context,
              label: 'Package',
              value: info.packageName,
              mono: true,
            ),
          ],
          const SizedBox(height: 8),
          if (loggedIn != null) ...[
            const _SectionHeader(label: 'Server'),
            _kv(
              context,
              label: 'URL',
              value: loggedIn.serverUrl,
              mono: true,
              onCopy: () => _copy('server URL', loggedIn.serverUrl),
            ),
            _kv(
              context,
              label: 'Signed in as',
              value: loggedIn.username,
            ),
            _kv(
              context,
              label: 'Token expires',
              value: DateFormat.yMMMd()
                  .add_Hms()
                  .format(loggedIn.expiresAt.toLocal()),
            ),
          ],
          const SizedBox(height: 24),
          Center(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Text(
                'opendray mobile — multi-CLI gateway control.\n'
                'Source: github.com/Opendray/opendray_v2',
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
          ),
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  Widget _kv(
    BuildContext context, {
    required String label,
    required String value,
    bool mono = false,
    VoidCallback? onCopy,
  }) {
    return ListTile(
      title: Text(label, style: Theme.of(context).textTheme.bodySmall),
      subtitle: Padding(
        padding: const EdgeInsets.only(top: 2),
        child: Text(
          value,
          style: TextStyle(
            fontSize: 14,
            fontFamily: mono ? 'monospace' : null,
          ),
        ),
      ),
      trailing: onCopy == null
          ? null
          : IconButton(
              icon: const Icon(Icons.copy_outlined, size: 18),
              tooltip: 'Copy',
              onPressed: onCopy,
            ),
      dense: true,
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 12, 16, 4),
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
