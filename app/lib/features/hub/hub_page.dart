import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';
import '../plugins/plugin_config_form.dart';

/// Marketplace entry point at `/hub`.
///
/// Lists every Entry returned by GET /api/marketplace/plugins and
/// drives the install flow: tap "Install" → POST /api/plugins/install
/// with `src=marketplace://NAME@VERSION` → consent preview dialog →
/// POST /api/plugins/install/confirm.
///
/// Already-installed entries render a disabled "Installed" button
/// rather than letting the user kick off a duplicate install.
class HubPage extends StatefulWidget {
  const HubPage({super.key});

  @override
  State<HubPage> createState() => _HubPageState();
}

enum _LoadState { loading, loaded, error }

class _HubPageState extends State<HubPage> {
  _LoadState _state = _LoadState.loading;
  List<MarketplaceEntry> _entries = const [];
  Set<String> _installedNames = const {};
  String? _error;
  bool _busy = false;

  ApiClient get _api => context.read<ApiClient>();

  /// Localized display name: picks the manifest's `displayName_zh`
  /// overlay when the user's locale is zh and it's non-empty; else
  /// falls back to the English source or, as a last resort, the
  /// plugin's bare `name`. Subscribes via watch so the widget
  /// rebuilds on language switch.
  String _resolveDisplayName(MarketplaceEntry entry) {
    final localized =
        context.pickL10n(entry.displayName, entry.displayNameZh);
    return localized.isEmpty ? entry.name : localized;
  }

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    if (!mounted) return;
    setState(() {
      _state = _LoadState.loading;
      _error = null;
    });
    try {
      // Fetch catalog + installed list in parallel so the "Installed"
      // badge is accurate on first paint.
      final results = await Future.wait([
        _api.listMarketplace(),
        _api.listProviders(),
      ]);
      final entries = results[0] as List<MarketplaceEntry>;
      final providers = results[1] as List;
      if (!mounted) return;
      setState(() {
        _entries = entries;
        _installedNames = {for (final p in providers) p.provider.name as String};
        _state = _LoadState.loaded;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _state = _LoadState.error;
      });
    }
  }

  void _notify(String msg, {bool isError = false}) {
    final messenger = ScaffoldMessenger.maybeOf(context);
    messenger?.showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: isError ? AppColors.error : null,
    ));
  }

  Future<void> _install(MarketplaceEntry entry) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      final pending = await _api.installPluginFromMarketplace(entry.ref);
      if (!mounted) return;
      final confirm = await _showConsentDialog(entry, pending);
      if (confirm != true) {
        // User declined — the token simply expires server-side.
        return;
      }
      final installedName = await _api.confirmPluginInstall(pending.token);

      // Plugins that declare a configSchema land in the Hub with
      // blank values. We open the configure form right after confirm
      // so the user doesn't have to dig for it — skip is allowed and
      // the plugin will just surface a "not configured" error until
      // the user comes back to Plugin → Configure.
      if (mounted && entry.configSchema.isNotEmpty) {
        await _promptConfigureAfterInstall(entry, installedName);
      }

      _notify('Installed $installedName');
      ProvidersBus.instance.notify();
      await _load();
    } catch (e) {
      _notify('Install failed: $e', isError: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// Post-install configuration dialog. Runs AFTER the install has
  /// been confirmed server-side so the plugin exists in /api/providers
  /// and has a consent row — the config PUT needs both. Cancelling
  /// is fine; the user can re-open the form from Plugin → Configure.
  Future<void> _promptConfigureAfterInstall(
      MarketplaceEntry entry, String installedName) async {
    final displayName = _resolveDisplayName(entry);
    await showDialog<void>(
      context: context,
      builder: (dialogCtx) {
        return AlertDialog(
          title: Text('Configure $displayName'),
          content: SingleChildScrollView(
            child: SizedBox(
              width: 380,
              child: PluginConfigForm(
                schema: entry.configSchema,
                initialValues: const {},
                submitLabel: 'Save',
                onCancel: () => Navigator.pop(dialogCtx),
                onSave: (drafts) async {
                  final body = PluginConfig(
                    schema: entry.configSchema,
                    values: const {},
                  ).toPutBody(drafts);
                  try {
                    await _api.putPluginConfig(installedName, body);
                    if (!dialogCtx.mounted) return;
                    Navigator.pop(dialogCtx);
                    _notify('Saved config for $installedName');
                  } catch (e) {
                    _notify('Save failed: $e', isError: true);
                  }
                },
              ),
            ),
          ),
        );
      },
    );
  }

  Future<bool?> _showConsentDialog(
      MarketplaceEntry entry, PendingInstall pending) {
    final displayName = _resolveDisplayName(entry);
    return showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Install $displayName?'),
        content: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('Version ${pending.version} · by ${entry.publisher}',
                  style: const TextStyle(
                      color: AppColors.textMuted, fontSize: 12)),
              Builder(builder: (ctx) {
                final desc =
                    ctx.pickL10n(entry.description, entry.descriptionZh);
                if (desc.isEmpty) return const SizedBox.shrink();
                return Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text(desc,
                      style: const TextStyle(fontSize: 13, height: 1.4)),
                );
              }),
              const SizedBox(height: 14),
              const Text('Grants:',
                  style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
              const SizedBox(height: 4),
              ..._permissionLines(pending.permissions),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
            child: const Text('Install'),
          ),
        ],
      ),
    );
  }

  /// Flatten the PermissionsV1 map into one Text widget per non-empty
  /// capability so the consent dialog shows the user exactly what
  /// they're accepting. Mirrors the subtitle logic in the runtime
  /// consent page for consistency.
  List<Widget> _permissionLines(Map<String, dynamic> perms) {
    if (perms.isEmpty) {
      return const [
        Text('No runtime permissions — trusted scope only.',
            style:
                TextStyle(fontSize: 12, color: AppColors.textMuted)),
      ];
    }
    final out = <Widget>[];
    perms.forEach((cap, value) {
      final summary = _summariseCap(value);
      if (summary.isEmpty) return;
      out.add(Padding(
        padding: const EdgeInsets.symmetric(vertical: 2),
        child: Text('• $cap: $summary',
            style: const TextStyle(fontSize: 12, height: 1.35)),
      ));
    });
    if (out.isEmpty) {
      out.add(const Text('(no non-empty capabilities declared)',
          style: TextStyle(fontSize: 12, color: AppColors.textMuted)));
    }
    return out;
  }

  String _summariseCap(dynamic v) {
    if (v == null) return '';
    if (v is bool) return v ? 'granted' : '';
    if (v is String) return v.isEmpty ? '' : v;
    if (v is List) {
      if (v.isEmpty) return '';
      final parts = v.map((e) => e.toString()).toList();
      return parts.length <= 3
          ? parts.join(', ')
          : '${parts.take(3).join(', ')} (+${parts.length - 3} more)';
    }
    if (v is Map) {
      if (v.isEmpty) return '';
      return v.keys.map((e) => e.toString()).join(', ');
    }
    return v.toString();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Hub'),
        actions: [
          if (_state == _LoadState.loaded)
            IconButton(
              icon: const Icon(Icons.refresh),
              onPressed: _busy ? null : _load,
            ),
        ],
      ),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    switch (_state) {
      case _LoadState.loading:
        return const Center(
          child: CircularProgressIndicator(color: AppColors.accent),
        );
      case _LoadState.error:
        return Padding(
          padding: const EdgeInsets.all(16),
          child: Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: AppColors.errorSoft,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('Failed to load marketplace',
                    style: TextStyle(
                        color: AppColors.error,
                        fontWeight: FontWeight.w500)),
                const SizedBox(height: 4),
                Text(_error ?? '',
                    style: const TextStyle(
                        color: AppColors.error, fontSize: 12)),
                const SizedBox(height: 10),
                FilledButton(
                  onPressed: _load,
                  style: FilledButton.styleFrom(
                      backgroundColor: AppColors.accent),
                  child: const Text('Retry'),
                ),
              ],
            ),
          ),
        );
      case _LoadState.loaded:
        if (_entries.isEmpty) {
          return const Center(
            child: Padding(
              padding: EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.storefront_outlined,
                      size: 48, color: AppColors.textMuted),
                  SizedBox(height: 12),
                  Text('Marketplace is empty',
                      style: TextStyle(
                          fontSize: 15, fontWeight: FontWeight.w500)),
                  SizedBox(height: 4),
                  Text(
                    'No plugins have been published to this server yet.',
                    textAlign: TextAlign.center,
                    style: TextStyle(
                        fontSize: 12, color: AppColors.textMuted),
                  ),
                ],
              ),
            ),
          );
        }
        return RefreshIndicator(
          onRefresh: _load,
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              for (final e in _entries) _entryCard(e),
            ],
          ),
        );
    }
  }

  Widget _entryCard(MarketplaceEntry entry) {
    final installed = _installedNames.contains(entry.name);
    final displayName = _resolveDisplayName(entry);
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 12, 12, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                _iconBadge(entry.icon),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(displayName,
                          style: const TextStyle(
                              fontWeight: FontWeight.w600, fontSize: 14)),
                      Text(
                        'v${entry.version} · ${entry.publisher}',
                        style: const TextStyle(
                            color: AppColors.textMuted, fontSize: 11),
                      ),
                    ],
                  ),
                ),
                _trustBadge(entry.trust),
                _formBadge(entry),
              ],
            ),
            Builder(builder: (ctx) {
              final desc =
                  ctx.pickL10n(entry.description, entry.descriptionZh);
              if (desc.isEmpty) return const SizedBox.shrink();
              return Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(desc,
                    style: const TextStyle(
                        fontSize: 12, color: AppColors.text, height: 1.4)),
              );
            }),
            const SizedBox(height: 10),
            Align(
              alignment: Alignment.centerRight,
              child: FilledButton.icon(
                onPressed: installed || _busy ? null : () => _install(entry),
                icon: Icon(installed ? Icons.check : Icons.download,
                    size: 16),
                label: Text(installed ? 'Installed' : 'Install'),
                style: FilledButton.styleFrom(
                    backgroundColor: installed
                        ? AppColors.textMuted
                        : AppColors.accent),
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// Trust badge for the Hub card. Renders:
  ///   official  → green "OFFICIAL" chip with shield icon
  ///   verified  → blue "VERIFIED" chip with checkmark
  ///   community → grey "COMMUNITY" chip
  /// Keeps sideload labelling to the post-install path since
  /// marketplace entries aren't sideloaded by definition.
  Widget _trustBadge(String trust) {
    Color color;
    IconData? icon;
    String label;
    switch (trust) {
      case 'official':
        color = const Color(0xFF16A34A); // green-600
        icon = Icons.shield;
        label = 'OFFICIAL';
        break;
      case 'verified':
        color = const Color(0xFF2563EB); // blue-600
        icon = Icons.verified;
        label = 'VERIFIED';
        break;
      default:
        color = AppColors.textMuted;
        icon = null;
        label = 'COMMUNITY';
    }
    return Container(
      margin: const EdgeInsets.only(right: 4),
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 10, color: color),
            const SizedBox(width: 3),
          ],
          Text(label,
              style: TextStyle(
                  fontSize: 10,
                  color: color,
                  fontWeight: FontWeight.w600,
                  letterSpacing: 0.4)),
        ],
      ),
    );
  }

  /// Form badge makes the plugin's code-ownership visible at a glance:
  ///
  ///   built-in   — publisher=opendray-builtin + form=declarative.
  ///                The "code" lives in the OpenDray gateway/Flutter
  ///                binaries; the marketplace entry is a registration
  ///                manifest. Cannot be replaced by a third party
  ///                without forking OpenDray. Upgrades ship with
  ///                OpenDray releases.
  ///   webview    — plugin ships HTML/JS bundle; runs in a sandboxed
  ///                WebView via the bridge. Third-party replaceable.
  ///   host       — plugin ships a sidecar process (Node, Python …);
  ///                speaks JSON-RPC over stdio. Third-party replaceable.
  ///
  /// Declarative plugins by a non-builtin publisher fall back to the
  /// raw form string — that combination is unusual (user forked a
  /// core panel?) and needs to stand out, not hide behind the
  /// built-in badge.
  Widget _formBadge(MarketplaceEntry entry) {
    if (entry.form.isEmpty) return const SizedBox.shrink();
    final isBuiltin = entry.publisher == 'opendray-builtin' &&
        entry.form == 'declarative';
    final (label, color, tooltip) = isBuiltin
        ? (
            'BUILT-IN',
            const Color(0xFF7C3AED), // violet-600
            'Ships with OpenDray. The marketplace entry is a registration '
                'manifest — the feature\'s code lives in the OpenDray gateway + '
                'Flutter app. Upgrades via OpenDray releases, not Hub.',
          )
        : (
            entry.form.toUpperCase(),
            AppColors.textMuted,
            entry.form == 'webview'
                ? 'Ships an HTML/JS bundle; runs in a sandboxed WebView.'
                : entry.form == 'host'
                    ? 'Ships a sidecar process; speaks JSON-RPC over stdio.'
                    : entry.form,
          );
    return Tooltip(
      message: tooltip,
      child: Container(
        margin: const EdgeInsets.only(left: 4, right: 8),
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.14),
          borderRadius: BorderRadius.circular(6),
        ),
        child: Text(label,
            style: TextStyle(
                fontSize: 10,
                color: color,
                fontWeight: FontWeight.w600,
                letterSpacing: 0.4)),
      ),
    );
  }

  Widget _iconBadge(String icon) {
    final isEmoji = icon.isNotEmpty && icon.length <= 4 && !icon.contains('/');
    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: AppColors.accent.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(8),
      ),
      alignment: Alignment.center,
      child: isEmoji
          ? Text(icon, style: const TextStyle(fontSize: 18))
          : const Icon(Icons.extension, color: AppColors.accent, size: 20),
    );
  }
}
