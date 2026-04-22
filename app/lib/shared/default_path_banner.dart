import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../core/api/api_client.dart';
import '../core/services/l10n.dart';
import '../features/plugins/plugin_configure_page.dart';
import 'theme/app_theme.dart';

/// Info banner shown at the top of the panel pages (File Browser, Log
/// Viewer, Task Runner) when the plugin is running with its manifest
/// default `allowedRoots` instead of a user-configured value.
///
/// First-install UX: the panel opens pointed at `$HOME` so there's
/// something to see, and this banner tells the user where they are and
/// offers one tap to narrow the scope. Dismissed state is remembered
/// per-plugin in SharedPreferences so returning users aren't nagged.
class DefaultPathBanner extends StatefulWidget {
  /// Plugin this banner is querying (e.g. `file-browser`, `log-viewer`,
  /// `task-runner`). Used both to pull GET /api/plugins/{name}/config
  /// and to key the dismiss preference.
  final String pluginName;

  /// Human-readable name for the Configure page header when the user
  /// taps "Change".
  final String displayName;

  /// Config key whose empty-vs-present state drives visibility.
  /// Defaults to `allowedRoots` — the one every panel plugin uses.
  final String configKey;

  const DefaultPathBanner({
    super.key,
    required this.pluginName,
    required this.displayName,
    this.configKey = 'allowedRoots',
  });

  @override
  State<DefaultPathBanner> createState() => _DefaultPathBannerState();
}

class _DefaultPathBannerState extends State<DefaultPathBanner> {
  bool _visible = false;
  String _defaultLabel = '';

  String get _dismissKey => 'banner_default_path.${widget.pluginName}';

  @override
  void initState() {
    super.initState();
    _probe();
  }

  Future<void> _probe() async {
    final prefs = await SharedPreferences.getInstance();
    if (prefs.getBool(_dismissKey) == true) return;
    if (!mounted) return;

    final api = context.read<ApiClient>();
    try {
      final cfg = await api.getPluginConfig(widget.pluginName);
      if (!mounted) return;

      // Empty / whitespace user value means the backend is resolving
      // via the manifest Default (see resolveRoots in gateway/). That's
      // exactly when the banner is useful; a user-saved value would be
      // misleading.
      final raw = (cfg.values[widget.configKey] ?? '').trim();
      if (raw.isNotEmpty) return;

      // Prefer the manifest default string verbatim — it's user-facing
      // ("$HOME", "/var/log") and self-describing without us having to
      // re-expand $HOME on the client. Fall back to a neutral string
      // when the field has no default declared.
      String label = '';
      for (final f in cfg.schema) {
        if (f.key == widget.configKey) {
          final dv = f.defaultValue;
          if (dv is String && dv.isNotEmpty) label = dv;
          break;
        }
      }
      if (label.isEmpty) return;

      setState(() {
        _visible = true;
        _defaultLabel = label;
      });
    } catch (_) {
      // Silent: plugin missing / server offline is surfaced by the page
      // itself. The banner is strictly additive.
    }
  }

  Future<void> _dismiss() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(_dismissKey, true);
    if (!mounted) return;
    setState(() => _visible = false);
  }

  void _openConfigure() {
    Navigator.of(context).push(MaterialPageRoute<void>(
      builder: (_) => PluginConfigurePage(
        pluginName: widget.pluginName,
        displayName: widget.displayName,
      ),
    ));
  }

  @override
  Widget build(BuildContext context) {
    if (!_visible) return const SizedBox.shrink();
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 0),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: AppColors.accentSoft,
        border: Border.all(color: AppColors.accent.withValues(alpha: 0.4)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          const Icon(Icons.info_outline, size: 16, color: AppColors.accent),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  context.tr('Using default directory'),
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: AppColors.text,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  _defaultLabel,
                  style: const TextStyle(
                    fontSize: 11,
                    fontFamily: 'monospace',
                    color: AppColors.textMuted,
                  ),
                ),
              ],
            ),
          ),
          TextButton(
            onPressed: _openConfigure,
            style: TextButton.styleFrom(
              foregroundColor: AppColors.accent,
              padding: const EdgeInsets.symmetric(horizontal: 10),
              minimumSize: const Size(0, 32),
            ),
            child: Text(context.tr('Change')),
          ),
          IconButton(
            tooltip: context.tr('Dismiss'),
            onPressed: _dismiss,
            icon: const Icon(Icons.close, size: 16, color: AppColors.textMuted),
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
          ),
        ],
      ),
    );
  }
}
