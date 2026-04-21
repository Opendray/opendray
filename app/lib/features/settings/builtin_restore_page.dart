import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// Settings → Built-in Plugins page.
///
/// Shows every manifest bundled in the OpenDray binary alongside its
/// current state (installed / disabled / uninstalled). The primary job
/// is the Restore action: when a user Uninstalls a built-in from the
/// Plugins page, the row is tombstoned and LoadAll stops re-seeding it
/// on boot — without this page there's no way back. Hub is hidden
/// during v1 and even after it re-opens it won't list built-ins
/// (they're shipped in-binary, not in the marketplace catalog).
///
/// Disabled built-ins show a status chip but no action — use the
/// Enable toggle on /plugins for those. This page is single-purpose:
/// undo an Uninstall.
class BuiltinRestorePage extends StatefulWidget {
  const BuiltinRestorePage({super.key});

  @override
  State<BuiltinRestorePage> createState() => _BuiltinRestorePageState();
}

class _BuiltinRestorePageState extends State<BuiltinRestorePage> {
  bool _loading = true;
  String? _error;
  List<BuiltinInfo> _items = const [];
  final Set<String> _restoring = {};

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    if (!mounted) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final items = await _api.listBuiltins();
      if (!mounted) return;
      setState(() {
        _items = items;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  void _notify(String msg, {bool isError = false}) {
    ScaffoldMessenger.maybeOf(context)?.showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: isError ? AppColors.error : null,
    ));
  }

