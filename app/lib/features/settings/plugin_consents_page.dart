import 'package:flutter/material.dart';
import '../../core/api/api_client.dart';
import '../../shared/theme/app_theme.dart';

/// Runtime consent toggles for a single plugin (M2 T21).
///
/// Lists the 11 PermissionsV1 capabilities and lets the user flip any
/// currently-granted cap OFF — driven by DELETE
/// /api/plugins/{name}/consents/{cap}. Re-granting requires
/// reinstalling the plugin in M2 (the install-time consent flow owns
/// the grant path), so ungranted switches render disabled with a helper
/// note rather than as actionable toggles.
///
/// The "Revoke all" action in the AppBar nukes the consent row
/// entirely via DELETE /api/plugins/{name}/consents and lands the page
/// in its empty state (the same one that renders when there was never
/// a consent row on file).
class PluginConsentsPage extends StatefulWidget {
  final String pluginName;
  final ApiClient api;

  /// Optional message sink — the caller (usually the settings host)
  /// pipes this through `ScaffoldMessenger.of(context).showSnackBar`.
  /// When null, messages are still shown inline via SnackBar on the
  /// page's own Scaffold so the page is self-contained.
  final void Function(String msg, {bool isError})? onMessage;

  const PluginConsentsPage({
    required this.pluginName,
    required this.api,
    this.onMessage,
    super.key,
  });

  @override
  State<PluginConsentsPage> createState() => _PluginConsentsPageState();
}

/// Per-cap render metadata. Kept as a static table because the set of
/// capabilities is closed at manifest-schema level — new caps require a
/// plugin/manifest.go change + a new entry here.
class _CapSpec {
  final String key;
  final String label;
  final IconData icon;
  const _CapSpec(this.key, this.label, this.icon);
}

const List<_CapSpec> _caps = [
  _CapSpec('fs', 'File access', Icons.folder_open),
  _CapSpec('exec', 'Run commands', Icons.terminal),
  _CapSpec('http', 'Network (HTTP)', Icons.public),
  _CapSpec('session', 'Sessions', Icons.sensors),
  _CapSpec('storage', 'Plugin storage', Icons.storage),
  _CapSpec('secret', 'Secrets', Icons.vpn_key),
  _CapSpec('clipboard', 'Clipboard', Icons.content_copy),
  _CapSpec('telegram', 'Telegram', Icons.send),
  _CapSpec('git', 'Git', Icons.commit),
  _CapSpec('llm', 'LLM providers', Icons.auto_awesome),
  _CapSpec('events', 'Events', Icons.bolt),
];

enum _LoadState { loading, loaded, empty, error }

