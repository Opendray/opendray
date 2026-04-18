import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// Claude Multi-Account management page.
///
/// Lists claude_accounts rows. Each maps to an on-disk `~/.claude-accounts/<name>/`
/// sandbox managed by the `claude-acc` host tool. Tokens never flow through
/// Flutter — they're written server-side with chmod 600 and read only at
/// session-spawn time.
class ClaudeAccountsPage extends StatefulWidget {
  const ClaudeAccountsPage({super.key});
  @override
  State<ClaudeAccountsPage> createState() => _ClaudeAccountsPageState();
}

class _ClaudeAccountsPageState extends State<ClaudeAccountsPage> {
  List<Map<String, dynamic>> _accounts = [];
  bool _loading = true;
  String? _error;
  bool _importing = false;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    try {
      final accounts = await _api.claudeAccounts();
      if (!mounted) return;
      setState(() {
        _accounts = accounts;
        _error = null;
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

  Future<void> _importLocal() async {
    setState(() => _importing = true);
    try {
      final summary = await ApiClient.describeErrors(_api.claudeAccountImportLocal);
      final imported = (summary['imported'] as List?)?.length ?? 0;
      final skipped = (summary['skipped'] as List?)?.length ?? 0;
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(imported == 0 && skipped == 0
            ? context.trOnce('No ~/.claude-accounts/tokens — run `claude-acc init` on the host first')
            : '${context.trOnce('Imported')} $imported · ${context.trOnce('Skipped')} $skipped'),
        backgroundColor: imported > 0 ? AppColors.success : AppColors.surface,
      ));
      _refresh();
      ProvidersBus.instance.notify();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    } finally {
      if (mounted) setState(() => _importing = false);
    }
  }

  Future<void> _openEditor([Map<String, dynamic>? account]) async {
    final changed = await showDialog<bool>(
      context: context,
      useRootNavigator: true,
      builder: (_) => _AccountEditorDialog(api: _api, initial: account),
    );
    if (changed == true) {
      _refresh();
      ProvidersBus.instance.notify();
    }
  }

  Future<void> _setToken(Map<String, dynamic> a) async {
    final token = await showDialog<String>(
      context: context,
      useRootNavigator: true,
      builder: (_) => _TokenDialog(
        accountName: a['name'] as String? ?? '',
        tokenFilled: a['tokenFilled'] as bool? ?? false,
      ),
    );
    if (token == null || token.isEmpty) return;
    try {
      await ApiClient.describeErrors(
          () => _api.claudeAccountSetToken(a['id'] as String, token));
      _refresh();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    }
  }

  Future<void> _toggle(Map<String, dynamic> a, bool enabled) async {
    try {
      await _api.claudeAccountToggle(a['id'] as String, enabled);
      setState(() => a['enabled'] = enabled);
      ProvidersBus.instance.notify();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
      }
    }
  }

