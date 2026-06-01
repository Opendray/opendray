import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/version_api.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:url_launcher/url_launcher.dart';

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
        content: Text(t.about.copied(label: label)),
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
      appBar: AppBar(title: Text(t.about.title)),
      body: ListView(
        padding: const EdgeInsets.symmetric(vertical: 8),
        children: [
          _SectionHeader(label: t.about.sections.app),
          if (info == null)
            ListTile(
              leading: const SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
              title: Text(t.about.loading),
            )
          else ...[
            _kv(
              context,
              label: t.about.fields.app,
              value: info.appName,
            ),
            _kv(
              context,
              label: t.about.fields.version,
              value: t.about.fields.versionFormat(
                version: info.version,
                build: info.buildNumber,
              ),
              mono: true,
              onCopy: () => _copy(
                t.about.copyLabels.version,
                '${info.version}+${info.buildNumber}',
              ),
            ),
            _kv(
              context,
              label: t.about.fields.package,
              value: info.packageName,
              mono: true,
            ),
          ],
          const SizedBox(height: 8),
          if (loggedIn != null) ...[
            _SectionHeader(label: t.about.sections.server),
            _kv(
              context,
              label: t.about.fields.url,
              value: loggedIn.serverUrl,
              mono: true,
              onCopy: () => _copy(t.about.copyLabels.serverUrl, loggedIn.serverUrl),
            ),
            _kv(
              context,
              label: t.about.fields.signedInAs,
              value: loggedIn.username,
            ),
            _kv(
              context,
              label: t.about.fields.tokenExpires,
              value: DateFormat.yMMMd()
                  .add_Hms()
                  .format(loggedIn.expiresAt.toLocal()),
            ),
            const SizedBox(height: 8),
            _SectionHeader(label: t.about.sections.gateway),
            const _GatewayVersion(),
          ],
          const SizedBox(height: 24),
          Center(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Text(
                t.about.tagline,
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
              tooltip: t.about.copyTooltip,
              onPressed: onCopy,
            ),
      dense: true,
    );
  }
}

// _GatewayVersion shows the running gateway's version + update status,
// read-only (no self-update trigger — see version_api.dart). The
// /version endpoint soft-fails, so an unreachable release check renders
// as "update check unavailable" rather than an error.
class _GatewayVersion extends ConsumerWidget {
  const _GatewayVersion();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final async = ref.watch(versionInfoProvider);
    return async.when(
      loading: () => ListTile(
        dense: true,
        leading: const SizedBox(
          width: 18,
          height: 18,
          child: CircularProgressIndicator(strokeWidth: 2),
        ),
        title: Text(t.about.gateway.checking,
            style: theme.textTheme.bodySmall),
      ),
      error: (e, _) => ListTile(
        dense: true,
        title: Text(t.about.gateway.checkFailed,
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.outline)),
      ),
      data: (v) {
        final muted = theme.colorScheme.onSurface.withValues(alpha: 0.6);
        final hasUpdate = v.checkError == null && v.updateAvailable;
        final statusText = v.checkError != null
            ? t.about.gateway.checkFailed
            : v.updateAvailable
                ? t.about.gateway.updateAvailable(version: v.latest ?? '')
                : t.about.gateway.upToDate;
        final statusColor = hasUpdate ? theme.colorScheme.primary : muted;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ListTile(
              dense: true,
              title: Text(t.about.gateway.version,
                  style: theme.textTheme.bodySmall),
              subtitle: Padding(
                padding: const EdgeInsets.only(top: 2),
                child: Text(
                  v.commit != null && v.commit!.isNotEmpty
                      ? '${v.current} (${_shortSha(v.commit!)})'
                      : v.current,
                  style: const TextStyle(fontSize: 14, fontFamily: 'monospace'),
                ),
              ),
              trailing: Text(statusText,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: statusColor)),
            ),
            if (v.updateAvailable && v.notesUrl != null)
              Padding(
                padding: const EdgeInsets.only(left: 16, bottom: 4),
                child: Align(
                  alignment: Alignment.centerLeft,
                  child: TextButton.icon(
                    icon: const Icon(Icons.open_in_new, size: 16),
                    label: Text(t.about.gateway.releaseNotes),
                    onPressed: () => _openNotes(v.notesUrl!),
                  ),
                ),
              ),
          ],
        );
      },
    );
  }

  static String _shortSha(String sha) =>
      sha.length > 7 ? sha.substring(0, 7) : sha;

  Future<void> _openNotes(String url) async {
    final uri = Uri.tryParse(url);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } on Object {
      // Best-effort convenience link.
    }
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