class _PluginConsentsPageState extends State<PluginConsentsPage> {
  _LoadState _state = _LoadState.loading;
  PluginConsents? _consents;
  String? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    if (!mounted) return;
    setState(() {
      _state = _LoadState.loading;
      _error = null;
    });
    try {
      final c = await widget.api.getPluginConsents(widget.pluginName);
      if (!mounted) return;
      setState(() {
        _consents = c;
        _state = _LoadState.loaded;
      });
    } on PluginConsentNotFoundException {
      if (!mounted) return;
      setState(() {
        _consents = null;
        _state = _LoadState.empty;
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
    final cb = widget.onMessage;
    if (cb != null) {
      cb(msg, isError: isError);
      return;
    }
    // Self-contained fallback so the page stays useful when embedded
    // without a message sink (e.g. during widget tests that don't
    // wire onMessage or in a one-off preview route).
    final messenger = ScaffoldMessenger.maybeOf(context);
    if (messenger != null) {
      messenger.showSnackBar(SnackBar(
        content: Text(msg),
        backgroundColor: isError ? AppColors.error : null,
      ));
    }
  }

  Future<void> _revokeCap(String cap, String label) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      await widget.api.revokePluginCapability(widget.pluginName, cap);
      _notify('Revoked $label');
      await _refresh();
    } on PluginConsentNotFoundException {
      // Race: consent row disappeared between load and revoke. Treat as
      // success and land in the empty state — the user's intent was
      // "take this away", and it's already gone.
      _notify('Revoked $label');
      if (!mounted) return;
      setState(() {
        _consents = null;
        _state = _LoadState.empty;
      });
    } catch (e) {
      _notify('Failed to revoke $label: $e', isError: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _confirmRevokeAll() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Revoke all permissions?'),
        content: Text(
          'This removes every granted capability for "${widget.pluginName}". '
          'The plugin will stop working until it is reinstalled.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(backgroundColor: AppColors.error),
            child: const Text('Revoke all'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    if (!mounted) return;
    setState(() => _busy = true);
    try {
      await widget.api.revokeAllPluginConsents(widget.pluginName);
      _notify('Revoked all permissions for ${widget.pluginName}');
      await _refresh();
    } catch (e) {
      _notify('Failed to revoke all: $e', isError: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.pluginName),
        actions: [
          if (_state == _LoadState.loaded)
            TextButton.icon(
              onPressed: _busy ? null : _confirmRevokeAll,
              icon: const Icon(Icons.block, size: 16, color: AppColors.error),
              label: const Text(
                'Revoke all',
                style: TextStyle(color: AppColors.error),
              ),
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
      case _LoadState.empty:
        return const Center(
          child: Padding(
            padding: EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.shield_outlined,
                    size: 48, color: AppColors.textMuted),
                SizedBox(height: 12),
                Text(
                  'No consent on record',
                  style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w500,
                      color: AppColors.text),
                ),
                SizedBox(height: 6),
                Text(
                  'This plugin has no granted capabilities. '
                  'Reinstall it to re-consent.',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                      fontSize: 12, color: AppColors.textMuted),
                ),
              ],
            ),
          ),
        );
      case _LoadState.error:
        return Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: AppColors.errorSoft,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      'Failed to load consents',
                      style: TextStyle(
                          color: AppColors.error,
                          fontWeight: FontWeight.w500),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      _error ?? '',
                      style: const TextStyle(
                          color: AppColors.error, fontSize: 12),
                    ),
                    const SizedBox(height: 10),
                    FilledButton(
                      onPressed: _refresh,
                      style: FilledButton.styleFrom(
                          backgroundColor: AppColors.accent),
                      child: const Text('Retry'),
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      case _LoadState.loaded:
        return _buildLoaded(_consents!);
    }
  }

  Widget _buildLoaded(PluginConsents c) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _header(c),
        const SizedBox(height: 12),
        Card(
          child: Column(
            children: [
              for (int i = 0; i < _caps.length; i++) ...[
                _capRow(_caps[i], c),
                if (i < _caps.length - 1)
                  const Divider(height: 1, indent: 56),
              ],
            ],
          ),
        ),
      ],
    );
  }

  Widget _header(PluginConsents c) {
    String fmt(DateTime? t) =>
        t == null ? '—' : t.toLocal().toString().split('.').first;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: const [
              Icon(Icons.shield, color: AppColors.accent, size: 18),
              SizedBox(width: 8),
              Text('Runtime permissions',
                  style:
                      TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
            ]),
            const SizedBox(height: 8),
            Text(
              'Flip a granted capability off to revoke it at runtime. '
              'Re-grant requires reinstalling the plugin.',
              style: TextStyle(
                  fontSize: 12, color: AppColors.textMuted),
            ),
            const SizedBox(height: 10),
            _kv('Granted', fmt(c.grantedAt)),
            _kv('Updated', fmt(c.updatedAt)),
          ],
        ),
      ),
    );
  }

  Widget _kv(String k, String v) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 2),
        child: Row(
          children: [
            SizedBox(
                width: 80,
                child: Text(k,
                    style: const TextStyle(
                        color: AppColors.textMuted, fontSize: 11))),
            Expanded(
                child: Text(v, style: const TextStyle(fontSize: 11))),
          ],
        ),
      );

  Widget _capRow(_CapSpec spec, PluginConsents c) {
    final granted = c.isCapGranted(spec.key);
    final subtitle = _subtitleFor(spec.key, c);
    return ListTile(
      leading: Icon(
        spec.icon,
        color: granted ? AppColors.accent : AppColors.textMuted,
        size: 22,
      ),
      title: Text(
        spec.label,
        style: TextStyle(
          fontSize: 14,
          color: granted ? AppColors.text : AppColors.textMuted,
        ),
      ),
      subtitle: Text(
        subtitle,
        style: const TextStyle(fontSize: 11, color: AppColors.textMuted),
      ),
      trailing: Switch(
        value: granted,
        activeTrackColor: AppColors.accent,
        // Flipping ON is intentionally disabled in M2 — the grant path
        // is tied to install-time consent. Only granted caps can be
        // flipped (to off). Everything else renders greyed out with a
        // helper subtitle explaining the re-install path.
        onChanged: (granted && !_busy)
            ? (_) => _revokeCap(spec.key, spec.label)
            : null,
      ),
    );
  }

  String _subtitleFor(String cap, PluginConsents c) {
    if (!c.isCapGranted(cap)) {
      return 'Not granted · reinstall to re-grant';
    }
    final v = c.perms[cap];
    // Render a short shape hint so the user sees what was actually
    // granted, not just a binary on/off. Matches the sketch in the
    // T21 spec ("exec: git *, npm *" / "storage: granted" / etc.).
    if (v is bool) return 'Granted';
    if (v is String) return v;
    if (v is List) {
      final parts = v.map((e) => e.toString()).toList();
      if (parts.length <= 3) return parts.join(', ');
      return '${parts.take(3).join(', ')} (+${parts.length - 3} more)';
    }
    if (v is Map) {
      final keys = v.keys.map((e) => e.toString()).toList();
      return keys.isEmpty ? 'Granted' : keys.join(', ');
    }
    return 'Granted';
  }
}