  Future<void> _delete(Map<String, dynamic> a) async {
    final ok = await showDialog<bool>(
      context: context,
      useRootNavigator: true,
      builder: (dialogCtx) => AlertDialog(
        title: Text(dialogCtx.tr('Delete Claude account?')),
        content: Text(
          dialogCtx.tr('This removes "@name" from OpenDray. The on-disk token file and config directory are left intact.')
              .replaceAll('@name', a['name'] as String? ?? ''),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogCtx, rootNavigator: true).pop(false),
            child: Text(dialogCtx.tr('Cancel')),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.of(dialogCtx, rootNavigator: true).pop(true),
            child: Text(dialogCtx.tr('Delete')),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.claudeAccountDelete(a['id'] as String);
      _refresh();
      ProvidersBus.instance.notify();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_error != null) {
      return _emptyState(
        icon: Icons.error_outline,
        title: context.tr('Failed to load Claude accounts'),
        body: _error!,
      );
    }
    return Scaffold(
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _openEditor(),
        backgroundColor: AppColors.accent,
        icon: const Icon(Icons.add),
        label: Text(context.tr('Add Claude account')),
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 96),
          children: [
            _intro(),
            const SizedBox(height: 8),
            _importButton(),
            const SizedBox(height: 12),
            if (_accounts.isEmpty)
              _emptyState(
                icon: Icons.person_outline,
                title: context.tr('No Claude accounts yet'),
                body: context.tr('Register a Claude subscription token so this server can launch sessions as that account. Tokens stay chmod 600 on the host — this UI only tracks metadata.'),
              )
            else
              ..._accounts.map((a) => Padding(
                    padding: const EdgeInsets.only(bottom: 10),
                    child: _accountCard(a),
                  )),
          ],
        ),
      ),
    );
  }

  Widget _intro() {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.accentSoft,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.accent.withValues(alpha: 0.35)),
      ),
      child: Row(children: [
        const Icon(Icons.info_outline, size: 18, color: AppColors.accent),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            context.tr('Each account maps to one OAuth token managed by the claude-acc host tool. Sessions pick an account at creation time; tokens never enter Postgres or this UI.'),
            style: const TextStyle(fontSize: 11, color: AppColors.text),
          ),
        ),
      ]),
    );
  }

  Widget _importButton() {
    return OutlinedButton.icon(
      onPressed: _importing ? null : _importLocal,
      icon: _importing
          ? const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2, color: AppColors.accent))
          : const Icon(Icons.download, size: 16),
      label: Text(context.tr('Import from ~/.claude-accounts')),
      style: OutlinedButton.styleFrom(
        foregroundColor: AppColors.accent,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      ),
    );
  }

  Widget _emptyState({required IconData icon, required String title, required String body}) {
    return Padding(
      padding: const EdgeInsets.all(28),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, size: 48, color: AppColors.textMuted),
          const SizedBox(height: 16),
          Text(title, style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 15)),
          const SizedBox(height: 8),
          Text(body, style: const TextStyle(color: AppColors.textMuted, fontSize: 12), textAlign: TextAlign.center),
        ],
      ),
    );
  }

  Widget _accountCard(Map<String, dynamic> a) {
    final enabled = a['enabled'] as bool? ?? true;
    final tokenFilled = a['tokenFilled'] as bool? ?? false;
    final display = (a['displayName'] as String?)?.isNotEmpty == true
        ? a['displayName'] as String
        : a['name'] as String? ?? '';

    return Card(
      color: AppColors.surface,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(
          color: enabled ? AppColors.accent.withValues(alpha: 0.25) : AppColors.border,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 12, 8, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      Flexible(
                        child: Text(
                          display,
                          style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 8),
                      _statusChip(tokenFilled),
                    ]),
                    const SizedBox(height: 2),
                    Text('claude-${a['name']}',
                        style: const TextStyle(
                            fontFamily: 'monospace', fontSize: 11, color: AppColors.textMuted)),
                    if ((a['description'] as String?)?.isNotEmpty == true) ...[
                      const SizedBox(height: 4),
                      Text(a['description'] as String,
                          style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
                    ],
                  ],
                ),
              ),
              Switch(
                value: enabled,
                onChanged: (v) => _toggle(a, v),
                activeThumbColor: AppColors.accent,
              ),
            ]),
            const SizedBox(height: 8),
            _kv(context.tr('Config dir'), a['configDir'] as String? ?? ''),
            _kv(context.tr('Token path'), a['tokenPath'] as String? ?? ''),
            const SizedBox(height: 4),
            Row(mainAxisAlignment: MainAxisAlignment.end, children: [
              TextButton.icon(
                onPressed: () => _setToken(a),
                icon: const Icon(Icons.vpn_key, size: 16),
                label: Text(tokenFilled
                    ? context.tr('Rotate token')
                    : context.tr('Set token')),
              ),
              TextButton.icon(
                onPressed: () => _openEditor(a),
                icon: const Icon(Icons.edit, size: 16),
                label: Text(context.tr('Edit')),
              ),
              TextButton.icon(
                onPressed: () => _delete(a),
                icon: const Icon(Icons.delete_outline, size: 16, color: Colors.red),
                label: Text(context.tr('Delete'), style: const TextStyle(color: Colors.red)),
              ),
            ]),
          ],
        ),
      ),
    );
  }

  Widget _statusChip(bool filled) {
    final color = filled ? AppColors.success : AppColors.warning;
    final bg = filled ? AppColors.successSoft : AppColors.warningSoft;
    final label = filled
        ? context.tr('token set')
        : context.tr('no token');
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(color: bg, borderRadius: BorderRadius.circular(4)),
      child: Text(label,
          style: TextStyle(fontSize: 10, fontWeight: FontWeight.w600, color: color)),
    );
  }

  Widget _kv(String k, String v) {
    return Padding(
      padding: const EdgeInsets.only(top: 2),
      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
        SizedBox(
          width: 92,
          child: Text(k,
              style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
        ),
        Expanded(
          child: Text(v,
              style: const TextStyle(
                  fontSize: 11, fontFamily: 'monospace', color: AppColors.text),
              overflow: TextOverflow.ellipsis),
        ),
      ]),
    );
  }
}

// ═════════════════════════════════════════════════════════════════
// Editor dialog
// ═════════════════════════════════════════════════════════════════

class _AccountEditorDialog extends StatefulWidget {
  final ApiClient api;
  final Map<String, dynamic>? initial;
  const _AccountEditorDialog({required this.api, required this.initial});

  @override
  State<_AccountEditorDialog> createState() => _AccountEditorDialogState();
}

class _AccountEditorDialogState extends State<_AccountEditorDialog> {
  late final TextEditingController _name;
  late final TextEditingController _displayName;
  late final TextEditingController _configDir;
  late final TextEditingController _tokenPath;
  late final TextEditingController _description;
  late final TextEditingController _token;
  late bool _enabled;

  @override
  void initState() {
    super.initState();
    final s = widget.initial ?? const <String, dynamic>{};
    _name        = TextEditingController(text: s['name']        as String? ?? '');
    _displayName = TextEditingController(text: s['displayName'] as String? ?? '');
    _configDir   = TextEditingController(text: s['configDir']   as String? ?? '');
    _tokenPath   = TextEditingController(text: s['tokenPath']   as String? ?? '');
    _description = TextEditingController(text: s['description'] as String? ?? '');
    _token       = TextEditingController();
    _enabled     = s['enabled'] as bool? ?? true;
  }