  Future<void> _restore(BuiltinInfo item) async {
    final name = item.provider.name;
    if (_restoring.contains(name)) return;
    // Snapshot the display name + translation strings BEFORE the async
    // call so we can use them afterwards without reaching back through
    // `context` (use_build_context_synchronously).
    final displayName = context.pickL10nOnce(
        item.provider.displayName, item.provider.displayNameZh);
    final restoredLabel = context.trOnce('Restored');
    final failedLabel = context.trOnce('Restore failed');
    setState(() => _restoring.add(name));
    try {
      await _api.restoreBuiltin(name);
      if (!mounted) return;
      _notify('$restoredLabel · $displayName');
      ProvidersBus.instance.notify();
      await _load();
    } catch (e) {
      if (!mounted) return;
      _notify('$failedLabel: $e', isError: true);
    } finally {
      if (mounted) setState(() => _restoring.remove(name));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(context.tr('Built-in plugins')),
        actions: [
          if (!_loading)
            IconButton(
              icon: const Icon(Icons.refresh),
              onPressed: _load,
            ),
        ],
      ),
      body: _loading
          ? const Center(
              child: CircularProgressIndicator(color: AppColors.accent))
          : RefreshIndicator(
              onRefresh: _load,
              child: _buildBody(),
            ),
    );
  }

  Widget _buildBody() {
    if (_error != null) {
      return ListView(
        padding: const EdgeInsets.all(16),
        children: [_errorBanner()],
      );
    }
    if (_items.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.inventory_2_outlined,
                  size: 40, color: AppColors.textMuted),
              const SizedBox(height: 10),
              Text(context.tr('No built-in plugins'),
                  style: const TextStyle(
                      fontWeight: FontWeight.w500,
                      fontSize: 14,
                      color: AppColors.text)),
              const SizedBox(height: 4),
              Text(
                context.tr(
                    'The server reported zero bundled manifests — this usually means the binary was built without the plugins/builtin tree.'),
                textAlign: TextAlign.center,
                style: const TextStyle(
                    fontSize: 12, color: AppColors.textMuted),
              ),
            ],
          ),
        ),
      );
    }
    final uninstalledCount = _items.where((e) => e.isUninstalled).length;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _header(uninstalledCount),
        const SizedBox(height: 12),
        for (final item in _items) _card(item),
      ],
    );
  }

  Widget _errorBanner() {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.errorSoft,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(context.tr('Failed to load built-in plugins'),
              style: const TextStyle(
                  color: AppColors.error, fontWeight: FontWeight.w500)),
          const SizedBox(height: 4),
          Text(_error ?? '',
              style: const TextStyle(color: AppColors.error, fontSize: 12)),
          const SizedBox(height: 10),
          FilledButton(
            onPressed: _load,
            style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
            child: Text(context.tr('Retry')),
          ),
        ],
      ),
    );
  }

  Widget _header(int uninstalledCount) {
    final subtitle = uninstalledCount == 0
        ? context.tr('All built-in plugins are currently installed.')
        : '$uninstalledCount ${context.tr('built-in plugin(s) uninstalled — tap Restore to bring them back.')}';
    return Padding(
      padding: const EdgeInsets.only(left: 2, right: 2),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            context.tr(
                'These plugins ship with OpenDray. Uninstalling one removes it from the Plugins page; restore it here.'),
            style: const TextStyle(
                fontSize: 12, color: AppColors.textMuted, height: 1.4),
          ),
          const SizedBox(height: 6),
          Text(subtitle,
              style: const TextStyle(
                  fontSize: 12,
                  color: AppColors.text,
                  fontWeight: FontWeight.w500)),
        ],
      ),
    );
  }

  Widget _card(BuiltinInfo item) {
    final prov = item.provider;
    final displayName =
        context.pickL10n(prov.displayName, prov.displayNameZh);
    final description =
        context.pickL10n(prov.description, prov.descriptionZh);
    // Greyed body for uninstalled plugins so the active installed ones
    // dominate visually; Restore button stays at full opacity to
    // signal it's the actionable element.
    final bodyOpacity = item.isUninstalled ? 0.55 : 1.0;

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 14, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Opacity(
                    opacity: bodyOpacity, child: _iconBadge(prov.icon)),
                const SizedBox(width: 14),
                Expanded(
                  child: Opacity(
                    opacity: bodyOpacity,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          displayName,
                          style: const TextStyle(
                              fontWeight: FontWeight.w600, fontSize: 15),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          'v${prov.version} · ${prov.publisher.isEmpty ? "opendray-builtin" : prov.publisher}',
                          style: const TextStyle(
                              color: AppColors.textMuted, fontSize: 11),
                        ),
                      ],
                    ),
                  ),
                ),
                _stateChip(item.state),
              ],
            ),
            if (description.isNotEmpty) ...[
              const SizedBox(height: 10),
              Opacity(
                opacity: bodyOpacity,
                child: Text(
                  description,
                  style: const TextStyle(
                      fontSize: 12, color: AppColors.text, height: 1.4),
                ),
              ),
            ],
            const SizedBox(height: 12),
            _action(item),
          ],
        ),
      ),
    );
  }

  Widget _stateChip(String state) {
    switch (state) {
      case 'installed':
        return _chip(context.trOnce('Installed'), AppColors.success);
      case 'disabled':
        return _chip(context.trOnce('Disabled'), AppColors.warning);
      default:
        return _chip(context.trOnce('Uninstalled'), AppColors.textMuted);
    }
  }

  Widget _chip(String label, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        label,
        style: TextStyle(
            color: color,
            fontSize: 10,
            fontWeight: FontWeight.w600,
            letterSpacing: 0.4),
      ),
    );
  }

  Widget _action(BuiltinInfo item) {
    if (item.isUninstalled) {
      final busy = _restoring.contains(item.provider.name);
      return Align(
        alignment: Alignment.centerRight,
        child: FilledButton.icon(
          onPressed: busy ? null : () => _restore(item),
          icon: busy
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child:
                      CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                )
              : const Icon(Icons.restore, size: 16),
          label: Text(
              busy ? context.tr('Restoring…') : context.tr('Restore')),
          style:
              FilledButton.styleFrom(backgroundColor: AppColors.accent),
        ),
      );
    }
    // Installed / disabled plugins — no action on this page.
    // Disabled ones are toggled via the Switch on /plugins; installed
    // ones are already fine. A hint line keeps the card visually
    // balanced without a misleading button.
    final hint = item.isDisabled
        ? context.tr('Toggle from Plugins page to enable.')
        : context.tr('Already active. Manage from Plugins page.');
    return Row(
      children: [
        Icon(Icons.info_outline, size: 14, color: AppColors.textMuted),
        const SizedBox(width: 6),
        Expanded(
          child: Text(hint,
              style: const TextStyle(
                  fontSize: 11, color: AppColors.textMuted)),
        ),
      ],
    );
  }

  Widget _iconBadge(String icon) {
    final isEmoji = icon.isNotEmpty && icon.length <= 4 && !icon.contains('/');
    return Container(
      width: 44,
      height: 44,
      decoration: BoxDecoration(
        color: AppColors.accent.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(10),
      ),
      alignment: Alignment.center,
      child: isEmoji
          ? Text(icon, style: const TextStyle(fontSize: 22))
          : const Icon(Icons.extension, color: AppColors.accent, size: 24),
    );
  }
}