  @override
  void dispose() {
    _name.dispose();
    _displayName.dispose();
    _configDir.dispose();
    _tokenPath.dispose();
    _description.dispose();
    _token.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (widget.initial == null) {
      final name = _name.text.trim();
      if (!RegExp(r'^[a-z0-9_-]{1,32}$').hasMatch(name)) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text(context.tr('Name must be 1-32 chars of [a-z0-9_-]'))));
        return;
      }
    }
    try {
      if (widget.initial == null) {
        await ApiClient.describeErrors(() => widget.api.claudeAccountCreate(
              name: _name.text.trim(),
              displayName: _displayName.text.trim(),
              configDir: _configDir.text.trim(),
              tokenPath: _tokenPath.text.trim(),
              token: _token.text.trim(),
              description: _description.text.trim(),
              enabled: _enabled,
            ));
      } else {
        await ApiClient.describeErrors(() => widget.api.claudeAccountUpdate(
              widget.initial!['id'] as String,
              {
                'displayName': _displayName.text.trim(),
                'configDir': _configDir.text.trim(),
                'tokenPath': _tokenPath.text.trim(),
                'description': _description.text.trim(),
                'enabled': _enabled,
              },
            ));
        final newToken = _token.text.trim();
        if (newToken.isNotEmpty) {
          await ApiClient.describeErrors(() =>
              widget.api.claudeAccountSetToken(widget.initial!['id'] as String, newToken));
        }
      }
      if (!mounted) return;
      Navigator.of(context, rootNavigator: true).pop(true);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final isNew = widget.initial == null;
    return AlertDialog(
      title: Text(isNew
          ? context.tr('New Claude account')
          : context.tr('Edit Claude account')),
      contentPadding: const EdgeInsets.fromLTRB(20, 12, 20, 0),
      content: SizedBox(
        width: 520,
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              if (isNew) _field(context.tr('Name'), _name, hint: 'kev',
                  help: context.tr('Lowercase alphanumeric, dash or underscore. Matches claude-<name> shortcut.')),
              _field(context.tr('Display Name'), _displayName,
                  hint: 'Kev (main)'),
              _field(context.tr('Config dir'), _configDir,
                  hint: '~/.claude-accounts/<name>',
                  help: context.tr('Leave empty to use the claude-acc default.')),
              _field(context.tr('Token path'), _tokenPath,
                  hint: '~/.claude-accounts/tokens/<name>.token',
                  help: context.tr('Leave empty to use the claude-acc default.')),
              _field(context.tr('Description'), _description),
              _field(
                isNew ? context.tr('OAuth token (optional)')
                      : context.tr('OAuth token (leave empty to keep)'),
                _token,
                obscure: true,
                hint: 'sk-ant-oat01-...',
                help: context.tr('Generated by `claude setup-token`. Written chmod 600 at the token path.'),
              ),
              const SizedBox(height: 8),
              SwitchListTile(
                value: _enabled,
                onChanged: (v) => setState(() => _enabled = v),
                title: Text(context.tr('Enabled')),
                contentPadding: EdgeInsets.zero,
                activeThumbColor: AppColors.accent,
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context, rootNavigator: true).pop(false),
          child: Text(context.tr('Cancel')),
        ),
        FilledButton(
          onPressed: _save,
          style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
          child: Text(context.tr('Save')),
        ),
      ],
    );
  }

  Widget _field(String label, TextEditingController c,
      {String? hint, String? help, bool obscure = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextField(
            controller: c,
            obscureText: obscure,
            decoration: InputDecoration(labelText: label, hintText: hint),
          ),
          if (help != null) ...[
            const SizedBox(height: 4),
            Text(help,
                style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
          ],
        ],
      ),
    );
  }
}

// ═════════════════════════════════════════════════════════════════
// Token-only dialog (quick rotate from a card action)
// ═════════════════════════════════════════════════════════════════

class _TokenDialog extends StatefulWidget {
  final String accountName;
  final bool tokenFilled;
  const _TokenDialog({required this.accountName, required this.tokenFilled});
  @override
  State<_TokenDialog> createState() => _TokenDialogState();
}

class _TokenDialogState extends State<_TokenDialog> {
  final _c = TextEditingController();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(widget.tokenFilled
          ? context.tr('Rotate OAuth token')
          : context.tr('Set OAuth token')),
      content: Column(mainAxisSize: MainAxisSize.min, children: [
        Text(
          context.tr('Paste the token for @name. Generated by `claude setup-token` on the host.')
              .replaceAll('@name', widget.accountName),
          style: const TextStyle(fontSize: 12, color: AppColors.textMuted),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _c,
          obscureText: true,
          decoration: const InputDecoration(hintText: 'sk-ant-oat01-...'),
        ),
      ]),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context, rootNavigator: true).pop(),
          child: Text(context.tr('Cancel')),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context, rootNavigator: true)
              .pop(_c.text.trim()),
          style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
          child: Text(context.tr('Save')),
        ),
      ],
    );
  }
}
